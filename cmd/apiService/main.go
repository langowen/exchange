package main

import (
	"context"
	"github.com/langowen/exchange/internal/apiService/http-server/server"
	"github.com/langowen/exchange/internal/apiService/service"
	"github.com/langowen/exchange/internal/apiService/storage/postgres"
	"github.com/langowen/exchange/internal/config"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

func main() {

	cfg := config.NewConfig()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	ctx, cancel := context.WithCancel(context.Background())

	pgStorage, err := postgres.New(ctx, cfg)
	if err != nil {
		log.Fatalln("Failed to initialize PostgresSQL storage", "error", err)
	}

	serviceRate := service.NewService(pgStorage, cfg)

	logger.With(
		"Config params", cfg,
		"go_version", runtime.Version(),
	).Info("starting server")

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	srv := server.Init(serviceRate, cfg)

	go func() {
		if err := srv.Server.ListenAndServe(); err != nil {
			logger.Error("failed to start server")
		}
	}()

	logger.Info("server started")

	<-done
	cancel()

	logger.Info("stopping server")

	if err := srv.Server.Shutdown(ctx); err != nil {
		logger.Error("failed to stop server", "error", err)
	}

	logger.Info("server stopped")

}
