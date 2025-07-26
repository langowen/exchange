package service

import (
	"context"
	"github.com/langowen/exchange/internal/entities"
	"time"
)

type Storage interface {
	GetRates(ctx context.Context, currency string, date time.Time, opt ...Option) (*entities.ExchangeRate, error)
	GetAllRates(ctx context.Context, date time.Time, option string) ([]entities.ExchangeRate, error)
}
