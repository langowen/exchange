package logger

import (
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

func New() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		log := slog.With(
			slog.String("component", "middleware/logger"),
		)

		slog.Info("logger middleware enabled")

		fn := func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/metrics" {
				next.ServeHTTP(w, r)
				return
			}
			entry := log.With(
				slog.String("method", r.Method),
				slog.String("protocol", r.URL.Scheme),
				slog.String("host", r.Host),
				slog.String("URL", decodeURI(r.RequestURI)),
				slog.String("remote_addr", r.RemoteAddr),
				slog.String("user_agent", r.UserAgent()),
				slog.String("request_id", middleware.GetReqID(r.Context())),
			)
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			t1 := time.Now()
			defer func() {
				entry.Info("request completed",
					slog.Int("status", ww.Status()),
					slog.Int("bytes", ww.BytesWritten()),
					slog.String("duration", time.Since(t1).String()),
				)
			}()
			next.ServeHTTP(ww, r)
		}

		return http.HandlerFunc(fn)
	}
}

func decodeURI(uri string) string {
	decoded, err := url.PathUnescape(uri)
	if err != nil {
		return uri
	}
	return decoded
}
