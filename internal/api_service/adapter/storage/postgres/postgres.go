package postgres

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/langowen/exchange/internal/api_service/service"
	"github.com/langowen/exchange/internal/entities"
	"github.com/pkg/errors"
	"log/slog"
	"time"
)

type Storage struct {
	db *pgxpool.Pool
}

func NewStorage(pool *pgxpool.Pool) *Storage {
	return &Storage{
		db: pool,
	}
}

func InitStorage(ctx context.Context, dsn string) (*Storage, error) {
	const op = "storage.postgres.New"

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, errors.Wrap(err, op)
	}
	poolConfig.MaxConns = 25
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = 10 * time.Minute
	poolConfig.MaxConnIdleTime = 5 * time.Minute

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, errors.Wrap(err, op)
	}

	if err = pool.Ping(ctx); err != nil {
		slog.Error("pgxpool ping failed", "error", err)
		pool.Close()
		return nil, errors.Wrap(err, op)
	}

	storageBD := NewStorage(pool)

	return storageBD, nil
}

func (s *Storage) GetRate(ctx context.Context, currency string, date time.Time, opts ...service.Option) (*entities.ExchangeRate, error) {
	const op = "storage.postgres.GetRates"

	options := &service.Options{}

	for _, opt := range opts {
		opt(options)
	}

	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	var query string
	switch options.FuncType {
	case service.Avg, service.Min, service.Max:
		query = fmt.Sprintf(`
             WITH AggRates AS (
                 SELECT c.code as crypto_code, f.code as fiat_code,
                     %s(er.amount) as amount,
                     MAX(er.timestamp) as max_timestamp,
                     c.id as crypto_id, f.id as fiat_id
                 FROM exchange_rates er
                 JOIN cryptocurrencies c ON er.crypto_id = c.id
			     JOIN fiat_currencies f ON er.fiat_id = f.id
			     WHERE c.code = $1 AND er.timestamp BETWEEN $2 AND $3 
			     GROUP BY c.code, f.code, c.id, f.id
             )
             SELECT crypto_code, fiat_code, amount, max_timestamp
             FROM AggRates
             ORDER BY fiat_code
		`, options.FuncType.String())
	default:
		query = `
            WITH RankedRates AS (
                SELECT c.code as crypto_code, f.code as fiat_code, 
                    er.amount, er.timestamp,
                    ROW_NUMBER() OVER (PARTITION BY c.code, f.code ORDER BY er.timestamp DESC) as rn
                FROM exchange_rates er
                JOIN cryptocurrencies c ON er.crypto_id = c.id
                JOIN fiat_currencies f ON er.fiat_id = f.id
                WHERE c.code = $1 AND er.timestamp BETWEEN $2 AND $3
            )
            SELECT crypto_code, fiat_code, amount, timestamp
            FROM RankedRates
            WHERE rn = 1
            ORDER BY fiat_code
        `
	}

	rows, err := s.db.Query(ctx, query, currency, startOfDay, endOfDay)
	if err != nil {
		return nil, errors.Wrap(err, op)
	}
	defer rows.Close()

	var fiatPrices []entities.FiatPrice
	var cryptoCode string
	var latestTimestamp time.Time

	for rows.Next() {
		var fiatCode string
		var amount float64
		var timestamp time.Time

		if err := rows.Scan(&cryptoCode, &fiatCode, &amount, &timestamp); err != nil {
			return nil, errors.Wrap(err, op)
		}

		fiatPrices = append(fiatPrices, entities.FiatPrice{
			Currency: fiatCode,
			Amount:   amount,
		})

		if timestamp.After(latestTimestamp) {
			latestTimestamp = timestamp
		}
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, op)
	}

	if len(fiatPrices) == 0 {
		return nil, fmt.Errorf("%s: no rates found for currency %s", op, currency)
	}

	rate, err := entities.NewRate(cryptoCode, fiatPrices, latestTimestamp)
	if err != nil {
		return nil, errors.Wrap(err, op)
	}

	return rate, nil
}

