package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMiddlewareFunc(t *testing.T) {
	tests := []struct {
		name     string
		setupFn  func() MiddlewareFunc
		wantBody string
	}{
		{
			name: "MiddlewareFunc_Wrap_Success",
			setupFn: func() MiddlewareFunc {
				return MiddlewareFunc(func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("X-Middleware", "test")
						next.ServeHTTP(w, r)
					})
				})
			},
			wantBody: "test response",
		},
		{
			name: "MiddlewareFunc_Wrap_ModifyResponse",
			setupFn: func() MiddlewareFunc {
				return MiddlewareFunc(func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("X-Modified", "true")
						w.WriteHeader(http.StatusOK)
						_, _ = w.Write([]byte("modified"))
					})
				})
			},
			wantBody: "modified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			middleware := tt.setupFn()
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("test response"))
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()

			// Act
			wrappedHandler := middleware.Wrap(handler)
			wrappedHandler.ServeHTTP(rec, req)

			// Assert
			if tt.name == "MiddlewareFunc_Wrap_Success" {
				assert.Equal(t, "test", rec.Header().Get("X-Middleware"))
				assert.Equal(t, tt.wantBody, rec.Body.String())
			} else {
				assert.Equal(t, "true", rec.Header().Get("X-Modified"))
				assert.Equal(t, tt.wantBody, rec.Body.String())
			}
		})
	}
}

func TestContextHelpers(t *testing.T) {
	tests := []struct {
		name         string
		setupCtx     func() context.Context
		testFunc     func(context.Context) interface{}
		expectedVal  interface{}
		expectedType string
	}{
		{
			name: "GetRequestID_WithValidID",
			setupCtx: func() context.Context {
				return WithRequestID(context.Background(), "test-req-123")
			},
			testFunc: func(ctx context.Context) interface{} {
				return GetRequestID(ctx)
			},
			expectedVal:  "test-req-123",
			expectedType: "string",
		},
		{
			name: "GetRequestID_WithEmptyContext",
			setupCtx: func() context.Context {
				return context.Background()
			},
			testFunc: func(ctx context.Context) interface{} {
				return GetRequestID(ctx)
			},
			expectedVal:  "",
			expectedType: "string",
		},
		{
			name: "GetStartTime_WithValidTime",
			setupCtx: func() context.Context {
				testTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
				return WithStartTime(context.Background(), testTime)
			},
			testFunc: func(ctx context.Context) interface{} {
				return GetStartTime(ctx)
			},
			expectedVal:  time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			expectedType: "time",
		},
		{
			name: "GetStartTime_WithEmptyContext",
			setupCtx: func() context.Context {
				return context.Background()
			},
			testFunc: func(ctx context.Context) interface{} {
				return GetStartTime(ctx)
			},
			expectedVal:  time.Time{},
			expectedType: "time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			ctx := tt.setupCtx()

			// Act
			result := tt.testFunc(ctx)

			// Assert
			switch tt.expectedType {
			case "string":
				assert.Equal(t, tt.expectedVal, result)
			case "time":
				assert.Equal(t, tt.expectedVal, result)
			}
		})
	}
}

func TestWithRequestID(t *testing.T) {
	tests := []struct {
		name      string
		requestID string
	}{
		{
			name:      "WithRequestID_ValidID",
			requestID: "req-12345",
		},
		{
			name:      "WithRequestID_EmptyID",
			requestID: "",
		},
		{
			name:      "WithRequestID_LongID",
			requestID: "very-long-request-id-with-many-characters-and-numbers-12345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			ctx := context.Background()

			// Act
			newCtx := WithRequestID(ctx, tt.requestID)
			retrievedID := GetRequestID(newCtx)

			// Assert
			assert.Equal(t, tt.requestID, retrievedID)
		})
	}
}

