package health

import (
	"testing"
	"time"

	"github.com/albedosehen/rwwwrse/internal/config"
	testingutils "github.com/albedosehen/rwwwrse/internal/testing"
)

func TestNewBackendTarget(t *testing.T) {
	tests := []struct {
		name        string
		backendName string
		route       config.BackendRoute
		wantErr     bool
		wantURL     string
		wantTimeout time.Duration
		validate    func(*testing.T, HealthTarget)
	}{
		{
			name:        "valid backend target with default health path",
			backendName: "api-backend",
			route: config.BackendRoute{
				URL:            "http://localhost:8080",
				HealthInterval: 30 * time.Second,
			},
			wantErr:     false,
			wantURL:     "http://localhost:8080/health",
			wantTimeout: 30 * time.Second,
			validate: func(t *testing.T, target HealthTarget) {
				testingutils.AssertEqual(t, "api-backend", target.Name())
				testingutils.AssertEqual(t, 200, target.ExpectedStatus())
				testingutils.AssertEmpty(t, target.Headers())
			},
		},
		{
			name:        "valid backend target with custom health path",
			backendName: "web-backend",
			route: config.BackendRoute{
				URL:            "https://example.com:8443",
				HealthPath:     "/status",
				HealthInterval: 15 * time.Second,
			},
			wantErr:     false,
			wantURL:     "https://example.com:8443/status",
			wantTimeout: 15 * time.Second,
			validate: func(t *testing.T, target HealthTarget) {
				testingutils.AssertEqual(t, "web-backend", target.Name())
				testingutils.AssertEqual(t, 200, target.ExpectedStatus())
			},
		},
		{
			name:        "health path without leading slash",
			backendName: "test-backend",
			route: config.BackendRoute{
				URL:        "http://test.com",
				HealthPath: "check",
			},
			wantErr:     false,
			wantURL:     "http://test.com/check",
			wantTimeout: 30 * time.Second,
		},
		{
			name:        "empty backend name",
			backendName: "",
			route: config.BackendRoute{
				URL: "http://localhost:8080",
			},
			wantErr: true,
		},
		{
			name:        "empty backend URL",
			backendName: "test-backend",
			route: config.BackendRoute{
				URL: "",
			},
			wantErr: true,
		},
		{
			name:        "invalid backend URL",
			backendName: "test-backend",
			route: config.BackendRoute{
				URL: "not-a-valid-url",
			},
			wantErr: true,
		},
		{
			name:        "zero timeout gets default",
			backendName: "timeout-backend",
			route: config.BackendRoute{
				URL:            "http://localhost:8080",
				HealthInterval: 0,
			},
			wantErr:     false,
			wantTimeout: 30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange & Act
			target, err := NewBackendTarget(tt.backendName, tt.route)

			// Assert
			if tt.wantErr {
				testingutils.AssertError(t, err)
				testingutils.AssertNil(t, target)
				return
			}

			testingutils.AssertNoError(t, err)
			testingutils.AssertNotNil(t, target)

			if tt.wantURL != "" {
				testingutils.AssertEqual(t, tt.wantURL, target.URL())
			}

			if tt.wantTimeout > 0 {
				testingutils.AssertEqual(t, tt.wantTimeout, target.Timeout())
			}

			if tt.validate != nil {
				tt.validate(t, target)
			}
		})
	}
}