func (s *Storage) GetAllRates(ctx context.Context, date time.Time, opts ...service.Option) ([]entities.ExchangeRate, error) {
	const op = "storage.postgres.GetAllRates"

	options := &service.Options{}
	for _, opt := range opts {
		opt(options)
	}

	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	var query string
	var isAggregate bool

	switch options.FuncType {
	case service.Avg, service.Min, service.Max:
		isAggregate = true
		query = fmt.Sprintf(`
            SELECT 
                c.code as crypto_code,
                f.code as fiat_code,
                %s(er.amount) as amount,
                MAX(er.timestamp) as timestamp
            FROM exchange_rates er
            JOIN cryptocurrencies c ON er.crypto_id = c.id
            JOIN fiat_currencies f ON er.fiat_id = f.id
            WHERE er.timestamp BETWEEN $1 AND $2
            GROUP BY c.code, f.code
            ORDER BY crypto_code, fiat_code
        `, options.FuncType.String())
	default:
		isAggregate = false
		query = `
            SELECT 
                c.code as crypto_code,
                array_agg(f.code) as fiat_codes,
                array_agg(er.amount) as amounts,
                MAX(er.timestamp) as timestamp
            FROM (
                SELECT 
                    crypto_id,
                    fiat_id,
                    amount,
                    timestamp,
                    ROW_NUMBER() OVER (PARTITION BY crypto_id, fiat_id ORDER BY timestamp DESC) as rn
                FROM exchange_rates
                WHERE timestamp BETWEEN $1 AND $2
            ) er
            JOIN cryptocurrencies c ON er.crypto_id = c.id
            JOIN fiat_currencies f ON er.fiat_id = f.id
            WHERE er.rn = 1
            GROUP BY c.code
            ORDER BY crypto_code
        `
	}

	rows, err := s.db.Query(ctx, query, startOfDay, endOfDay)
	if err != nil {
		return nil, errors.Wrap(err, op)
	}
	defer rows.Close()

	var rates []entities.ExchangeRate

	for rows.Next() {
		if isAggregate {
			var cryptoCode, fiatCode string
			var amount float64
			var timestamp time.Time

			if err := rows.Scan(&cryptoCode, &fiatCode, &amount, &timestamp); err != nil {
				return nil, errors.Wrap(err, op)
			}

			var found bool
			for i := range rates {
				if rates[i].Title == cryptoCode {
					rates[i].FiatValues = append(rates[i].FiatValues, entities.FiatPrice{
						Currency: fiatCode,
						Amount:   amount,
					})
					if timestamp.After(rates[i].DateUpdate) {
						rates[i].DateUpdate = timestamp
					}
					found = true
					break
				}
			}

			if !found {
				rates = append(rates, entities.ExchangeRate{
					Title: cryptoCode,
					FiatValues: []entities.FiatPrice{{
						Currency: fiatCode,
						Amount:   amount,
					}},
					DateUpdate: timestamp,
				})
			}
		} else {
			var cryptoCode string
			var fiatCodes []string
			var amounts []float64
			var timestamp time.Time

			if err := rows.Scan(&cryptoCode, &fiatCodes, &amounts, &timestamp); err != nil {
				return nil, errors.Wrap(err, op)
			}

			fiatPrices := make([]entities.FiatPrice, len(fiatCodes))
			for i := range fiatCodes {
				fiatPrices[i] = entities.FiatPrice{
					Currency: fiatCodes[i],
					Amount:   amounts[i],
				}
			}

			rates = append(rates, entities.ExchangeRate{
				Title:      cryptoCode,
				FiatValues: fiatPrices,
				DateUpdate: timestamp,
			})
		}
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, op)
	}

	return rates, nil
}

func (s *Storage) ExistsRate(ctx context.Context, currency string) (bool, error) {
	const op = "storage.postgres.ExistsRate"

	query := `SELECT EXISTS(SELECT 1 FROM cryptocurrencies WHERE code = $1)`

	var exists bool
	err := s.db.QueryRow(ctx, query, currency).Scan(&exists)
	if err != nil {
		return false, errors.Wrap(err, op)
	}

	return exists, nil
}
