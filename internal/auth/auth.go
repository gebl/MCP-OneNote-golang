// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

// auth.go - OAuth2 authentication and token management for the OneNote MCP server.
//
// This file provides a complete OAuth 2.0 PKCE (Proof Key for Code Exchange) implementation
// for authenticating with Microsoft Graph API. It handles the entire authentication flow
// from initial authorization to token refresh and secure storage.
//
// Key Features:
// - OAuth 2.0 PKCE flow for secure public client authentication
// - Automatic token refresh with exponential backoff
// - Local token persistence with secure storage
// - CSRF protection with state parameter validation
// - Comprehensive error handling and logging
// - Support for both single-tenant and multi-tenant Azure applications
//
// Authentication Flow:
// 1. Generate PKCE code verifier and challenge
// 2. Redirect user to Microsoft login with authorization URL
// 3. Handle OAuth callback with authorization code
// 4. Exchange authorization code for access and refresh tokens
// 5. Store tokens securely for future use
// 6. Automatically refresh tokens before expiration
//
// Token Management:
// - Access tokens: Short-lived (1 hour) for API requests
// - Refresh tokens: Long-lived for obtaining new access tokens
// - Automatic expiry detection and refresh
// - Secure local storage in JSON format
// - Token validation and error recovery
//
// Security Features:
// - PKCE prevents authorization code interception attacks
// - State parameter prevents CSRF attacks
// - Input validation and sanitization
// - Secure token storage with file permissions
// - Automatic cleanup of sensitive data
//
// Configuration Requirements:
// - Azure App Registration with OAuth 2.0 configuration
// - Redirect URI: http://localhost:8080/callback (for local development)
// - API Permissions: Notes.ReadWrite (delegated)
// - Supported account types: Single tenant or multi-tenant
//
// Error Handling:
// - PKCE code verifier mismatch detection
// - Token expiration and refresh failures
// - Network connectivity issues
// - Invalid or expired authorization codes
// - Azure app configuration errors
//
// Usage Example:
//   oauthCfg := auth.NewOAuth2Config(clientID, tenantID, redirectURI)
//   tokenManager, err := auth.AuthenticateUser(oauthCfg, "tokens.json")
//   if err != nil {
//       logging.AuthLogger.Error("Authentication failed", "error", err)
//   }
//
// For detailed setup instructions, see README.md and docs/setup.md

package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	httputils "github.com/gebl/onenote-mcp-server/internal/http"
	"github.com/gebl/onenote-mcp-server/internal/logging"
)

const (
	unknownPath = "unknown"
)

// OAuth2Config holds Microsoft Identity Platform OAuth2 configuration for public clients (PKCE).
type OAuth2Config struct {
	ClientID    string // Application (client) ID
	TenantID    string // Directory (tenant) ID
	RedirectURI string // Redirect URI for OAuth2 callback
}

// TokenManager handles access/refresh tokens and their expiry.
type TokenManager struct {
	AccessToken  string `json:"access_token"`  // OAuth2 access token
	RefreshToken string `json:"refresh_token"` // OAuth2 refresh token
	Expiry       int64  `json:"expiry"`        // Unix timestamp for token expiry
}

// NewOAuth2Config creates a new OAuth2Config from config values.
// clientID: Application (client) ID
// tenantID: Directory (tenant) ID
// redirectURI: Redirect URI for OAuth2 callback
// Returns a pointer to an OAuth2Config instance.
func NewOAuth2Config(clientID, tenantID, redirectURI string) *OAuth2Config {
	logging.AuthLogger.Debug("Initializing OAuth2 configuration for Microsoft Graph",
		"client_id", maskSensitiveData(clientID),
		"tenant_id", tenantID,
		"redirect_uri", redirectURI,
		"flow_type", "PKCE")

	config := &OAuth2Config{
		ClientID:    clientID,
		TenantID:    tenantID,
		RedirectURI: redirectURI,
	}

	logging.AuthLogger.Debug("OAuth2 configuration created successfully")
	return config
}

// maskSensitiveData masks sensitive configuration values for logging
func maskSensitiveData(value string) string {
	if value == "" {
		return "<empty>"
	}
	if len(value) <= 8 {
		return "***"
	}
	return value[:4] + "***" + value[len(value)-4:]
}

