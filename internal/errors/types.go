package errors

import (
	"errors"
	"fmt"
	"net/http"
)

type ProxyError struct {
	Code       ProxyErrorCode         `json:"code"`
	Message    string                 `json:"message"`
	Cause      error                  `json:"-"`
	Context    map[string]interface{} `json:"context,omitempty"`
	HTTPStatus int                    `json:"http_status,omitempty"`
}

// ProxyErrorCode defines specific error conditions within the proxy system.
type ProxyErrorCode int

// Error code constants for different proxy error conditions.
const (
	// Backend-related errors
	ErrCodeBackendUnavailable ProxyErrorCode = iota + 1000
	ErrCodeBackendTimeout
	ErrCodeBackendConnectionFailed
	ErrCodeBackendInvalidResponse

	// Routing-related errors
	ErrCodeInvalidHost
	ErrCodeHostNotConfigured
	ErrCodeRoutingFailed

	// TLS-related errors
	ErrCodeTLSHandshake
	ErrCodeCertificateNotFound
	ErrCodeCertificateExpired
	ErrCodeCertificateInvalid

	// Security-related errors
	ErrCodeRateLimited
	ErrCodeAccessDenied
	ErrCodeInvalidOrigin

	// Configuration-related errors
	ErrCodeConfigInvalid
	ErrCodeConfigMissing
	ErrCodeConfigValidation

	// Health check-related errors
	ErrCodeHealthCheckFailed
	ErrCodeHealthCheckTimeout
	ErrCodeCircuitBreakerOpen

	// Request processing errors
	ErrCodeRequestInvalid
	ErrCodeRequestTooLarge
	ErrCodeRequestTimeout

	// Internal errors
	ErrCodeInternalError
	ErrCodeServiceUnavailable
	ErrCodeNotImplemented
)

func (e *ProxyError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *ProxyError) Unwrap() error {
	return e.Cause
}

func (e *ProxyError) Is(target error) bool {
	if t, ok := target.(*ProxyError); ok {
		return e.Code == t.Code
	}
	return false
}

func (e *ProxyError) As(target interface{}) bool {
	if t, ok := target.(**ProxyError); ok {
		*t = e
		return true
	}
	return false
}

func (e *ProxyError) WithContext(key string, value interface{}) *ProxyError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

func (e *ProxyError) WithHTTPStatus(status int) *ProxyError {
	e.HTTPStatus = status
	return e
}

func (code ProxyErrorCode) String() string {
	switch code {
	case ErrCodeBackendUnavailable:
		return "backend_unavailable"
	case ErrCodeBackendTimeout:
		return "backend_timeout"
	case ErrCodeBackendConnectionFailed:
		return "backend_connection_failed"
	case ErrCodeBackendInvalidResponse:
		return "backend_invalid_response"
	case ErrCodeInvalidHost:
		return "invalid_host"
	case ErrCodeHostNotConfigured:
		return "host_not_configured"
	case ErrCodeRoutingFailed:
		return "routing_failed"
	case ErrCodeTLSHandshake:
		return "tls_handshake"
	case ErrCodeCertificateNotFound:
		return "certificate_not_found"
	case ErrCodeCertificateExpired:
		return "certificate_expired"
	case ErrCodeCertificateInvalid:
		return "certificate_invalid"
	case ErrCodeRateLimited:
		return "rate_limited"
	case ErrCodeAccessDenied:
		return "access_denied"
	case ErrCodeInvalidOrigin:
		return "invalid_origin"
	case ErrCodeConfigInvalid:
		return "config_invalid"
	case ErrCodeConfigMissing:
		return "config_missing"
	case ErrCodeConfigValidation:
		return "config_validation"
	case ErrCodeHealthCheckFailed:
		return "health_check_failed"
	case ErrCodeHealthCheckTimeout:
		return "health_check_timeout"
	case ErrCodeCircuitBreakerOpen:
		return "circuit_breaker_open"
	case ErrCodeRequestInvalid:
		return "request_invalid"
	case ErrCodeRequestTooLarge:
		return "request_too_large"
	case ErrCodeRequestTimeout:
		return "request_timeout"
	case ErrCodeInternalError:
		return "internal_error"
	case ErrCodeServiceUnavailable:
		return "service_unavailable"
	case ErrCodeNotImplemented:
		return "not_implemented"
	default:
		return "unknown_error"
	}
}

