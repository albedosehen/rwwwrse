package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/albedosehen/rwwwrse/internal/observability"
)

type contextKey string

type httpServer struct {
	server   *http.Server
	listener net.Listener
	handler  http.Handler
	config   ServerConfig
	metrics  ServerMetrics
	logger   observability.Logger

	running   int32
	startTime time.Time

	activeConnections int64
	totalConnections  int64
	totalRequests     int64
}

func NewHTTPServer(config ServerConfig, handler http.Handler, logger observability.Logger) HTTPServer {
	s := &httpServer{
		config:  config,
		handler: handler,
		logger:  logger,
	}

	s.setupServer()
	return s
}

func (s *httpServer) setupServer() {
	readTimeout, _ := time.ParseDuration(s.config.ReadTimeout)
	if readTimeout == 0 {
		readTimeout = 30 * time.Second
	}

	writeTimeout, _ := time.ParseDuration(s.config.WriteTimeout)
	if writeTimeout == 0 {
		writeTimeout = 30 * time.Second
	}

	idleTimeout, _ := time.ParseDuration(s.config.IdleTimeout)
	if idleTimeout == 0 {
		idleTimeout = 60 * time.Second
	}

	maxHeaderBytes := s.config.MaxHeaderBytes
	if maxHeaderBytes == 0 {
		maxHeaderBytes = 1 << 20 // 1MB default
	}

	wrappedHandler := s.wrapHandler(s.handler)

	s.server = &http.Server{
		Addr:           s.getAddress(),
		Handler:        wrappedHandler,
		ReadTimeout:    readTimeout,
		WriteTimeout:   writeTimeout,
		IdleTimeout:    idleTimeout,
		MaxHeaderBytes: maxHeaderBytes,
		ConnState:      s.onConnStateChange,
	}

	if !s.config.KeepAlivesEnabled {
		s.server.SetKeepAlivesEnabled(false)
	}
}

func (s *httpServer) wrapHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		atomic.AddInt64(&s.totalRequests, 1)

		// Wrap response writer to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Add context
		ctx := r.Context()
		const requestStartKey contextKey = "request_start"
		const serverNameKey contextKey = "server_name"

		ctx = context.WithValue(ctx, requestStartKey, start)
		ctx = context.WithValue(ctx, serverNameKey, s.getServerName())
		r = r.WithContext(ctx)

		handler.ServeHTTP(rw, r)

		// Record metrics
		duration := time.Since(start)
		if s.metrics != nil {
			s.metrics.RecordRequest(r.Method, r.URL.Path, rw.statusCode, duration.Seconds())
		}

		if s.logger != nil {
			s.logger.Info(ctx, "HTTP request processed",
				observability.String("method", r.Method),
				observability.String("path", r.URL.Path),
				observability.Int("status", rw.statusCode),
				observability.Duration("duration", duration),
				observability.String("remote_addr", r.RemoteAddr),
				observability.String("user_agent", r.UserAgent()),
			)
		}
	})
}

func (s *httpServer) onConnStateChange(conn net.Conn, state http.ConnState) {
	switch state {
	case http.StateNew:
		atomic.AddInt64(&s.activeConnections, 1)
		atomic.AddInt64(&s.totalConnections, 1)
		if s.metrics != nil {
			s.metrics.RecordConnection("open")
		}
	case http.StateClosed, http.StateHijacked:
		atomic.AddInt64(&s.activeConnections, -1)
		if s.metrics != nil {
			s.metrics.RecordConnection("close")
		}
	}
}

func (s *httpServer) getAddress() string {
	host := s.config.Host
	if host == "" {
		host = "0.0.0.0"
	}
	return fmt.Sprintf("%s:%d", host, s.config.Port)
}

func (s *httpServer) getServerName() string {
	return fmt.Sprintf("http-%s:%d", s.config.Host, s.config.Port)
}

