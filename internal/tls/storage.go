package tls

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type fileStorage struct {
	basePath string
	mu       sync.RWMutex
}

type fileLock struct {
	filePath string
	file     *os.File
}

func NewFileStorageImpl(basePath string) (CertificateStorage, error) {
	if basePath == "" {
		return nil, fmt.Errorf("base path cannot be empty")
	}

	// Ensure the directory exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &fileStorage{
		basePath: basePath,
	}, nil
}

func (fs *fileStorage) StoreCertificate(domain string, cert *Certificate) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}

	if cert == nil {
		return fmt.Errorf("certificate cannot be nil")
	}

	// Create domain directory
	domainPath := filepath.Join(fs.basePath, domain)
	if err := os.MkdirAll(domainPath, 0755); err != nil {
		return fmt.Errorf("failed to create domain directory: %w", err)
	}

	// Store certificate
	certPath := filepath.Join(domainPath, "cert.pem")
	if err := os.WriteFile(certPath, cert.CertificatePEM, 0644); err != nil {
		return fmt.Errorf("failed to write certificate: %w", err)
	}

	// Store private key
	keyPath := filepath.Join(domainPath, "key.pem")
	if err := os.WriteFile(keyPath, cert.PrivateKeyPEM, 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	// Store certificate info
	infoPath := filepath.Join(domainPath, "info.json")
	infoData, err := json.Marshal(cert)
	if err != nil {
		return fmt.Errorf("failed to marshal certificate info: %w", err)
	}
	if err := os.WriteFile(infoPath, infoData, 0644); err != nil {
		return fmt.Errorf("failed to write certificate info: %w", err)
	}

	return nil
}

func (fs *fileStorage) LoadCertificate(domain string) (*Certificate, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	if domain == "" {
		return nil, fmt.Errorf("domain cannot be empty")
	}

	domainPath := filepath.Join(fs.basePath, domain)
	infoPath := filepath.Join(domainPath, "info.json")

	// Check if certificate info exists
	if _, err := os.Stat(infoPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("certificate not found for domain %s", domain)
	}

	// Load certificate info
	infoData, err := os.ReadFile(infoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate info: %w", err)
	}

	var cert Certificate
	if err := json.Unmarshal(infoData, &cert); err != nil {
		return nil, fmt.Errorf("failed to unmarshal certificate info: %w", err)
	}

	return &cert, nil
}

func (fs *fileStorage) DeleteCertificate(domain string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}

	domainPath := filepath.Join(fs.basePath, domain)
	if err := os.RemoveAll(domainPath); err != nil {
		return fmt.Errorf("failed to delete certificate directory: %w", err)
	}

	return nil
}

func (fs *fileStorage) ListCertificates() ([]string, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	entries, err := os.ReadDir(fs.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to list storage directory: %w", err)
	}

	var domains []string
	for _, entry := range entries {
		if entry.IsDir() {
			// Check if this directory contains a certificate
			infoPath := filepath.Join(fs.basePath, entry.Name(), "info.json")
			if _, err := os.Stat(infoPath); err == nil {
				domains = append(domains, entry.Name())
			}
		}
	}

	return domains, nil
}

func (fs *fileStorage) StoreMetadata(domain string, metadata *CertificateMetadata) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}

	if metadata == nil {
		return fmt.Errorf("metadata cannot be nil")
	}

	domainPath := filepath.Join(fs.basePath, domain)
	if err := os.MkdirAll(domainPath, 0755); err != nil {
		return fmt.Errorf("failed to create domain directory: %w", err)
	}

	metadataPath := filepath.Join(domainPath, "metadata.json")
	metadataData, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(metadataPath, metadataData, 0644); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

func (fs *fileStorage) LoadMetadata(domain string) (*CertificateMetadata, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	if domain == "" {
		return nil, fmt.Errorf("domain cannot be empty")
	}

	metadataPath := filepath.Join(fs.basePath, domain, "metadata.json")

	// Check if metadata exists
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("metadata not found for domain %s", domain)
	}

	// Load metadata
	metadataData, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	var metadata CertificateMetadata
	if err := json.Unmarshal(metadataData, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &metadata, nil
}

func (fs *fileStorage) Lock(ctx context.Context, key string) (Lock, error) {
	if key == "" {
		return nil, fmt.Errorf("lock key cannot be empty")
	}

	lockPath := filepath.Join(fs.basePath, ".locks", key+".lock")
	lockDir := filepath.Dir(lockPath)

	// Ensure lock directory exists
	if err := os.MkdirAll(lockDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create lock directory: %w", err)
	}

	// Try to create lock file with exclusive access
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}

	// Write lock timestamp
	if _, err := file.WriteString(time.Now().Format(time.RFC3339)); err != nil {
		file.Close()
		os.Remove(lockPath)
		return nil, fmt.Errorf("failed to write lock timestamp: %w", err)
	}

	return &fileLock{
		filePath: lockPath,
		file:     file,
	}, nil
}

func (fl *fileLock) Unlock() error {
	if fl.file != nil {
		fl.file.Close()
		fl.file = nil
	}

	if err := os.Remove(fl.filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove lock file: %w", err)
	}

	return nil
}

func (fl *fileLock) Refresh(ctx context.Context) error {
	if fl.file == nil {
		return fmt.Errorf("lock is not valid")
	}

	// Update timestamp in lock file
	if _, err := fl.file.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek lock file: %w", err)
	}

	if err := fl.file.Truncate(0); err != nil {
		return fmt.Errorf("failed to truncate lock file: %w", err)
	}

	if _, err := fl.file.WriteString(time.Now().Format(time.RFC3339)); err != nil {
		return fmt.Errorf("failed to update lock timestamp: %w", err)
	}

	return nil
}

func (fl *fileLock) IsValid() bool {
	if fl.file == nil {
		return false
	}

	// Check if lock file still exists
	_, err := os.Stat(fl.filePath)
	return err == nil
}
