// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

// manager.go - Authentication state manager for MCP tools.
//
// This file provides an authentication manager that allows MCP tools to check,
// refresh, and manage authentication state independently of the main server startup.
// It maintains a global view of authentication status while ensuring thread safety.
//
// Key Features:
// - Thread-safe authentication state management
// - Authentication status reporting without exposing tokens
// - Manual token refresh capabilities
// - Re-authentication flow initiation
// - Authentication clearing (logout)
//
// Security Features:
// - Never exposes actual access/refresh tokens in responses
// - Secure state parameter generation for auth sessions
// - Timeout handling for auth sessions
// - Safe concurrent access to auth state

package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gebl/onenote-mcp-server/internal/logging"
)

// AuthStatus represents the current authentication state without exposing sensitive tokens
type AuthStatus struct {
	Authenticated         bool       `json:"authenticated"`
	TokenExpiry           *time.Time `json:"tokenExpiry,omitempty"`
	TokenExpiresIn        string     `json:"tokenExpiresIn,omitempty"`
	RefreshTokenAvailable bool       `json:"refreshTokenAvailable"`
	LastRefresh           *time.Time `json:"lastRefresh,omitempty"`
	AuthMethod            string     `json:"authMethod"`
	Message               string     `json:"message,omitempty"`
}

// AuthSession represents an active authentication session
type AuthSession struct {
	State           string    `json:"state"`
	CodeVerifier    string    `json:"codeVerifier"`
	CodeChallenge   string    `json:"codeChallenge"`
	AuthURL         string    `json:"authUrl"`
	CreatedAt       time.Time `json:"createdAt"`
	TimeoutMinutes  int       `json:"timeoutMinutes"`
	LocalServerPort int       `json:"localServerPort"`
}

// AuthManager manages authentication state and operations for MCP tools
type AuthManager struct {
	mu             sync.RWMutex
	oauthConfig    *OAuth2Config
	tokenManager   *TokenManager
	tokenPath      string
	activeSession  *AuthSession
	lastRefresh    *time.Time
	onTokenRefresh func(string) // Callback for when token is refreshed
	serverMode     string        // "stdio" or "http" - determines callback handling
	callbackChan   chan string   // Channel for receiving OAuth callback codes in HTTP mode
}

// NewAuthManager creates a new authentication manager
func NewAuthManager(oauthConfig *OAuth2Config, tokenManager *TokenManager, tokenPath string) *AuthManager {
	return &AuthManager{
		oauthConfig:  oauthConfig,
		tokenManager: tokenManager,
		tokenPath:    tokenPath,
		serverMode:   "stdio", // Default to stdio mode
		callbackChan: make(chan string, 1),
	}
}

// SetTokenRefreshCallback sets a callback function that will be called when tokens are refreshed
func (am *AuthManager) SetTokenRefreshCallback(callback func(string)) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.onTokenRefresh = callback
}

// SetServerMode sets the server mode ("stdio" or "http")
func (am *AuthManager) SetServerMode(mode string) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.serverMode = mode
	logging.AuthLogger.Info("Server mode configured for auth manager", "mode", mode)
}

// GetAuthStatus returns the current authentication status without exposing sensitive data
func (am *AuthManager) GetAuthStatus() *AuthStatus {
	am.mu.RLock()
	defer am.mu.RUnlock()

	status := &AuthStatus{
		AuthMethod: "OAuth2_PKCE",
	}

	if am.tokenManager == nil || am.tokenManager.AccessToken == "" {
		status.Authenticated = false
		status.Message = "No authentication tokens found"
		return status
	}

	status.Authenticated = true
	status.RefreshTokenAvailable = am.tokenManager.RefreshToken != ""

	if am.tokenManager.Expiry > 0 {
		expiry := time.Unix(am.tokenManager.Expiry, 0)
		status.TokenExpiry = &expiry

		timeUntilExpiry := time.Until(expiry)
		if timeUntilExpiry > 0 {
			status.TokenExpiresIn = formatDuration(timeUntilExpiry)
		} else {
			status.TokenExpiresIn = "expired"
		}
	}

	if am.lastRefresh != nil {
		status.LastRefresh = am.lastRefresh
	}

	// Check if token is expired
	if am.tokenManager.IsExpired() {
		status.Message = "Token is expired but can be refreshed"
	} else {
		status.Message = "Authentication is valid"
	}

	return status
}

