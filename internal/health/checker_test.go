package health

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/albedosehen/rwwwrse/internal/config"
	testingutils "github.com/albedosehen/rwwwrse/internal/testing"
)

// TestNewHealthChecker tests the constructor
func TestNewHealthChecker(t *testing.T) {
	// Arrange
	cfg := config.HealthConfig{
		Enabled:  true,
		Timeout:  5 * time.Second,
		Interval: 30 * time.Second,
	}
	logger := testingutils.NewMockLogger()
	metrics := testingutils.NewMockMetricsCollector()

	// Act
	checker := NewHealthChecker(cfg, logger, metrics)

	// Assert
	testingutils.AssertNotNil(t, checker)
}

// TestHealthChecker_Check tests the health checking functionality
func TestHealthChecker_Check(t *testing.T) {
	tests := []struct {
		name           string
		target         HealthTarget
		serverHandler  http.HandlerFunc
		wantHealthy    bool
		wantStatusCode int
		wantError      bool
		validate       func(t *testing.T, status HealthStatus)
	}{
		{
			name: "successful health check",
			serverHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("OK"))
			}),
			wantHealthy:    true,
			wantStatusCode: 200,
			wantError:      false,
		},
		{
			name: "failed health check - 500 status",
			serverHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("Internal Server Error"))
			}),
			wantHealthy:    false,
			wantStatusCode: 500,
			wantError:      true,
		},
		{
			name: "failed health check - timeout",
			serverHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(200 * time.Millisecond) // Longer than test timeout
				w.WriteHeader(http.StatusOK)
			}),
			wantHealthy: false,
			wantError:   true,
			validate: func(t *testing.T, status HealthStatus) {
				testingutils.AssertContains(t, status.Error.Error(), "health check request failed")
			},
		},
		{
			name:        "nil target",
			target:      nil,
			wantHealthy: false,
			wantError:   true,
			validate: func(t *testing.T, status HealthStatus) {
				testingutils.AssertContains(t, status.Error.Error(), "target cannot be nil")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			cfg := config.HealthConfig{
				Enabled:  true,
				Timeout:  5 * time.Second,
				Interval: 30 * time.Second,
			}
			logger := testingutils.NewMockLogger()
			metrics := testingutils.NewMockMetricsCollector()

			// Setup mock expectations for all possible log levels
			logger.On("Debug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
			logger.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
			logger.On("Warn", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
			logger.On("Error", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
			metrics.On("RecordHealthCheck", mock.AnythingOfType("string"), mock.AnythingOfType("bool"), mock.AnythingOfType("time.Duration")).Return()

			checker := NewHealthChecker(cfg, logger, metrics)

			var server *httptest.Server
			if tt.serverHandler != nil {
				server = httptest.NewServer(tt.serverHandler)
				defer server.Close()

				// Create target with server URL
				if tt.target == nil {
					tt.target = createTestTarget("test", server.URL, 100*time.Millisecond, 200)
				}
			}

			ctx := context.Background()

			// Act
			status := checker.Check(ctx, tt.target)

			// Assert
			testingutils.AssertEqual(t, tt.wantHealthy, status.Healthy)

			if tt.wantStatusCode > 0 {
				testingutils.AssertEqual(t, tt.wantStatusCode, status.StatusCode)
			}

			if tt.wantError {
				testingutils.AssertNotNil(t, status.Error)
			} else {
				testingutils.AssertNil(t, status.Error)
			}

			if tt.validate != nil {
				tt.validate(t, status)
			}
		})
	}
}

func TestHealthChecker_StartStopMonitoring(t *testing.T) {
	// Arrange
	cfg := config.HealthConfig{
		Enabled:  true,
		Timeout:  1 * time.Second,
		Interval: 100 * time.Millisecond,
	}
	logger := testingutils.NewMockLogger()
	metrics := testingutils.NewMockMetricsCollector()

	// Setup mock expectations for all possible log calls
	logger.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
	logger.On("Info", mock.Anything, mock.Anything).Return()
	logger.On("Debug", mock.Anything, mock.Anything, mock.Anything).Return()

	checker := NewHealthChecker(cfg, logger, metrics)

	hc := checker.(*healthChecker)

	t.Run("start monitoring when enabled", func(t *testing.T) {
		// Act
		err := checker.StartMonitoring(context.Background())

		// Assert
		testingutils.AssertNoError(t, err)
		testingutils.AssertTrue(t, hc.running)
	})

	t.Run("start monitoring when already running", func(t *testing.T) {
		// Act
		err := checker.StartMonitoring(context.Background())

		// Assert
		testingutils.AssertError(t, err)
		testingutils.AssertContains(t, err.Error(), "already running")
	})

	t.Run("stop monitoring", func(t *testing.T) {
		// Act
		err := checker.StopMonitoring()

		// Assert
		testingutils.AssertNoError(t, err)
		testingutils.AssertFalse(t, hc.running)
	})

	t.Run("stop monitoring when not running", func(t *testing.T) {
		// Act
		err := checker.StopMonitoring()

		// Assert
		testingutils.AssertNoError(t, err)
	})
}

func TestHealthChecker_StartMonitoring_Disabled(t *testing.T) {
	// Arrange
	cfg := config.HealthConfig{
		Enabled: false,
	}
	logger := testingutils.NewMockLogger()
	metrics := testingutils.NewMockMetricsCollector()

	// Setup mock expectations
	logger.On("Info", mock.Anything, mock.Anything).Return()
	logger.On("Debug", mock.Anything, mock.Anything).Return()

	checker := NewHealthChecker(cfg, logger, metrics)

	// Act
	err := checker.StartMonitoring(context.Background())

	// Assert
	testingutils.AssertNoError(t, err)

	hc := checker.(*healthChecker)
	testingutils.AssertFalse(t, hc.running)
}

func TestHealthChecker_Subscribe_Unsubscribe(t *testing.T) {
	// Arrange
	cfg := config.HealthConfig{Enabled: true}
	logger := testingutils.NewMockLogger()
	metrics := testingutils.NewMockMetricsCollector()

	// Setup mock expectations
	logger.On("Debug", mock.Anything, mock.Anything, mock.Anything).Return()
	logger.On("Info", mock.Anything, mock.Anything, mock.Anything).Return()
	logger.On("Error", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

	checker := NewHealthChecker(cfg, logger, metrics)

	eventCh1 := make(chan HealthEvent, 10)
	eventCh2 := make(chan HealthEvent, 10)

	t.Run("subscribe valid channel", func(t *testing.T) {
		// Act
		err := checker.Subscribe(eventCh1)

		// Assert
		testingutils.AssertNoError(t, err)
	})

	t.Run("subscribe multiple channels", func(t *testing.T) {
		// Act
		err := checker.Subscribe(eventCh2)

		// Assert
		testingutils.AssertNoError(t, err)
	})

	t.Run("subscribe nil channel", func(t *testing.T) {
		// Act
		err := checker.Subscribe(nil)

		// Assert
		testingutils.AssertError(t, err)
		testingutils.AssertContains(t, err.Error(), "channel cannot be nil")
	})

	t.Run("unsubscribe existing channel", func(t *testing.T) {
		// Act
		err := checker.Unsubscribe(eventCh1)

		// Assert
		testingutils.AssertNoError(t, err)
	})

	t.Run("unsubscribe non-existent channel", func(t *testing.T) {
		// Arrange
		nonExistentCh := make(chan HealthEvent, 1)

		// Act
		err := checker.Unsubscribe(nonExistentCh)

		// Assert
		testingutils.AssertError(t, err)
		testingutils.AssertContains(t, err.Error(), "channel not found")
	})

	t.Run("unsubscribe nil channel", func(t *testing.T) {
		// Act
		err := checker.Unsubscribe(nil)

		// Assert
		testingutils.AssertError(t, err)
		testingutils.AssertContains(t, err.Error(), "channel cannot be nil")
	})
}

func TestHealthChecker_RegisterUnregisterTarget(t *testing.T) {
	// Arrange
	cfg := config.HealthConfig{Enabled: true}
	logger := testingutils.NewMockLogger()
	metrics := testingutils.NewMockMetricsCollector()

	// Setup mock expectations
	logger.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
	logger.On("Info", mock.Anything, mock.Anything, mock.Anything).Return()
	logger.On("Error", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

	checker := NewHealthChecker(cfg, logger, metrics)

	hc := checker.(*healthChecker)
	target := createTestTarget("test-target", "http://localhost:8080", 5*time.Second, 200)

	t.Run("register valid target", func(t *testing.T) {
		// Act
		err := hc.RegisterTarget(target)

		// Assert
		testingutils.AssertNoError(t, err)
		testingutils.AssertContains(t, hc.targets, "test-target")
	})

	t.Run("register nil target", func(t *testing.T) {
		// Act
		err := hc.RegisterTarget(nil)

		// Assert
		testingutils.AssertError(t, err)
		testingutils.AssertContains(t, err.Error(), "target cannot be nil")
	})

	t.Run("register target with empty name", func(t *testing.T) {
		// Arrange - create a mock target that returns empty name
		emptyTarget := &mockTarget{
			name:           "",
			url:            "http://localhost:8080",
			timeout:        5 * time.Second,
			expectedStatus: 200,
			headers:        make(map[string]string),
		}

		// Act
		err := hc.RegisterTarget(emptyTarget)

		// Assert
		testingutils.AssertError(t, err)
		testingutils.AssertContains(t, err.Error(), "target name cannot be empty")
	})

	t.Run("unregister existing target", func(t *testing.T) {
		// Act
		err := hc.UnregisterTarget("test-target")

		// Assert
		testingutils.AssertNoError(t, err)
		testingutils.AssertNotContains(t, hc.targets, "test-target")
		testingutils.AssertNotContains(t, hc.status, "test-target")
	})

	t.Run("unregister with empty name", func(t *testing.T) {
		// Act
		err := hc.UnregisterTarget("")

		// Assert
		testingutils.AssertError(t, err)
		testingutils.AssertContains(t, err.Error(), "target name cannot be empty")
	})
}

func TestHealthChecker_GetTargetStatus(t *testing.T) {
	// Arrange
	cfg := config.HealthConfig{Enabled: true}
	logger := testingutils.NewMockLogger()
	metrics := testingutils.NewMockMetricsCollector()
	checker := NewHealthChecker(cfg, logger, metrics)

	hc := checker.(*healthChecker)

	// Add test status
	testStatus := HealthStatus{
		Healthy:   true,
		LastCheck: time.Now(),
	}
	hc.status["test-target"] = testStatus

	t.Run("get existing target status", func(t *testing.T) {
		// Act
		status, exists := hc.GetTargetStatus("test-target")

		// Assert
		testingutils.AssertTrue(t, exists)
		testingutils.AssertEqual(t, testStatus.Healthy, status.Healthy)
	})

	t.Run("get non-existent target status", func(t *testing.T) {
		// Act
		_, exists := hc.GetTargetStatus("non-existent")

		// Assert
		testingutils.AssertFalse(t, exists)
	})
}

func TestHealthChecker_GetAllTargetStatus(t *testing.T) {
	// Arrange
	cfg := config.HealthConfig{Enabled: true}
	logger := testingutils.NewMockLogger()
	metrics := testingutils.NewMockMetricsCollector()
	checker := NewHealthChecker(cfg, logger, metrics)

	hc := checker.(*healthChecker)

	// Add test statuses
	status1 := HealthStatus{Healthy: true, LastCheck: time.Now()}
	status2 := HealthStatus{Healthy: false, LastCheck: time.Now()}

	hc.status["target1"] = status1
	hc.status["target2"] = status2

	// Act
	allStatus := hc.GetAllTargetStatus()

	// Assert
	testingutils.AssertLen(t, allStatus, 2)
	testingutils.AssertEqual(t, status1.Healthy, allStatus["target1"].Healthy)
	testingutils.AssertEqual(t, status2.Healthy, allStatus["target2"].Healthy)
}

func TestHealthChecker_MonitoringIntegration(t *testing.T) {
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := config.HealthConfig{
		Enabled:            true,
		Timeout:            1 * time.Second,
		Interval:           50 * time.Millisecond,
		UnhealthyThreshold: 2,
		HealthyThreshold:   2,
	}
	logger := testingutils.NewMockLogger()
	metrics := testingutils.NewMockMetricsCollector()

	// Setup mock expectations for all possible log levels
	logger.On("Debug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
	logger.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
	logger.On("Warn", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
	logger.On("Error", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
	metrics.On("RecordHealthCheck", mock.AnythingOfType("string"), mock.AnythingOfType("bool"), mock.AnythingOfType("time.Duration")).Return()

	checker := NewHealthChecker(cfg, logger, metrics)

	hc := checker.(*healthChecker)
	target := createTestTarget("integration-test", server.URL, 1*time.Second, 200)

	// Subscribe to events
	eventCh := make(chan HealthEvent, 100)
	err := checker.Subscribe(eventCh)
	testingutils.AssertNoError(t, err)

	// Register target
	err = hc.RegisterTarget(target)
	testingutils.AssertNoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Act
	err = checker.StartMonitoring(ctx)
	testingutils.AssertNoError(t, err)

	// Wait for some health checks to occur
	time.Sleep(200 * time.Millisecond)

	// Stop monitoring
	err = checker.StopMonitoring()
	testingutils.AssertNoError(t, err)

	// Assert
	// Should have received some events
	select {
	case event := <-eventCh:
		testingutils.AssertEqual(t, HealthCheckStarted, event.Type)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected to receive HealthCheckStarted event")
	}

	// Check that target status was updated
	status, exists := hc.GetTargetStatus("integration-test")
	testingutils.AssertTrue(t, exists)
	testingutils.AssertTrue(t, status.Healthy)
	testingutils.AssertTrue(t, status.ConsecutiveSuccesses > 0)
}

// mockTarget implements HealthTarget for testing
type mockTarget struct {
	name           string
	url            string
	timeout        time.Duration
	expectedStatus int
	headers        map[string]string
}

func (m *mockTarget) Name() string               { return m.name }
func (m *mockTarget) URL() string                { return m.url }
func (m *mockTarget) Timeout() time.Duration     { return m.timeout }
func (m *mockTarget) ExpectedStatus() int        { return m.expectedStatus }
func (m *mockTarget) Headers() map[string]string { return m.headers }

// Helper function to create test targets
func createTestTarget(name, url string, timeout time.Duration, expectedStatus int) HealthTarget {
	target, _ := NewCustomTarget(name, url, timeout, expectedStatus, nil)
	return target
}

// Benchmark tests
func BenchmarkHealthChecker_Check(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := config.HealthConfig{
		Enabled: true,
		Timeout: 5 * time.Second,
	}
	logger := testingutils.NewMockLogger()
	metrics := testingutils.NewMockMetricsCollector()

	// Setup mock expectations for all possible log levels
	logger.On("Debug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
	logger.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
	logger.On("Warn", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
	logger.On("Error", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
	metrics.On("RecordHealthCheck", mock.AnythingOfType("string"), mock.AnythingOfType("bool"), mock.AnythingOfType("time.Duration")).Return()

	checker := NewHealthChecker(cfg, logger, metrics)

	target := createTestTarget("bench-test", server.URL, 5*time.Second, 200)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = checker.Check(ctx, target)
	}
}

func BenchmarkHealthChecker_RegisterTarget(b *testing.B) {
	cfg := config.HealthConfig{Enabled: true}
	logger := testingutils.NewMockLogger()
	metrics := testingutils.NewMockMetricsCollector()
	checker := NewHealthChecker(cfg, logger, metrics)

	hc := checker.(*healthChecker)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		target := createTestTarget(fmt.Sprintf("target-%d", i), "http://localhost:8080", 5*time.Second, 200)
		_ = hc.RegisterTarget(target)
	}
}

func BenchmarkHealthChecker_GetAllTargetStatus(b *testing.B) {
	cfg := config.HealthConfig{Enabled: true}
	logger := testingutils.NewMockLogger()
	metrics := testingutils.NewMockMetricsCollector()
	checker := NewHealthChecker(cfg, logger, metrics)

	hc := checker.(*healthChecker)

	// Add some test data
	for i := 0; i < 100; i++ {
		hc.status[fmt.Sprintf("target-%d", i)] = HealthStatus{
			Healthy:   i%2 == 0,
			LastCheck: time.Now(),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = hc.GetAllTargetStatus()
	}
}
