package main

import (
	"context"
	"github.com/langowen/exchange/deploy/config"
	"github.com/langowen/exchange/internal/currency_fetcher/adapter/api_client/coin_desk"
	"github.com/langowen/exchange/internal/currency_fetcher/adapter/storage/postgres"
	service "github.com/langowen/exchange/internal/currency_fetcher/fetcher"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

// TODO 1. Прокидывать не конфиг а конкретные параметры везде
// TODO 2. Сделать пакет APP в котором сделать фукнция запуска всех сервисов
func main() {
	cfg := config.NewConfig()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	ctx, cancel := context.WithCancel(context.Background())

	pgStorage, err := postgres.New(ctx, cfg)
	if err != nil {
		log.Fatalln("Failed to initialize PostgresSQL storage", "error", err)
	}

	httClient := coin_desk.NewHTTPClient()

	fetch := service.NewFetcher(pgStorage, httClient, cfg)

	if err := service.InitFetcher(ctx, fetch); err != nil {
		log.Fatalln("Failed to initialize fetcher", "error", err)
	}

	slog.Info("starting fetcher")

	done := make(chan os.Signal, 1)

	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	<-done
	cancel()

}
