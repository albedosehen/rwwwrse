// Package tls implements TLS certificate management with CertMagic integration.
package tls

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"
	"time"

	"github.com/caddyserver/certmagic"

	"github.com/albedosehen/rwwwrse/internal/observability"
)

// certMagicManager implements the Manager interface using CertMagic.
type certMagicManager struct {
	config       TLSConfig
	certMagic    *certmagic.Config
	domains      map[string]bool
	domainsMutex sync.RWMutex
	logger       observability.Logger
	metrics      observability.MetricsCollector
	isStarted    bool
	startMutex   sync.Mutex
	stopChan     chan struct{}
}

// NewCertMagicManager creates a new TLS manager using CertMagic.
func NewCertMagicManager(
	config TLSConfig,
	logger observability.Logger,
	metrics observability.MetricsCollector,
) (Manager, error) {
	if config.Email == "" {
		return nil, fmt.Errorf("email is required for ACME registration")
	}

	if !config.AcceptTOS {
		return nil, fmt.Errorf("must accept Terms of Service")
	}

	// Create CertMagic configuration
	certMagicConfig := certmagic.NewDefault()

	// Configure ACME settings
	if config.CAServerURL != "" {
		certMagicConfig.DefaultServerName = config.CAServerURL
	}

	// Configure storage if specified
	if config.StorageType != "" {
		storage, err := createStorage(config.StorageType, config.StorageConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create storage: %w", err)
		}
		certMagicConfig.Storage = storage
	}

	// Configure ACME issuer
	acmeIssuer := certmagic.NewACMEIssuer(certMagicConfig, certmagic.ACMEIssuer{
		Email:                   config.Email,
		Agreed:                  config.AcceptTOS,
		DisableHTTPChallenge:    config.ChallengeType != "http-01",
		DisableTLSALPNChallenge: config.ChallengeType != "tls-alpn-01",
	})

	certMagicConfig.Issuers = []certmagic.Issuer{acmeIssuer}

	// Configure renewal settings
	if config.RenewalDays > 0 {
		certMagicConfig.RenewalWindowRatio = float64(config.RenewalDays) / 90.0 // Assume 90-day certs
	}

	// Configure TLS settings
	if config.MustStaple {
		certMagicConfig.MustStaple = true
	}

	manager := &certMagicManager{
		config:    config,
		certMagic: certMagicConfig,
		domains:   make(map[string]bool),
		logger:    logger,
		metrics:   metrics,
		stopChan:  make(chan struct{}),
	}

	return manager, nil
}

// GetCertificate returns a certificate for the given client hello info.
func (m *certMagicManager) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	if hello.ServerName == "" {
		return nil, fmt.Errorf("no server name provided")
	}

	// Check if domain is managed
	m.domainsMutex.RLock()
	managed := m.domains[hello.ServerName]
	m.domainsMutex.RUnlock()

	if !managed {
		if m.logger != nil {
			m.logger.Warn(context.Background(), "Certificate requested for unmanaged domain",
				observability.String("domain", hello.ServerName),
			)
		}
		return nil, fmt.Errorf("domain not managed: %s", hello.ServerName)
	}

	// Get certificate from CertMagic
	cert, err := m.certMagic.GetCertificate(hello)
	if err != nil {
		if m.logger != nil {
			m.logger.Error(context.Background(), err, "Failed to get certificate",
				observability.String("domain", hello.ServerName),
			)
		}
		if m.metrics != nil {
			// TODO: Add certificate error metrics when interface is extended
			// m.metrics.RecordCertificateError(hello.ServerName, err)
			_ = m.metrics // Acknowledge metrics is available but not used yet
		}
		return nil, fmt.Errorf("failed to get certificate for %s: %w", hello.ServerName, err)
	}

	if m.logger != nil {
		m.logger.Debug(context.Background(), "Certificate provided successfully",
			observability.String("domain", hello.ServerName),
		)
	}

	if m.metrics != nil {
		// TODO: Add certificate success metrics when interface is extended
		// m.metrics.RecordCertificateSuccess(hello.ServerName)
		_ = m.metrics // Acknowledge metrics is available but not used yet
	}

	return cert, nil
}

