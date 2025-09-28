package config

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/albedosehen/rwwwrse/internal/observability"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Debug(ctx context.Context, msg string, fields ...observability.Field) {
	args := []interface{}{ctx, msg}
	for _, field := range fields {
		args = append(args, field)
	}
	m.Called(args...)
}

func (m *MockLogger) Info(ctx context.Context, msg string, fields ...observability.Field) {
	args := []interface{}{ctx, msg}
	for _, field := range fields {
		args = append(args, field)
	}
	m.Called(args...)
}

func (m *MockLogger) Warn(ctx context.Context, msg string, fields ...observability.Field) {
	args := []interface{}{ctx, msg}
	for _, field := range fields {
		args = append(args, field)
	}
	m.Called(args...)
}

func (m *MockLogger) Error(ctx context.Context, err error, msg string, fields ...observability.Field) {
	args := []interface{}{ctx, err, msg}
	for _, field := range fields {
		args = append(args, field)
	}
	m.Called(args...)
}

func (m *MockLogger) WithFields(fields ...observability.Field) observability.Logger {
	args := make([]interface{}, len(fields))
	for i, field := range fields {
		args[i] = field
	}
	m.Called(args...)
	return m
}

func (m *MockLogger) WithContext(ctx context.Context) observability.Logger {
	m.Called(ctx)
	return m
}

func TestDopplerProviderAvailability(t *testing.T) {
	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	mockLogger := new(MockLogger)
	mockLogger.On("Debug", mock.Anything, mock.Anything).Return()

	tests := []struct {
		name          string
		dopplerToken  string
		commandExists bool
		expectCLI     bool
		expectConfig  bool
	}{
		{
			name:          "available when token exists and command exists",
			dopplerToken:  "test-token",
			commandExists: true,
			expectCLI:     true,
			expectConfig:  true,
		},
		{
			name:          "not configured when token doesn't exist",
			dopplerToken:  "",
			commandExists: true,
			expectCLI:     true,
			expectConfig:  false,
		},
		{
			name:          "CLI not available when command doesn't exist",
			dopplerToken:  "test-token",
			commandExists: false,
			expectCLI:     false,
			expectConfig:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment for this test
			if tt.dopplerToken != "" {
				os.Setenv("DOPPLER_TOKEN", tt.dopplerToken)
			} else {
				os.Unsetenv("DOPPLER_TOKEN")
			}

			// Temporarily modify PATH to control command availability
			if tt.commandExists {
				// Keep original PATH to ensure command exists
				// (Assumes doppler is installed)
			} else {
				// Set PATH to empty to ensure command doesn't exist
				os.Setenv("PATH", "")
			}

			provider := NewDopplerProvider(mockLogger)
			assert.Equal(t, tt.expectCLI, provider.isDopplerCLIAvailable())
			assert.Equal(t, tt.expectConfig, provider.isDopplerConfigured())
		})
	}
}

// TestLoadSecrets tests the LoadSecrets method
// This is a more complex test that requires mocking exec.Command
// In a real-world scenario, you might want to use a library like github.com/jarcoal/httpmock
// to mock the HTTP calls to Doppler API
func TestLoadSecrets(t *testing.T) {
	// Skip if doppler command not available in test environment
	_, err := exec.LookPath("doppler")
	if err != nil {
		t.Skip("Skipping test: doppler command not available")
	}

	// Create a mock logger
	mockLogger := new(MockLogger)
	mockLogger.On("Debug", mock.Anything, mock.Anything).Return()
	mockLogger.On("Info", mock.Anything, mock.Anything).Return()
	mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything).Return() // For call with secret_count
	mockLogger.On("Warn", mock.Anything, mock.Anything).Return()
	mockLogger.On("Error", mock.Anything, mock.Anything, mock.Anything).Return()

	// Set DOPPLER_TOKEN for testing
	// In a real CI/CD environment, you'd want to use a test token
	origToken := os.Getenv("DOPPLER_TOKEN")
	defer func() {
		if origToken != "" {
			os.Setenv("DOPPLER_TOKEN", origToken)
		} else {
			os.Unsetenv("DOPPLER_TOKEN")
		}
	}()

	// We're only testing that the interface works without errors
	// In a real test, we would mock the doppler command response
	// and verify the environment variables are set correctly

	// If you have a real token for testing, uncomment this:
	// os.Setenv("DOPPLER_TOKEN", "your-test-token")

	// For now, we'll assume no token is available, so this should be a no-op
	os.Unsetenv("DOPPLER_TOKEN")

	provider := NewDopplerProvider(mockLogger)
	err = provider.LoadSecrets()
	assert.NoError(t, err)
}
