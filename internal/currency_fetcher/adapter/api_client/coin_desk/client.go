package coin_desk

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/langowen/exchange/internal/entities"
	"github.com/pkg/errors"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type HTTPClient struct {
	client *http.Client
}

func NewHTTPClient() *HTTPClient {
	return &HTTPClient{client: &http.Client{}}
}

func (c *HTTPClient) ApiClient(ctx context.Context, rates []entities.ExchangeRate, url string) ([]entities.ExchangeRate, error) {
	const op = "coin_desk.ApiClient"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, op)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, op)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			slog.Error(op, "error", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: bad status: %s", op, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, op)
	}

	var apiResponse map[string]map[string]float64
	if err = json.Unmarshal(body, &apiResponse); err != nil {
		return nil, errors.Wrap(err, op)
	}

	for i, cryptoRate := range rates {
		cryptoData, exists := apiResponse[cryptoRate.Title]
		if !exists {
			return nil, fmt.Errorf("%s: dont found rate for %s", op, cryptoRate.Title)
		}

		for j, fiat := range cryptoRate.FiatValues {
			if amount, ok := cryptoData[fiat.Currency]; ok {
				rates[i].FiatValues[j].Amount = amount
			}
		}

		rates[i].DateUpdate = time.Now()
	}

	return rates, nil
}
