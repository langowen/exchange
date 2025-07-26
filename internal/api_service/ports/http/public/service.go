package public

import (
	"context"
	"github.com/langowen/exchange/internal/entities"
)

type Service interface {
	FetchRate(ctx context.Context, currency string, date string, options string) (rate *entities.ExchangeRate, err error)
	FetchAllRates(ctx context.Context, date string, options string) (rates []entities.ExchangeRate, err error)
}