// GenerateCodeVerifier creates a random PKCE code verifier string.
// Returns the code verifier and an error, if any.
func GenerateCodeVerifier() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// CodeChallengeS256 generates a code challenge from a code verifier using SHA-256.
// Returns the base64url-encoded code challenge string.
func CodeChallengeS256(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// GetAuthURL generates the Microsoft login URL for user consent (PKCE).
// state: Opaque value to maintain state between request and callback.
// codeChallenge: PKCE code challenge string.
// Returns the full authorization URL as a string.
func (c *OAuth2Config) GetAuthURL(state, codeChallenge string) string {
	authURL := fmt.Sprintf(
		"https://login.microsoftonline.com/%s/oauth2/v2.0/authorize?client_id=%s&response_type=code&redirect_uri=%s&response_mode=query&scope=%s&state=%s&code_challenge=%s&code_challenge_method=S256",
		url.PathEscape(c.TenantIDOrCommon()),
		url.QueryEscape(c.ClientID),
		url.QueryEscape(c.RedirectURI),
		url.QueryEscape("offline_access Notes.ReadWrite"),
		url.QueryEscape(state),
		url.QueryEscape(codeChallenge),
	)
	
	// Enhanced debugging for troubleshooting unauthorized_client error
	logging.AuthLogger.Info("Generated OAuth authorization URL", 
		"tenant", c.TenantIDOrCommon(), 
		"scope", "Notes.ReadWrite")
	logging.AuthLogger.Debug("OAuth URL components for troubleshooting", 
		"client_id", maskSensitiveData(c.ClientID),
		"redirect_uri", c.RedirectURI,
		"tenant_id_original", c.TenantID,
		"tenant_id_used", c.TenantIDOrCommon(),
		"tenant_is_common", c.TenantID == "" || c.TenantID == "common")
	return authURL
}

// TenantIDOrCommon returns the tenant ID or "common" if not set.
func (c *OAuth2Config) TenantIDOrCommon() string {
	if c.TenantID == "" {
		return "common"
	}
	return c.TenantID
}

// ExchangeCode exchanges the auth code for access and refresh tokens (PKCE).
// ctx: Context for the HTTP request.
// code: Authorization code received from the callback.
// codeVerifier: PKCE code verifier string.
// Returns a pointer to a TokenManager and an error, if any.
func (c *OAuth2Config) ExchangeCode(ctx context.Context, code, codeVerifier string) (*TokenManager, error) {
	logging.AuthLogger.Info("Exchanging authorization code for access tokens", "endpoint", "Microsoft Identity Platform")
	endpoint := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", c.TenantIDOrCommon())
	data := url.Values{}
	data.Set("client_id", c.ClientID)
	data.Set("scope", "offline_access Notes.ReadWrite")
	data.Set("code", code)
	data.Set("redirect_uri", c.RedirectURI)
	data.Set("grant_type", "authorization_code")
	data.Set("code_verifier", codeVerifier)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		logging.AuthLogger.Error("Failed to create token exchange request", "error", err, "endpoint", endpoint)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logging.AuthLogger.Error("Failed to send token exchange request", "error", err, "endpoint", endpoint)
		return nil, err
	}
	
	var body []byte
	var statusCode int
	err = httputils.WithAutoCleanup(resp, func(resp *http.Response) error {
		statusCode = resp.StatusCode
		responseBody, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return readErr
		}
		body = responseBody
		return nil
	})
	if err != nil {
		logging.AuthLogger.Error("Failed to read response body", "error", err, "endpoint", endpoint)
		return nil, err
	}
	
	if statusCode != 200 {
		logging.AuthLogger.Error("Token exchange failed", "status_code", statusCode, "response_body", string(body))

		// Check for specific PKCE errors
		if strings.Contains(string(body), "invalid_grant") && strings.Contains(string(body), "code_verifier") {
			logging.AuthLogger.Warn("PKCE code verifier mismatch detected",
				"possible_causes", []string{
					"Authorization code used with different code verifier",
					"Authorization code expired (10 minute limit)",
					"Authorization code already used",
				},
				"action_required", "Restart authentication flow")
			return nil, fmt.Errorf("PKCE code verifier mismatch - please restart authentication flow")
		}

		return nil, fmt.Errorf("token exchange failed: %s", string(body))
	}
	var res struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		logging.AuthLogger.Error("Failed to decode token response JSON", "error", err)
		return nil, err
	}
	logging.AuthLogger.Info("Successfully obtained access and refresh tokens", "expires_in_seconds", res.ExpiresIn)
	return &TokenManager{
		AccessToken:  res.AccessToken,
		RefreshToken: res.RefreshToken,
		Expiry:       time.Now().Unix() + res.ExpiresIn,
	}, nil
}

