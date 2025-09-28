// Package health provides health target implementation.
package health

import (
	"fmt"
	"maps"
	"net/url"
	"time"

	"github.com/albedosehen/rwwwrse/internal/config"
)

// backendTarget implements HealthTarget for backend services.
type backendTarget struct {
	name           string
	url            string
	timeout        time.Duration
	expectedStatus int
	headers        map[string]string
}

// NewBackendTarget creates a health target from backend configuration.
func NewBackendTarget(name string, route config.BackendRoute) (HealthTarget, error) {
	if name == "" {
		return nil, fmt.Errorf("backend name cannot be empty")
	}

	if route.URL == "" {
		return nil, fmt.Errorf("backend URL cannot be empty")
	}

	// Parse the backend URL to validate it
	baseURL, err := url.Parse(route.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid backend URL: %w", err)
	}

	// Construct health check URL
	healthPath := route.HealthPath
	if healthPath == "" {
		healthPath = "/health"
	}

	// Ensure health path starts with /
	if healthPath[0] != '/' {
		healthPath = "/" + healthPath
	}

	healthURL := fmt.Sprintf("%s://%s%s", baseURL.Scheme, baseURL.Host, healthPath)

	// Set timeout
	timeout := route.HealthInterval
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &backendTarget{
		name:           name,
		url:            healthURL,
		timeout:        timeout,
		expectedStatus: 200, // Default to 200 OK
		headers:        make(map[string]string),
	}, nil
}

// NewCustomTarget creates a health target with custom parameters.
func NewCustomTarget(name, targetURL string, timeout time.Duration, expectedStatus int, headers map[string]string) (HealthTarget, error) {
	if name == "" {
		return nil, fmt.Errorf("target name cannot be empty")
	}

	if targetURL == "" {
		return nil, fmt.Errorf("target URL cannot be empty")
	}

	// Validate URL
	if _, err := url.Parse(targetURL); err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	if expectedStatus <= 0 {
		expectedStatus = 200
	}

	if headers == nil {
		headers = make(map[string]string)
	}

	return &backendTarget{
		name:           name,
		url:            targetURL,
		timeout:        timeout,
		expectedStatus: expectedStatus,
		headers:        headers,
	}, nil
}

// Name returns a unique identifier for this health target.
func (bt *backendTarget) Name() string {
	return bt.name
}

// URL returns the health check endpoint URL.
func (bt *backendTarget) URL() string {
	return bt.url
}

// Timeout returns the maximum time to wait for a health check response.
func (bt *backendTarget) Timeout() time.Duration {
	return bt.timeout
}

// ExpectedStatus returns the HTTP status code that indicates good health.
func (bt *backendTarget) ExpectedStatus() int {
	return bt.expectedStatus
}

// Headers returns any custom headers to include in health check requests.
func (bt *backendTarget) Headers() map[string]string {
	// Return a copy to prevent modification
	headers := make(map[string]string, len(bt.headers))
	maps.Copy(headers, bt.headers)
	return headers
}

// SetTimeout updates the timeout for this target.
func (bt *backendTarget) SetTimeout(timeout time.Duration) {
	if timeout > 0 {
		bt.timeout = timeout
	}
}

// SetExpectedStatus updates the expected status code for this target.
func (bt *backendTarget) SetExpectedStatus(status int) {
	if status > 0 {
		bt.expectedStatus = status
	}
}

// AddHeader adds a custom header for health check requests.
func (bt *backendTarget) AddHeader(key, value string) {
	if key != "" && value != "" {
		bt.headers[key] = value
	}
}

// RemoveHeader removes a custom header.
func (bt *backendTarget) RemoveHeader(key string) {
	delete(bt.headers, key)
}

// String returns a string representation of the target.
func (bt *backendTarget) String() string {
	return fmt.Sprintf("HealthTarget{name=%s, url=%s, timeout=%v, status=%d}",
		bt.name, bt.url, bt.timeout, bt.expectedStatus)
}