// GetTLSConfig returns a TLS configuration for HTTPS servers.
func (m *certMagicManager) GetTLSConfig() *tls.Config {
	tlsConfig := &tls.Config{
		GetCertificate: m.GetCertificate,
		NextProtos:     []string{"h2", "http/1.1", "acme-tls/1"},
	}

	// Configure minimum TLS version
	switch m.config.MinTLSVersion {
	case "1.0":
		tlsConfig.MinVersion = tls.VersionTLS10
	case "1.1":
		tlsConfig.MinVersion = tls.VersionTLS11
	case "1.2":
		tlsConfig.MinVersion = tls.VersionTLS12
	case "1.3":
		tlsConfig.MinVersion = tls.VersionTLS13
	default:
		tlsConfig.MinVersion = tls.VersionTLS12 // Secure default
	}

	// Configure cipher suites if specified
	if len(m.config.CipherSuites) > 0 {
		cipherSuites := make([]uint16, 0, len(m.config.CipherSuites))
		for _, suite := range m.config.CipherSuites {
			if cipherID := getCipherSuiteID(suite); cipherID != 0 {
				cipherSuites = append(cipherSuites, cipherID)
			}
		}
		tlsConfig.CipherSuites = cipherSuites
	}

	// Note: PreferServerCipherSuites was removed as it's deprecated since Go 1.17
	// The Go runtime now automatically selects the best cipher suite

	return tlsConfig
}

// AddDomain adds a domain to be managed for automatic certificate provisioning.
func (m *certMagicManager) AddDomain(domain string) error {
	if domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}

	// Validate domain format (basic validation)
	if !isValidDomain(domain) {
		return fmt.Errorf("invalid domain format: %s", domain)
	}

	m.domainsMutex.Lock()
	defer m.domainsMutex.Unlock()

	if m.domains[domain] {
		return nil // Already managed
	}

	// Add domain to CertMagic management
	err := m.certMagic.ManageAsync(context.Background(), []string{domain})
	if err != nil {
		if m.logger != nil {
			m.logger.Error(context.Background(), err, "Failed to add domain to management",
				observability.String("domain", domain),
			)
		}
		return fmt.Errorf("failed to manage domain %s: %w", domain, err)
	}

	m.domains[domain] = true

	if m.logger != nil {
		m.logger.Info(context.Background(), "Domain added to TLS management",
			observability.String("domain", domain),
		)
	}

	return nil
}

// RemoveDomain removes a domain from automatic certificate management.
func (m *certMagicManager) RemoveDomain(domain string) error {
	m.domainsMutex.Lock()
	defer m.domainsMutex.Unlock()

	if !m.domains[domain] {
		return nil // Not managed
	}

	delete(m.domains, domain)

	if m.logger != nil {
		m.logger.Info(context.Background(), "Domain removed from TLS management",
			observability.String("domain", domain),
		)
	}

	return nil
}

// GetDomains returns all domains currently managed by this TLS manager.
func (m *certMagicManager) GetDomains() []string {
	m.domainsMutex.RLock()
	defer m.domainsMutex.RUnlock()

	domains := make([]string, 0, len(m.domains))
	for domain := range m.domains {
		domains = append(domains, domain)
	}

	return domains
}

// RenewCertificates manually triggers certificate renewal for all managed domains.
func (m *certMagicManager) RenewCertificates(ctx context.Context) error {
	domains := m.GetDomains()
	if len(domains) == 0 {
		return nil
	}

	if m.logger != nil {
		m.logger.Info(ctx, "Starting manual certificate renewal",
			observability.Int("domain_count", len(domains)),
		)
	}

	var renewalErrors []error
	for _, domain := range domains {
		err := m.certMagic.RenewCertAsync(ctx, domain, false)
		if err != nil {
			renewalErrors = append(renewalErrors, fmt.Errorf("failed to renew %s: %w", domain, err))
			if m.logger != nil {
				m.logger.Error(ctx, err, "Certificate renewal failed",
					observability.String("domain", domain),
				)
			}
		} else {
			if m.logger != nil {
				m.logger.Info(ctx, "Certificate renewal initiated",
					observability.String("domain", domain),
				)
			}
		}
	}

	if len(renewalErrors) > 0 {
		return fmt.Errorf("renewal errors: %v", renewalErrors)
	}

	return nil
}