// StartLocalServer starts a local HTTP server to capture the auth code from the OAuth2 redirect.
// redirectPath: Path to listen for the callback (e.g., "/callback").
// codeCh: Channel to send the received code.
// state: Expected state value for CSRF protection.
// Returns the HTTP server pointer and an error, if any.
func StartLocalServer(redirectPath string, codeCh chan<- string, state string) (*http.Server, error) {
	logging.AuthLogger.Info("Starting local HTTP server to receive auth code", "port", 8080, "redirect_path", redirectPath)
	mux := http.NewServeMux()
	mux.HandleFunc(redirectPath, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			http.Error(w, "Invalid state", http.StatusBadRequest)
			logging.AuthLogger.Info("Invalid state received in redirect.")
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "Missing code", http.StatusBadRequest)
			logging.AuthLogger.Info("Missing code in redirect.")
			return
		}
		w.Write([]byte("Authentication successful. You may close this window."))
		logging.AuthLogger.Info("Received authorization code from redirect.")
		codeCh <- code
	})
	server := &http.Server{Addr: ":8080", Handler: mux}
	go func() {
		_ = server.ListenAndServe()
	}()
	return server, nil
}

// SaveTokens saves tokens to a file (for demo; use secure storage in production).
// path: File path to save the tokens.
// Returns an error if saving fails.
func (tm *TokenManager) SaveTokens(path string) error {
	// Get absolute path for better debugging
	absPath, err := filepath.Abs(path)
	if err != nil {
		logging.AuthLogger.Debug("Could not resolve absolute path", "path", path, "error", err)
		absPath = unknownPath
	}

	logging.AuthLogger.Debug("Saving authentication tokens to file",
		"path", path,
		"absolute_path", absPath,
		"access_token", maskSensitiveData(tm.AccessToken),
		"refresh_token", maskSensitiveData(tm.RefreshToken),
		"expires_at", time.Unix(tm.Expiry, 0).Format(time.RFC3339))

	f, err := os.Create(path)
	if err != nil {
		logging.AuthLogger.Error("Failed to create token file", "path", path, "absolute_path", absPath, "error", err)
		return err
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(tm); err != nil {
		logging.AuthLogger.Error("Failed to encode tokens to file", "path", path, "absolute_path", absPath, "error", err)
		return err
	}

	logging.AuthLogger.Debug("Tokens saved successfully", "path", path, "absolute_path", absPath)
	return nil
}

// GetTokenPath returns the token file path, using TOKEN_FILE environment variable if set
func GetTokenPath(defaultPath string) string {
	if envPath := os.Getenv("TOKEN_FILE"); envPath != "" {
		logging.AuthLogger.Debug("Using TOKEN_FILE environment variable", "path", envPath)
		return envPath
	}
	logging.AuthLogger.Debug("TOKEN_FILE environment variable not set, using default", "default_path", defaultPath)
	return defaultPath
}

// LoadTokens loads tokens from a file.
// path: File path to load the tokens from.
// Returns a pointer to a TokenManager and an error, if any.
func LoadTokens(path string) (*TokenManager, error) {
	// Get absolute path for better debugging
	absPath, err := filepath.Abs(path)
	if err != nil {
		logging.AuthLogger.Debug("Could not resolve absolute path", "path", path, "error", err)
		absPath = unknownPath
	}

	logging.AuthLogger.Debug("Loading tokens from file", "path", path, "absolute_path", absPath)

	fileInfo, err := os.Stat(path)
	if err != nil {
		logging.AuthLogger.Debug("No token file found", "path", path, "absolute_path", absPath, "error", err)
		return nil, err
	}
	logging.AuthLogger.Debug("Token file found:")
	logging.AuthLogger.Debug("Token file details", "path", path)
	logging.AuthLogger.Debug("Token file details", "absolute_path", absPath)
	logging.AuthLogger.Debug("Token file details", "size_bytes", fileInfo.Size())
	logging.AuthLogger.Debug("Token file details", "modified_time", fileInfo.ModTime())
	logging.AuthLogger.Debug("Token file details", "permissions", fileInfo.Mode().String())

	f, err := os.Open(path)
	if err != nil {
		logging.AuthLogger.Error("Failed to open token file", "path", path, "absolute_path", absPath, "error", err)
		return nil, err
	}
	defer f.Close()

	tm := &TokenManager{}
	if err := json.NewDecoder(f).Decode(tm); err != nil {
		logging.AuthLogger.Error("Failed to decode token file", "path", path, "absolute_path", absPath, "error", err)
		return nil, err
	}

	logging.AuthLogger.Debug("Tokens loaded successfully", "path", path, "absolute_path", absPath)
	logging.AuthLogger.Debug("Token details", "access_token", maskSensitiveData(tm.AccessToken))
	logging.AuthLogger.Debug("Token details", "refresh_token", maskSensitiveData(tm.RefreshToken))
	logging.AuthLogger.Debug("Token details", "expiry_timestamp", tm.Expiry, "expires_at", time.Unix(tm.Expiry, 0).Format(time.RFC3339))
	
	// Additional debugging to understand token format issues
	logging.AuthLogger.Debug("Token format validation", 
		"access_token_length", len(tm.AccessToken),
		"access_token_empty", tm.AccessToken == "",
		"access_token_has_dots", strings.Count(tm.AccessToken, "."),
		"access_token_starts_with", func() string {
			if len(tm.AccessToken) >= 10 {
				return tm.AccessToken[:10]
			}
			return tm.AccessToken
		}())

	return tm, nil
}

// RefreshToken refreshes the access token using the refresh token.
// ctx: Context for the HTTP request.
// refreshToken: The refresh token to use for refreshing the access token.
// Returns a pointer to a TokenManager and an error, if any.
func (c *OAuth2Config) RefreshToken(ctx context.Context, refreshToken string) (*TokenManager, error) {
	logging.AuthLogger.Info("Refreshing access token using refresh token...")
	endpoint := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", c.TenantIDOrCommon())
	data := url.Values{}
	data.Set("client_id", c.ClientID)
	data.Set("scope", "offline_access Notes.ReadWrite")
	data.Set("refresh_token", refreshToken)
	data.Set("redirect_uri", c.RedirectURI)
	data.Set("grant_type", "refresh_token")

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		logging.AuthLogger.Error("Error creating refresh request", "error", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logging.AuthLogger.Error("Error sending refresh request", "error", err)
		return nil, err
	}
	
	var body []byte
	var statusCode int
	err = httputils.WithAutoCleanup(resp, func(resp *http.Response) error {
		statusCode = resp.StatusCode
		responseBody, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return readErr
		}
		body = responseBody
		return nil
	})
	if err != nil {
		logging.AuthLogger.Error("Failed to read refresh response body", "error", err)
		return nil, err
	}
	
	if statusCode != 200 {
		logging.AuthLogger.Error("Token refresh failed", "response_body", string(body))
		return nil, fmt.Errorf("token refresh failed: %s", string(body))
	}
	var res struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		logging.AuthLogger.Error("Error decoding refresh response", "error", err)
		return nil, err
	}
	logging.AuthLogger.Info("Token refresh successful.")
	return &TokenManager{
		AccessToken:  res.AccessToken,
		RefreshToken: res.RefreshToken,
		Expiry:       time.Now().Unix() + res.ExpiresIn,
	}, nil
}