func TestNewCustomTarget(t *testing.T) {
	tests := []struct {
		name           string
		targetName     string
		targetURL      string
		timeout        time.Duration
		expectedStatus int
		headers        map[string]string
		wantErr        bool
		validate       func(*testing.T, HealthTarget)
	}{
		{
			name:           "valid custom target",
			targetName:     "custom-service",
			targetURL:      "http://localhost:9000/custom-health",
			timeout:        10 * time.Second,
			expectedStatus: 204,
			headers: map[string]string{
				"Authorization": "Bearer token",
				"X-Custom":      "value",
			},
			wantErr: false,
			validate: func(t *testing.T, target HealthTarget) {
				testingutils.AssertEqual(t, "custom-service", target.Name())
				testingutils.AssertEqual(t, "http://localhost:9000/custom-health", target.URL())
				testingutils.AssertEqual(t, 10*time.Second, target.Timeout())
				testingutils.AssertEqual(t, 204, target.ExpectedStatus())

				headers := target.Headers()
				testingutils.AssertEqual(t, "Bearer token", headers["Authorization"])
				testingutils.AssertEqual(t, "value", headers["X-Custom"])
			},
		},
		{
			name:           "empty target name",
			targetName:     "",
			targetURL:      "http://localhost:9000",
			timeout:        5 * time.Second,
			expectedStatus: 200,
			wantErr:        true,
		},
		{
			name:           "empty target URL",
			targetName:     "test",
			targetURL:      "",
			timeout:        5 * time.Second,
			expectedStatus: 200,
			wantErr:        true,
		},
		{
			name:           "invalid URL",
			targetName:     "test",
			targetURL:      "not-a-url",
			timeout:        5 * time.Second,
			expectedStatus: 200,
			wantErr:        true,
		},
		{
			name:           "zero timeout gets default",
			targetName:     "test",
			targetURL:      "http://localhost:9000",
			timeout:        0,
			expectedStatus: 200,
			wantErr:        false,
			validate: func(t *testing.T, target HealthTarget) {
				testingutils.AssertEqual(t, 30*time.Second, target.Timeout())
			},
		},
		{
			name:           "negative timeout gets default",
			targetName:     "test",
			targetURL:      "http://localhost:9000",
			timeout:        -5 * time.Second,
			expectedStatus: 200,
			wantErr:        false,
			validate: func(t *testing.T, target HealthTarget) {
				testingutils.AssertEqual(t, 30*time.Second, target.Timeout())
			},
		},
		{
			name:           "zero status gets default",
			targetName:     "test",
			targetURL:      "http://localhost:9000",
			timeout:        5 * time.Second,
			expectedStatus: 0,
			wantErr:        false,
			validate: func(t *testing.T, target HealthTarget) {
				testingutils.AssertEqual(t, 200, target.ExpectedStatus())
			},
		},
		{
			name:           "negative status gets default",
			targetName:     "test",
			targetURL:      "http://localhost:9000",
			timeout:        5 * time.Second,
			expectedStatus: -1,
			wantErr:        false,
			validate: func(t *testing.T, target HealthTarget) {
				testingutils.AssertEqual(t, 200, target.ExpectedStatus())
			},
		},
		{
			name:           "nil headers map",
			targetName:     "test",
			targetURL:      "http://localhost:9000",
			timeout:        5 * time.Second,
			expectedStatus: 200,
			headers:        nil,
			wantErr:        false,
			validate: func(t *testing.T, target HealthTarget) {
				testingutils.AssertNotNil(t, target.Headers())
				testingutils.AssertEmpty(t, target.Headers())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange & Act
			target, err := NewCustomTarget(tt.targetName, tt.targetURL, tt.timeout, tt.expectedStatus, tt.headers)

			// Assert
			if tt.wantErr {
				testingutils.AssertError(t, err)
				testingutils.AssertNil(t, target)
				return
			}

			testingutils.AssertNoError(t, err)
			testingutils.AssertNotNil(t, target)

			if tt.validate != nil {
				tt.validate(t, target)
			}
		})
	}
}

func TestBackendTarget_Methods(t *testing.T) {
	// Arrange
	route := config.BackendRoute{
		URL:            "http://localhost:8080",
		HealthPath:     "/health",
		HealthInterval: 15 * time.Second,
	}

	target, err := NewBackendTarget("test-backend", route)
	testingutils.AssertNoError(t, err)
	testingutils.AssertNotNil(t, target)

	bt := target.(*backendTarget)

	t.Run("Name", func(t *testing.T) {
		testingutils.AssertEqual(t, "test-backend", bt.Name())
	})

	t.Run("URL", func(t *testing.T) {
		testingutils.AssertEqual(t, "http://localhost:8080/health", bt.URL())
	})

	t.Run("Timeout", func(t *testing.T) {
		testingutils.AssertEqual(t, 15*time.Second, bt.Timeout())
	})

	t.Run("ExpectedStatus", func(t *testing.T) {
		testingutils.AssertEqual(t, 200, bt.ExpectedStatus())
	})

	t.Run("Headers returns copy", func(t *testing.T) {
		// Act
		headers1 := bt.Headers()
		headers2 := bt.Headers()

		// Assert - should be different map instances but same content
		testingutils.AssertNotNil(t, headers1)
		testingutils.AssertNotNil(t, headers2)

		// Modify one map
		headers1["test"] = "value"

		// Other map should not be affected
		_, exists := headers2["test"]
		testingutils.AssertFalse(t, exists)
	})
}

func TestBackendTarget_SetTimeout(t *testing.T) {
	// Arrange
	route := config.BackendRoute{URL: "http://localhost:8080"}
	target, err := NewBackendTarget("test", route)
	testingutils.AssertNoError(t, err)

	bt := target.(*backendTarget)

	tests := []struct {
		name        string
		newTimeout  time.Duration
		wantTimeout time.Duration
	}{
		{
			name:        "valid positive timeout",
			newTimeout:  45 * time.Second,
			wantTimeout: 45 * time.Second,
		},
		{
			name:        "zero timeout ignored",
			newTimeout:  0,
			wantTimeout: 30 * time.Second, // should remain unchanged
		},
		{
			name:        "negative timeout ignored",
			newTimeout:  -10 * time.Second,
			wantTimeout: 30 * time.Second, // should remain unchanged
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			bt.SetTimeout(tt.newTimeout)

			// Assert
			testingutils.AssertEqual(t, tt.wantTimeout, bt.Timeout())
		})
	}
}

