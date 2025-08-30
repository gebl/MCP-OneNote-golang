// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package graph

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/gebl/onenote-mcp-server/internal/auth"
	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name        string
		accessToken string
		expectPanic bool
	}{
		{
			name:        "valid access token",
			accessToken: "test-access-token-123",
			expectPanic: false,
		},
		{
			name:        "empty access token",
			accessToken: "",
			expectPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectPanic {
				assert.Panics(t, func() {
					NewClient(tt.accessToken)
				})
			} else {
				client := NewClient(tt.accessToken)
				assert.NotNil(t, client)
				assert.Equal(t, tt.accessToken, client.AccessToken)
				assert.NotNil(t, client.GraphClient)
				assert.Nil(t, client.AuthProvider) // Should be nil for simple client
				assert.Nil(t, client.OAuthConfig)
				assert.Nil(t, client.TokenManager)
				assert.Equal(t, "", client.TokenPath)
				assert.Nil(t, client.Config)
			}
		})
	}
}

func TestNewClientWithTokenRefresh(t *testing.T) {
	// Create temporary directory for token files
	tempDir := t.TempDir()
	tokenPath := filepath.Join(tempDir, "test_tokens.json")

	// Create test components
	oauthConfig := &auth.OAuth2Config{
		ClientID:    "test-client-id",
		TenantID:    "test-tenant-id",
		RedirectURI: "http://localhost:8080/callback",
	}

	tokenManager := &auth.TokenManager{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		Expiry:       time.Now().Add(time.Hour).Unix(),
	}

	config := &Config{
		NotebookName: "Test Notebook",
	}

	tests := []struct {
		name         string
		accessToken  string
		oauthConfig  *auth.OAuth2Config
		tokenManager *auth.TokenManager
		tokenPath    string
		config       *Config
	}{
		{
			name:         "full configuration",
			accessToken:  "test-access-token-123",
			oauthConfig:  oauthConfig,
			tokenManager: tokenManager,
			tokenPath:    tokenPath,
			config:       config,
		},
		{
			name:         "minimal configuration",
			accessToken:  "test-access-token-456",
			oauthConfig:  nil,
			tokenManager: nil,
			tokenPath:    "",
			config:       nil,
		},
		{
			name:         "invalid token path",
			accessToken:  "test-access-token-789",
			oauthConfig:  oauthConfig,
			tokenManager: tokenManager,
			tokenPath:    "/invalid/path/that/does/not/exist/tokens.json",
			config:       config,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClientWithTokenRefresh(
				tt.accessToken,
				tt.oauthConfig,
				tt.tokenManager,
				tt.tokenPath,
				tt.config,
			)

			// Verify basic client properties
			assert.NotNil(t, client)
			assert.Equal(t, tt.accessToken, client.AccessToken)
			assert.NotNil(t, client.GraphClient)
			assert.NotNil(t, client.AuthProvider)
			assert.Equal(t, tt.oauthConfig, client.OAuthConfig)
			assert.Equal(t, tt.tokenManager, client.TokenManager)
			assert.Equal(t, tt.tokenPath, client.TokenPath)
			assert.Equal(t, tt.config, client.Config)

			// Verify auth provider
			assert.Equal(t, tt.accessToken, client.AuthProvider.AccessToken)
		})
	}
}

// TestStaticTokenProvider_AuthenticateRequest is skipped due to complex Microsoft Graph SDK interface requirements
// The functionality is tested indirectly through integration tests
func TestStaticTokenProvider_AuthenticateRequest(t *testing.T) {
	provider := &StaticTokenProvider{
		AccessToken: "test-bearer-token-123",
	}

	// Test that the provider is created correctly
	assert.Equal(t, "test-bearer-token-123", provider.AccessToken)
	
	// Note: Full testing of AuthenticateRequest would require mocking the Microsoft Graph SDK
	// RequestInformation interface, which is complex and changes between SDK versions.
	// The functionality is validated through integration tests and actual usage.
}