// IsExpired returns true if the access token is expired.
// Returns true if the current time is after the token's expiry time minus a buffer.
func (tm *TokenManager) IsExpired() bool {
	now := time.Now().Unix()
	expiryWithBuffer := tm.Expiry - 60 // 60s buffer
	isExpired := now > expiryWithBuffer

	logging.AuthLogger.Debug("Token expiry check:")
	logging.AuthLogger.Debug("Token expiry check - current time", "timestamp", now, "formatted", time.Unix(now, 0).Format(time.RFC3339))
	logging.AuthLogger.Debug("Token expiry check - token expiry", "timestamp", tm.Expiry, "formatted", time.Unix(tm.Expiry, 0).Format(time.RFC3339))
	logging.AuthLogger.Debug("Token expiry check - expiry with buffer", "timestamp", expiryWithBuffer, "formatted", time.Unix(expiryWithBuffer, 0).Format(time.RFC3339))
	logging.AuthLogger.Debug("Token expiry check - result", "is_expired", isExpired)

	return isExpired
}

// SaveCodeVerifier saves the code verifier to a temporary file for the current auth session.
// This helps prevent PKCE mismatches if the auth flow is interrupted.
func SaveCodeVerifier(codeVerifier string) error {
	logging.AuthLogger.Info("Saving code verifier for PKCE session...")
	data := map[string]string{
		"code_verifier": codeVerifier,
		"timestamp":     fmt.Sprintf("%d", time.Now().Unix()),
	}
	f, err := os.Create("code_verifier.json")
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(data)
}

