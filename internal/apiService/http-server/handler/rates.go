package handler

import (
	"context"
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/langowen/exchange/internal/apiService/storage"
	"log/slog"
	"net/http"
)

type Service interface {
	FetchRate(ctx context.Context, currency string) (rate *storage.Rate, err error)
	FetchAllRates(ctx context.Context) (rates []storage.Rate, err error)
}

type RateRequest struct {
	Currency string `json:"currency"`
}

type RateResponse struct {
	Currency string  `json:"currency"`
	Value    float64 `json:"value"`
}

func GetAllRates(service Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		rates, err := service.FetchAllRates(ctx)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, err.Error())
		}

		response := make([]RateResponse, len(rates))
		for i, rate := range rates {
			response[i] = RateResponse{
				Currency: rate.Currency,
				Value:    rate.Value,
			}
		}

		RespondWithJSON(w, http.StatusOK, response)
	}
}

func GetRateByCurrency(service Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var request RateRequest

		request.Currency = chi.URLParam(r, "cryptocurrency")

		rate, err := service.FetchRate(ctx, request.Currency)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, err.Error())
		}

		response := RateResponse{
			Currency: rate.Currency,
			Value:    rate.Value,
		}

		RespondWithJSON(w, http.StatusOK, response)
	}
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
