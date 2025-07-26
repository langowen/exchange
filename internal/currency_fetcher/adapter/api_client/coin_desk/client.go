package coin_desk

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type HTTPClient struct {
	client *http.Client
}

func NewHTTPClient() *HTTPClient {
	return &HTTPClient{client: &http.Client{}}
}

func (c *HTTPClient) ApiClient(ctx context.Context, url string) (map[string]float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request error: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("api_client get error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body error: %w", err)
	}

	var result map[string]float64
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("json unmarshal error: %w", err)
	}
	return result, nil
}

type T struct {
	BTC struct {
		USD float64 `json:"USD"`
		JPY float64 `json:"JPY"`
		EUR float64 `json:"EUR"`
	} `json:"BTC"`
	ETH struct {
		USD float64 `json:"USD"`
		JPY float64 `json:"JPY"`
		EUR float64 `json:"EUR"`
	} `json:"ETH"`
}