func (code ProxyErrorCode) HTTPStatus() int {
	switch code {
	case ErrCodeBackendUnavailable, ErrCodeBackendConnectionFailed:
		return http.StatusBadGateway
	case ErrCodeBackendTimeout, ErrCodeHealthCheckTimeout, ErrCodeRequestTimeout:
		return http.StatusGatewayTimeout
	case ErrCodeInvalidHost, ErrCodeHostNotConfigured:
		return http.StatusNotFound
	case ErrCodeRateLimited:
		return http.StatusTooManyRequests
	case ErrCodeAccessDenied, ErrCodeInvalidOrigin:
		return http.StatusForbidden
	case ErrCodeRequestInvalid, ErrCodeBackendInvalidResponse:
		return http.StatusBadRequest
	case ErrCodeRequestTooLarge:
		return http.StatusRequestEntityTooLarge
	case ErrCodeTLSHandshake, ErrCodeCertificateNotFound, ErrCodeCertificateExpired, ErrCodeCertificateInvalid:
		return http.StatusBadGateway
	case ErrCodeCircuitBreakerOpen, ErrCodeServiceUnavailable:
		return http.StatusServiceUnavailable
	case ErrCodeNotImplemented:
		return http.StatusNotImplemented
	case ErrCodeConfigInvalid, ErrCodeConfigMissing, ErrCodeConfigValidation, ErrCodeHealthCheckFailed:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

func NewBackendError(code ProxyErrorCode, backend string, cause error) *ProxyError {
	err := &ProxyError{
		Code:       code,
		Message:    fmt.Sprintf("backend error: %s", code.String()),
		Cause:      cause,
		Context:    map[string]interface{}{"backend": backend},
		HTTPStatus: code.HTTPStatus(),
	}

	if cause != nil {
		err.Message = fmt.Sprintf("backend %s: %s", backend, cause.Error())
	}

	return err
}

func NewRoutingError(code ProxyErrorCode, host string, cause error) *ProxyError {
	err := &ProxyError{
		Code:       code,
		Message:    fmt.Sprintf("routing error: %s", code.String()),
		Cause:      cause,
		Context:    map[string]interface{}{"host": host},
		HTTPStatus: code.HTTPStatus(),
	}

	if cause != nil {
		err.Message = fmt.Sprintf("routing for host %s: %s", host, cause.Error())
	}

	return err
}

func NewTLSError(code ProxyErrorCode, domain string, cause error) *ProxyError {
	err := &ProxyError{
		Code:       code,
		Message:    fmt.Sprintf("TLS error: %s", code.String()),
		Cause:      cause,
		Context:    map[string]interface{}{"domain": domain},
		HTTPStatus: code.HTTPStatus(),
	}

	if cause != nil {
		err.Message = fmt.Sprintf("TLS for domain %s: %s", domain, cause.Error())
	}

	return err
}

func NewConfigError(code ProxyErrorCode, field string, cause error) *ProxyError {
	err := &ProxyError{
		Code:       code,
		Message:    fmt.Sprintf("configuration error: %s", code.String()),
		Cause:      cause,
		Context:    map[string]interface{}{"field": field},
		HTTPStatus: code.HTTPStatus(),
	}

	if cause != nil {
		err.Message = fmt.Sprintf("configuration field %s: %s", field, cause.Error())
	}

	return err
}

func NewSecurityError(code ProxyErrorCode, reason string, cause error) *ProxyError {
	err := &ProxyError{
		Code:       code,
		Message:    fmt.Sprintf("security error: %s", code.String()),
		Cause:      cause,
		Context:    map[string]interface{}{"reason": reason},
		HTTPStatus: code.HTTPStatus(),
	}

	if cause != nil {
		err.Message = fmt.Sprintf("security violation (%s): %s", reason, cause.Error())
	}

	return err
}

func WrapError(code ProxyErrorCode, message string, cause error) *ProxyError {
	return &ProxyError{
		Code:       code,
		Message:    message,
		Cause:      cause,
		Context:    make(map[string]interface{}),
		HTTPStatus: code.HTTPStatus(),
	}
}

var (
	ErrBackendUnavailable = &ProxyError{
		Code:       ErrCodeBackendUnavailable,
		Message:    "no healthy backend available",
		HTTPStatus: http.StatusBadGateway,
	}

	ErrHostNotConfigured = &ProxyError{
		Code:       ErrCodeHostNotConfigured,
		Message:    "host not configured in routing table",
		HTTPStatus: http.StatusNotFound,
	}

	ErrRateLimited = &ProxyError{
		Code:       ErrCodeRateLimited,
		Message:    "rate limit exceeded",
		HTTPStatus: http.StatusTooManyRequests,
	}

	ErrCircuitBreakerOpen = &ProxyError{
		Code:       ErrCodeCircuitBreakerOpen,
		Message:    "circuit breaker is open",
		HTTPStatus: http.StatusServiceUnavailable,
	}
)

func IsTemporary(err error) bool {
	var proxyErr *ProxyError
	if errors.As(err, &proxyErr) {
		switch proxyErr.Code {
		case ErrCodeBackendTimeout, ErrCodeBackendConnectionFailed,
			ErrCodeHealthCheckTimeout, ErrCodeRequestTimeout,
			ErrCodeCircuitBreakerOpen, ErrCodeServiceUnavailable:
			return true
		}
	}
	return false
}

func IsRetryable(err error) bool {
	var proxyErr *ProxyError
	if errors.As(err, &proxyErr) {
		switch proxyErr.Code {
		case ErrCodeBackendTimeout, ErrCodeBackendConnectionFailed,
			ErrCodeBackendUnavailable, ErrCodeServiceUnavailable:
			return true
		}
	}
	return false
}

func IsSecurity(err error) bool {
	var proxyErr *ProxyError
	if errors.As(err, &proxyErr) {
		switch proxyErr.Code {
		case ErrCodeRateLimited, ErrCodeAccessDenied, ErrCodeInvalidOrigin:
			return true
		}
	}
	return false
}
