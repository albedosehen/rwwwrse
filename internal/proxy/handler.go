package proxy

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"net/http/httputil"
	"time"

	proxyerrors "github.com/albedosehen/rwwwrse/internal/errors"
	"github.com/albedosehen/rwwwrse/internal/observability"
)

const (
	// HeaderRequestID is the header name for request correlation IDs
	HeaderRequestID = "X-Request-ID"

	// HeaderForwardedFor is the header name for forwarded IP addresses
	HeaderForwardedFor = "X-Forwarded-For"

	// HeaderForwardedProto is the header name for forwarded protocol
	HeaderForwardedProto = "X-Forwarded-Proto"

	// HeaderForwardedHost is the header name for forwarded host
	HeaderForwardedHost = "X-Forwarded-Host"
)

type proxyHandler struct {
	router  Router
	logger  observability.Logger
	metrics observability.MetricsCollector
}

func NewProxyHandlerImpl(
	router Router,
	logger observability.Logger,
	metrics observability.MetricsCollector,
) (ProxyHandler, error) {
	if router == nil {
		return nil, proxyerrors.NewConfigError(
			proxyerrors.ErrCodeConfigInvalid,
			"router",
			nil,
		)
	}

	if logger == nil {
		return nil, proxyerrors.NewConfigError(
			proxyerrors.ErrCodeConfigInvalid,
			"logger",
			nil,
		)
	}

	ph := &proxyHandler{
		router:  router,
		logger:  logger,
		metrics: metrics,
	}

	logger.Info(context.Background(), "Proxy handler initialized")

	return ph, nil
}

func (ph *proxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	ctx := r.Context()

	// Generate request ID if not present
	requestID := r.Header.Get(HeaderRequestID)
	if requestID == "" {
		requestID = generateRequestID()
		r.Header.Set(HeaderRequestID, requestID)
	}

	type contextKey string
	const requestIDKey contextKey = "request_id"
	ctx = context.WithValue(ctx, requestIDKey, requestID)
	r = r.WithContext(ctx)

	ph.logger.Info(ctx, "Processing proxy request",
		observability.String("request_id", requestID),
		observability.String("method", r.Method),
		observability.String("host", r.Host),
		observability.String("path", r.URL.Path),
		observability.String("remote_addr", r.RemoteAddr),
		observability.String("user_agent", r.Header.Get("User-Agent")),
	)

	// Increment active connections
	if ph.metrics != nil {
		ph.metrics.IncActiveConnections()
		defer ph.metrics.DecActiveConnections()
	}

	// Find backend for the request
	backend, err := ph.router.Route(ctx, r.Host)
	if err != nil {
		ph.handleError(w, r, err, startTime)
		return
	}

	// Create reverse proxy for the backend
	proxy := ph.createReverseProxy(backend)

	// Set up forwarded headers
	ph.setupForwardedHeaders(r)

	// Record metrics for backend request
	if ph.metrics != nil {
		defer func() {
			duration := time.Since(startTime)
			status := "unknown"
			if w.Header().Get("X-Status-Code") != "" {
				status = w.Header().Get("X-Status-Code")
			}
			ph.metrics.RecordBackendRequest(backend.Name(), status, duration)
		}()
	}

	ph.logger.Debug(ctx, "Forwarding request to backend",
		observability.String("request_id", requestID),
		observability.String("backend", backend.Name()),
		observability.String("backend_url", backend.URL().String()),
	)

	// Serve the request through the reverse proxy
	proxy.ServeHTTP(w, r)
}

func (ph *proxyHandler) createReverseProxy(backend Backend) *httputil.ReverseProxy {
	targetURL := backend.URL()

	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(targetURL)
			r.SetXForwarded()

			// Add custom headers
			r.Out.Header.Set("X-Forwarded-By", "rwwwrse")
		},
		Transport: backend.Transport(),
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			ph.handleProxyError(w, r, backend, err)
		},
		ModifyResponse: func(resp *http.Response) error {
			// Store status code for metrics
			resp.Header.Set("X-Status-Code", resp.Status)
			return nil
		},
	}

	return proxy
}