// RefreshToken manually triggers a token refresh
func (am *AuthManager) RefreshToken() (*AuthStatus, error) {
	am.mu.Lock()
	defer am.mu.Unlock()

	if am.oauthConfig == nil || am.tokenManager == nil {
		return nil, fmt.Errorf("authentication not configured")
	}

	if am.tokenManager.RefreshToken == "" {
		return nil, fmt.Errorf("no refresh token available")
	}

	logging.AuthLogger.Info("Manually refreshing token via MCP tool")

	// Refresh the token
	newTokenManager, err := am.oauthConfig.RefreshToken(context.Background(), am.tokenManager.RefreshToken)
	if err != nil {
		logging.AuthLogger.Error("Token refresh failed", "error", err)
		return nil, fmt.Errorf("failed to refresh token: %v", err)
	}

	// Update the token manager
	am.tokenManager.AccessToken = newTokenManager.AccessToken
	am.tokenManager.RefreshToken = newTokenManager.RefreshToken
	am.tokenManager.Expiry = newTokenManager.Expiry

	// Save the new tokens
	if am.tokenPath != "" {
		if err := newTokenManager.SaveTokens(am.tokenPath); err != nil {
			logging.AuthLogger.Warn("Failed to save refreshed tokens", "error", err)
		}
	}

	now := time.Now()
	am.lastRefresh = &now

	// Call the token refresh callback if set
	if am.onTokenRefresh != nil {
		am.onTokenRefresh(newTokenManager.AccessToken)
	}

	logging.AuthLogger.Info("Token refresh successful via MCP tool")

	// Return updated status
	return am.GetAuthStatus(), nil
}

// InitiateAuth starts a new authentication flow with a local HTTP server
func (am *AuthManager) InitiateAuth() (*AuthSession, error) {
	am.mu.Lock()
	defer am.mu.Unlock()

	if am.oauthConfig == nil {
		return nil, fmt.Errorf("OAuth configuration not available")
	}

	logging.AuthLogger.Info("Initiating new authentication flow via MCP tool")

	// Generate PKCE parameters
	codeVerifier, err := GenerateCodeVerifier()
	if err != nil {
		return nil, fmt.Errorf("failed to generate code verifier: %v", err)
	}

	codeChallenge := CodeChallengeS256(codeVerifier)

	// Generate secure state parameter
	state, err := generateSecureState()
	if err != nil {
		return nil, fmt.Errorf("failed to generate state parameter: %v", err)
	}

	// Create auth URL
	authURL := am.oauthConfig.GetAuthURL(state, codeChallenge)

	// Create session
	session := &AuthSession{
		State:           state,
		CodeVerifier:    codeVerifier,
		CodeChallenge:   codeChallenge,
		AuthURL:         authURL,
		CreatedAt:       time.Now(),
		TimeoutMinutes:  10, // 10 minute timeout
		LocalServerPort: 8080,
	}

	am.activeSession = session

	// Only start a separate callback server in stdio mode
	// In HTTP mode, the main server will handle callbacks
	logging.AuthLogger.Info("Callback handling mode decision", "server_mode", am.serverMode, "will_start_separate_server", am.serverMode == "stdio")
	if am.serverMode == "stdio" {
		logging.AuthLogger.Info("Starting separate OAuth callback server for stdio mode")
		// Start the OAuth callback server in the background
		go am.startOAuthCallbackServer(session)
	} else {
		logging.AuthLogger.Info("Using main HTTP server for OAuth callbacks in HTTP mode")
		// In HTTP mode, start a goroutine to wait for the callback
		go am.waitForOAuthCallback(session)
	}

	logging.AuthLogger.Debug("Authentication session created", "state", state, "server_mode", am.serverMode)

	return session, nil
}

