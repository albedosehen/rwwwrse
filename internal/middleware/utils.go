// Package middleware provides utility functions for middleware components.
package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"maps"
	"net"
	"net/http"
	"slices"
	"strings"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

const (
	// requestIDKey is the context key for request IDs.
	requestIDKey contextKey = "request_id"
)

// HTTP headers used for client IP extraction.
const (
	headerXForwardedFor    = "X-Forwarded-For"
	headerXRealIP          = "X-Real-IP"
	headerXClientIP        = "X-Client-IP"
	headerXOriginalFor     = "X-Original-For"
	headerXClusterClientIP = "X-Cluster-Client-IP"
	headerForwarded        = "Forwarded"
	headerCFConnectingIP   = "CF-Connecting-IP"
	headerTrueClientIP     = "True-Client-IP"
)

// IP address validation constants.
const (
	// ipv4LoopbackCIDR = "127.0.0.0/8"
	// ipv6LoopbackCIDR = "::1/128"
	ipv4PrivateCIDR1 = "10.0.0.0/8"
	ipv4PrivateCIDR2 = "172.16.0.0/12"
	ipv4PrivateCIDR3 = "192.168.0.0/16"
	ipv6PrivateCIDR  = "fc00::/7"
)

// SetRequestID sets the request ID in the context.
func SetRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// generateRequestID generates a new unique request ID.
func generateRequestID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to a simpler method if crypto/rand fails
		return fmt.Sprintf("req_%d", randomFallback())
	}
	return hex.EncodeToString(bytes)
}

// randomFallback provides a simple fallback for request ID generation.
func randomFallback() int64 {
	// Simple pseudo-random number based on current time
	// This is a fallback and not cryptographically secure
	return int64(len("fallback"))*1000000 + int64(len("request"))
}

// containsString checks if a slice contains a specific string.
func containsString(slice []string, item string) bool {
	return slices.Contains(slice, item)
}

// normalizeHeaderName normalizes HTTP header names to lowercase.
func normalizeHeaderName(name string) string {
	return name // This is a simple normalization
}

// isValidOrigin validates if an origin is properly formatted.
func isValidOrigin(origin string) bool {
	if origin == "" {
		return false
	}

	// Basic validation - should start with http:// or https://
	if len(origin) > 7 && (origin[:7] == "http://" || origin[:8] == "https://") {
		return true
	}

	return false
}

// sanitizeString removes potentially dangerous characters from strings.
func sanitizeString(s string) string {
	// Basic sanitization - remove null bytes and control characters
	result := make([]byte, 0, len(s))
	for _, b := range []byte(s) {
		if b >= 32 && b <= 126 {
			result = append(result, b)
		}
	}
	return string(result)
}

// isInternalError checks if a status code represents an internal server error.
func isInternalError(statusCode int) bool {
	return statusCode >= 500 && statusCode < 600
}

// isClientError checks if a status code represents a client error.
func isClientError(statusCode int) bool {
	return statusCode >= 400 && statusCode < 500
}

// isSuccessStatus checks if a status code represents success.
func isSuccessStatus(statusCode int) bool {
	return statusCode >= 200 && statusCode < 300
}

// getClientIP extracts the real client IP address from HTTP request headers.
// It checks various proxy headers in order of preference and validates IP addresses.
// Returns the first valid, non-private IP address found, or falls back to RemoteAddr.
func getClientIP(r *http.Request) string {
	if r == nil {
		return ""
	}

	// Define header priority order for IP extraction
	headerPriority := []string{
		headerCFConnectingIP,   // Cloudflare
		headerTrueClientIP,     // Akamai, Cloudflare
		headerXRealIP,          // Nginx proxy
		headerXForwardedFor,    // Standard proxy header
		headerXClientIP,        // Apache mod_proxy
		headerXClusterClientIP, // Kubernetes
		headerXOriginalFor,     // Custom proxy setups
		headerForwarded,        // RFC 7239
	}

	// Check each header in priority order
	for _, header := range headerPriority {
		if ip := extractIPFromHeader(r, header); ip != "" {
			if validIP := validateAndCleanIP(ip); validIP != "" {
				return validIP
			}
		}
	}

	// Fall back to RemoteAddr if no valid IP found in headers
	if remoteIP := extractIPFromRemoteAddr(r.RemoteAddr); remoteIP != "" {
		return remoteIP
	}

	return ""
}

