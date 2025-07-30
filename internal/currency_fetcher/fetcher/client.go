package fetcher

import (
	"context"
	"github.com/langowen/exchange/internal/entities"
)

type HTTPClient interface {
	ApiClient(ctx context.Context, rates []entities.ExchangeRate, url string) ([]entities.ExchangeRate, error)
}
