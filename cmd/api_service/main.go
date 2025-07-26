package main

import (
	"context"
	"github.com/langowen/exchange/deploy/config"
	"github.com/langowen/exchange/internal/api_service/adapter/storage/postgres"
	"github.com/langowen/exchange/internal/api_service/ports/http/public"
	"github.com/langowen/exchange/internal/api_service/service"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

//сделать waping, сделать энититес сервис, постаратся реализовать патерн для вызова бд из сервиса
//googsy

func main() {

	cfg := config.NewConfig()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	ctx, cancel := context.WithCancel(context.Background())

	pgStorage, err := postgres.New(ctx, cfg)
	if err != nil {
		log.Fatalln("Failed to initialize PostgresSQL storage", "error", err)
	}

	serviceRate, err := service.NewService(pgStorage, cfg)
	if err != nil {
		log.Fatalln("Failed to initialize service rate", "error", err)
	}

	done := make(chan os.Signal, 1)

	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	logger.With(
		"Config params", cfg,
		"go_version", runtime.Version(),
	).Info("starting server")

	serverDone := public.StartServer(ctx, serviceRate, cfg)

	logger.Info("server started")

	<-done
	cancel()
	logger.Info("stopping server")

	<-serverDone
	logger.Info("server stopped")

}
