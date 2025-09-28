package middleware

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetRequestID(t *testing.T) {
	tests := []struct {
		name      string
		requestID string
	}{
		{
			name:      "SetRequestID_ValidID",
			requestID: "test-request-123",
		},
		{
			name:      "SetRequestID_EmptyID",
			requestID: "",
		},
		{
			name:      "SetRequestID_LongID",
			requestID: "very-long-request-id-with-many-characters-and-numbers-12345678901234567890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			ctx := context.Background()

			// Act
			newCtx := SetRequestID(ctx, tt.requestID)

			// Assert
			value := newCtx.Value(requestIDKey)
			assert.Equal(t, tt.requestID, value)
		})
	}
}

func TestGenerateRequestID(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "GenerateRequestID_UniqueValues",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			id1 := generateRequestID()
			id2 := generateRequestID()
			id3 := generateRequestID()

			// Assert
			assert.NotEmpty(t, id1)
			assert.NotEmpty(t, id2)
			assert.NotEmpty(t, id3)

			// IDs should be unique
			assert.NotEqual(t, id1, id2)
			assert.NotEqual(t, id2, id3)
			assert.NotEqual(t, id1, id3)

			// IDs should be hex encoded (length should be even and contain only hex characters)
			assert.Equal(t, 16, len(id1)) // 8 bytes = 16 hex characters
			assert.Regexp(t, "^[0-9a-f]+$", id1)
		})
	}
}

func TestRandomFallback(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "RandomFallback_ConsistentValue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			value1 := randomFallback()
			value2 := randomFallback()

			// Assert
			assert.NotZero(t, value1)
			assert.NotZero(t, value2)
			// This is a deterministic fallback, so values should be the same
			assert.Equal(t, value1, value2)
		})
	}
}