// startOAuthCallbackServer starts a temporary HTTP server to handle OAuth callback
func (am *AuthManager) startOAuthCallbackServer(session *AuthSession) {
	codeCh := make(chan string, 1)

	// Start local server to receive OAuth2 code
	server, err := StartLocalServer("/callback", codeCh, session.State)
	if err != nil {
		logging.AuthLogger.Error("Failed to start OAuth callback server", "error", err)
		return
	}

	// Set up timeout
	timeout := time.Duration(session.TimeoutMinutes) * time.Minute
	timeoutTimer := time.NewTimer(timeout)

	select {
	case code := <-codeCh:
		// We received the auth code
		timeoutTimer.Stop()

		// Shutdown the server gracefully
		logging.AuthLogger.Info("OAuth callback received, shutting down local HTTP server")
		if err := server.Shutdown(context.Background()); err != nil {
			logging.AuthLogger.Warn("Failed to shutdown HTTP server gracefully", "error", err)
		} else {
			logging.AuthLogger.Debug("Local HTTP server shut down successfully")
		}

		// Complete the authentication
		if err := am.CompleteAuth(code); err != nil {
			logging.AuthLogger.Error("Failed to complete authentication", "error", err)
		}

	case <-timeoutTimer.C:
		// Timeout occurred
		logging.AuthLogger.Info("OAuth session timed out", "timeout_minutes", session.TimeoutMinutes)

		// Shutdown the server
		if err := server.Shutdown(context.Background()); err != nil {
			logging.AuthLogger.Warn("Failed to shutdown HTTP server after timeout", "error", err)
		}

		// Clear the active session
		am.mu.Lock()
		am.activeSession = nil
		am.mu.Unlock()
	}
}

// CompleteAuth completes an authentication flow with the received code
func (am *AuthManager) CompleteAuth(code string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	if am.activeSession == nil {
		return fmt.Errorf("no active authentication session")
	}

	// Check if session has expired
	if time.Since(am.activeSession.CreatedAt) > time.Duration(am.activeSession.TimeoutMinutes)*time.Minute {
		am.activeSession = nil
		return fmt.Errorf("authentication session has expired")
	}

	logging.AuthLogger.Info("Completing authentication with received code")

	// Exchange code for tokens
	tokenManager, err := am.oauthConfig.ExchangeCode(context.Background(), code, am.activeSession.CodeVerifier)
	if err != nil {
		am.activeSession = nil
		return fmt.Errorf("failed to exchange code for tokens: %v", err)
	}

	// Update token manager
	am.tokenManager.AccessToken = tokenManager.AccessToken
	am.tokenManager.RefreshToken = tokenManager.RefreshToken
	am.tokenManager.Expiry = tokenManager.Expiry

	// Save tokens
	if am.tokenPath != "" {
		if err := tokenManager.SaveTokens(am.tokenPath); err != nil {
			logging.AuthLogger.Warn("Failed to save tokens", "error", err)
		}
	}

	now := time.Now()
	am.lastRefresh = &now
	am.activeSession = nil

	// Call the token refresh callback if set
	if am.onTokenRefresh != nil {
		am.onTokenRefresh(tokenManager.AccessToken)
	}

	logging.AuthLogger.Info("Authentication completed successfully via MCP tool")

	return nil
}

// ClearAuth clears stored authentication tokens (logout)
func (am *AuthManager) ClearAuth() error {
	am.mu.Lock()
	defer am.mu.Unlock()

	logging.AuthLogger.Info("Clearing authentication tokens via MCP tool")

	// Clear in-memory tokens
	if am.tokenManager != nil {
		am.tokenManager.AccessToken = ""
		am.tokenManager.RefreshToken = ""
		am.tokenManager.Expiry = 0
	}

	am.lastRefresh = nil
	am.activeSession = nil

	// Remove token file
	if am.tokenPath != "" {
		// Create empty token manager and save it (effectively clearing the file)
		emptyTokenManager := &TokenManager{}
		if err := emptyTokenManager.SaveTokens(am.tokenPath); err != nil {
			logging.AuthLogger.Warn("Failed to clear token file", "error", err)
		}
	}

	// Call the token refresh callback with empty token to clear Graph client
	if am.onTokenRefresh != nil {
		am.onTokenRefresh("")
	}

	logging.AuthLogger.Info("Authentication cleared successfully")

	return nil
}

