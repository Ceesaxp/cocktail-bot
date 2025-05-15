package api

import (
	"crypto/subtle"
	"sync"
)

// AuthProvider handles API authentication
type AuthProvider struct {
	tokens map[string]bool
	mu     sync.RWMutex
}

// NewAuthProvider creates a new authentication provider with the given tokens
func NewAuthProvider(tokens []string) *AuthProvider {
	// Create a map for O(1) lookups
	tokenMap := make(map[string]bool, len(tokens))
	for _, token := range tokens {
		if token != "" {
			tokenMap[token] = true
		}
	}

	return &AuthProvider{
		tokens: tokenMap,
	}
}

// Authenticate validates the provided token
// Returns true if the token is valid
func (a *AuthProvider) Authenticate(token string) bool {
	// If we have no tokens, reject all requests
	a.mu.RLock()
	defer a.mu.RUnlock()

	if len(a.tokens) == 0 {
		return false
	}

	// Check for token in the map
	// Note: We use a constant-time comparison approach to 
	// prevent timing attacks, even though that's overkill 
	// for a simple API key check like this
	for validToken := range a.tokens {
		if subtle.ConstantTimeCompare([]byte(token), []byte(validToken)) == 1 {
			return true
		}
	}

	return false
}

// AddToken adds a new token to the provider
func (a *AuthProvider) AddToken(token string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if token != "" {
		a.tokens[token] = true
	}
}

// RemoveToken removes a token from the provider
func (a *AuthProvider) RemoveToken(token string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	delete(a.tokens, token)
}

// HasTokens returns true if there are tokens configured
func (a *AuthProvider) HasTokens() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return len(a.tokens) > 0
}