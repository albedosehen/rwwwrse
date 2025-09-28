package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/albedosehen/rwwwrse/internal/observability"
)

type serverManager struct {
	servers map[string]Server
	logger  observability.Logger
	mutex   sync.RWMutex

	// State management
	running bool
	wg      sync.WaitGroup
}

func NewServerManager(logger observability.Logger) ServerManager {
	return &serverManager{
		servers: make(map[string]Server),
		logger:  logger,
	}
}

func (sm *serverManager) AddServer(name string, server Server) error {
	if name == "" {
		return fmt.Errorf("server name cannot be empty")
	}

	if server == nil {
		return fmt.Errorf("server cannot be nil")
	}

	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if _, exists := sm.servers[name]; exists {
		return fmt.Errorf("server with name %q already exists", name)
	}

	sm.servers[name] = server

	if sm.logger != nil {
		sm.logger.Info(context.Background(), "Server added to manager",
			observability.String("server_name", name),
			observability.String("listen_addr", server.ListenAddr()),
		)
	}

	return nil
}

func (sm *serverManager) RemoveServer(name string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	server, exists := sm.servers[name]
	if !exists {
		return fmt.Errorf("server with name %q not found", name)
	}

	if sm.running {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Stop(ctx); err != nil {
			if sm.logger != nil {
				sm.logger.Error(ctx, err, "Error stopping server during removal",
					observability.String("server_name", name),
				)
			}
		}
	}

	delete(sm.servers, name)

	if sm.logger != nil {
		sm.logger.Info(context.Background(), "Server removed from manager",
			observability.String("server_name", name),
		)
	}

	return nil
}

func (sm *serverManager) StartAll(ctx context.Context) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if sm.running {
		return fmt.Errorf("servers are already running")
	}

	if len(sm.servers) == 0 {
		return fmt.Errorf("no servers to start")
	}

	sm.running = true

	// Start each server in its own goroutine
	for name, server := range sm.servers {
		sm.wg.Add(1)
		go func(serverName string, srv Server) {
			defer sm.wg.Done()

			if sm.logger != nil {
				sm.logger.Info(ctx, "Starting server",
					observability.String("server_name", serverName),
					observability.String("listen_addr", srv.ListenAddr()),
				)
			}

			if err := srv.Start(ctx); err != nil {
				if sm.logger != nil {
					sm.logger.Error(ctx, err, "Server startup error",
						observability.String("server_name", serverName),
					)
				}
			}
		}(name, server)
	}

	if sm.logger != nil {
		sm.logger.Info(ctx, "All servers started",
			observability.Int("server_count", len(sm.servers)),
		)
	}

	return nil
}

func (sm *serverManager) StopAll(ctx context.Context) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if !sm.running {
		return nil // Already stopped
	}

	if sm.logger != nil {
		sm.logger.Info(ctx, "Stopping all servers",
			observability.Int("server_count", len(sm.servers)),
		)
	}

	var errors []error
	for name, server := range sm.servers {
		if err := server.Stop(ctx); err != nil {
			errors = append(errors, fmt.Errorf("failed to stop server %q: %w", name, err))
			if sm.logger != nil {
				sm.logger.Error(ctx, err, "Error stopping server",
					observability.String("server_name", name),
				)
			}
		} else if sm.logger != nil {
			sm.logger.Info(ctx, "Server stopped",
				observability.String("server_name", name),
			)
		}
	}

	sm.wg.Wait()
	sm.running = false

	if len(errors) > 0 {
		return fmt.Errorf("errors occurred while stopping servers: %v", errors)
	}

	if sm.logger != nil {
		sm.logger.Info(ctx, "All servers stopped")
	}

	return nil
}

func (sm *serverManager) GetServer(name string) (Server, bool) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	server, exists := sm.servers[name]
	return server, exists
}

func (sm *serverManager) ListServers() []string {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	names := make([]string, 0, len(sm.servers))
	for name := range sm.servers {
		names = append(names, name)
	}

	return names
}

func (sm *serverManager) IsRunning() bool {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	return sm.running
}

func (sm *serverManager) GetServerCount() int {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	return len(sm.servers)
}

func (sm *serverManager) GetServerStats() map[string]ServerStats {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	stats := make(map[string]ServerStats)
	for name, server := range sm.servers {
		if metricsServer, ok := server.(interface{ GetStats() ServerStats }); ok {
			stats[name] = metricsServer.GetStats()
		} else {
			stats[name] = ServerStats{
				IsRunning: sm.running,
				StartTime: time.Now().Format(time.RFC3339),
			}
		}
	}

	return stats
}

type serverMetrics struct {
	activeConnections   int64
	totalConnections    int64
	totalRequests       int64
	connectionErrors    int64
	requestErrors       int64
	averageResponseTime float64
	requestsPerSecond   float64
	startTime           time.Time
	mutex               sync.RWMutex
}

func NewServerMetrics() ServerMetrics {
	return &serverMetrics{
		startTime: time.Now(),
	}
}

func (sm *serverMetrics) RecordRequest(method, path string, statusCode int, duration float64) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.totalRequests++

	if sm.averageResponseTime == 0 {
		sm.averageResponseTime = duration
	} else {
		sm.averageResponseTime = (sm.averageResponseTime + duration) / 2
	}

	uptime := time.Since(sm.startTime).Seconds()
	if uptime > 0 {
		sm.requestsPerSecond = float64(sm.totalRequests) / uptime
	}

	if statusCode >= 400 {
		sm.requestErrors++
	}
}

func (sm *serverMetrics) RecordConnection(action string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	switch action {
	case "open":
		sm.activeConnections++
		sm.totalConnections++
	case "close":
		sm.activeConnections--
		if sm.activeConnections < 0 {
			sm.activeConnections = 0
		}
	case "error":
		sm.connectionErrors++
	}
}

func (sm *serverMetrics) RecordError(errorType, context string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	switch errorType {
	case "connection":
		sm.connectionErrors++
	case "request":
		sm.requestErrors++
	}
}

func (sm *serverMetrics) GetStats() ServerStats {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	uptime := time.Since(sm.startTime)

	return ServerStats{
		ActiveConnections:   sm.activeConnections,
		TotalConnections:    sm.totalConnections,
		TotalRequests:       sm.totalRequests,
		ConnectionErrors:    sm.connectionErrors,
		RequestErrors:       sm.requestErrors,
		AverageResponseTime: sm.averageResponseTime,
		RequestsPerSecond:   sm.requestsPerSecond,
		IsRunning:           true, // This would need to be set by the server
		UpTime:              uptime.String(),
		StartTime:           sm.startTime.Format(time.RFC3339),
	}
}

type basicHealthChecker struct {
	server Server
}

func NewBasicHealthChecker(server Server) HealthChecker {
	return &basicHealthChecker{
		server: server,
	}
}

func (hc *basicHealthChecker) IsHealthy(ctx context.Context) bool {
	return hc.server != nil && hc.server.Handler() != nil
}

func (hc *basicHealthChecker) HealthDetails(ctx context.Context) map[string]interface{} {
	details := make(map[string]interface{})

	if hc.server == nil {
		details["status"] = "unhealthy"
		details["reason"] = "server is nil"
		return details
	}

	details["status"] = "healthy"
	details["listen_addr"] = hc.server.ListenAddr()
	details["has_handler"] = hc.server.Handler() != nil

	if httpsServer, ok := hc.server.(HTTPSServer); ok {
		details["is_https"] = httpsServer.IsHTTPS()
	} else if httpServer, ok := hc.server.(HTTPServer); ok {
		details["is_https"] = httpServer.IsHTTPS()
	}

	return details
}
