package fetcher

import (
	"context"
	"fmt"
	"github.com/langowen/exchange/deploy/config"
	"github.com/langowen/exchange/internal/entities"
	"github.com/pkg/errors"
	"log/slog"
	"net/url"
	"strings"
	"time"
)

type Fetcher struct {
	storage    Storage
	httpClient HTTPClient
	redis      RedisStorage
	config     *config.Config
}

func NewFetcher(storage Storage, client HTTPClient, redis RedisStorage, cfg *config.Config) *Fetcher {
	return &Fetcher{
		storage:    storage,
		httpClient: client,
		redis:      redis,
		config:     cfg,
	}
}

func (f *Fetcher) StartFetcher(ctx context.Context) error {
	const op = "fetcher.StartFetcher"

	ticker := time.NewTicker(f.config.Fetcher.TimeTickers)
	defer ticker.Stop()

	go f.getNewRate(ctx)

	for {
		select {
		case <-ticker.C:
			rates, err := f.storage.GetRates(ctx)
			if err != nil {
				return errors.Wrap(err, op)
			}

			if err := f.fetchRate(ctx, rates); err != nil {
				slog.Error("Ошибка при обновлении валютного курса", "rate", rates, "error", err)
			}

		case <-ctx.Done():
			return errors.Wrap(ctx.Err(), op)
		}
	}
}

func (f *Fetcher) getNewRate(ctx context.Context) {
	const op = "fetcher.getNewRate"

	for {
		select {
		case <-ctx.Done():
			slog.Error("Обновление валютных курсов остановлено", "op", op, "error", ctx.Err())
			return
		default:
			currency, err := f.redis.ListenNew(ctx)
			if err != nil {
				slog.Error(op, "error", err)
			}

			if err = f.storage.SaveNewCurrency(ctx, currency); err != nil {
				slog.Error(op, "error", err)
			}

			rates, err := f.storage.GetRates(ctx)
			if err != nil {
				slog.Error(op, "error", err)
			}

			if err = f.fetchRate(ctx, rates); err != nil {
				slog.Error(op, "error", err)
			}

			err = f.redis.PublishUpd(ctx, currency)
			if err != nil {
				slog.Error(op, "error", err)
			}
		}
	}
}

func (f *Fetcher) fetchRate(ctx context.Context, rates []entities.ExchangeRate) error {
	const op = "fetcher.fetchRate"

	ctx, cancel := context.WithTimeout(ctx, f.config.Fetcher.Timeout)
	defer cancel()

	apiURL, err := f.getUrl(rates)
	if err != nil {
		return errors.Wrap(err, op)
	}

	result, err := f.httpClient.ApiClient(ctx, rates, apiURL)
	if err != nil {
		return errors.Wrap(err, op)
	}

	if err := f.storage.SaveRates(ctx, result); err != nil {
		return errors.Wrap(err, op)
	}

	return nil
}

func (f *Fetcher) getUrl(rates []entities.ExchangeRate) (string, error) {
	const op = "fetcher.getUrl"

	if len(rates) == 0 {
		return "", fmt.Errorf("%s: пустой список валют", op)
	}

	fsyms := make([]string, len(rates))
	for i, rate := range rates {
		fsyms[i] = rate.Title
	}

	tsyms := make([]string, len(rates[0].FiatValues))
	for i, fiat := range rates[0].FiatValues {
		tsyms[i] = fiat.Currency
	}

	u, err := url.Parse(f.config.Fetcher.URL)
	if err != nil {
		return "", errors.Wrap(err, op)
	}

	q := u.Query()
	q.Set("fsyms", strings.Join(fsyms, ","))
	q.Set("tsyms", strings.Join(tsyms, ","))
	u.RawQuery = q.Encode()

	return u.String(), nil
}
