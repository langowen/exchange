package public

import (
	"context"
	"github.com/langowen/exchange/internal/entities"
)

type Service interface {
	GetRate(ctx context.Context, currency string, date string, options string) (rate *entities.ExchangeRate, err error)
	GetAllRates(ctx context.Context, date string, options string) (rates []entities.ExchangeRate, err error)
}
