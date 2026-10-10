package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"
)

type ListRecommendationsRequest struct {
	UserId     string   `json:"userId"`
	ProductIds []string `json:"productIds"`
}

type ListRecommendationsResponse struct {
	ProductIds []string `json:"productIds"`
}

type Product struct {
	Id string `json:"id"`
}

type ListProductsResponse struct {
	Products []Product `json:"products"`
}

var (
	httpClient         = &http.Client{Timeout: 5 * time.Second}
	productCatalogsURL string
)

func sample(source []string, count int) []string {
	if len(source) <= count {
		return source
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	indices := r.Perm(len(source))
	result := make([]string, count)
	for i := 0; i < count; i++ {
		result[i] = source[indices[i]]
	}
	return result
}

func handleRecommendations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ListRecommendationsRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	resp, err := httpClient.Get(fmt.Sprintf("%s/products", productCatalogsURL))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("downstream catalog error: %v", err)})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("downstream catalog returned %d: %s", resp.StatusCode, string(body))})
		return
	}

	var prodList ListProductsResponse
	if err := json.NewDecoder(resp.Body).Decode(&prodList); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	exclude := make(map[string]bool)
	for _, id := range req.ProductIds {
		exclude[id] = true
	}

	var filtered []string
	for _, p := range prodList.Products {
		if !exclude[p.Id] {
			filtered = append(filtered, p.Id)
		}
	}

	recommended := sample(filtered, 4)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(ListRecommendationsResponse{ProductIds: recommended})
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9008"
	}
	productCatalogsURL = os.Getenv("PRODUCT_CATALOGS_SERVICE_URL")
	if productCatalogsURL == "" {
		productCatalogsURL = "http://127.0.0.1:9007"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/recommendations", handleRecommendations)

	addr := fmt.Sprintf("127.0.0.1:%s", port)
	log.Printf("recommendation listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
