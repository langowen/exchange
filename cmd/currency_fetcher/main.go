package main

import (
	"context"
	"github.com/langowen/exchange/deploy/config"
	"github.com/langowen/exchange/internal/currency_fetcher/app"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	cfg := config.NewConfig()

	app := apiApp.NewApiApp(cfg)

	go func() {
		done := make(chan os.Signal, 1)

		signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

		<-done
		cancel()

		slog.Info("gracefully shutting down")
	}()

	app.Start(ctx)

	slog.Info("stopping application")

}
