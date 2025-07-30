package apiApp

import (
	"context"
	"fmt"
	"github.com/langowen/exchange/deploy/config"
	"github.com/langowen/exchange/internal/currency_fetcher/adapter/api_client/coin_desk"
	"github.com/langowen/exchange/internal/currency_fetcher/fetcher"
	"os"

	"github.com/langowen/exchange/internal/currency_fetcher/adapter/storage/postgres"
	"github.com/langowen/exchange/internal/currency_fetcher/adapter/storage/redis"
	"log"
	"log/slog"

	redisPack "github.com/redis/go-redis/v9"
)

type ApiApp struct {
	cfg *config.Config
}

func NewApiApp(cfg *config.Config) *ApiApp {
	return &ApiApp{cfg: cfg}
}

func (a *ApiApp) Start(ctx context.Context) {
	a.initLogger()
	slog.Info("Logger initialized")

	slog.With("config", a.cfg).Info("starting application")

	pgStorage := a.initDatabase(ctx)
	slog.Info("Storage initialized")

	httpClient := a.initHTTPClient()
	slog.Info("HTTP client initialized")

	rdStorage := a.initRedis(ctx)
	slog.Info("Redis client initialized")

	slog.Info("starting application")
	if err := a.initFetcher(ctx, pgStorage, httpClient, rdStorage); err != nil {
		log.Fatal(err)
	}

}

func (a *ApiApp) initLogger() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: false,
	}))
	slog.SetDefault(logger)
}

func (a *ApiApp) initDatabase(ctx context.Context) *postgres.Storage {

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s search_path=%s",
		a.cfg.Storage.Host,
		a.cfg.Storage.Port,
		a.cfg.Storage.User,
		a.cfg.Storage.Password,
		a.cfg.Storage.DBName,
		a.cfg.Storage.SSLMode,
		a.cfg.Storage.Schema,
	)

	pgStorage, err := postgres.InitStorage(ctx, dsn)
	if err != nil {
		log.Fatalln("Failed to initialize PostgresSQL storage", "error", err)
	}

	return pgStorage
}

func (a *ApiApp) initHTTPClient() *coin_desk.HTTPClient {
	httClient := coin_desk.NewHTTPClient()

	return httClient
}

func (a *ApiApp) initRedis(ctx context.Context) *redis.Storage {
	options := &redisPack.Options{
		Addr:     a.cfg.Redis.Host,
		Password: a.cfg.Redis.Password,
		DB:       a.cfg.Redis.DB,
	}

	rdStorage, err := redis.InitStorage(ctx, options)
	if err != nil {
		log.Fatalln("Failed to initialize Redis storage", "error", err)
	}

	return rdStorage
}

func (a *ApiApp) initFetcher(ctx context.Context, storage *postgres.Storage, client *coin_desk.HTTPClient, redis *redis.Storage) error {
	fetch := fetcher.NewFetcher(storage, client, redis, a.cfg)

	if err := fetch.StartFetcher(ctx); err != nil {
		slog.Error("Failed to fetcher", "error", err)
		return err
	}

	return nil
}
