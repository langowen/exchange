package service

import (
	"context"
	"github.com/langowen/exchange/deploy/config"
	"github.com/langowen/exchange/internal/entities"
	"time"
)

type Service struct {
	storage Storage
	cfg     *config.Config
}

func NewService(storage Storage, cfg *config.Config) (*Service, error) {
	return &Service{
		storage: storage,
		cfg:     cfg,
	}, nil
}

// TODO переделать методы на OPtions и дописать на min, max
func (s *Service) FetchRate(ctx context.Context, currency string, date string, option string) (*entities.ExchangeRate, error) {
	if option == "" {
		option = "last"
	}

	dateTime := time.Now()
	if date != "" {
		parsedTime, err := time.Parse("2006-01-02", date)
		if err == nil {
			dateTime = parsedTime
		}
	}

	return s.storage.GetRates(ctx, currency, dateTime, option)
}

func (s *Service) FetchAllRates(ctx context.Context, date string, option string) ([]entities.ExchangeRate, error) {
	if option == "" {
		option = "last"
	}

	dateTime := time.Now()
	if date != "" {
		parsedTime, err := time.Parse("2006-01-02", date)
		if err == nil {
			dateTime = parsedTime
		}
	}

	return s.storage.GetAllRates(ctx, dateTime, option)
}

type AggFunc int

const (
	_ AggFunc = iota
	Avg
	Min
	Max
)

type Options struct {
	FuncType AggFunc
}

type Option func(o *Options)

func (a AggFunc) String() string {
	return [...]string{"", "avg", "min", "max"}[a]
}

func WithAggFunc() Option {
	return func(o *Options) {
		o.FuncType = Avg
	}
}

func (s *Service) FetchRateWithAvg(ctx context.Context, currency string, date string) (*entities.ExchangeRate, error) {
	s.storage.GetAllRates(ctx, date, WithAggFunc())
}
