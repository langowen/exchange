package postgres

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/langowen/exchange/deploy/config"
	"github.com/langowen/exchange/internal/api_service/service"
	"github.com/langowen/exchange/internal/entities"
	"log/slog"
	"time"
)

type Storage struct {
	db  *pgxpool.Pool
	cfg *config.Config
}

func NewStorage(pool *pgxpool.Pool, cfg *config.Config) *Storage {
	return &Storage{
		db:  pool,
		cfg: cfg,
	}
}

func New(ctx context.Context, cfg *config.Config) (*Storage, error) {
	const op = "storage.postgres.New"

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s search_path=%s",
		cfg.Storage.Host,
		cfg.Storage.Port,
		cfg.Storage.User,
		cfg.Storage.Password,
		cfg.Storage.DBName,
		cfg.Storage.SSLMode,
		cfg.Storage.Schema,
	)

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("%s: parse config failed: %w", op, err)
	}
	poolConfig.MaxConns = 25
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = 10 * time.Minute
	poolConfig.MaxConnIdleTime = 5 * time.Minute

	ctx, cancel := context.WithTimeout(ctx, cfg.Storage.Timeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		slog.Error("pgxpool connect failed", "error", err)
		return nil, fmt.Errorf("%s: pgxpool connect failed: %w", op, err)
	}

	if err := pool.Ping(ctx); err != nil {
		slog.Error("pgxpool ping failed", "error", err)
		pool.Close()
		return nil, fmt.Errorf("%s: ping failed: %w", op, err)
	}

	storageBD := NewStorage(pool, cfg)

	slog.Info("PostgresSQL storage initialized successfully")
	return storageBD, nil
}