// UpdateTokenManager updates the internal token manager (called by external refresh operations)
func (am *AuthManager) UpdateTokenManager(tokenManager *TokenManager) {
	am.mu.Lock()
	defer am.mu.Unlock()

	am.tokenManager = tokenManager
	now := time.Now()
	am.lastRefresh = &now
}

// GetActiveSession returns the current active authentication session, if any
func (am *AuthManager) GetActiveSession() *AuthSession {
	am.mu.RLock()
	defer am.mu.RUnlock()

	return am.activeSession
}

// HandleOAuthCallback handles OAuth callbacks when running in HTTP mode
func (am *AuthManager) HandleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	am.mu.RLock()
	session := am.activeSession
	am.mu.RUnlock()

	if session == nil {
		http.Error(w, "No active authentication session", http.StatusBadRequest)
		logging.AuthLogger.Warn("OAuth callback received with no active session")
		return
	}

	// Validate state parameter
	state := r.URL.Query().Get("state")
	if state != session.State {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		logging.AuthLogger.Warn("OAuth callback received with invalid state", "expected", session.State, "received", state)
		return
	}

	// Get authorization code
	code := r.URL.Query().Get("code")
	if code == "" {
		error := r.URL.Query().Get("error")
		errorDescription := r.URL.Query().Get("error_description")
		http.Error(w, fmt.Sprintf("OAuth error: %s - %s", error, errorDescription), http.StatusBadRequest)
		logging.AuthLogger.Error("OAuth callback error", "error", error, "description", errorDescription)
		return
	}

	// Send success response to user
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
	<title>Authentication Successful</title>
	<style>
		body { font-family: Arial, sans-serif; text-align: center; padding: 50px; }
		h1 { color: #4CAF50; }
	</style>
</head>
<body>
	<h1>Authentication Successful!</h1>
	<p>You may now close this window and return to your application.</p>
</body>
</html>`))

	// Send code through channel for processing
	select {
	case am.callbackChan <- code:
		logging.AuthLogger.Info("OAuth callback code sent for processing")
	default:
		logging.AuthLogger.Warn("OAuth callback channel is full, discarding code")
	}
}

// waitForOAuthCallback waits for OAuth callback when running in HTTP mode
func (am *AuthManager) waitForOAuthCallback(session *AuthSession) {
	// Set up timeout
	timeout := time.Duration(session.TimeoutMinutes) * time.Minute
	timeoutTimer := time.NewTimer(timeout)

	select {
	case code := <-am.callbackChan:
		// We received the auth code
		timeoutTimer.Stop()

		logging.AuthLogger.Info("OAuth callback code received in HTTP mode")

		// Complete the authentication
		if err := am.CompleteAuth(code); err != nil {
			logging.AuthLogger.Error("Failed to complete authentication", "error", err)
		}

	case <-timeoutTimer.C:
		// Timeout occurred
		logging.AuthLogger.Info("OAuth session timed out", "timeout_minutes", session.TimeoutMinutes)

		// Clear the active session
		am.mu.Lock()
		am.activeSession = nil
		am.mu.Unlock()
	}
}

// generateSecureState generates a cryptographically secure state parameter
func generateSecureState() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// formatDuration formats a duration in a human-readable format
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d seconds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%d minutes", int(d.Minutes()))
	} else if d < 24*time.Hour {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		return fmt.Sprintf("%d hours %d minutes", hours, minutes)
	} else {
		days := int(d.Hours()) / 24
		hours := int(d.Hours()) % 24
		return fmt.Sprintf("%d days %d hours", days, hours)
	}
}
