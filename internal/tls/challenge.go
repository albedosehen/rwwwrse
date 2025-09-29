package tls

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"
)

type httpChallengeHandler struct {
	challenges map[string]string // domain+token -> keyAuth
	mu         sync.RWMutex
}

func NewHTTPChallengeHandlerImpl() ChallengeHandler {
	return &httpChallengeHandler{
		challenges: make(map[string]string),
	}
}

func (h *httpChallengeHandler) HandleHTTP01Challenge(w http.ResponseWriter, r *http.Request) {
	// Extract token from URL path
	// Expected path: /.well-known/acme-challenge/{token}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := r.URL.Path
	if len(path) < 29 { // /.well-known/acme-challenge/ = 28 chars + token
		http.Error(w, "Invalid challenge path", http.StatusNotFound)
		return
	}

	token := path[28:] // Extract token after /.well-known/acme-challenge/
	if token == "" {
		http.Error(w, "Missing challenge token", http.StatusNotFound)
		return
	}

	// Get the host from the request
	host := r.Host
	if host == "" {
		http.Error(w, "Missing host header", http.StatusBadRequest)
		return
	}

	// Look up the key authorization
	keyAuth, err := h.GetChallengeData(host, token)
	if err != nil {
		http.Error(w, "Challenge not found", http.StatusNotFound)
		return
	}

	// Return the key authorization
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(keyAuth))
}

func (h *httpChallengeHandler) HandleTLSALPN01Challenge(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	// TLS-ALPN-01 challenge handling would go here
	// For now, we don't support TLS-ALPN-01 challenges
	return nil, fmt.Errorf("TLS-ALPN-01 challenges not implemented")
}

func (h *httpChallengeHandler) SetChallengeData(domain, token, keyAuth string) error {
	if domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}
	if token == "" {
		return fmt.Errorf("token cannot be empty")
	}
	if keyAuth == "" {
		return fmt.Errorf("keyAuth cannot be empty")
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	key := domain + ":" + token
	h.challenges[key] = keyAuth

	return nil
}

func (h *httpChallengeHandler) GetChallengeData(domain, token string) (string, error) {
	if domain == "" {
		return "", fmt.Errorf("domain cannot be empty")
	}
	if token == "" {
		return "", fmt.Errorf("token cannot be empty")
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	key := domain + ":" + token
	keyAuth, exists := h.challenges[key]
	if !exists {
		return "", fmt.Errorf("challenge data not found for domain %s and token %s", domain, token)
	}

	return keyAuth, nil
}

func (h *httpChallengeHandler) ClearChallengeData(domain, token string) error {
	if domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}
	if token == "" {
		return fmt.Errorf("token cannot be empty")
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	key := domain + ":" + token
	delete(h.challenges, key)

	return nil
}
