package public

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/langowen/exchange/deploy/config"
	mwLogger "github.com/langowen/exchange/internal/api_service/ports/http/public/middleware/logger"
	"github.com/langowen/exchange/internal/api_service/service"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log/slog"
	"net/http"
	"time"
)

type Server struct {
	Server  *http.Server
	cfg     *config.Config
	service Service
}

func NewServer(server *http.Server, cfg *config.Config, service2 *service.Service) *Server {
	return &Server{
		Server:  server,
		cfg:     cfg,
		service: service2,
	}
}

func StartServer(ctx context.Context, service *service.Service, cfg *config.Config) <-chan struct{} {

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(mwLogger.New())
	r.Use(middleware.Recoverer)

	r.Handle("/metrics", promhttp.Handler())

	serverConfig := &http.Server{
		Addr:         ":" + cfg.HTTPServer.Port,
		Handler:      r,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	server := NewServer(serverConfig, cfg, service)

	doneChan := make(chan struct{})

	go func() {
		if err := server.Server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Http server error", "error", err)
		}
	}()

	r.Get("/rates", server.GetAllRates)
	r.Get("/rates/{cryptocurrency}", server.GetRateByCurrency)

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Server.Shutdown(shutdownCtx); err != nil {
			slog.Error("Failed to stop server", "error", err)
		}

		close(doneChan)
	}()

	return doneChan
}

func (s *Server) GetAllRates(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	options := r.URL.Query().Get("option")

	date := r.URL.Query().Get("date")

	rates, err := s.service.FetchAllRates(ctx, date, options)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, err.Error())
	}

	RespondWithJSON(w, http.StatusOK, rates)

}

func (s *Server) GetRateByCurrency(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	currency := chi.URLParam(r, "cryptocurrency")

	options := r.URL.Query().Get("option")

	date := r.URL.Query().Get("date")

	rate, err := s.service.FetchRate(ctx, currency, date, options)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, err.Error())
	}

	RespondWithJSON(w, http.StatusOK, rate)

}

func RespondWithJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(code)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("Failed to encode response", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func RespondWithError(w http.ResponseWriter, code int, message string, details ...string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(code)

	errorText := message
	if len(details) > 0 {
		errorText += "\nDetails: " + details[0]
	}

	if _, err := w.Write([]byte(errorText)); err != nil {
		slog.Error("Failed to write error response", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