func (s *httpServer) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&s.running, 0, 1) {
		return fmt.Errorf("server is already running")
	}

	s.startTime = time.Now()

	listener, err := net.Listen("tcp", s.getAddress())
	if err != nil {
		atomic.StoreInt32(&s.running, 0)
		return fmt.Errorf("failed to create listener: %w", err)
	}

	s.listener = listener

	if s.logger != nil {
		s.logger.Info(ctx, "HTTP server starting",
			observability.String("address", s.getAddress()),
			observability.String("server", s.getServerName()),
		)
	}

	err = s.server.Serve(listener)
	if err != nil && err != http.ErrServerClosed {
		atomic.StoreInt32(&s.running, 0)
		if s.logger != nil {
			s.logger.Error(ctx, err, "HTTP server error")
		}
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

func (s *httpServer) Stop(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&s.running, 1, 0) {
		return nil // Already stopped
	}

	if s.logger != nil {
		s.logger.Info(ctx, "HTTP server stopping",
			observability.String("server", s.getServerName()),
		)
	}

	shutdownTimeout, _ := time.ParseDuration(s.config.ShutdownTimeout)
	if shutdownTimeout == 0 {
		shutdownTimeout = 30 * time.Second
	}

	shutdownCtx, cancel := context.WithTimeout(ctx, shutdownTimeout)
	defer cancel()

	if err := s.server.Shutdown(shutdownCtx); err != nil {
		if s.logger != nil {
			s.logger.Error(ctx, err, "Error during server shutdown")
		}
		return fmt.Errorf("server shutdown error: %w", err)
	}

	if s.logger != nil {
		s.logger.Info(ctx, "HTTP server stopped",
			observability.String("server", s.getServerName()),
			observability.Duration("uptime", time.Since(s.startTime)),
		)
	}

	return nil
}

func (s *httpServer) ListenAddr() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return s.getAddress()
}

func (s *httpServer) Handler() http.Handler {
	return s.handler
}

func (s *httpServer) IsHTTPS() bool {
	return false
}

type httpsServer struct {
	*httpServer
	tlsConfig *tls.Config
	// Note: certManager removed as it was unused
	// TODO: Add certificate manager when TLS management is implemented
}

func NewHTTPSServer(
	config ServerConfig,
	handler http.Handler,
	tlsConfig *tls.Config,
	logger observability.Logger,
) HTTPSServer {
	baseServer := &httpServer{
		config:  config,
		handler: handler,
		logger:  logger,
	}

	s := &httpsServer{
		httpServer: baseServer,
		tlsConfig:  tlsConfig,
	}

	s.setupHTTPSServer()
	return s
}

func (s *httpsServer) setupHTTPSServer() {
	s.setupServer()

	if s.config.TLSPort > 0 {
		host := s.config.Host
		if host == "" {
			host = "0.0.0.0"
		}
		s.server.Addr = fmt.Sprintf("%s:%d", host, s.config.TLSPort)
	}

	// Configure TLS
	if s.tlsConfig != nil {
		s.server.TLSConfig = s.tlsConfig
	}
}

func (s *httpsServer) getAddress() string {
	host := s.config.Host
	if host == "" {
		host = "0.0.0.0"
	}
	port := s.config.TLSPort
	if port == 0 {
		port = 8443 // Default HTTPS port
	}
	return fmt.Sprintf("%s:%d", host, port)
}

func (s *httpsServer) getServerName() string {
	port := s.config.TLSPort
	if port == 0 {
		port = 8443
	}
	return fmt.Sprintf("https-%s:%d", s.config.Host, port)
}

func (s *httpsServer) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&s.running, 0, 1) {
		return fmt.Errorf("server is already running")
	}

	s.startTime = time.Now()

	listener, err := net.Listen("tcp", s.getAddress())
	if err != nil {
		atomic.StoreInt32(&s.running, 0)
		return fmt.Errorf("failed to create listener: %w", err)
	}

	tlsListener := tls.NewListener(listener, s.server.TLSConfig)
	s.listener = tlsListener

	if s.logger != nil {
		s.logger.Info(ctx, "HTTPS server starting",
			observability.String("address", s.getAddress()),
			observability.String("server", s.getServerName()),
		)
	}

	err = s.server.Serve(tlsListener)
	if err != nil && err != http.ErrServerClosed {
		atomic.StoreInt32(&s.running, 0)
		if s.logger != nil {
			s.logger.Error(ctx, err, "HTTPS server error")
		}
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

func (s *httpsServer) IsHTTPS() bool {
	return true
}

// GetCertificate returns the current TLS certificate function.
func (s *httpsServer) GetCertificate() func(*http.Request) (*http.Request, error) {
	// This is a placeholder - in a real implementation, this would
	// integrate with the TLS certificate manager
	return func(r *http.Request) (*http.Request, error) {
		return r, nil
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
