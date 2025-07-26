package service

import (
	"context"
	"github.com/langowen/exchange/internal/entities"
)

type Storage interface {
	SaveRate(ctx context.Context, rate *entities.ExchangeRate) error
}