func TestWithStartTime(t *testing.T) {
	tests := []struct {
		name      string
		startTime time.Time
	}{
		{
			name:      "WithStartTime_ValidTime",
			startTime: time.Date(2023, 6, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			name:      "WithStartTime_ZeroTime",
			startTime: time.Time{},
		},
		{
			name:      "WithStartTime_CurrentTime",
			startTime: time.Now(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			ctx := context.Background()

			// Act
			newCtx := WithStartTime(ctx, tt.startTime)
			retrievedTime := GetStartTime(newCtx)

			// Assert
			if tt.name == "WithStartTime_CurrentTime" {
				// For current time, check it's close to the expected time
				assert.WithinDuration(t, tt.startTime, retrievedTime, time.Millisecond)
			} else {
				assert.Equal(t, tt.startTime, retrievedTime)
			}
		})
	}
}

func TestRateLimitStats(t *testing.T) {
	tests := []struct {
		name  string
		stats RateLimitStats
	}{
		{
			name: "RateLimitStats_ValidData",
			stats: RateLimitStats{
				Requests:   10,
				Remaining:  5,
				ResetTime:  time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
				RetryAfter: 30 * time.Second,
			},
		},
		{
			name: "RateLimitStats_ZeroValues",
			stats: RateLimitStats{
				Requests:   0,
				Remaining:  0,
				ResetTime:  time.Time{},
				RetryAfter: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange & Act
			stats := tt.stats

			// Assert
			assert.Equal(t, tt.stats.Requests, stats.Requests)
			assert.Equal(t, tt.stats.Remaining, stats.Remaining)
			assert.Equal(t, tt.stats.ResetTime, stats.ResetTime)
			assert.Equal(t, tt.stats.RetryAfter, stats.RetryAfter)
		})
	}
}

func TestSecurityPolicy(t *testing.T) {
	tests := []struct {
		name   string
		policy SecurityPolicy
	}{
		{
			name: "SecurityPolicy_FullyPopulated",
			policy: SecurityPolicy{
				ContentTypeNosniff:      true,
				FrameOptions:            "DENY",
				ContentSecurityPolicy:   "default-src 'self'",
				StrictTransportSecurity: "max-age=31536000",
				ReferrerPolicy:          "strict-origin",
				PermissionsPolicy:       "geolocation=(), microphone=()",
				CrossOriginEmbedder:     "require-corp",
				CrossOriginOpener:       "same-origin",
				CrossOriginResource:     "same-site",
			},
		},
		{
			name: "SecurityPolicy_Defaults",
			policy: SecurityPolicy{
				ContentTypeNosniff:      false,
				FrameOptions:            "",
				ContentSecurityPolicy:   "",
				StrictTransportSecurity: "",
				ReferrerPolicy:          "",
				PermissionsPolicy:       "",
				CrossOriginEmbedder:     "",
				CrossOriginOpener:       "",
				CrossOriginResource:     "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange & Act
			policy := tt.policy

			// Assert
			assert.Equal(t, tt.policy.ContentTypeNosniff, policy.ContentTypeNosniff)
			assert.Equal(t, tt.policy.FrameOptions, policy.FrameOptions)
			assert.Equal(t, tt.policy.ContentSecurityPolicy, policy.ContentSecurityPolicy)
			assert.Equal(t, tt.policy.StrictTransportSecurity, policy.StrictTransportSecurity)
			assert.Equal(t, tt.policy.ReferrerPolicy, policy.ReferrerPolicy)
			assert.Equal(t, tt.policy.PermissionsPolicy, policy.PermissionsPolicy)
			assert.Equal(t, tt.policy.CrossOriginEmbedder, policy.CrossOriginEmbedder)
			assert.Equal(t, tt.policy.CrossOriginOpener, policy.CrossOriginOpener)
			assert.Equal(t, tt.policy.CrossOriginResource, policy.CrossOriginResource)
		})
	}
}

func TestCORSPolicy(t *testing.T) {
	tests := []struct {
		name   string
		policy CORSPolicy
	}{
		{
			name: "CORSPolicy_FullyConfigured",
			policy: CORSPolicy{
				AllowedOrigins:     []string{"http://localhost:3000", "https://example.com"},
				AllowedMethods:     []string{"GET", "POST", "PUT", "DELETE"},
				AllowedHeaders:     []string{"Content-Type", "Authorization"},
				ExposedHeaders:     []string{"X-Total-Count"},
				AllowCredentials:   true,
				MaxAge:             3600,
				OptionsPassthrough: false,
			},
		},
		{
			name: "CORSPolicy_Minimal",
			policy: CORSPolicy{
				AllowedOrigins:     []string{"*"},
				AllowedMethods:     []string{"GET"},
				AllowedHeaders:     []string{},
				ExposedHeaders:     []string{},
				AllowCredentials:   false,
				MaxAge:             0,
				OptionsPassthrough: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange & Act
			policy := tt.policy

			// Assert
			assert.Equal(t, tt.policy.AllowedOrigins, policy.AllowedOrigins)
			assert.Equal(t, tt.policy.AllowedMethods, policy.AllowedMethods)
			assert.Equal(t, tt.policy.AllowedHeaders, policy.AllowedHeaders)
			assert.Equal(t, tt.policy.ExposedHeaders, policy.ExposedHeaders)
			assert.Equal(t, tt.policy.AllowCredentials, policy.AllowCredentials)
			assert.Equal(t, tt.policy.MaxAge, policy.MaxAge)
			assert.Equal(t, tt.policy.OptionsPassthrough, policy.OptionsPassthrough)
		})
	}
}

func TestContextKeys(t *testing.T) {
	tests := []struct {
		name        string
		contextKey  ContextKey
		expectedVal string
	}{
		{
			name:        "ContextKeyRequestID",
			contextKey:  ContextKeyRequestID,
			expectedVal: "request_id",
		},
		{
			name:        "ContextKeyStartTime",
			contextKey:  ContextKeyStartTime,
			expectedVal: "start_time",
		},
		{
			name:        "ContextKeyUserAgent",
			contextKey:  ContextKeyUserAgent,
			expectedVal: "user_agent",
		},
		{
			name:        "ContextKeyRemoteAddr",
			contextKey:  ContextKeyRemoteAddr,
			expectedVal: "remote_addr",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act & Assert
			assert.Equal(t, tt.expectedVal, string(tt.contextKey))
		})
	}
}

// Benchmark tests for context operations
func BenchmarkWithRequestID(b *testing.B) {
	ctx := context.Background()
	requestID := "test-request-123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = WithRequestID(ctx, requestID)
	}
}

func BenchmarkGetRequestID(b *testing.B) {
	ctx := WithRequestID(context.Background(), "test-request-123")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetRequestID(ctx)
	}
}

func BenchmarkWithStartTime(b *testing.B) {
	ctx := context.Background()
	startTime := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = WithStartTime(ctx, startTime)
	}
}

func BenchmarkGetStartTime(b *testing.B) {
	ctx := WithStartTime(context.Background(), time.Now())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetStartTime(ctx)
	}
}
