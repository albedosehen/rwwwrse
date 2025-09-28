package tls

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"
)

// Manager manages TLS certificates and provides HTTPS server configuration.
// It handles automatic certificate provisioning, renewal, and storage.
type Manager interface {
	// GetCertificate returns a certificate for the given client hello info.
	// This is used as the GetCertificate function in tls.Config.
	GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error)

	// GetTLSConfig returns a TLS configuration for HTTPS servers.
	GetTLSConfig() *tls.Config

	// AddDomain adds a domain to be managed for automatic certificate provisioning.
	AddDomain(domain string) error

	// RemoveDomain removes a domain from automatic certificate management.
	RemoveDomain(domain string) error

	// GetDomains returns all domains currently managed by this TLS manager.
	GetDomains() []string

	// RenewCertificates manually triggers certificate renewal for all managed domains.
	RenewCertificates(ctx context.Context) error

	// GetCertificateInfo returns information about the certificate for a domain.
	GetCertificateInfo(domain string) (*CertificateInfo, error)

	// Start starts the TLS manager background processes.
	Start(ctx context.Context) error

	// Stop stops the TLS manager and cleans up resources.
	Stop(ctx context.Context) error

	// IsHealthy returns whether the TLS manager is functioning properly.
	IsHealthy() bool
}

// CertificateProvider handles certificate provisioning and renewal.
// It abstracts the certificate authority (CA) interaction.
type CertificateProvider interface {
	// ObtainCertificate obtains a new certificate for the given domains.
	ObtainCertificate(ctx context.Context, domains []string) (*Certificate, error)

	// RenewCertificate renews an existing certificate.
	RenewCertificate(ctx context.Context, cert *Certificate) (*Certificate, error)

	// RevokeCertificate revokes a certificate.
	RevokeCertificate(ctx context.Context, cert *Certificate) error

	// ValidateDomains validates that domains can be used for certificate issuance.
	ValidateDomains(domains []string) error

	// GetAccountInfo returns information about the ACME account.
	GetAccountInfo() (*AccountInfo, error)
}

// CertificateStorage handles persistent storage of certificates and keys.
// It ensures certificates survive application restarts and can be shared across instances.
type CertificateStorage interface {
	// StoreCertificate stores a certificate and its private key.
	StoreCertificate(domain string, cert *Certificate) error

	// LoadCertificate loads a certificate and its private key for a domain.
	LoadCertificate(domain string) (*Certificate, error)

	// DeleteCertificate removes a certificate from storage.
	DeleteCertificate(domain string) error

	// ListCertificates returns all stored certificate domains.
	ListCertificates() ([]string, error)

	// StoreMetadata stores certificate metadata (expiry, renewal info, etc.).
	StoreMetadata(domain string, metadata *CertificateMetadata) error

	// LoadMetadata loads certificate metadata for a domain.
	LoadMetadata(domain string) (*CertificateMetadata, error)

	// Lock acquires a distributed lock for certificate operations.
	Lock(ctx context.Context, key string) (Lock, error)
}

// Lock represents a distributed lock for coordinating certificate operations
// across multiple application instances.
type Lock interface {
	// Unlock releases the lock.
	Unlock() error

	// Refresh extends the lock duration.
	Refresh(ctx context.Context) error

	// IsValid returns whether the lock is still valid.
	IsValid() bool
}

// ChallengeHandler handles ACME challenge responses for domain validation.
// It supports HTTP-01 and TLS-ALPN-01 challenge types.
type ChallengeHandler interface {
	// HandleHTTP01Challenge handles HTTP-01 challenge requests.
	HandleHTTP01Challenge(w http.ResponseWriter, r *http.Request)

	// HandleTLSALPN01Challenge handles TLS-ALPN-01 challenge certificates.
	HandleTLSALPN01Challenge(hello *tls.ClientHelloInfo) (*tls.Certificate, error)

	// SetChallengeData sets challenge data for a domain and token.
	SetChallengeData(domain, token, keyAuth string) error

	// GetChallengeData retrieves challenge data for a domain and token.
	GetChallengeData(domain, token string) (string, error)

	// ClearChallengeData removes challenge data for a domain and token.
	ClearChallengeData(domain, token string) error
}

// Certificate represents a TLS certificate with its private key and metadata.
type Certificate struct {
	// Domain is the primary domain name for this certificate.
	Domain string `json:"domain"`

	// SANs are the Subject Alternative Names (additional domains).
	SANs []string `json:"sans"`

	// CertificatePEM is the PEM-encoded certificate chain.
	CertificatePEM []byte `json:"certificate_pem"`

	// PrivateKeyPEM is the PEM-encoded private key.
	PrivateKeyPEM []byte `json:"private_key_pem"`

	// IssuedAt is when the certificate was issued.
	IssuedAt time.Time `json:"issued_at"`

	// ExpiresAt is when the certificate expires.
	ExpiresAt time.Time `json:"expires_at"`

	// Issuer is the certificate authority that issued this certificate.
	Issuer string `json:"issuer"`

	// SerialNumber is the certificate serial number.
	SerialNumber string `json:"serial_number"`
}

