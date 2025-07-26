package service

import (
	"context"
	"fmt"
	"github.com/langowen/exchange/deploy/config"
	"github.com/langowen/exchange/internal/entities"
	"log/slog"
	"net/url"
	"time"
)

type Fetcher struct {
	storage    Storage
	httpClient HTTPClient
	config     *config.Config
}

func NewFetcher(storage Storage, client HTTPClient, cfg *config.Config) *Fetcher {
	return &Fetcher{
		storage:    storage,
		httpClient: client,
		config:     cfg,
	}
}

func InitFetcher(ctx context.Context, fetcher *Fetcher) error {
	rates := fetcher.config.Split("Rate")

	for _, rate := range rates {

		go func(currency string) {
			ticker := time.NewTicker(fetcher.config.Fetcher.TimeTickers)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					if err := fetcher.fetchRate(ctx, currency); err != nil {
						slog.Error("Ошибка при обновлении валютного курса", "rate", currency, "error", err)
					}
				case <-ctx.Done():
					slog.Info("Обновление валютных курсов остановлено", "error", ctx.Err())
				}
			}
		}(rate)
	}
	slog.Info("Запущен fetcher для обновления валютных курсов", "url", fetcher.config.Fetcher.URL, "rates", rates)
	return nil
}

// TODO реализовать функционал для добавления новой валюты из запроса если ее нет в БД
func (f *Fetcher) fetchRate(ctx context.Context, currency string) error {
	ctx, cancel := context.WithTimeout(ctx, f.config.Fetcher.Timeout)
	defer cancel()

	apiURL, err := f.getUrl(currency)
	if err != nil {
		return fmt.Errorf("не удалось получить URL для валюты %s: %w", currency, err)
	}

	rates, err := f.httpClient.Fetch(ctx, apiURL)
	if err != nil {
		return err
	}

	now := time.Now()

	var fiatValues []entities.FiatPrice
	for title, amount := range rates {
		fiatValues = append(fiatValues, entities.FiatPrice{
			Currency: title,
			Amount:   amount,
		})
	}

	rate, err := entities.NewRate(currency, fiatValues, now)
	if err != nil {
		return err
	}

	if err := f.storage.SaveRate(ctx, rate); err != nil {
		return err
	}

	return nil
}

func (f *Fetcher) getUrl(rate string) (string, error) {
	u, err := url.Parse(f.config.Fetcher.URL)
	if err != nil {
		return "", fmt.Errorf("невалидный base url: %w", err)
	}
	q := u.Query()
	q.Set("fsym", rate)
	q.Set("tsyms", f.config.Fetcher.ValueRate)
	u.RawQuery = q.Encode()
	apiURL := u.String()

	return apiURL, nil
}