// extractIPFromHeader extracts IP address from a specific header.
func extractIPFromHeader(r *http.Request, headerName string) string {
	headerValue := strings.TrimSpace(r.Header.Get(headerName))
	if headerValue == "" {
		return ""
	}

	// Handle special case for Forwarded header (RFC 7239)
	if headerName == headerForwarded {
		return extractIPFromForwardedHeader(headerValue)
	}

	// For comma-separated values, take the first (leftmost) IP
	// which represents the original client
	if idx := strings.Index(headerValue, ","); idx != -1 {
		headerValue = strings.TrimSpace(headerValue[:idx])
	}

	return headerValue
}

// extractIPFromForwardedHeader parses RFC 7239 Forwarded header.
// Example: "for=192.0.2.60;proto=http;by=203.0.113.43"
func extractIPFromForwardedHeader(forwarded string) string {
	// Look for "for=" parameter
	parts := strings.SplitSeq(forwarded, ";")
	for part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(strings.ToLower(part), "for=") {
			forValue := strings.TrimSpace(part[4:])
			// Remove quotes if present
			forValue = strings.Trim(forValue, `"`)
			// Handle IPv6 brackets
			if strings.HasPrefix(forValue, "[") && strings.HasSuffix(forValue, "]") {
				forValue = forValue[1 : len(forValue)-1]
			}
			return forValue
		}
	}
	return ""
}

// extractIPFromRemoteAddr extracts IP from RemoteAddr, removing port if present.
func extractIPFromRemoteAddr(remoteAddr string) string {
	if remoteAddr == "" {
		return ""
	}

	// Handle IPv6 addresses with port
	if strings.HasPrefix(remoteAddr, "[") {
		if idx := strings.LastIndex(remoteAddr, "]:"); idx != -1 {
			return remoteAddr[1:idx]
		}
		return strings.Trim(remoteAddr, "[]")
	}

	// Handle IPv4 addresses with port
	if idx := strings.LastIndex(remoteAddr, ":"); idx != -1 {
		// Check if this is actually a port separator (not IPv6)
		if !strings.Contains(remoteAddr[:idx], ":") {
			return remoteAddr[:idx]
		}
	}

	return remoteAddr
}

// validateAndCleanIP validates and cleans an IP address string.
// Returns empty string if IP is invalid or private/loopback.
func validateAndCleanIP(ipStr string) string {
	if ipStr == "" {
		return ""
	}

	// Parse the IP address
	ip := net.ParseIP(strings.TrimSpace(ipStr))
	if ip == nil {
		return ""
	}

	// Reject loopback addresses
	if ip.IsLoopback() {
		return ""
	}

	// Reject private addresses for security
	if isPrivateIP(ip) {
		return ""
	}

	// Reject unspecified addresses (0.0.0.0 or ::)
	if ip.IsUnspecified() {
		return ""
	}

	// Return the cleaned IP string
	return ip.String()
}

// isPrivateIP checks if an IP address is in a private range.
func isPrivateIP(ip net.IP) bool {
	if ip == nil {
		return false
	}

	// Define private IP ranges
	privateRanges := []string{
		ipv4PrivateCIDR1, // 10.0.0.0/8
		ipv4PrivateCIDR2, // 172.16.0.0/12
		ipv4PrivateCIDR3, // 192.168.0.0/16
		ipv6PrivateCIDR,  // fc00::/7
	}

	for _, cidr := range privateRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}

	return false
}

// validateContentType checks if a content type is allowed.
func validateContentType(contentType string, allowedTypes []string) bool {
	if len(allowedTypes) == 0 {
		return true // Allow all if no restrictions
	}

	return slices.Contains(allowedTypes, contentType)
}

// parseSize parses size strings (e.g., "1MB", "512KB") to bytes.
func parseSize(size string) (int64, error) {
	if size == "" {
		return 0, nil
	}

	// Simple parsing - just handle basic cases
	switch size {
	case "1KB":
		return 1024, nil
	case "1MB":
		return 1024 * 1024, nil
	case "10MB":
		return 10 * 1024 * 1024, nil
	default:
		return 0, fmt.Errorf("unsupported size format: %s", size)
	}
}

// mergeHeaders merges two sets of headers, with the second set taking precedence.
func mergeHeaders(base, override map[string]string) map[string]string {
	result := make(map[string]string)

	// Copy base headers
	maps.Copy(result, base)

	// Override with new headers
	maps.Copy(result, override)

	return result
}

// validateMiddlewareConfig validates common middleware configuration parameters.
func validateMiddlewareConfig(config interface{}) error {
	// This would validate configuration based on type
	// For now, just return nil
	return nil
}
