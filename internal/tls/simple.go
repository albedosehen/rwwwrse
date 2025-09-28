package tls

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"
	"time"

	"github.com/albedosehen/rwwwrse/internal/observability"
)

type simpleTLSManager struct {
	config       TLSConfig
	certificates map[string]*tls.Certificate
	certMutex    sync.RWMutex
	domains      map[string]bool
	domainsMutex sync.RWMutex
	logger       observability.Logger
	metrics      observability.MetricsCollector
	isStarted    bool
	startMutex   sync.Mutex
	stopChan     chan struct{}
}

func NewSimpleTLSManager(
	config TLSConfig,
	logger observability.Logger,
	metrics observability.MetricsCollector,
) (Manager, error) {
	manager := &simpleTLSManager{
		config:       config,
		certificates: make(map[string]*tls.Certificate),
		domains:      make(map[string]bool),
		logger:       logger,
		metrics:      metrics,
		stopChan:     make(chan struct{}),
	}

	return manager, nil
}

func (m *simpleTLSManager) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	if hello.ServerName == "" {
		return nil, fmt.Errorf("no server name provided")
	}

	m.certMutex.RLock()
	cert, exists := m.certificates[hello.ServerName]
	m.certMutex.RUnlock()

	if !exists {
		if m.logger != nil {
			m.logger.Warn(context.Background(), "No certificate available for domain",
				observability.String("domain", hello.ServerName),
			)
		}
		return nil, fmt.Errorf("no certificate available for domain: %s", hello.ServerName)
	}

	return cert, nil
}

func (m *simpleTLSManager) GetTLSConfig() *tls.Config {
	tlsConfig := &tls.Config{
		GetCertificate: m.GetCertificate,
		NextProtos:     []string{"h2", "http/1.1"},
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

func (m *simpleTLSManager) AddDomain(domain string) error {
	if domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}

	m.domainsMutex.Lock()
	defer m.domainsMutex.Unlock()

	m.domains[domain] = true

	if m.logger != nil {
		m.logger.Info(context.Background(), "Domain added to simple TLS management",
			observability.String("domain", domain),
		)
	}

	return nil
}

func (m *simpleTLSManager) RemoveDomain(domain string) error {
	m.domainsMutex.Lock()
	defer m.domainsMutex.Unlock()

	delete(m.domains, domain)

	// Also remove certificate
	m.certMutex.Lock()
	delete(m.certificates, domain)
	m.certMutex.Unlock()

	if m.logger != nil {
		m.logger.Info(context.Background(), "Domain removed from simple TLS management",
			observability.String("domain", domain),
		)
	}

	return nil
}

func (m *simpleTLSManager) GetDomains() []string {
	m.domainsMutex.RLock()
	defer m.domainsMutex.RUnlock()

	domains := make([]string, 0, len(m.domains))
	for domain := range m.domains {
		domains = append(domains, domain)
	}

	return domains
}

func (m *simpleTLSManager) RenewCertificates(ctx context.Context) error {
	if m.logger != nil {
		m.logger.Info(ctx, "Certificate renewal not supported in simple TLS manager")
	}
	return nil
}

func (m *simpleTLSManager) GetCertificateInfo(domain string) (*CertificateInfo, error) {
	m.certMutex.RLock()
	cert, exists := m.certificates[domain]
	m.certMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no certificate for domain: %s", domain)
	}

	if len(cert.Certificate) == 0 {
		return nil, fmt.Errorf("invalid certificate for domain: %s", domain)
	}

	// This is a simplified implementation
	info := &CertificateInfo{
		Domain:          domain,
		SANs:            []string{},
		Status:          CertificateStatusValid,
		AutoRenew:       false,
		Issuer:          "Manual",
		IssuedAt:        time.Now().Add(-time.Hour * 24 * 30), // Placeholder
		ExpiresAt:       time.Now().Add(time.Hour * 24 * 365), // Placeholder
		DaysUntilExpiry: 365,                                  // Placeholder
	}

	return info, nil
}

func (m *simpleTLSManager) Start(ctx context.Context) error {
	m.startMutex.Lock()
	defer m.startMutex.Unlock()

	if m.isStarted {
		return nil
	}

	if m.logger != nil {
		m.logger.Info(ctx, "Starting simple TLS manager")
	}

	m.isStarted = true

	if m.logger != nil {
		m.logger.Info(ctx, "Simple TLS manager started")
	}

	return nil
}

func (m *simpleTLSManager) Stop(ctx context.Context) error {
	m.startMutex.Lock()
	defer m.startMutex.Unlock()

	if !m.isStarted {
		return nil
	}

	if m.logger != nil {
		m.logger.Info(ctx, "Stopping simple TLS manager")
	}

	close(m.stopChan)
	m.isStarted = false

	if m.logger != nil {
		m.logger.Info(ctx, "Simple TLS manager stopped")
	}

	return nil
}

func (m *simpleTLSManager) IsHealthy() bool {
	m.startMutex.Lock()
	started := m.isStarted
	m.startMutex.Unlock()

	return started
}

func (m *simpleTLSManager) SetCertificate(domain string, cert *tls.Certificate) error {
	if domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}
	if cert == nil {
		return fmt.Errorf("certificate cannot be nil")
	}

	m.certMutex.Lock()
	defer m.certMutex.Unlock()

	m.certificates[domain] = cert

	if m.logger != nil {
		m.logger.Info(context.Background(), "Certificate set for domain",
			observability.String("domain", domain),
		)
	}

	return nil
}
