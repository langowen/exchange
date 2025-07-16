package service

import (
	"context"
	"github.com/langowen/exchange/internal/apiService/storage"
	"github.com/langowen/exchange/internal/config"
)

type Storage interface {
	GetRate(ctx context.Context, currency string) (rate *storage.Rate, err error)
	GetAllRates(ctx context.Context) (rates []storage.Rate, err error)
}

type Service struct {
	storage Storage
	cfg     *config.Config
}

func NewService(storage Storage, cfg *config.Config) *Service {
	return &Service{
		storage: storage,
		cfg:     cfg,
	}
}

func (s *Service) FetchRate(ctx context.Context, currency string) (rate *storage.Rate, err error) {
	return s.storage.GetRate(ctx, currency)
}

func (s *Service) FetchAllRates(ctx context.Context) (rates []storage.Rate, err error) {
	return s.storage.GetAllRates(ctx)
}
