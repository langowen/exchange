package main

import (
	"context"
	"github.com/langowen/exchange/deploy/config"
	fetcherApp "github.com/langowen/exchange/internal/api_service/app"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfg := config.NewConfig()

	ctx, cancel := context.WithCancel(context.Background())

	app := fetcherApp.NewFetcherApp(cfg)
	serverDone := app.Start(ctx)

	done := make(chan os.Signal, 1)

	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	<-done
	slog.Info("Gracefully shutting down")

	cancel()
	slog.Info("stopping server")

	<-serverDone
	slog.Info("server stopped")

}