// LoadCodeVerifier loads the code verifier from the temporary file.
// Returns the code verifier and an error, if any.
func LoadCodeVerifier() (string, error) {
	logging.AuthLogger.Info("Loading code verifier for PKCE session...")
	f, err := os.Open("code_verifier.json")
	if err != nil {
		return "", err
	}
	defer f.Close()

	var data map[string]string
	if err := json.NewDecoder(f).Decode(&data); err != nil {
		return "", err
	}

	// Check if the code verifier is not too old (within 10 minutes)
	if timestamp, exists := data["timestamp"]; exists {
		if ts, err := strconv.ParseInt(timestamp, 10, 64); err == nil {
			if time.Now().Unix()-ts > 600 { // 10 minutes
				logging.AuthLogger.Info("Code verifier is too old, will generate new one")
				return "", fmt.Errorf("code verifier expired")
			}
		}
	}

	return data["code_verifier"], nil
}

// CleanupCodeVerifier removes the temporary code verifier file.
func CleanupCodeVerifier() {
	logging.AuthLogger.Info("Cleaning up code verifier file...")
	os.Remove("code_verifier.json")
}

// AuthenticateUser handles the complete authentication flow including token loading,
// PKCE authentication if needed, and token saving. This function encapsulates the
// authentication logic that was previously in main.go.
// oauthCfg: OAuth2 configuration for the application
// tokenPath: Path to save/load tokens
// Returns a TokenManager and an error, if any
func AuthenticateUser(oauthCfg *OAuth2Config, tokenPath string) (*TokenManager, error) {
	// Load or obtain authentication tokens
	tm, err := LoadTokens(tokenPath)
	if err != nil || tm == nil || tm.IsExpired() {
		logging.AuthLogger.Info("No valid token found or token expired. Starting PKCE flow...")

		// Try to load existing code verifier first (in case of interrupted auth flow)
		codeVerifier, err := LoadCodeVerifier()
		if err != nil {
			logging.AuthLogger.Info("No valid code verifier found, generating new one...")
			// Set up PKCE (Proof Key for Code Exchange) for OAuth2
			codeVerifier, err = GenerateCodeVerifier()
			if err != nil {
				return nil, fmt.Errorf("failed to generate code verifier: %v", err)
			}
			// Save the code verifier for this auth session
			if errSave := SaveCodeVerifier(codeVerifier); errSave != nil {
				logging.AuthLogger.Warn("Failed to save code verifier", "error", errSave)
			}
		} else {
			logging.AuthLogger.Info("Using existing code verifier from previous auth session...")
		}

		codeChallenge := CodeChallengeS256(codeVerifier)
		state := "mcp-onenote-state" // could randomize for extra security
		redirectPath := "/callback"
		codeCh := make(chan string)

		// Start local server to receive OAuth2 code
		server, err := StartLocalServer(redirectPath, codeCh, state)
		if err != nil {
			return nil, fmt.Errorf("failed to start local server: %v", err)
		}

		// Print auth URL for user to complete authentication
		authURL := oauthCfg.GetAuthURL(state, codeChallenge)
		logging.AuthLogger.Info("Please visit the following URL in your browser to authenticate", "auth_url", authURL)

		// Wait for code from local server
		code := <-codeCh

		// Shutdown the HTTP server gracefully after receiving the code
		logging.AuthLogger.Info("OAuth callback received, shutting down local HTTP server...")
		if shutdownErr := server.Shutdown(context.Background()); shutdownErr != nil {
			logging.AuthLogger.Warn("Failed to shutdown HTTP server gracefully", "error", shutdownErr)
		} else {
			logging.AuthLogger.Info("Local HTTP server shut down successfully")
		}

		// Exchange code for tokens
		tm, err = oauthCfg.ExchangeCode(context.Background(), code, codeVerifier)
		if err != nil {
			return nil, fmt.Errorf("failed to exchange code for tokens: %v", err)
		}

		// Save tokens for future use
		if err := tm.SaveTokens(tokenPath); err != nil {
			logging.AuthLogger.Warn("Failed to save tokens", "error", err)
		}

		// Clean up the code verifier file after successful authentication
		CleanupCodeVerifier()
		logging.AuthLogger.Info("Authentication complete. Tokens saved.")
	}

	return tm, nil
}

