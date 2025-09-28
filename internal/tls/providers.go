package tls

import (
	"fmt"
	"slices"

	"github.com/google/wire"

	"github.com/albedosehen/rwwwrse/internal/config"
	"github.com/albedosehen/rwwwrse/internal/observability"
)

var ProviderSet = wire.NewSet(
	NewTLSManager,
	NewFileStorage,
	NewHTTPChallengeHandler,
)

func NewTLSManager(
	cfg *config.Config,
	logger observability.Logger,
	metrics observability.MetricsCollector,
) (Manager, error) {
	configTLS := cfg.TLS
	if configTLS.Email == "" {
		return nil, fmt.Errorf("TLS email is required for automatic certificate management")
	}

	// Convert config.TLSConfig to tls.TLSConfig
	tlsConfig := TLSConfig{
		Email:               configTLS.Email,
		AcceptTOS:           true, // Assume acceptance when email is provided
		CAServerURL:         "https://acme-v02.api.letsencrypt.org/directory",
		ChallengeType:       "http-01",
		KeyType:             "ECDSA",
		RenewalDays:         30,
		StorageType:         "file",
		StorageConfig:       map[string]interface{}{"path": configTLS.CacheDir},
		DisableHTTPRedirect: false,
		MinTLSVersion:       "1.2",
		MustStaple:          false,
	}

	if configTLS.Staging {
		tlsConfig.CAServerURL = "https://acme-staging-v02.api.letsencrypt.org/directory"
	}

	// For now, create a simple TLS manager
	// In production, this would use CertMagic when available
	return NewSimpleTLSManager(tlsConfig, logger, metrics)
}

func NewFileStorage(cfg *config.Config) (CertificateStorage, error) {
	storagePath := cfg.TLS.CacheDir
	if storagePath == "" {
		storagePath = "./certs"
	}

	return NewFileStorageImpl(storagePath)
}

func NewHTTPChallengeHandler() ChallengeHandler {
	return NewHTTPChallengeHandlerImpl()
}

type TLSManagerConfig struct {
	// Provider specifies the TLS manager implementation to use.
	Provider string `mapstructure:"provider"`

	// Config contains the TLS configuration.
	Config TLSConfig `mapstructure:"config"`
}

func DefaultTLSManagerConfig() TLSManagerConfig {
	return TLSManagerConfig{
		Provider: "simple",
		Config:   DefaultTLSConfig(),
	}
}

func CreateTLSManager(
	config TLSManagerConfig,
	logger observability.Logger,
	metrics observability.MetricsCollector,
) (Manager, error) {
	switch config.Provider {
	case "simple":
		return NewSimpleTLSManager(config.Config, logger, metrics)
	case "certmagic":
		// This would use CertMagic when available
		return NewCertMagicManager(config.Config, logger, metrics)
	default:
		return nil, fmt.Errorf("unsupported TLS provider: %s", config.Provider)
	}
}

func ValidateTLSConfig(config TLSConfig) error {
	if config.Email == "" {
		return fmt.Errorf("email is required for ACME registration")
	}

	if !config.AcceptTOS {
		return fmt.Errorf("must accept Terms of Service")
	}

	if config.RenewalDays <= 0 {
		return fmt.Errorf("renewal days must be positive")
	}

	if config.MinTLSVersion != "" {
		validVersions := []string{"1.0", "1.1", "1.2", "1.3"}
		valid := slices.Contains(validVersions, config.MinTLSVersion)
		if !valid {
			return fmt.Errorf("invalid TLS version: %s", config.MinTLSVersion)
		}
	}

	return nil
}
