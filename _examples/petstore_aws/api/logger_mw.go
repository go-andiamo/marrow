package api

import (
	"github.com/go-chi/chi/v5/middleware"
	"log/slog"
	"net/http"
	"os"
	"time"
)

func slogMiddleware() func(next http.Handler) http.Handler {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			duration := time.Since(start)
			reqID := middleware.GetReqID(r.Context())
			logger.Info("apiRequest",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", ww.Status()),
				slog.Int("bytes", ww.BytesWritten()),
				slog.String("remoteIp", r.RemoteAddr),
				slog.String("userAgent", r.UserAgent()),
				slog.Duration("duration", duration),
				slog.String("requestId", reqID),
			)
		}
		return http.HandlerFunc(fn)
	}
}
