package prometheusclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type PrometheusResult struct {
	Metric map[string]string `json:"metric"`
	Value  []interface{}     `json:"value"`
}

type PrometheusClient struct {
	URL    string
	Client *http.Client
}

func NewPrometheusClient(url string) *PrometheusClient {
	return &PrometheusClient{
		URL:    url,
		Client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (pc *PrometheusClient) Query(query string) ([]PrometheusResult, error) {
	return pc.QueryWithContext(context.Background(), query)
}

func (pc *PrometheusClient) QueryWithContext(ctx context.Context, query string) ([]PrometheusResult, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/api/v1/query", pc.URL), nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("query", query)
	req.URL.RawQuery = q.Encode()

	resp, err := pc.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string             `json:"resultType"`
			Result     []PrometheusResult `json:"result"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("query failed with status: %s", result.Status)
	}

	return result.Data.Result, nil
}