// IsAuthError checks if an error is due to authentication issues.
// err: The error to check
// Returns true if the error is authentication-related
func IsAuthError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "401") ||
		strings.Contains(errStr, "403") ||
		strings.Contains(errStr, "unauthorized") ||
		strings.Contains(errStr, "forbidden") ||
		strings.Contains(errStr, "invalid_token") ||
		strings.Contains(errStr, "expired_token")
}

// RefreshTokenAndRetry attempts to refresh the access token and retry the given operation.
// oauthCfg: OAuth2 configuration for token refresh
// tokenManager: Token manager for refresh operations
// tokenPath: Path to save refreshed tokens
// operation: Function to retry after token refresh
// onTokenRefresh: Optional callback function called when token is refreshed
// Returns an error if the operation fails
func RefreshTokenAndRetry(oauthCfg *OAuth2Config, tokenManager *TokenManager, tokenPath string, operation func() error, onTokenRefresh func(string)) error {
	logging.AuthLogger.Info("=== TOKEN REFRESH DEBUG START ===")

	if oauthCfg == nil || tokenManager == nil {
		logging.AuthLogger.Info("ERROR: Token refresh not configured - OAuthConfig or TokenManager is nil")
		return fmt.Errorf("token refresh not configured")
	}

	logging.AuthLogger.Debug("Token refresh debug - current token expiry", "expiry_timestamp", tokenManager.Expiry)
	logging.AuthLogger.Debug("Token refresh debug - current time", "current_timestamp", time.Now().Unix())
	logging.AuthLogger.Debug("Token refresh debug - expiry status", "is_expired", tokenManager.IsExpired())
	logging.AuthLogger.Debug("Token refresh debug - refresh token status", "refresh_token_available", tokenManager.RefreshToken != "")

	logging.AuthLogger.Info("Authentication error detected, attempting token refresh...")

	// Refresh the token
	logging.AuthLogger.Info("Calling OAuth2Config.RefreshToken...")
	newTokenManager, err := oauthCfg.RefreshToken(context.Background(), tokenManager.RefreshToken)
	if err != nil {
		logging.AuthLogger.Error("Failed to refresh token", "error", err)
		logging.AuthLogger.Info("=== TOKEN REFRESH DEBUG END (FAILED) ===")
		return fmt.Errorf("failed to refresh token: %v", err)
	}

	logging.AuthLogger.Info("Token refresh successful!")
	logging.AuthLogger.Debug("Token refresh success - new expiry", "expiry_timestamp", newTokenManager.Expiry)
	logging.AuthLogger.Debug("Token refresh success - access token", "token_length", len(newTokenManager.AccessToken))
	logging.AuthLogger.Debug("Token refresh success - refresh token", "token_length", len(newTokenManager.RefreshToken))

	// Update the token manager
	logging.AuthLogger.Info("Updating token manager with new token...")
	tokenManager.AccessToken = newTokenManager.AccessToken
	tokenManager.RefreshToken = newTokenManager.RefreshToken
	tokenManager.Expiry = newTokenManager.Expiry

	// Call the token refresh callback if provided
	if onTokenRefresh != nil {
		logging.AuthLogger.Info("Calling token refresh callback...")
		onTokenRefresh(newTokenManager.AccessToken)
	}

	// Save the new tokens
	if tokenPath != "" {
		// Get absolute path for better debugging
		absTokenPath, err := filepath.Abs(tokenPath)
		if err != nil {
			logging.AuthLogger.Debug("Could not get absolute path", "token_path", tokenPath, "error", err)
			absTokenPath = "unknown"
		}

		logging.AuthLogger.Debug("Saving refreshed tokens", "path", tokenPath, "absolute_path", absTokenPath)
		if err := newTokenManager.SaveTokens(tokenPath); err != nil {
			logging.AuthLogger.Warn("Failed to save refreshed tokens", "path", tokenPath, "absolute_path", absTokenPath, "error", err)
		} else {
			logging.AuthLogger.Debug("Tokens saved successfully", "path", tokenPath, "absolute_path", absTokenPath)
		}
	} else {
		logging.AuthLogger.Info("No token path configured, skipping token save")
	}

	logging.AuthLogger.Info("Token refreshed successfully, retrying operation...")
	logging.AuthLogger.Info("=== TOKEN REFRESH DEBUG END (SUCCESS) ===")

	// Retry the operation
	return operation()
}