func (ph *proxyHandler) setupForwardedHeaders(r *http.Request) {
	// Set X-Forwarded-For
	if clientIP := r.Header.Get("X-Real-IP"); clientIP != "" {
		r.Header.Set(HeaderForwardedFor, clientIP)
	} else if r.RemoteAddr != "" {
		r.Header.Set(HeaderForwardedFor, r.RemoteAddr)
	}

	// Set X-Forwarded-Proto
	if r.TLS != nil {
		r.Header.Set(HeaderForwardedProto, "https")
	} else {
		r.Header.Set(HeaderForwardedProto, "http")
	}

	// Set X-Forwarded-Host
	if r.Host != "" {
		r.Header.Set(HeaderForwardedHost, r.Host)
	}
}

func (ph *proxyHandler) handleProxyError(w http.ResponseWriter, r *http.Request, backend Backend, err error) {
	requestID := r.Header.Get(HeaderRequestID)

	ph.logger.Error(r.Context(), err, "Proxy error occurred",
		observability.String("request_id", requestID),
		observability.String("backend", backend.Name()),
		observability.String("backend_url", backend.URL().String()),
		observability.String("method", r.Method),
		observability.String("path", r.URL.Path),
	)

	// Create appropriate proxy error
	proxyErr := proxyerrors.NewBackendError(
		proxyerrors.ErrCodeBackendConnectionFailed,
		backend.Name(),
		err,
	)

	ph.writeErrorResponse(w, r, proxyErr)
}

func (ph *proxyHandler) handleError(w http.ResponseWriter, r *http.Request, err error, startTime time.Time) {
	requestID := r.Header.Get(HeaderRequestID)
	duration := time.Since(startTime)

	ph.logger.Error(r.Context(), err, "Request error",
		observability.String("request_id", requestID),
		observability.String("method", r.Method),
		observability.String("host", r.Host),
		observability.String("path", r.URL.Path),
		observability.Duration("duration", duration),
	)

	// Record metrics
	if ph.metrics != nil {
		status := "500"
		if proxyErr, ok := err.(*proxyerrors.ProxyError); ok {
			status = string(rune(proxyErr.Code.HTTPStatus()))
		}
		ph.metrics.RecordRequest(r.Method, r.Host, status, duration)
	}

	ph.writeErrorResponse(w, r, err)
}

func (ph *proxyHandler) writeErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	var proxyErr *proxyerrors.ProxyError

	// Convert to proxy error if needed
	if !isProxyError(err) {
		proxyErr = proxyerrors.WrapError(
			proxyerrors.ErrCodeInternalError,
			"internal server error",
			err,
		)
	} else {
		proxyErr = err.(*proxyerrors.ProxyError)
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", r.Header.Get(HeaderRequestID))

	// Write status code
	statusCode := proxyErr.Code.HTTPStatus()
	w.WriteHeader(statusCode)

	// Write error response body
	response := map[string]any{
		"error":      proxyErr.Message,
		"status":     statusCode,
		"request_id": r.Header.Get(HeaderRequestID),
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	}

	// Include error context if available
	if len(proxyErr.Context) > 0 {
		response["context"] = proxyErr.Context
	}

	// Write JSON response
	w.Write([]byte(`{"error":"` + proxyErr.Message + `","status":` + string(rune(statusCode)) + `,"request_id":"` + r.Header.Get(HeaderRequestID) + `","timestamp":"` + time.Now().UTC().Format(time.RFC3339) + `"}`))
}

func (ph *proxyHandler) Shutdown(ctx context.Context) error {
	ph.logger.Info(ctx, "Shutting down proxy handler")

	// Wait for any ongoing requests to complete
	// This is handled by the HTTP server's graceful shutdown

	ph.logger.Info(ctx, "Proxy handler shutdown complete")
	return nil
}

func generateRequestID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID
		return hex.EncodeToString([]byte(time.Now().Format("20060102150405.000000")))
	}
	return hex.EncodeToString(bytes)
}

func isProxyError(err error) bool {
	_, ok := err.(*proxyerrors.ProxyError)
	return ok
}
