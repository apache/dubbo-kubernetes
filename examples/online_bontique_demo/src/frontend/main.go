package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	Port               string
	StaticDir          string
	AdURL              string
	CartURL            string
	CheckoutURL        string
	CurrencyURL        string
	EmailURL           string
	PaymentURL         string
	ProductCatalogsURL string
	RecommendURL       string
	ShippingURL        string
}

var (
	cfg        Config
	httpClient = &http.Client{Timeout: 5 * time.Second}
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
		Port:               getEnv("PORT", "8080"),
		StaticDir:          getEnv("STATIC_DIR", "../../web/dist"),
		AdURL:              getEnvAny([]string{"AD_URL", "AD_SERVICE_URL"}, "http://127.0.0.1:9001"),
		CartURL:            getEnvAny([]string{"CART_URL", "CART_SERVICE_URL"}, "http://127.0.0.1:9002"),
		CheckoutURL:        getEnvAny([]string{"CHECKOUT_URL", "CHECKOUT_SERVICE_URL"}, "http://127.0.0.1:9003"),
		CurrencyURL:        getEnvAny([]string{"CURRENCY_URL", "CURRENCY_SERVICE_URL"}, "http://127.0.0.1:9004"),
		EmailURL:           getEnvAny([]string{"EMAIL_URL", "EMAIL_SERVICE_URL"}, "http://127.0.0.1:9005"),
		PaymentURL:         getEnvAny([]string{"PAYMENT_URL", "PAYMENT_SERVICE_URL"}, "http://127.0.0.1:9006"),
		ProductCatalogsURL: getEnvAny([]string{"PRODUCT_CATALOGS_URL", "PRODUCT_CATALOGS_SERVICE_URL"}, "http://127.0.0.1:9007"),
		RecommendURL:       getEnvAny([]string{"RECOMMENDATION_URL", "RECOMMENDATION_SERVICE_URL"}, "http://127.0.0.1:9008"),
		ShippingURL:        getEnvAny([]string{"SHIPPING_URL", "SHIPPING_SERVICE_URL"}, "http://127.0.0.1:9009"),
	}
}

func proxyRequest(targetURL string, w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	req, err := http.NewRequest(r.Method, targetURL, bytes.NewReader(body))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	for k, vv := range r.Header {
		for _, v := range vv {
			req.Header.Add(k, v)
		}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("downstream failure: %v", err)})
		return
	}
	defer resp.Body.Close()

	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func handleAPI(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	query := ""
	if r.URL.RawQuery != "" {
		query = "?" + r.URL.RawQuery
	}

	switch {
	case strings.HasPrefix(path, "/api/products"):
		sub := strings.TrimPrefix(path, "/api/products")
		proxyRequest(fmt.Sprintf("%s/products%s%s", cfg.ProductCatalogsURL, sub, query), w, r)
	case strings.HasPrefix(path, "/api/cart"):
		sub := strings.TrimPrefix(path, "/api/cart")
		proxyRequest(fmt.Sprintf("%s/cart%s%s", cfg.CartURL, sub, query), w, r)
	case strings.HasPrefix(path, "/api/checkout"):
		proxyRequest(fmt.Sprintf("%s/checkout%s", cfg.CheckoutURL, query), w, r)
	case strings.HasPrefix(path, "/api/currencies/convert"):
		proxyRequest(fmt.Sprintf("%s/convert%s", cfg.CurrencyURL, query), w, r)
	case strings.HasPrefix(path, "/api/currencies"):
		proxyRequest(fmt.Sprintf("%s/currencies%s", cfg.CurrencyURL, query), w, r)
	case strings.HasPrefix(path, "/api/shipping/quote"):
		proxyRequest(fmt.Sprintf("%s/quote%s", cfg.ShippingURL, query), w, r)
	case strings.HasPrefix(path, "/api/shipping/ship"):
		proxyRequest(fmt.Sprintf("%s/ship%s", cfg.ShippingURL, query), w, r)
	case strings.HasPrefix(path, "/api/payment/charge"):
		proxyRequest(fmt.Sprintf("%s/charge%s", cfg.PaymentURL, query), w, r)
	case strings.HasPrefix(path, "/api/email/confirmation"):
		proxyRequest(fmt.Sprintf("%s/send-order-confirmation%s", cfg.EmailURL, query), w, r)
	case strings.HasPrefix(path, "/api/recommendations"):
		proxyRequest(fmt.Sprintf("%s/recommendations%s", cfg.RecommendURL, query), w, r)
	case strings.HasPrefix(path, "/api/ads"):
		proxyRequest(fmt.Sprintf("%s/ads%s", cfg.AdURL, query), w, r)
	default:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "API route not found"})
	}
}

func handleStatic(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		handleAPI(w, r)
		return
	}

	cleanPath := filepath.Clean(r.URL.Path)
	fullPath := filepath.Join(cfg.StaticDir, cleanPath)

	fi, err := os.Stat(fullPath)
	if err == nil && !fi.IsDir() {
		http.ServeFile(w, r, fullPath)
		return
	}

	// SPA fallback
	indexPath := filepath.Join(cfg.StaticDir, "index.html")
	if _, err := os.Stat(indexPath); err == nil {
		http.ServeFile(w, r, indexPath)
		return
	}

	// If no dist yet, return simple message
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Frontend gateway active. Run npm run build in web/ to serve assets."))
}

func main() {
	initConfig()
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/api/", handleAPI)
	mux.HandleFunc("/", handleStatic)

	addr := fmt.Sprintf("127.0.0.1:%s", cfg.Port)
	log.Printf("frontend gateway listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("gateway error: %v", err)
	}
}
