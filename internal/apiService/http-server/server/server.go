package server

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/langowen/exchange/internal/apiService/http-server/handler"
	mwLogger "github.com/langowen/exchange/internal/apiService/http-server/middleware/logger"
	"github.com/langowen/exchange/internal/apiService/service"
	"github.com/langowen/exchange/internal/config"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

type Server struct {
	Server *http.Server
	cfg    *config.Config
}

func NewServer(server *http.Server, cfg *config.Config) *Server {
	return &Server{
		Server: server,
		cfg:    cfg,
	}
}

func Init(service *service.Service, cfg *config.Config) *Server {

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(mwLogger.New())
	r.Use(middleware.Recoverer)

	r.Handle("/metrics", promhttp.Handler())

	r.Get("/rates", handler.GetAllRates(service))
	r.Get("/rates/{cryptocurrency}", handler.GetRateByCurrency(service))

	serverConfig := &http.Server{
		Addr:         ":" + cfg.HTTPServer.Port,
		Handler:      r,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	return NewServer(serverConfig, cfg)
}
