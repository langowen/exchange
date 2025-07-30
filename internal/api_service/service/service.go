package service

import (
	"context"
	"fmt"
	"github.com/langowen/exchange/internal/entities"
	"github.com/pkg/errors"
	"time"
)

type Service struct {
	storage Storage
	redis   RedisStorage
}

func NewService(storage Storage, redis RedisStorage) (*Service, error) {
	return &Service{
		storage: storage,
		redis:   redis,
	}, nil
}

func (s *Service) GetRate(ctx context.Context, currency string, date string, option string) (*entities.ExchangeRate, error) {
	const op = "service.GetRate"

	dateTime := time.Now()
	if date != "" {
		parsedTime, err := time.Parse("2006-01-02", date)
		if err != nil {
			return nil, errors.Wrap(err, op)
		}
		dateTime = parsedTime
	}

	exists, err := s.storage.ExistsRate(ctx, currency)
	if err != nil {
		return nil, errors.Wrap(err, op)
	}
	if !exists {
		if err = s.getNewRate(ctx, currency); err != nil {
			return nil, errors.Wrap(err, op)
		}
	}

	switch option {
	case "avg":
		return s.GetRateWithAvg(ctx, currency, dateTime)
	case "min":
		return s.GetRateWithMin(ctx, currency, dateTime)
	case "max":
		return s.GetRateWithMax(ctx, currency, dateTime)
	default:
		return s.storage.GetRate(ctx, currency, dateTime)
	}

}

func (s *Service) getNewRate(ctx context.Context, currency string) error {
	const op = "service.GetNewRate"

	err := s.redis.PublishNew(ctx, currency)
	if err != nil {
		return errors.Wrap(err, op)
	}

	ctxListen, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	res, err := s.redis.ListenUdp(ctxListen)
	if err != nil {
		if errors.Is(err, entities.ErrRedisTimeout) {
			return err
		}
		return errors.Wrap(err, op)
	}
	if res != currency {
		return fmt.Errorf("invalid currency is redis format: %s != %s", res, currency)
	}

	return nil
}

func (s *Service) GetAllRates(ctx context.Context, date string, option string) ([]entities.ExchangeRate, error) {
	const op = "service.GetAllRates"

	dateTime := time.Now()
	if date != "" {
		parsedTime, err := time.Parse("2006-01-02", date)
		if err != nil {
			return nil, errors.Wrap(err, op)
		}
		dateTime = parsedTime
	}

	switch option {
	case "avg":
		return s.GetAllRatesWithAvg(ctx, dateTime)
	case "min":
		return s.GetAllRatesWithMin(ctx, dateTime)
	case "max":
		return s.GetAllRatesWithMax(ctx, dateTime)
	default:
		return s.storage.GetAllRates(ctx, dateTime)
	}

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

func WithAggFunc(f AggFunc) Option {
	return func(o *Options) {
		o.FuncType = f
	}
}

func (s *Service) GetRateWithAvg(ctx context.Context, currency string, date time.Time) (*entities.ExchangeRate, error) {
	return s.storage.GetRate(ctx, currency, date, WithAggFunc(Avg))
}

func (s *Service) GetRateWithMin(ctx context.Context, currency string, date time.Time) (*entities.ExchangeRate, error) {
	return s.storage.GetRate(ctx, currency, date, WithAggFunc(Min))
}

func (s *Service) GetRateWithMax(ctx context.Context, currency string, date time.Time) (*entities.ExchangeRate, error) {
	return s.storage.GetRate(ctx, currency, date, WithAggFunc(Max))
}

func (s *Service) GetAllRatesWithAvg(ctx context.Context, date time.Time) ([]entities.ExchangeRate, error) {
	return s.storage.GetAllRates(ctx, date, WithAggFunc(Avg))
}

func (s *Service) GetAllRatesWithMin(ctx context.Context, date time.Time) ([]entities.ExchangeRate, error) {
	return s.storage.GetAllRates(ctx, date, WithAggFunc(Min))
}

func (s *Service) GetAllRatesWithMax(ctx context.Context, date time.Time) ([]entities.ExchangeRate, error) {
	return s.storage.GetAllRates(ctx, date, WithAggFunc(Max))
}