func (s *Storage) GetRates(ctx context.Context, currency string, date time.Time, opt ...service.Option) (*entities.ExchangeRate, error) {
	const op = "storage.postgres.GetRates"

	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	var query string
	switch option {
	case "max":
		query = `
			WITH MaxRates AS (
				SELECT c.code as crypto_code, f.code as fiat_code,
					MAX(er.amount) as amount,
					c.id as crypto_id, f.id as fiat_id
				FROM exchange_rates er
				JOIN cryptocurrencies c ON er.crypto_id = c.id
				JOIN fiat_currencies f ON er.fiat_id = f.id
				WHERE c.code = $1 AND er.timestamp BETWEEN $2 AND $3
				GROUP BY c.code, f.code, c.id, f.id
			),
			MaxRatesWithTimestamp AS (
				SELECT mr.crypto_code, mr.fiat_code, mr.amount, 
					(SELECT timestamp 
					FROM exchange_rates 
					WHERE crypto_id = mr.crypto_id 
					AND fiat_id = mr.fiat_id 
					AND amount = mr.amount 
					AND timestamp BETWEEN $2 AND $3 
					ORDER BY timestamp DESC LIMIT 1) as timestamp
				FROM MaxRates mr
			)
			SELECT crypto_code, fiat_code, amount, timestamp
			FROM MaxRatesWithTimestamp
			ORDER BY fiat_code
		`
	case "min":
		query = `
			WITH MinRates AS (
				SELECT c.code as crypto_code, f.code as fiat_code,
					MIN(er.amount) as amount,
					c.id as crypto_id, f.id as fiat_id
				FROM exchange_rates er
				JOIN cryptocurrencies c ON er.crypto_id = c.id
				JOIN fiat_currencies f ON er.fiat_id = f.id
				WHERE c.code = $1 AND er.timestamp BETWEEN $2 AND $3
				GROUP BY c.code, f.code, c.id, f.id
			),
			MinRatesWithTimestamp AS (
				SELECT mr.crypto_code, mr.fiat_code, mr.amount, 
					(SELECT timestamp 
					FROM exchange_rates 
					WHERE crypto_id = mr.crypto_id 
					AND fiat_id = mr.fiat_id 
					AND amount = mr.amount 
					AND timestamp BETWEEN $2 AND $3 
					ORDER BY timestamp DESC LIMIT 1) as timestamp
				FROM MinRates mr
			)
			SELECT crypto_code, fiat_code, amount, timestamp
			FROM MinRatesWithTimestamp
			ORDER BY fiat_code
		`
	case "avg":
		query = `
			WITH AvgRates AS (
				SELECT c.code as crypto_code, f.code as fiat_code, 
				    AVG(er.amount) as amount, 
				    MAX(er.timestamp) as timestamp
				FROM exchange_rates er
				JOIN cryptocurrencies c ON er.crypto_id = c.id
				JOIN fiat_currencies f ON er.fiat_id = f.id
				WHERE c.code = $1 AND er.timestamp BETWEEN $2 AND $3
				GROUP BY c.code, f.code
				ORDER BY f.code
			)
			SELECT crypto_code, fiat_code, amount, timestamp
			FROM AvgRates
		`
	case "last":
		fallthrough
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
		return nil, fmt.Errorf("%s: %w", op, err)
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
			return nil, fmt.Errorf("%s: %w", op, err)
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
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if len(fiatPrices) == 0 {
		return nil, fmt.Errorf("%s: no rates found for currency %s", op, currency)
	}

	rate := &entities.ExchangeRate{
		Title:      cryptoCode,
		FiatValues: fiatPrices,
		DateUpdate: latestTimestamp,
	}

	return rate, nil
}

func (s *Storage) GetAllRates(ctx context.Context, date time.Time, option string) ([]entities.ExchangeRate, error) {
	const op = "storage.postgres.GetAllRates"

	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	var query string
	switch option {
	case "max":
		query = `
			WITH RatesWithMax AS (
				SELECT 
					c.code as crypto_code,
					f.code as fiat_code,
					MAX(er.amount) as amount,
					MAX(er.timestamp) as timestamp,
					c.id as crypto_id,
					f.id as fiat_id
				FROM exchange_rates er
				JOIN cryptocurrencies c ON er.crypto_id = c.id
				JOIN fiat_currencies f ON er.fiat_id = f.id
				WHERE er.timestamp BETWEEN $1 AND $2
				GROUP BY c.code, f.code, c.id, f.id
			),
			GroupedRates AS (
				SELECT 
					crypto_code,
					array_agg(fiat_code) as fiat_codes,
					array_agg(amount) as amounts,
					MAX(timestamp) as latest_timestamp
				FROM RatesWithMax
				GROUP BY crypto_code
				ORDER BY crypto_code
			)
			SELECT crypto_code, fiat_codes, amounts, latest_timestamp
			FROM GroupedRates
		`
	case "min":
		query = `
			WITH RatesWithMin AS (
				SELECT 
					c.code as crypto_code,
					f.code as fiat_code,
					MIN(er.amount) as amount,
					MAX(er.timestamp) as timestamp,
					c.id as crypto_id,
					f.id as fiat_id
				FROM exchange_rates er
				JOIN cryptocurrencies c ON er.crypto_id = c.id
				JOIN fiat_currencies f ON er.fiat_id = f.id
				WHERE er.timestamp BETWEEN $1 AND $2
				GROUP BY c.code, f.code, c.id, f.id
			),
			GroupedRates AS (
				SELECT 
					crypto_code,
					array_agg(fiat_code) as fiat_codes,
					array_agg(amount) as amounts,
					MAX(timestamp) as latest_timestamp
				FROM RatesWithMin
				GROUP BY crypto_code
				ORDER BY crypto_code
			)
			SELECT crypto_code, fiat_codes, amounts, latest_timestamp
			FROM GroupedRates
		`
	case "avg":
		query = `
			WITH RatesWithAvg AS (
				SELECT 
					c.code as crypto_code,
					f.code as fiat_code,
					AVG(er.amount) as amount,
					MAX(er.timestamp) as timestamp,
					c.id as crypto_id,
					f.id as fiat_id
				FROM exchange_rates er
				JOIN cryptocurrencies c ON er.crypto_id = c.id
				JOIN fiat_currencies f ON er.fiat_id = f.id
				WHERE er.timestamp BETWEEN $1 AND $2
				GROUP BY c.code, f.code, c.id, f.id
			),
			GroupedRates AS (
				SELECT 
					crypto_code,
					array_agg(fiat_code) as fiat_codes,
					array_agg(amount) as amounts,
					MAX(timestamp) as latest_timestamp
				FROM RatesWithAvg
				GROUP BY crypto_code
				ORDER BY crypto_code
			)
			SELECT crypto_code, fiat_codes, amounts, latest_timestamp
			FROM GroupedRates
		`
	case "last":
		fallthrough
	default:
		query = `
			WITH RankedRates AS (
				SELECT 
					c.code as crypto_code,
					f.code as fiat_code,
					er.amount,
					er.timestamp,
					ROW_NUMBER() OVER (PARTITION BY c.code, f.code ORDER BY er.timestamp DESC) as rn
				FROM exchange_rates er
				JOIN cryptocurrencies c ON er.crypto_id = c.id
				JOIN fiat_currencies f ON er.fiat_id = f.id
				WHERE er.timestamp BETWEEN $1 AND $2
			),
			FilteredRates AS (
				SELECT 
					crypto_code, 
					fiat_code,
					amount,
					timestamp
				FROM RankedRates
				WHERE rn = 1
			),
			GroupedRates AS (
				SELECT 
					crypto_code,
					array_agg(fiat_code) as fiat_codes,
					array_agg(amount) as amounts,
					MAX(timestamp) as latest_timestamp
				FROM FilteredRates
				GROUP BY crypto_code
				ORDER BY crypto_code
			)
			SELECT crypto_code, fiat_codes, amounts, latest_timestamp
			FROM GroupedRates
		`
	}

	rows, err := s.db.Query(ctx, query, startOfDay, endOfDay)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var rates []entities.ExchangeRate

	for rows.Next() {
		var cryptoCode string
		var fiatCodes []string
		var amounts []float64
		var timestamp time.Time

		if err := rows.Scan(&cryptoCode, &fiatCodes, &amounts, &timestamp); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		var fiatPrices []entities.FiatPrice
		for i := 0; i < len(fiatCodes); i++ {
			fiatPrices = append(fiatPrices, entities.FiatPrice{
				Currency: fiatCodes[i],
				Amount:   amounts[i],
			})
		}

		rates = append(rates, entities.ExchangeRate{
			Title:      cryptoCode,
			FiatValues: fiatPrices,
			DateUpdate: timestamp,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return rates, nil
}
