package fetcherApp

import (
	"context"
	"fmt"
	"github.com/langowen/exchange/deploy/config"
	"github.com/langowen/exchange/internal/api_service/adapter/storage/postgres"
	"github.com/langowen/exchange/internal/api_service/adapter/storage/redis"
	"github.com/langowen/exchange/internal/api_service/ports/http/public"
	"github.com/langowen/exchange/internal/api_service/service"
	redisPack "github.com/redis/go-redis/v9"
	"log"
	"log/slog"
	"os"
)

type FetcherApp struct {
	cfg *config.Config
}

func NewFetcherApp(cfg *config.Config) *FetcherApp {
	return &FetcherApp{cfg: cfg}
}

func (f *FetcherApp) Start(ctx context.Context) <-chan struct{} {
	f.initLogger()
	slog.Info("Logger initialized")

	slog.With("config", f.cfg).Info("starting server")

	pgStorage := f.initDatabase(ctx)
	slog.Info("Storage initialized")

	rdStorage := f.initRedis(ctx)
	slog.Info("Redis client initialized")

	apiService := f.initService(pgStorage, rdStorage)
	slog.Info("Service initialized")

	serverDone := f.StartServer(ctx, apiService)
	slog.Info("server started")

	return serverDone
}

func (f *FetcherApp) initLogger() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: false,
	}))
	slog.SetDefault(logger)
}

func (f *FetcherApp) initDatabase(ctx context.Context) *postgres.Storage {

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s search_path=%s",
		f.cfg.Storage.Host,
		f.cfg.Storage.Port,
		f.cfg.Storage.User,
		f.cfg.Storage.Password,
		f.cfg.Storage.DBName,
		f.cfg.Storage.SSLMode,
		f.cfg.Storage.Schema,
	)

	pgStorage, err := postgres.InitStorage(ctx, dsn)
	if err != nil {
		log.Fatalln("Failed to initialize PostgresSQL storage", "error", err)
	}

	return pgStorage
}

func (f *FetcherApp) initRedis(ctx context.Context) *redis.Storage {
	options := &redisPack.Options{
		Addr:     f.cfg.Redis.Host,
		Password: f.cfg.Redis.Password,
		DB:       f.cfg.Redis.DB,
	}

	rdStorage, err := redis.InitStorage(ctx, options)
	if err != nil {
		log.Fatalln("Failed to initialize Redis storage", "error", err)
	}

	return rdStorage
}

func (f *FetcherApp) initService(storage *postgres.Storage, redis *redis.Storage) *service.Service {
	apiService, err := service.NewService(storage, redis)
	if err != nil {
		log.Fatalln("Failed to initialize service rate", "error", err)
	}

	return apiService
}

func (f *FetcherApp) StartServer(ctx context.Context, apiService *service.Service) <-chan struct{} {
	serverDone := public.StartServer(ctx, apiService, f.cfg)

	return serverDone
}
