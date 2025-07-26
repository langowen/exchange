package postgres

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/langowen/exchange/deploy/config"
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
		return nil, fmt.Errorf("%s: pgxpool connect failed: %w", op, err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("%s: ping failed: %w", op, err)
	}

	storageBD := NewStorage(pool, cfg)

	slog.Info("PostgresSQL storage initialized successfully")
	return storageBD, nil
}

func (storage *Storage) SaveRate(ctx context.Context, rate *entities.ExchangeRate) error {
	const op = "storage.postgres.SaveRate"

	var cryptoID int
	err := storage.db.QueryRow(ctx, `INSERT INTO cryptocurrencies (code) VALUES ($1) ON CONFLICT (code) DO UPDATE SET code=EXCLUDED.code RETURNING id`, rate.Title).Scan(&cryptoID)
	if err != nil {
		return fmt.Errorf("%s: upsert cryptocurrency failed: %w", op, err)
	}

	tx, err := storage.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("%s: begin transaction failed: %w", op, err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	for _, fiatValue := range rate.FiatValues {
		var fiatID int
		err = tx.QueryRow(ctx, `INSERT INTO fiat_currencies (code) VALUES ($1) ON CONFLICT (code) DO UPDATE SET code=EXCLUDED.code RETURNING id`, fiatValue.Currency).Scan(&fiatID)
		if err != nil {
			return fmt.Errorf("%s: upsert fiat currency failed: %w", op, err)
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO exchange_rates (crypto_id, fiat_id, amount, timestamp)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (crypto_id, fiat_id, timestamp) DO UPDATE SET amount=EXCLUDED.amount
		`, cryptoID, fiatID, fiatValue.Amount, rate.DateUpdate)
		if err != nil {
			return fmt.Errorf("%s: save exchange rate failed: %w", op, err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("%s: commit transaction failed: %w", op, err)
	}

	slog.Debug("rates saved successfully", "crypto", rate.Title, "fiat_count", len(rate.FiatValues))
	return nil
}