func TestContainsString(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{
			name:     "ContainsString_Found",
			slice:    []string{"apple", "banana", "cherry"},
			item:     "banana",
			expected: true,
		},
		{
			name:     "ContainsString_NotFound",
			slice:    []string{"apple", "banana", "cherry"},
			item:     "orange",
			expected: false,
		},
		{
			name:     "ContainsString_EmptySlice",
			slice:    []string{},
			item:     "apple",
			expected: false,
		},
		{
			name:     "ContainsString_EmptyItem",
			slice:    []string{"apple", "", "cherry"},
			item:     "",
			expected: true,
		},
		{
			name:     "ContainsString_CaseSensitive",
			slice:    []string{"Apple", "Banana", "Cherry"},
			item:     "apple",
			expected: false,
		},
		{
			name:     "ContainsString_SingleItem",
			slice:    []string{"onlyitem"},
			item:     "onlyitem",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			result := containsString(tt.slice, tt.item)

			// Assert
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizeHeaderName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "NormalizeHeaderName_StandardHeader",
			input:    "Content-Type",
			expected: "Content-Type",
		},
		{
			name:     "NormalizeHeaderName_CustomHeader",
			input:    "X-Custom-Header",
			expected: "X-Custom-Header",
		},
		{
			name:     "NormalizeHeaderName_EmptyHeader",
			input:    "",
			expected: "",
		},
		{
			name:     "NormalizeHeaderName_LowercaseHeader",
			input:    "content-type",
			expected: "content-type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			result := normalizeHeaderName(tt.input)

			// Assert
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidOrigin(t *testing.T) {
	tests := []struct {
		name     string
		origin   string
		expected bool
	}{
		{
			name:     "IsValidOrigin_HTTPSValid",
			origin:   "https://example.com",
			expected: true,
		},
		{
			name:     "IsValidOrigin_HTTPValid",
			origin:   "http://localhost:3000",
			expected: true,
		},
		{
			name:     "IsValidOrigin_HTTPSWithPort",
			origin:   "https://api.example.com:8443",
			expected: true,
		},
		{
			name:     "IsValidOrigin_Empty",
			origin:   "",
			expected: false,
		},
		{
			name:     "IsValidOrigin_NoProtocol",
			origin:   "example.com",
			expected: false,
		},
		{
			name:     "IsValidOrigin_InvalidProtocol",
			origin:   "ftp://example.com",
			expected: false,
		},
		{
			name:     "IsValidOrigin_OnlyProtocol",
			origin:   "https://",
			expected: true,
		},
		{
			name:     "IsValidOrigin_ShortHTTP",
			origin:   "http://x",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			result := isValidOrigin(tt.origin)

			// Assert
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "SanitizeString_NormalText",
			input:    "Hello, World!",
			expected: "Hello, World!",
		},
		{
			name:     "SanitizeString_WithControlCharacters",
			input:    "Hello\x00\x01\x02World",
			expected: "HelloWorld",
		},
		{
			name:     "SanitizeString_WithNewlines",
			input:    "Hello\nWorld\r\n",
			expected: "HelloWorld",
		},
		{
			name:     "SanitizeString_WithTabs",
			input:    "Hello\tWorld",
			expected: "HelloWorld",
		},
		{
			name:     "SanitizeString_Numbers",
			input:    "123456789",
			expected: "123456789",
		},
		{
			name:     "SanitizeString_SpecialCharacters",
			input:    "!@#$%^&*()_+-=[]{}|;:,.<>?",
			expected: "!@#$%^&*()_+-=[]{}|;:,.<>?",
		},
		{
			name:     "SanitizeString_Empty",
			input:    "",
			expected: "",
		},
		{
			name:     "SanitizeString_OnlyControlCharacters",
			input:    "\x00\x01\x02\x03",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			result := sanitizeString(tt.input)

			// Assert
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsInternalError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		expected   bool
	}{
		{
			name:       "IsInternalError_500",
			statusCode: 500,
			expected:   true,
		},
		{
			name:       "IsInternalError_502",
			statusCode: 502,
			expected:   true,
		},
		{
			name:       "IsInternalError_599",
			statusCode: 599,
			expected:   true,
		},
		{
			name:       "IsInternalError_400",
			statusCode: 400,
			expected:   false,
		},
		{
			name:       "IsInternalError_200",
			statusCode: 200,
			expected:   false,
		},
		{
			name:       "IsInternalError_600",
			statusCode: 600,
			expected:   false,
		},
		{
			name:       "IsInternalError_499",
			statusCode: 499,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			result := isInternalError(tt.statusCode)

			// Assert
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsClientError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		expected   bool
	}{
		{
			name:       "IsClientError_400",
			statusCode: 400,
			expected:   true,
		},
		{
			name:       "IsClientError_404",
			statusCode: 404,
			expected:   true,
		},
		{
			name:       "IsClientError_499",
			statusCode: 499,
			expected:   true,
		},
		{
			name:       "IsClientError_200",
			statusCode: 200,
			expected:   false,
		},
		{
			name:       "IsClientError_500",
			statusCode: 500,
			expected:   false,
		},
		{
			name:       "IsClientError_399",
			statusCode: 399,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			result := isClientError(tt.statusCode)

			// Assert
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsSuccessStatus(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		expected   bool
	}{
		{
			name:       "IsSuccessStatus_200",
			statusCode: 200,
			expected:   true,
		},
		{
			name:       "IsSuccessStatus_201",
			statusCode: 201,
			expected:   true,
		},
		{
			name:       "IsSuccessStatus_299",
			statusCode: 299,
			expected:   true,
		},
		{
			name:       "IsSuccessStatus_199",
			statusCode: 199,
			expected:   false,
		},
		{
			name:       "IsSuccessStatus_300",
			statusCode: 300,
			expected:   false,
		},
		{
			name:       "IsSuccessStatus_400",
			statusCode: 400,
			expected:   false,
		},
		{
			name:       "IsSuccessStatus_500",
			statusCode: 500,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			result := isSuccessStatus(tt.statusCode)

			// Assert
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name     string
		setupReq func() *http.Request
		expected string
	}{
		{
			name: "GetClientIP_NilRequest",
			setupReq: func() *http.Request {
				return nil
			},
			expected: "",
		},
		{
			name: "GetClientIP_CloudflareConnectingIP",
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("CF-Connecting-IP", "203.0.113.1")
				req.RemoteAddr = "192.168.1.1:8080"
				return req
			},
			expected: "203.0.113.1",
		},
		{
			name: "GetClientIP_TrueClientIP",
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("True-Client-IP", "203.0.113.2")
				req.RemoteAddr = "192.168.1.1:8080"
				return req
			},
			expected: "203.0.113.2",
		},
		{
			name: "GetClientIP_XRealIP",
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("X-Real-IP", "203.0.113.3")
				req.RemoteAddr = "192.168.1.1:8080"
				return req
			},
			expected: "203.0.113.3",
		},
		{
			name: "GetClientIP_XForwardedFor_Single",
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("X-Forwarded-For", "203.0.113.4")
				req.RemoteAddr = "192.168.1.1:8080"
				return req
			},
			expected: "203.0.113.4",
		},
		{
			name: "GetClientIP_XForwardedFor_Multiple",
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("X-Forwarded-For", "203.0.113.5, 192.168.1.2, 10.0.0.1")
				req.RemoteAddr = "192.168.1.1:8080"
				return req
			},
			expected: "203.0.113.5",
		},
		{
			name: "GetClientIP_ForwardedHeader",
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("Forwarded", "for=203.0.113.6;proto=https;by=192.168.1.1")
				req.RemoteAddr = "192.168.1.1:8080"
				return req
			},
			expected: "203.0.113.6",
		},
		{
			name: "GetClientIP_ForwardedHeader_IPv6",
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("Forwarded", `for="[2001:db8::1]";proto=https`)
				req.RemoteAddr = "192.168.1.1:8080"
				return req
			},
			expected: "2001:db8::1",
		},
		{
			name: "GetClientIP_RemoteAddr_IPv4",
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.RemoteAddr = "203.0.113.7:8080"
				return req
			},
			expected: "203.0.113.7",
		},
		{
			name: "GetClientIP_RemoteAddr_IPv6",
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.RemoteAddr = "[2001:db8::2]:8080"
				return req
			},
			expected: "2001:db8::2",
		},
		{
			name: "GetClientIP_PrivateIP_Filtered",
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("X-Forwarded-For", "192.168.1.100")
				req.RemoteAddr = "203.0.113.8:8080"
				return req
			},
			expected: "203.0.113.8",
		},
		{
			name: "GetClientIP_LoopbackIP_Filtered",
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("X-Real-IP", "127.0.0.1")
				req.RemoteAddr = "203.0.113.9:8080"
				return req
			},
			expected: "203.0.113.9",
		},
		{
			name: "GetClientIP_InvalidIP_Filtered",
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("X-Forwarded-For", "invalid-ip")
				req.RemoteAddr = "203.0.113.10:8080"
				return req
			},
			expected: "203.0.113.10",
		},
		{
			name: "GetClientIP_EmptyHeaders_FallbackToRemoteAddr",
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.RemoteAddr = "203.0.113.11:8080"
				return req
			},
			expected: "203.0.113.11",
		},
		{
			name: "GetClientIP_HeaderPriority",
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("X-Forwarded-For", "203.0.113.12")
				req.Header.Set("CF-Connecting-IP", "203.0.113.13")
				req.RemoteAddr = "192.168.1.1:8080"
				return req
			},
			expected: "203.0.113.13", // CF-Connecting-IP has higher priority
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			req := tt.setupReq()

			// Act
			result := getClientIP(req)

			// Assert
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractIPFromHeader(t *testing.T) {
	tests := []struct {
		name       string
		headerName string
		setupReq   func() *http.Request
		expected   string
	}{
		{
			name:       "ExtractIPFromHeader_XForwardedFor",
			headerName: "X-Forwarded-For",
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("X-Forwarded-For", "203.0.113.1")
				return req
			},
			expected: "203.0.113.1",
		},
		{
			name:       "ExtractIPFromHeader_EmptyHeader",
			headerName: "X-Real-IP",
			setupReq: func() *http.Request {
				return httptest.NewRequest("GET", "/", nil)
			},
			expected: "",
		},
		{
			name:       "ExtractIPFromHeader_CommaSeparated",
			headerName: "X-Forwarded-For",
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("X-Forwarded-For", "203.0.113.1, 192.168.1.1, 10.0.0.1")
				return req
			},
			expected: "203.0.113.1",
		},
		{
			name:       "ExtractIPFromHeader_ForwardedSpecial",
			headerName: "Forwarded",
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("Forwarded", "for=203.0.113.1;proto=https")
				return req
			},
			expected: "203.0.113.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			req := tt.setupReq()

			// Act
			result := extractIPFromHeader(req, tt.headerName)

			// Assert
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractIPFromForwardedHeader(t *testing.T) {
	tests := []struct {
		name      string
		forwarded string
		expected  string
	}{
		{
			name:      "ExtractIPFromForwardedHeader_Simple",
			forwarded: "for=203.0.113.1",
			expected:  "203.0.113.1",
		},
		{
			name:      "ExtractIPFromForwardedHeader_WithQuotes",
			forwarded: `for="203.0.113.1"`,
			expected:  "203.0.113.1",
		},
		{
			name:      "ExtractIPFromForwardedHeader_IPv6",
			forwarded: `for="[2001:db8::1]"`,
			expected:  "2001:db8::1",
		},
		{
			name:      "ExtractIPFromForwardedHeader_Multiple",
			forwarded: "for=203.0.113.1;proto=https;by=192.168.1.1",
			expected:  "203.0.113.1",
		},
		{
			name:      "ExtractIPFromForwardedHeader_NoFor",
			forwarded: "proto=https;by=192.168.1.1",
			expected:  "",
		},
		{
			name:      "ExtractIPFromForwardedHeader_Empty",
			forwarded: "",
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			result := extractIPFromForwardedHeader(tt.forwarded)

			// Assert
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractIPFromRemoteAddr(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		expected   string
	}{
		{
			name:       "ExtractIPFromRemoteAddr_IPv4WithPort",
			remoteAddr: "203.0.113.1:8080",
			expected:   "203.0.113.1",
		},
		{
			name:       "ExtractIPFromRemoteAddr_IPv6WithPort",
			remoteAddr: "[2001:db8::1]:8080",
			expected:   "2001:db8::1",
		},
		{
			name:       "ExtractIPFromRemoteAddr_IPv4NoPort",
			remoteAddr: "203.0.113.1",
			expected:   "203.0.113.1",
		},
		{
			name:       "ExtractIPFromRemoteAddr_IPv6NoPort",
			remoteAddr: "2001:db8::1",
			expected:   "2001:db8::1",
		},
		{
			name:       "ExtractIPFromRemoteAddr_Empty",
			remoteAddr: "",
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			result := extractIPFromRemoteAddr(tt.remoteAddr)

			// Assert
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateAndCleanIP(t *testing.T) {
	tests := []struct {
		name     string
		ipStr    string
		expected string
	}{
		{
			name:     "ValidateAndCleanIP_ValidPublicIPv4",
			ipStr:    "203.0.113.1",
			expected: "203.0.113.1",
		},
		{
			name:     "ValidateAndCleanIP_ValidPublicIPv6",
			ipStr:    "2001:db8::1",
			expected: "2001:db8::1",
		},
		{
			name:     "ValidateAndCleanIP_PrivateIPv4",
			ipStr:    "192.168.1.1",
			expected: "",
		},
		{
			name:     "ValidateAndCleanIP_LoopbackIPv4",
			ipStr:    "127.0.0.1",
			expected: "",
		},
		{
			name:     "ValidateAndCleanIP_LoopbackIPv6",
			ipStr:    "::1",
			expected: "",
		},
		{
			name:     "ValidateAndCleanIP_UnspecifiedIPv4",
			ipStr:    "0.0.0.0",
			expected: "",
		},
		{
			name:     "ValidateAndCleanIP_UnspecifiedIPv6",
			ipStr:    "::",
			expected: "",
		},
		{
			name:     "ValidateAndCleanIP_InvalidIP",
			ipStr:    "invalid-ip",
			expected: "",
		},
		{
			name:     "ValidateAndCleanIP_Empty",
			ipStr:    "",
			expected: "",
		},
		{
			name:     "ValidateAndCleanIP_WithWhitespace",
			ipStr:    "  203.0.113.1  ",
			expected: "203.0.113.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			result := validateAndCleanIP(tt.ipStr)

			// Assert
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		name     string
		ipStr    string
		expected bool
	}{
		{
			name:     "IsPrivateIP_PublicIPv4",
			ipStr:    "203.0.113.1",
			expected: false,
		},
		{
			name:     "IsPrivateIP_PrivateIPv4_10",
			ipStr:    "10.0.0.1",
			expected: true,
		},
		{
			name:     "IsPrivateIP_PrivateIPv4_172",
			ipStr:    "172.16.0.1",
			expected: true,
		},
		{
			name:     "IsPrivateIP_PrivateIPv4_192",
			ipStr:    "192.168.1.1",
			expected: true,
		},
		{
			name:     "IsPrivateIP_PrivateIPv6",
			ipStr:    "fc00::1",
			expected: true,
		},
		{
			name:     "IsPrivateIP_PublicIPv6",
			ipStr:    "2001:db8::1",
			expected: false,
		},
		{
			name:     "IsPrivateIP_NilIP",
			ipStr:    "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			var ip net.IP
			if tt.ipStr != "" {
				ip = net.ParseIP(tt.ipStr)
			}

			// Act
			result := isPrivateIP(ip)

			// Assert
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Benchmark tests for the new getClientIP function
func BenchmarkGetClientIP(b *testing.B) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.1, 192.168.1.1")
	req.RemoteAddr = "192.168.1.1:8080"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = getClientIP(req)
	}
}

func BenchmarkValidateAndCleanIP(b *testing.B) {
	ipStr := "203.0.113.1"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validateAndCleanIP(ipStr)
	}
}

func TestValidateContentType(t *testing.T) {
	tests := []struct {
		name         string
		contentType  string
		allowedTypes []string
		expected     bool
	}{
		{
			name:         "ValidateContentType_AllowedType",
			contentType:  "application/json",
			allowedTypes: []string{"application/json", "text/plain"},
			expected:     true,
		},
		{
			name:         "ValidateContentType_NotAllowedType",
			contentType:  "application/xml",
			allowedTypes: []string{"application/json", "text/plain"},
			expected:     false,
		},
		{
			name:         "ValidateContentType_EmptyAllowedTypes",
			contentType:  "application/json",
			allowedTypes: []string{},
			expected:     true,
		},
		{
			name:         "ValidateContentType_EmptyContentType",
			contentType:  "",
			allowedTypes: []string{"application/json"},
			expected:     false,
		},
		{
			name:         "ValidateContentType_ExactMatch",
			contentType:  "text/html",
			allowedTypes: []string{"text/html"},
			expected:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			result := validateContentType(tt.contentType, tt.allowedTypes)

			// Assert
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseSize(t *testing.T) {
	tests := []struct {
		name        string
		size        string
		expected    int64
		expectError bool
	}{
		{
			name:        "ParseSize_1KB",
			size:        "1KB",
			expected:    1024,
			expectError: false,
		},
		{
			name:        "ParseSize_1MB",
			size:        "1MB",
			expected:    1024 * 1024,
			expectError: false,
		},
		{
			name:        "ParseSize_10MB",
			size:        "10MB",
			expected:    10 * 1024 * 1024,
			expectError: false,
		},
		{
			name:        "ParseSize_Empty",
			size:        "",
			expected:    0,
			expectError: false,
		},
		{
			name:        "ParseSize_Unsupported",
			size:        "1GB",
			expected:    0,
			expectError: true,
		},
		{
			name:        "ParseSize_InvalidFormat",
			size:        "invalid",
			expected:    0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			result, err := parseSize(tt.size)

			// Assert
			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, int64(0), result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestMergeHeaders(t *testing.T) {
	tests := []struct {
		name     string
		base     map[string]string
		override map[string]string
		expected map[string]string
	}{
		{
			name: "MergeHeaders_NoOverlap",
			base: map[string]string{
				"Content-Type": "application/json",
				"Accept":       "application/json",
			},
			override: map[string]string{
				"Authorization": "Bearer token",
				"User-Agent":    "test-agent",
			},
			expected: map[string]string{
				"Content-Type":  "application/json",
				"Accept":        "application/json",
				"Authorization": "Bearer token",
				"User-Agent":    "test-agent",
			},
		},
		{
			name: "MergeHeaders_WithOverride",
			base: map[string]string{
				"Content-Type": "application/json",
				"Accept":       "application/json",
			},
			override: map[string]string{
				"Content-Type": "application/xml",
				"User-Agent":   "test-agent",
			},
			expected: map[string]string{
				"Content-Type": "application/xml",
				"Accept":       "application/json",
				"User-Agent":   "test-agent",
			},
		},
		{
			name: "MergeHeaders_EmptyBase",
			base: map[string]string{},
			override: map[string]string{
				"Content-Type": "application/json",
			},
			expected: map[string]string{
				"Content-Type": "application/json",
			},
		},
		{
			name: "MergeHeaders_EmptyOverride",
			base: map[string]string{
				"Content-Type": "application/json",
			},
			override: map[string]string{},
			expected: map[string]string{
				"Content-Type": "application/json",
			},
		},
		{
			name:     "MergeHeaders_BothEmpty",
			base:     map[string]string{},
			override: map[string]string{},
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			result := mergeHeaders(tt.base, tt.override)

			// Assert
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateMiddlewareConfig(t *testing.T) {
	tests := []struct {
		name   string
		config interface{}
	}{
		{
			name:   "ValidateMiddlewareConfig_AnyConfig",
			config: "any config",
		},
		{
			name:   "ValidateMiddlewareConfig_NilConfig",
			config: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			err := validateMiddlewareConfig(tt.config)

			// Assert
			assert.NoError(t, err)
		})
	}
}

// Benchmark tests for utility functions
func BenchmarkGenerateRequestID(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generateRequestID()
	}
}

func BenchmarkContainsString(b *testing.B) {
	slice := []string{"apple", "banana", "cherry", "date", "elderberry", "fig", "grape"}
	item := "cherry"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = containsString(slice, item)
	}
}

func BenchmarkSanitizeString(b *testing.B) {
	input := "Hello\x00\x01\x02World\n\r\tThis is a test string with control characters"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sanitizeString(input)
	}
}

func BenchmarkIsValidOrigin(b *testing.B) {
	origin := "https://api.example.com:8443"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = isValidOrigin(origin)
	}
}

func BenchmarkMergeHeaders(b *testing.B) {
	base := map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/json",
		"User-Agent":   "test-agent",
	}
	override := map[string]string{
		"Authorization": "Bearer token",
		"Content-Type":  "application/xml",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mergeHeaders(base, override)
	}
}

func BenchmarkStatusChecks(b *testing.B) {
	statusCodes := []int{200, 400, 500, 201, 404, 502}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, code := range statusCodes {
			_ = isSuccessStatus(code)
			_ = isClientError(code)
			_ = isInternalError(code)
		}
	}
}