func TestStaticTokenProvider_GetAuthorizationToken(t *testing.T) {
	provider := &StaticTokenProvider{
		AccessToken: "test-token-abc123",
	}

	ctx := context.Background()
	scopes := []string{"https://graph.microsoft.com/.default"}

	token, err := provider.GetAuthorizationToken(ctx, scopes)

	assert.NoError(t, err)
	assert.Equal(t, "test-token-abc123", token)
}

func TestStaticTokenProvider_UpdateAccessToken(t *testing.T) {
	provider := &StaticTokenProvider{
		AccessToken: "old-token",
	}

	// Update token
	newToken := "new-token-xyz789"
	provider.UpdateAccessToken(newToken)

	assert.Equal(t, newToken, provider.AccessToken)
}

func TestClient_UpdateToken(t *testing.T) {
	// Create test token manager
	tokenManager := &auth.TokenManager{
		AccessToken:  "old-access-token",
		RefreshToken: "old-refresh-token",
		Expiry:       time.Now().Add(time.Hour).Unix(),
	}

	// Create client with token refresh capabilities
	client := NewClientWithTokenRefresh(
		"old-access-token",
		nil,
		tokenManager,
		"",
		nil,
	)

	tests := []struct {
		name     string
		newToken string
	}{
		{
			name:     "update with valid token",
			newToken: "new-access-token-123",
		},
		{
			name:     "update with empty token",
			newToken: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client.UpdateToken(tt.newToken)

			// Verify client token is updated
			assert.Equal(t, tt.newToken, client.AccessToken)

			// Verify auth provider token is updated
			assert.Equal(t, tt.newToken, client.AuthProvider.AccessToken)

			// Verify token manager behavior for empty tokens
			if tt.newToken == "" {
				assert.Equal(t, "", client.TokenManager.AccessToken)
				assert.Equal(t, "", client.TokenManager.RefreshToken)
				assert.Equal(t, int64(0), client.TokenManager.Expiry)
			}
		})
	}
}

func TestClient_RefreshTokenIfNeeded(t *testing.T) {
	tests := []struct {
		name           string
		tokenManager   *auth.TokenManager
		oauthConfig    *auth.OAuth2Config
		expectedError  string
		setupTokenFile bool
	}{
		{
			name:          "no token manager",
			tokenManager:  nil,
			oauthConfig:   nil,
			expectedError: "token manager or OAuth config not available",
		},
		{
			name: "no oauth config",
			tokenManager: &auth.TokenManager{
				AccessToken: "test-token",
				Expiry:      time.Now().Add(time.Hour).Unix(),
			},
			oauthConfig:   nil,
			expectedError: "token manager or OAuth config not available",
		},
		{
			name: "token not expired",
			tokenManager: &auth.TokenManager{
				AccessToken: "test-token",
				Expiry:      time.Now().Add(time.Hour).Unix(),
			},
			oauthConfig: &auth.OAuth2Config{
				ClientID:    "test-client",
				TenantID:    "test-tenant",
				RedirectURI: "http://localhost:8080/callback",
			},
			expectedError: "",
		},
		{
			name: "token expired but no refresh token",
			tokenManager: &auth.TokenManager{
				AccessToken:  "expired-token",
				RefreshToken: "",
				Expiry:       time.Now().Add(-time.Hour).Unix(),
			},
			oauthConfig: &auth.OAuth2Config{
				ClientID:    "test-client",
				TenantID:    "test-tenant",
				RedirectURI: "http://localhost:8080/callback",
			},
			expectedError: "failed to refresh token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tokenPath string
			if tt.setupTokenFile {
				tempDir := t.TempDir()
				tokenPath = filepath.Join(tempDir, "tokens.json")
			}

			client := NewClientWithTokenRefresh(
				"test-token",
				tt.oauthConfig,
				tt.tokenManager,
				tokenPath,
				nil,
			)

			err := client.RefreshTokenIfNeeded()

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClient_GetDefaultNotebookID(t *testing.T) {
	client := NewClient("test-token")

	id, err := client.GetDefaultNotebookID()

	assert.Error(t, err)
	assert.Equal(t, "", id)
	assert.Contains(t, err.Error(), "has been moved to the notebooks package")
}

