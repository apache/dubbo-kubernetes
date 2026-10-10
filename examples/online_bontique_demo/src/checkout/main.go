package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
)

type Address struct {
	StreetAddress string `json:"streetAddress"`
	City          string `json:"city"`
	State         string `json:"state"`
	Country       string `json:"country"`
	ZipCode       int    `json:"zipCode"`
}

type CreditCardInfo struct {
	CreditCardNumber          string `json:"creditCardNumber"`
	CreditCardCvv             int    `json:"creditCardCvv"`
	CreditCardExpirationYear  int    `json:"creditCardExpirationYear"`
	CreditCardExpirationMonth int    `json:"creditCardExpirationMonth"`
}

type CartItem struct {
	ProductId string `json:"productId"`
	Quantity  int    `json:"quantity"`
}

type Cart struct {
	UserId string     `json:"userId"`
	Items  []CartItem `json:"items"`
}

type Product struct {
	Id          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Picture     string   `json:"picture"`
	PriceUsd    Money    `json:"priceUsd"`
	Categories  []string `json:"categories"`
}

type OrderItem struct {
	Item CartItem `json:"item"`
	Cost Money    `json:"cost"`
}

type OrderResult struct {
	OrderId            string      `json:"orderId"`
	ShippingTrackingId string      `json:"shippingTrackingId"`
	ShippingCost       Money       `json:"shippingCost"`
	ShippingAddress    Address     `json:"shippingAddress"`
	Items              []OrderItem `json:"items"`
}

type PlaceOrderRequest struct {
	UserId       string         `json:"userId"`
	UserCurrency string         `json:"userCurrency"`
	Address      Address        `json:"address"`
	Email        string         `json:"email"`
	CreditCard   CreditCardInfo `json:"creditCard"`
}

type PlaceOrderResponse struct {
	Order OrderResult `json:"order"`
}

type Config struct {
	Port               string
	CartURL            string
	ProductCatalogsURL string
	CurrencyURL        string
	ShippingURL        string
	PaymentURL         string
	EmailURL           string
}

var (
	httpClient = &http.Client{Timeout: 5 * time.Second}
	cfg        Config
)

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getEnvAny(keys []string, fallback string) string {
	for _, k := range keys {
		if val := os.Getenv(k); val != "" {
			return val
		}
	}
	return fallback
}

func initConfig() {
	cfg = Config{
		Port:               getEnv("PORT", "9003"),
		CartURL:            getEnvAny([]string{"CART_URL", "CART_SERVICE_URL"}, "http://127.0.0.1:9002"),
		ProductCatalogsURL: getEnvAny([]string{"PRODUCT_CATALOGS_URL", "PRODUCT_CATALOGS_SERVICE_URL"}, "http://127.0.0.1:9007"),
		CurrencyURL:        getEnvAny([]string{"CURRENCY_URL", "CURRENCY_SERVICE_URL"}, "http://127.0.0.1:9004"),
		ShippingURL:        getEnvAny([]string{"SHIPPING_URL", "SHIPPING_SERVICE_URL"}, "http://127.0.0.1:9009"),
		PaymentURL:         getEnvAny([]string{"PAYMENT_URL", "PAYMENT_SERVICE_URL"}, "http://127.0.0.1:9006"),
		EmailURL:           getEnvAny([]string{"EMAIL_URL", "EMAIL_SERVICE_URL"}, "http://127.0.0.1:9005"),
	}
}

func postJSON(url string, reqBody interface{}, respBody interface{}) error {
	b, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	resp, err := httpClient.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	if respBody != nil {
		return json.NewDecoder(resp.Body).Decode(respBody)
	}
	return nil
}

func getJSON(url string, respBody interface{}) error {
	resp, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return json.NewDecoder(resp.Body).Decode(respBody)
}