func TestBackendTarget_SetExpectedStatus(t *testing.T) {
	// Arrange
	route := config.BackendRoute{URL: "http://localhost:8080"}
	target, err := NewBackendTarget("test", route)
	testingutils.AssertNoError(t, err)

	bt := target.(*backendTarget)

	tests := []struct {
		name       string
		newStatus  int
		wantStatus int
	}{
		{
			name:       "valid positive status",
			newStatus:  204,
			wantStatus: 204,
		},
		{
			name:       "zero status ignored",
			newStatus:  0,
			wantStatus: 200, // should remain unchanged
		},
		{
			name:       "negative status ignored",
			newStatus:  -1,
			wantStatus: 200, // should remain unchanged
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			bt.SetExpectedStatus(tt.newStatus)

			// Assert
			testingutils.AssertEqual(t, tt.wantStatus, bt.ExpectedStatus())
		})
	}
}

func TestBackendTarget_HeaderOperations(t *testing.T) {
	// Arrange
	route := config.BackendRoute{URL: "http://localhost:8080"}
	target, err := NewBackendTarget("test", route)
	testingutils.AssertNoError(t, err)

	bt := target.(*backendTarget)

	t.Run("AddHeader valid", func(t *testing.T) {
		// Act
		bt.AddHeader("Authorization", "Bearer token")
		bt.AddHeader("X-Custom", "value")

		// Assert
		headers := bt.Headers()
		testingutils.AssertEqual(t, "Bearer token", headers["Authorization"])
		testingutils.AssertEqual(t, "value", headers["X-Custom"])
	})

	t.Run("AddHeader with empty key ignored", func(t *testing.T) {
		// Arrange
		initialCount := len(bt.Headers())

		// Act
		bt.AddHeader("", "value")

		// Assert
		testingutils.AssertEqual(t, initialCount, len(bt.Headers()))
	})

	t.Run("AddHeader with empty value ignored", func(t *testing.T) {
		// Arrange
		initialCount := len(bt.Headers())

		// Act
		bt.AddHeader("key", "")

		// Assert
		testingutils.AssertEqual(t, initialCount, len(bt.Headers()))
	})

	t.Run("RemoveHeader", func(t *testing.T) {
		// Arrange
		bt.AddHeader("ToRemove", "value")
		headers := bt.Headers()
		testingutils.AssertContains(t, headers, "ToRemove")

		// Act
		bt.RemoveHeader("ToRemove")

		// Assert
		headers = bt.Headers()
		testingutils.AssertNotContains(t, headers, "ToRemove")
	})

	t.Run("RemoveHeader non-existent key", func(t *testing.T) {
		// Act & Assert - should not panic
		bt.RemoveHeader("NonExistent")
	})
}

func TestBackendTarget_String(t *testing.T) {
	// Arrange
	route := config.BackendRoute{
		URL:            "http://localhost:8080",
		HealthPath:     "/health",
		HealthInterval: 15 * time.Second,
	}

	target, err := NewBackendTarget("test-backend", route)
	testingutils.AssertNoError(t, err)

	bt := target.(*backendTarget)

	// Act
	str := bt.String()

	// Assert
	testingutils.AssertContains(t, str, "test-backend")
	testingutils.AssertContains(t, str, "http://localhost:8080/health")
	testingutils.AssertContains(t, str, "15s")
	testingutils.AssertContains(t, str, "200")
}

// Benchmark tests
func BenchmarkNewBackendTarget(b *testing.B) {
	route := config.BackendRoute{
		URL:            "http://localhost:8080",
		HealthPath:     "/health",
		HealthInterval: 30 * time.Second,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := NewBackendTarget("test-backend", route)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNewCustomTarget(b *testing.B) {
	headers := map[string]string{
		"Authorization": "Bearer token",
		"X-Custom":      "value",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := NewCustomTarget("test", "http://localhost:8080", 30*time.Second, 200, headers)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBackendTarget_Headers(b *testing.B) {
	route := config.BackendRoute{URL: "http://localhost:8080"}
	target, err := NewBackendTarget("test", route)
	if err != nil {
		b.Fatal(err)
	}

	bt := target.(*backendTarget)
	bt.AddHeader("Authorization", "Bearer token")
	bt.AddHeader("X-Custom", "value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bt.Headers()
	}
}

func BenchmarkBackendTarget_AddHeader(b *testing.B) {
	route := config.BackendRoute{URL: "http://localhost:8080"}
	target, err := NewBackendTarget("test", route)
	if err != nil {
		b.Fatal(err)
	}

	bt := target.(*backendTarget)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bt.AddHeader("Header"+string(rune(i)), "value")
	}
}
