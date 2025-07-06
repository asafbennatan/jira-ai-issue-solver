package services

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"jira-ai-issue-solver/models"
)

// GitHubAppService handles GitHub App authentication and token generation
type GitHubAppService interface {
	// GetInstallationToken returns a valid installation access token (cached or fresh)
	GetInstallationToken() (string, error)

	// GetAppToken generates a new app-level JWT token
	GetAppToken() (string, error)
}

// tokenCache represents a cached installation token
type tokenCache struct {
	Token     string
	ExpiresAt time.Time
	mutex     sync.RWMutex
}

// GitHubAppServiceImpl implements the GitHubAppService interface
type GitHubAppServiceImpl struct {
	config     *models.Config
	client     *http.Client
	tokenCache *tokenCache
}

// NewGitHubAppService creates a new GitHubAppService
func NewGitHubAppService(config *models.Config) GitHubAppService {
	return &GitHubAppServiceImpl{
		config:     config,
		client:     &http.Client{},
		tokenCache: &tokenCache{},
	}
}

// GetAppToken generates a new app-level JWT token
func (s *GitHubAppServiceImpl) GetAppToken() (string, error) {
	// Parse the private key
	privateKey, err := s.parsePrivateKey()
	if err != nil {
		return "", fmt.Errorf("failed to parse private key: %w", err)
	}

	// Create JWT claims
	now := time.Now()
	claims := map[string]interface{}{
		"iat": now.Unix(),
		"exp": now.Add(10 * time.Minute).Unix(), // Token expires in 10 minutes
		"iss": s.config.GitHub.AppID,
	}

	// Generate JWT token
	token, err := s.generateJWT(privateKey, claims)
	if err != nil {
		return "", fmt.Errorf("failed to generate JWT: %w", err)
	}

	return token, nil
}

// GetInstallationToken returns a valid installation access token (cached or fresh)
func (s *GitHubAppServiceImpl) GetInstallationToken() (string, error) {
	// Check if we have a valid cached token
	s.tokenCache.mutex.RLock()
	if s.tokenCache.Token != "" && time.Now().Before(s.tokenCache.ExpiresAt.Add(-5*time.Minute)) {
		// Token is still valid (with 5-minute buffer)
		token := s.tokenCache.Token
		s.tokenCache.mutex.RUnlock()
		return token, nil
	}
	s.tokenCache.mutex.RUnlock()

	// Need to get a fresh token
	return s.refreshInstallationToken()
}

// refreshInstallationToken generates a new installation token and caches it
func (s *GitHubAppServiceImpl) refreshInstallationToken() (string, error) {
	// Acquire write lock for token refresh
	s.tokenCache.mutex.Lock()
	defer s.tokenCache.mutex.Unlock()

	// Double-check if another goroutine already refreshed the token
	if s.tokenCache.Token != "" && time.Now().Before(s.tokenCache.ExpiresAt.Add(-5*time.Minute)) {
		return s.tokenCache.Token, nil
	}

	// First get an app token
	appToken, err := s.GetAppToken()
	if err != nil {
		return "", fmt.Errorf("failed to get app token: %w", err)
	}

	// Use the app token to get an installation token
	url := fmt.Sprintf("https://api.github.com/app/installations/%d/access_tokens", s.config.GitHub.InstallationID)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", appToken))
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get installation token: %s, status code: %d", string(body), resp.StatusCode)
	}

	var tokenResponse struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Cache the token
	s.tokenCache.Token = tokenResponse.Token
	s.tokenCache.ExpiresAt = tokenResponse.ExpiresAt

	return tokenResponse.Token, nil
}

// parsePrivateKey parses the RSA private key from PEM format
func (s *GitHubAppServiceImpl) parsePrivateKey() (*rsa.PrivateKey, error) {
	// Decode the PEM block
	block, _ := pem.Decode([]byte(s.config.GitHub.AppPrivateKey))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	// Parse the private key
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS8 format
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}

		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("private key is not an RSA key")
		}

		return rsaKey, nil
	}

	return privateKey, nil
}

// generateJWT generates a JWT token using the provided private key and claims
func (s *GitHubAppServiceImpl) generateJWT(privateKey *rsa.PrivateKey, claims map[string]interface{}) (string, error) {
	// For simplicity, we'll use a basic JWT implementation
	// In a production environment, you might want to use a proper JWT library like github.com/golang-jwt/jwt

	// Create header
	header := map[string]string{
		"alg": "RS256",
		"typ": "JWT",
	}

	// Encode header and claims
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("failed to marshal header: %w", err)
	}

	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("failed to marshal claims: %w", err)
	}

	// Base64 encode header and claims
	headerB64 := base64URLEncode(headerJSON)
	claimsB64 := base64URLEncode(claimsJSON)

	// Create the payload to sign
	payload := headerB64 + "." + claimsB64

	// Sign the payload
	signature, err := s.signPayload(privateKey, payload)
	if err != nil {
		return "", fmt.Errorf("failed to sign payload: %w", err)
	}

	// Create the JWT token
	token := payload + "." + base64URLEncode(signature)

	return token, nil
}

// signPayload signs the payload using the RSA private key
func (s *GitHubAppServiceImpl) signPayload(privateKey *rsa.PrivateKey, payload string) ([]byte, error) {
	// Hash the payload
	hasher := sha256.New()
	hasher.Write([]byte(payload))
	hash := hasher.Sum(nil)

	// Sign the hash
	signature, err := rsa.SignPKCS1v15(nil, privateKey, crypto.SHA256, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to sign hash: %w", err)
	}

	return signature, nil
}

// base64URLEncode encodes data using base64url encoding (RFC 4648)
func base64URLEncode(data []byte) string {
	// Use base64url encoding (no padding, URL-safe characters)
	return base64.RawURLEncoding.EncodeToString(data)
}