func handleCheckout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PlaceOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// 1. Get Cart
	var cart Cart
	cartURL := fmt.Sprintf("%s/cart?userId=%s", cfg.CartURL, req.UserId)
	if err := getJSON(cartURL, &cart); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("failed to get cart: %v", err)})
		return
	}

	// 2. Prep order items
	orderItems := make([]OrderItem, 0, len(cart.Items))
	for _, item := range cart.Items {
		var prod Product
		prodURL := fmt.Sprintf("%s/products/%s", cfg.ProductCatalogsURL, item.ProductId)
		if err := getJSON(prodURL, &prod); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("failed to get product %s: %v", item.ProductId, err)})
			return
		}

		convReq := map[string]interface{}{
			"from":   prod.PriceUsd,
			"toCode": req.UserCurrency,
		}
		var convertedPrice Money
		if err := postJSON(fmt.Sprintf("%s/convert", cfg.CurrencyURL), convReq, &convertedPrice); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("currency conversion failed: %v", err)})
			return
		}

		orderItems = append(orderItems, OrderItem{
			Item: item,
			Cost: convertedPrice,
		})
	}

	// 3. Shipping quote
	quoteReq := map[string]interface{}{
		"address": req.Address,
		"items":   cart.Items,
	}
	var quoteResp struct {
		CostUsd Money `json:"costUsd"`
	}
	if err := postJSON(fmt.Sprintf("%s/quote", cfg.ShippingURL), quoteReq, &quoteResp); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("shipping quote failed: %v", err)})
		return
	}

	convShippingReq := map[string]interface{}{
		"from":   quoteResp.CostUsd,
		"toCode": req.UserCurrency,
	}
	var shippingPrice Money
	if err := postJSON(fmt.Sprintf("%s/convert", cfg.CurrencyURL), convShippingReq, &shippingPrice); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("shipping conversion failed: %v", err)})
		return
	}

	// 4. Calculate total
	total := Money{CurrencyCode: req.UserCurrency, Units: 0, Nanos: 0}
	var err error
	total, err = Sum(total, shippingPrice)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	for _, oi := range orderItems {
		itemTotal, err := MultiplySlow(oi.Cost, oi.Item.Quantity)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		total, err = Sum(total, itemTotal)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
	}

	// 5. Payment charge
	chargeReq := map[string]interface{}{
		"amount":     total,
		"creditCard": req.CreditCard,
	}
	var chargeResp struct {
		TransactionId string `json:"transactionId"`
	}
	if err := postJSON(fmt.Sprintf("%s/charge", cfg.PaymentURL), chargeReq, &chargeResp); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("payment charge failed: %v", err)})
		return
	}

	// 6. Ship order
	shipReq := map[string]interface{}{
		"address": req.Address,
		"items":   cart.Items,
	}
	var shipResp struct {
		TrackingId string `json:"trackingId"`
	}
	if err := postJSON(fmt.Sprintf("%s/ship", cfg.ShippingURL), shipReq, &shipResp); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("ship order failed: %v", err)})
		return
	}

	// 7. Empty cart
	emptyReq := map[string]interface{}{"userId": req.UserId}
	_ = postJSON(fmt.Sprintf("%s/cart/empty", cfg.CartURL), emptyReq, nil)

	// 8. Order result & confirmation
	orderId := uuid.New().String()
	orderResult := OrderResult{
		OrderId:            orderId,
		ShippingTrackingId: shipResp.TrackingId,
		ShippingCost:       shippingPrice,
		ShippingAddress:    req.Address,
		Items:              orderItems,
	}

	emailReq := map[string]interface{}{
		"email": req.Email,
		"order": orderId,
	}
	_ = postJSON(fmt.Sprintf("%s/send-order-confirmation", cfg.EmailURL), emailReq, nil)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(PlaceOrderResponse{Order: orderResult})
}

func main() {
	initConfig()
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/checkout", handleCheckout)

	addr := fmt.Sprintf("127.0.0.1:%s", cfg.Port)
	log.Printf("checkout listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