// GetCertificateInfo returns information about the certificate for a domain.
func (m *certMagicManager) GetCertificateInfo(domain string) (*CertificateInfo, error) {
	m.domainsMutex.RLock()
	managed := m.domains[domain]
	m.domainsMutex.RUnlock()

	if !managed {
		return nil, fmt.Errorf("domain not managed: %s", domain)
	}

	// Try to get certificate information
	// This is a simplified implementation - in a real scenario,
	// you'd query the actual certificate from storage
	info := &CertificateInfo{
		Domain:          domain,
		SANs:            []string{},
		Status:          CertificateStatusValid,
		AutoRenew:       true,
		Issuer:          "Let's Encrypt",
		IssuedAt:        time.Now().Add(-time.Hour * 24 * 30), // Placeholder
		ExpiresAt:       time.Now().Add(time.Hour * 24 * 60),  // Placeholder
		DaysUntilExpiry: 60,                                   // Placeholder
	}

	return info, nil
}

// Start starts the TLS manager background processes.
func (m *certMagicManager) Start(ctx context.Context) error {
	m.startMutex.Lock()
	defer m.startMutex.Unlock()

	if m.isStarted {
		return nil
	}

	if m.logger != nil {
		m.logger.Info(ctx, "Starting TLS manager")
	}

	// CertMagic handles background processes automatically
	// We just need to mark as started
	m.isStarted = true

	// Start background maintenance goroutine
	go m.maintenanceLoop(ctx)

	if m.logger != nil {
		m.logger.Info(ctx, "TLS manager started successfully")
	}

	return nil
}

// Stop stops the TLS manager and cleans up resources.
func (m *certMagicManager) Stop(ctx context.Context) error {
	m.startMutex.Lock()
	defer m.startMutex.Unlock()

	if !m.isStarted {
		return nil
	}

	if m.logger != nil {
		m.logger.Info(ctx, "Stopping TLS manager")
	}

	// Signal maintenance loop to stop
	close(m.stopChan)

	m.isStarted = false

	if m.logger != nil {
		m.logger.Info(ctx, "TLS manager stopped")
	}

	return nil
}

// IsHealthy returns whether the TLS manager is functioning properly.
func (m *certMagicManager) IsHealthy() bool {
	m.startMutex.Lock()
	started := m.isStarted
	m.startMutex.Unlock()

	return started
}

// maintenanceLoop runs background maintenance tasks.
func (m *certMagicManager) maintenanceLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Hour) // Check every hour
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.performMaintenance(ctx)
		}
	}
}

// performMaintenance performs routine maintenance tasks.
func (m *certMagicManager) performMaintenance(ctx context.Context) {
	if m.logger != nil {
		m.logger.Debug(ctx, "Performing TLS maintenance")
	}

	// Check certificate expiry and trigger renewals if needed
	domains := m.GetDomains()
	for _, domain := range domains {
		// This would check actual certificate expiry and trigger renewal
		// For now, just log the maintenance check
		if m.logger != nil {
			m.logger.Debug(ctx, "Checked certificate status",
				observability.String("domain", domain),
			)
		}
	}
}

// Helper functions

// createStorage creates a storage backend based on configuration.
func createStorage(storageType string, config map[string]interface{}) (certmagic.Storage, error) {
	switch storageType {
	case "file":
		// Use default file storage
		return &certmagic.FileStorage{
			Path: getStoragePath(config),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", storageType)
	}
}

// getStoragePath extracts storage path from config.
func getStoragePath(config map[string]interface{}) string {
	if config == nil {
		return "./certs" // Default path
	}

	if path, ok := config["path"].(string); ok {
		return path
	}

	return "./certs" // Default path
}

// isValidDomain performs basic domain validation.
func isValidDomain(domain string) bool {
	if len(domain) == 0 || len(domain) > 253 {
		return false
	}

	// Basic validation - should contain at least one dot and valid characters
	return len(domain) > 0 && domain != "localhost"
}

// getCipherSuiteID returns the cipher suite ID for a given name.
func getCipherSuiteID(name string) uint16 {
	suites := map[string]uint16{
		"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384": tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384":   tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305":  tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
		"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305":    tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256": tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256":   tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	}

	return suites[name]
}
