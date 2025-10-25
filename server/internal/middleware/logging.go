package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
)

type logContextKey string

const (
	requestIDKey logContextKey = "request_id"
	loggerKey    logContextKey = "logger"
)

// LoggingMiddleware attaches a request ID and a contextual slog.Logger to each request.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := uuid.New().String()

		// Create a contextual logger
		logger := slog.With(
			"request_id", reqID,
			"method", r.Method,
			"path", r.URL.Path,
		)

		ctx := context.WithValue(r.Context(), requestIDKey, reqID)
		ctx = context.WithValue(ctx, loggerKey, logger)

		logger.Info("request started")

		next.ServeHTTP(w, r.WithContext(ctx))

		logger.Info("request completed")
	})
}

// GetLogger retrieves the contextual slog.Logger from context
func GetLogger(ctx context.Context) *slog.Logger {
	logger, ok := ctx.Value(loggerKey).(*slog.Logger)
	if !ok {
		return slog.Default()
	}
	return logger
}

// GetRequestID retrieves the request ID from context
func GetRequestID(ctx context.Context) string {
	reqID, ok := ctx.Value(requestIDKey).(string)
	if !ok {
		return ""
	}
	return reqID
}