// CertificateInfo provides information about a certificate's status and metadata.
type CertificateInfo struct {
	// Domain is the primary domain name.
	Domain string `json:"domain"`

	// SANs are the Subject Alternative Names.
	SANs []string `json:"sans"`

	// Status indicates the certificate status (valid, expired, revoked, etc.).
	Status CertificateStatus `json:"status"`

	// IssuedAt is when the certificate was issued.
	IssuedAt time.Time `json:"issued_at"`

	// ExpiresAt is when the certificate expires.
	ExpiresAt time.Time `json:"expires_at"`

	// DaysUntilExpiry is the number of days until expiration.
	DaysUntilExpiry int `json:"days_until_expiry"`

	// AutoRenew indicates if automatic renewal is enabled.
	AutoRenew bool `json:"auto_renew"`

	// LastRenewalAttempt is when renewal was last attempted.
	LastRenewalAttempt *time.Time `json:"last_renewal_attempt,omitempty"`

	// RenewalError is the last renewal error, if any.
	RenewalError string `json:"renewal_error,omitempty"`

	// Issuer is the certificate authority.
	Issuer string `json:"issuer"`
}

// CertificateMetadata contains metadata about certificate management.
type CertificateMetadata struct {
	// Domain is the primary domain name.
	Domain string `json:"domain"`

	// CreatedAt is when the certificate was first created.
	CreatedAt time.Time `json:"created_at"`

	// LastUpdated is when the certificate was last updated.
	LastUpdated time.Time `json:"last_updated"`

	// RenewalCount is the number of times the certificate has been renewed.
	RenewalCount int `json:"renewal_count"`

	// NextRenewal is when the next renewal should occur.
	NextRenewal time.Time `json:"next_renewal"`

	// ChallengeType is the ACME challenge type used.
	ChallengeType string `json:"challenge_type"`

	// ManagementEnabled indicates if automatic management is enabled.
	ManagementEnabled bool `json:"management_enabled"`

	// Tags are custom tags for certificate organization.
	Tags []string `json:"tags,omitempty"`
}

// AccountInfo contains information about the ACME account.
type AccountInfo struct {
	// Email is the account email address.
	Email string `json:"email"`

	// URL is the account URL.
	URL string `json:"url"`

	// Status is the account status.
	Status string `json:"status"`

	// CreatedAt is when the account was created.
	CreatedAt time.Time `json:"created_at"`

	// Contact contains contact information.
	Contact []string `json:"contact"`

	// TermsOfServiceAgreed indicates if ToS has been agreed to.
	TermsOfServiceAgreed bool `json:"terms_of_service_agreed"`
}

// CertificateStatus represents the status of a certificate.
type CertificateStatus string

const (
	// CertificateStatusValid indicates the certificate is valid and current.
	CertificateStatusValid CertificateStatus = "valid"

	// CertificateStatusExpired indicates the certificate has expired.
	CertificateStatusExpired CertificateStatus = "expired"

	// CertificateStatusExpiring indicates the certificate is nearing expiry.
	CertificateStatusExpiring CertificateStatus = "expiring"

	// CertificateStatusRevoked indicates the certificate has been revoked.
	CertificateStatusRevoked CertificateStatus = "revoked"

	// CertificateStatusPending indicates certificate issuance is in progress.
	CertificateStatusPending CertificateStatus = "pending"

	// CertificateStatusError indicates there was an error with the certificate.
	CertificateStatusError CertificateStatus = "error"
)

// TLSConfig holds configuration for TLS management.
type TLSConfig struct {
	// Email is the email address for ACME account registration.
	Email string `mapstructure:"email"`

	// AcceptTOS indicates acceptance of the CA's Terms of Service.
	AcceptTOS bool `mapstructure:"accept_tos"`

	// CAServerURL is the ACME CA server URL (defaults to Let's Encrypt production).
	CAServerURL string `mapstructure:"ca_server_url"`

	// ChallengeType specifies the ACME challenge type (http-01, tls-alpn-01).
	ChallengeType string `mapstructure:"challenge_type"`

	// KeyType specifies the private key type (RSA, ECDSA).
	KeyType string `mapstructure:"key_type"`

	// RenewalDays is the number of days before expiry to start renewal.
	RenewalDays int `mapstructure:"renewal_days"`

	// StorageType specifies the certificate storage backend.
	StorageType string `mapstructure:"storage_type"`

	// StorageConfig contains storage-specific configuration.
	StorageConfig map[string]interface{} `mapstructure:"storage_config"`

	// DisableHTTPRedirect disables automatic HTTP to HTTPS redirects.
	DisableHTTPRedirect bool `mapstructure:"disable_http_redirect"`

	// MinTLSVersion specifies the minimum TLS version.
	MinTLSVersion string `mapstructure:"min_tls_version"`

	// CipherSuites specifies allowed cipher suites.
	CipherSuites []string `mapstructure:"cipher_suites"`

	// MustStaple enables OCSP Must-Staple for certificates.
	MustStaple bool `mapstructure:"must_staple"`

	// Note: PreferServerCipherSuites was removed as it's deprecated since Go 1.17
	// The Go runtime now automatically selects the best cipher suite based on hardware and security considerations
}

// DefaultTLSConfig returns a secure default TLS configuration.
func DefaultTLSConfig() TLSConfig {
	return TLSConfig{
		CAServerURL:         "https://acme-v02.api.letsencrypt.org/directory",
		ChallengeType:       "http-01",
		KeyType:             "ECDSA",
		RenewalDays:         30,
		StorageType:         "file",
		DisableHTTPRedirect: false,
		MinTLSVersion:       "1.2",
		MustStaple:          false,
		CipherSuites: []string{
			"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
			"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
			"TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305",
			"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305",
			"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
			"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
		},
	}
}
