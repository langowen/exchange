package postgres

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/langowen/exchange/internal/apiService/storage"
	"github.com/langowen/exchange/internal/config"
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
		return nil, fmt.Errorf("%s: ping failed: %w", op, "error", err)
	}

	storageBD := NewStorage(pool, cfg)

	/*if err := storageBD.initSchema(ctx); err != nil {
		slog.Error("failed to init database schema", "error", err)
		pool.Close()
		return nil, fmt.Errorf("%s: init schema failed: %w", op, err)
	}*/

	slog.Info("PostgresSQL storage initialized successfully")
	return storageBD, nil
}

func (s *Storage) GetRate(ctx context.Context, currency string) (rate *storage.Rate, err error) {
	return nil, nil
}

func (s *Storage) GetAllRates(ctx context.Context) (rates []storage.Rate, err error) {
	return nil, nil
}