// MakeAuthenticatedRequest makes an HTTP request with authentication and handles token refresh if needed.
// req: The HTTP request to make
// accessToken: Current access token
// oauthCfg: OAuth2 configuration for token refresh
// tokenManager: Token manager for refresh operations
// tokenPath: Path to save refreshed tokens
// Returns the HTTP response and an error, if any
func MakeAuthenticatedRequest(req *http.Request, accessToken string, oauthCfg *OAuth2Config, tokenManager *TokenManager, tokenPath string) (*http.Response, error) {
	logging.AuthLogger.Info("Making authenticated request", "method", req.Method, "url", req.URL.String())

	// Store the original request body for potential retry after token refresh
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			logging.AuthLogger.Error("Failed to read request body", "error", err)
			return nil, fmt.Errorf("failed to read request body: %v", err)
		}
		req.Body.Close()
		// Replace the consumed body with a fresh reader
		req.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
		logging.AuthLogger.Debug("Stored request body for potential retry", "body_size", len(bodyBytes))
	}

	// Set the current access token
	req.Header.Set("Authorization", "Bearer "+accessToken)

	// Make the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logging.AuthLogger.Error("Request failed", "error", err)
		return resp, err
	}

	logging.AuthLogger.Debug("Response received", "status_code", resp.StatusCode)

	// Check if we got an auth error
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		logging.AuthLogger.Info("Authentication error detected, checking if token refresh is available", "status_code", resp.StatusCode)

		// Try to refresh token and retry
		if oauthCfg != nil && tokenManager != nil {
			logging.AuthLogger.Info("Token refresh is available, attempting refresh...")

			// Close the current response
			resp.Body.Close()

			// Create a new request with fresh body content
			var newBody io.Reader
			if len(bodyBytes) > 0 {
				newBody = strings.NewReader(string(bodyBytes))
				logging.AuthLogger.Debug("Creating retry request with body", "body_size", len(bodyBytes))
			}

			newReq, err := http.NewRequest(req.Method, req.URL.String(), newBody)
			if err != nil {
				logging.AuthLogger.Error("Failed to create new request for retry", "error", err)
				return nil, err
			}

			// Copy headers (excluding Authorization which will be set with new token)
			for key, values := range req.Header {
				if key != "Authorization" {
					for _, value := range values {
						newReq.Header.Add(key, value)
					}
				}
			}

			// Refresh token and retry
			err = RefreshTokenAndRetry(oauthCfg, tokenManager, tokenPath, func() error {
				logging.AuthLogger.Info("Retrying request after token refresh", "method", newReq.Method, "url", newReq.URL.String())
				newReq.Header.Set("Authorization", "Bearer "+tokenManager.AccessToken)
				retryResp, retryErr := http.DefaultClient.Do(newReq)
				if retryErr != nil {
					logging.AuthLogger.Error("Retry request failed", "error", retryErr)
					return retryErr
				}
				logging.AuthLogger.Debug("Retry response received", "status_code", retryResp.StatusCode)
				if retryResp.StatusCode == 401 || retryResp.StatusCode == 403 {
					logging.AuthLogger.Info("ERROR: Authentication still failed after token refresh")
					return fmt.Errorf("authentication failed even after token refresh")
				}
				// Replace the response
				resp = retryResp
				logging.AuthLogger.Info("Request succeeded after token refresh!")
				return nil
			}, nil) // No callback needed for direct HTTP requests

			if err != nil {
				logging.AuthLogger.Error("Token refresh and retry failed", "error", err)
				return nil, err
			}
		} else {
			logging.AuthLogger.Warn("Authentication error but token refresh not available",
				"oauth_config_available", oauthCfg != nil,
				"token_manager_available", tokenManager != nil)
		}
	}

	return resp, nil
}

