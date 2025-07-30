package service

import (
	"context"
	"github.com/langowen/exchange/internal/entities"
	"time"
)

type Storage interface {
	GetRate(ctx context.Context, currency string, date time.Time, opts ...Option) (*entities.ExchangeRate, error)
	GetAllRates(ctx context.Context, date time.Time, opts ...Option) ([]entities.ExchangeRate, error)
	ExistsRate(ctx context.Context, currency string) (bool, error)
}
