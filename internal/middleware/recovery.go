// Package middleware implements recovery middleware for panic handling.
package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/albedosehen/rwwwrse/internal/observability"
)

// recoveryMiddleware implements panic recovery for HTTP handlers.
type recoveryMiddleware struct {
	logger observability.Logger
}

// NewRecoveryMiddleware creates a new recovery middleware.
func NewRecoveryMiddleware(logger observability.Logger) Middleware {
	return &recoveryMiddleware{
		logger: logger,
	}
}

// Wrap implements the Middleware interface.
func (m *recoveryMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				m.handlePanic(w, r, err)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// handlePanic handles a panic that occurred during request processing.
func (m *recoveryMiddleware) handlePanic(w http.ResponseWriter, r *http.Request, err interface{}) {
	stack := debug.Stack()
	requestID := GetRequestID(r.Context())

	m.logger.Error(r.Context(), fmt.Errorf("panic: %v", err), "Panic recovered",
		observability.String("request_id", requestID),
		observability.String("method", r.Method),
		observability.String("path", r.URL.Path),
		observability.String("remote_addr", r.RemoteAddr),
		observability.String("stack", string(stack)),
	)

	// Return 500 Internal Server Error
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)

	response := fmt.Sprintf(`{
		"error": "Internal server error",
		"status": 500,
		"request_id": "%s",
		"timestamp": "%s"
	}`, requestID, time.Now().UTC().Format(time.RFC3339))

	_, _ = w.Write([]byte(response))
}