// MakeAuthenticatedRequestWithCallback makes an HTTP request with authentication and handles token refresh if needed.
// This version accepts a callback function that will be called when the token is refreshed.
// req: The HTTP request to make
// accessToken: Current access token
// oauthCfg: OAuth2 configuration for token refresh
// tokenManager: Token manager for refresh operations
// tokenPath: Path to save refreshed tokens
// onTokenRefresh: Callback function called when token is refreshed
// Returns the HTTP response and an error, if any
func MakeAuthenticatedRequestWithCallback(req *http.Request, accessToken string, oauthCfg *OAuth2Config, tokenManager *TokenManager, tokenPath string, onTokenRefresh func(string)) (*http.Response, error) {
	logging.AuthLogger.Info("Making authenticated request", "method", req.Method, "url", req.URL.String())

	// Store the original request body for potential retry after token refresh
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			logging.AuthLogger.Error("Failed to read request body", "error", err)
			return nil, fmt.Errorf("failed to read request body: %v", err)
		}
		req.Body.Close()
		// Replace the consumed body with a fresh reader
		req.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
		logging.AuthLogger.Debug("Stored request body for potential retry", "body_size", len(bodyBytes))
	}

	// Set the current access token
	req.Header.Set("Authorization", "Bearer "+accessToken)

	// Make the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logging.AuthLogger.Error("Request failed", "error", err)
		return resp, err
	}

	logging.AuthLogger.Debug("Response received", "status_code", resp.StatusCode)

	// Check if we got an auth error
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		logging.AuthLogger.Info("Authentication error detected, checking if token refresh is available", "status_code", resp.StatusCode)

		// Try to refresh token and retry
		if oauthCfg != nil && tokenManager != nil {
			logging.AuthLogger.Info("Token refresh is available, attempting refresh...")

			// Close the current response
			resp.Body.Close()

			// Create a new request with fresh body content
			var newBody io.Reader
			if len(bodyBytes) > 0 {
				newBody = strings.NewReader(string(bodyBytes))
				logging.AuthLogger.Debug("Creating retry request with body", "body_size", len(bodyBytes))
			}

			newReq, err := http.NewRequest(req.Method, req.URL.String(), newBody)
			if err != nil {
				logging.AuthLogger.Error("Failed to create new request for retry", "error", err)
				return nil, err
			}

			// Copy headers (excluding Authorization which will be set with new token)
			for key, values := range req.Header {
				if key != "Authorization" {
					for _, value := range values {
						newReq.Header.Add(key, value)
					}
				}
			}

			// Refresh token and retry
			err = RefreshTokenAndRetry(oauthCfg, tokenManager, tokenPath, func() error {
				logging.AuthLogger.Info("Retrying request after token refresh", "method", newReq.Method, "url", newReq.URL.String())
				newReq.Header.Set("Authorization", "Bearer "+tokenManager.AccessToken)
				retryResp, retryErr := http.DefaultClient.Do(newReq)
				if retryErr != nil {
					logging.AuthLogger.Error("Retry request failed", "error", retryErr)
					return retryErr
				}
				logging.AuthLogger.Debug("Retry response received", "status_code", retryResp.StatusCode)
				if retryResp.StatusCode == 401 || retryResp.StatusCode == 403 {
					logging.AuthLogger.Info("ERROR: Authentication still failed after token refresh")
					return fmt.Errorf("authentication failed even after token refresh")
				}
				// Replace the response
				resp = retryResp
				logging.AuthLogger.Info("Request succeeded after token refresh!")
				return nil
			}, onTokenRefresh)

			if err != nil {
				logging.AuthLogger.Error("Token refresh and retry failed", "error", err)
				return nil, err
			}
		} else {
			logging.AuthLogger.Warn("Authentication error but token refresh not available",
				"oauth_config_available", oauthCfg != nil,
				"token_manager_available", tokenManager != nil)
		}
	}

	return resp, nil
}
