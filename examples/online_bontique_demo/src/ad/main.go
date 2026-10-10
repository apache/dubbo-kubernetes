package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"
)

type Ad struct {
	RedirectUrl string `json:"redirectUrl"`
	Text        string `json:"text"`
}

type AdRequest struct {
	ContextKeys []string `json:"contextKeys"`
}

type AdResponse struct {
	Ads []Ad `json:"ads"`
}

var adsMap = map[string][]Ad{
	"clothing":    {{"/product/66VCHSJNUP", "Tank top for sale. 20% off."}},
	"accessories": {{"/product/1YMWWN1N4O", "Watch for sale. Buy one, get second kit for free"}},
	"footwear":    {{"/product/L9ECAV7KIM", "Loafers for sale. Buy one, get second one for free"}},
	"hair":        {{"/product/2ZYFJ3GM2N", "Hairdryer for sale. 50% off."}},
	"decor":       {{"/product/0PUK6V6EV0", "Candle holder for sale. 30% off."}},
	"kitchen": {
		{"/product/9SIQT8TOJO", "Bamboo glass jar for sale. 10% off."},
		{"/product/6E92ZMYYFZ", "Mug for sale. Buy two, get third one for free"},
	},
}

func getAllAds() []Ad {
	var all []Ad
	for _, ads := range adsMap {
		all = append(all, ads...)
	}
	return all
}

func getRandomAds(count int) []Ad {
	all := getAllAds()
	if len(all) <= count {
		return all
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	indices := r.Perm(len(all))
	result := make([]Ad, count)
	for i := 0; i < count; i++ {
		result[i] = all[indices[i]]
	}
	return result
}

func GetAds(req AdRequest) AdResponse {
	var res []Ad
	if len(req.ContextKeys) > 0 {
		for _, cat := range req.ContextKeys {
			if ads, ok := adsMap[cat]; ok {
				res = append(res, ads...)
			}
		}
		if len(res) == 0 {
			res = getRandomAds(2)
		}
	} else {
		res = getRandomAds(2)
	}
	return AdResponse{Ads: res}
}

func handleAds(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	var req AdRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	resp := GetAds(req)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9001"
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/ads", handleAds)

	addr := fmt.Sprintf("127.0.0.1:%s", port)
	log.Printf("ad listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
