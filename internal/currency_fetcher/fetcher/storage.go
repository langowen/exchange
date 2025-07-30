package fetcher

import (
	"context"
	"github.com/langowen/exchange/internal/entities"
)

type Storage interface {
	SaveRates(ctx context.Context, rates []entities.ExchangeRate) error
	GetRates(ctx context.Context) ([]entities.ExchangeRate, error)
	SaveNewCurrency(ctx context.Context, currency string) error
}
