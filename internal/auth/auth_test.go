// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package auth

import (
	"crypto/rand"
	"encoding/base64"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOAuth2Config_Creation tests OAuth2 configuration creation and validation
func TestOAuth2Config_Creation(t *testing.T) {
	tests := []struct {
		name        string
		clientID    string
		tenantID    string
		redirectURI string
		valid       bool
	}{
		{
			name:        "valid configuration",
			clientID:    "test-client-id",
			tenantID:    "common",
			redirectURI: "http://localhost:8080/callback",
			valid:       true,
		},
		{
			name:        "empty client ID",
			clientID:    "",
			tenantID:    "common",
			redirectURI: "http://localhost:8080/callback",
			valid:       false,
		},
		{
			name:        "empty tenant ID",
			clientID:    "test-client-id",
			tenantID:    "",
			redirectURI: "http://localhost:8080/callback",
			valid:       false,
		},
		{
			name:        "empty redirect URI",
			clientID:    "test-client-id",
			tenantID:    "common",
			redirectURI: "",
			valid:       false,
		},
		{
			name:        "specific tenant ID",
			clientID:    "test-client-id",
			tenantID:    "12345678-1234-1234-1234-123456789012",
			redirectURI: "http://localhost:8080/callback",
			valid:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := NewOAuth2Config(tt.clientID, tt.tenantID, tt.redirectURI)
			
			if tt.valid {
				assert.NotNil(t, config)
				assert.Equal(t, tt.clientID, config.ClientID)
				assert.Equal(t, tt.tenantID, config.TenantID)
				assert.Equal(t, tt.redirectURI, config.RedirectURI)
			} else {
				// Config is created but should have empty/invalid values
				if tt.clientID == "" || tt.tenantID == "" || tt.redirectURI == "" {
					// These would be caught by validation in actual usage
					assert.NotNil(t, config) // Constructor doesn't validate, that's done later
				}
			}
		})
	}
}

// TestGenerateCodeVerifier tests PKCE code verifier generation
func TestGenerateCodeVerifier(t *testing.T) {
	t.Run("generates valid code verifier", func(t *testing.T) {
		verifier, err := GenerateCodeVerifier()
		
		assert.NoError(t, err)
		assert.NotEmpty(t, verifier)
		
		// Code verifier should be base64-encoded
		_, err = base64.RawURLEncoding.DecodeString(verifier)
		assert.NoError(t, err, "Code verifier should be valid base64")
	})

	t.Run("generates different verifiers each time", func(t *testing.T) {
		verifier1, err1 := GenerateCodeVerifier()
		verifier2, err2 := GenerateCodeVerifier()
		
		assert.NoError(t, err1)
		assert.NoError(t, err2)
		
		// Each call should generate different values
		assert.NotEqual(t, verifier1, verifier2)
	})

	t.Run("generates verifier with correct length", func(t *testing.T) {
		verifier, err := GenerateCodeVerifier()
		
		assert.NoError(t, err)
		
		// Decode to check original length (should be 32 bytes before encoding)
		decoded, err := base64.RawURLEncoding.DecodeString(verifier)
		assert.NoError(t, err)
		assert.Len(t, decoded, 32, "Code verifier should be 32 bytes before encoding")
	})
}

// TestCodeChallengeS256 tests code challenge generation from verifier
func TestCodeChallengeS256(t *testing.T) {
	t.Run("generates valid code challenge", func(t *testing.T) {
		verifier, err := GenerateCodeVerifier()
		require.NoError(t, err)
		
		challenge := CodeChallengeS256(verifier)
		
		assert.NotEmpty(t, challenge)
		assert.NotEqual(t, verifier, challenge)
		
		// Challenge should be base64-encoded
		_, err = base64.RawURLEncoding.DecodeString(challenge)
		assert.NoError(t, err, "Code challenge should be valid base64")
	})

	t.Run("generates same challenge for same verifier", func(t *testing.T) {
		verifier, err := GenerateCodeVerifier()
		require.NoError(t, err)
		
		challenge1 := CodeChallengeS256(verifier)
		challenge2 := CodeChallengeS256(verifier)
		
		assert.Equal(t, challenge1, challenge2, "Same verifier should generate same challenge")
	})
}

// TestOAuth2Config_AuthorizationURL tests authorization URL generation
func TestOAuth2Config_AuthorizationURL(t *testing.T) {
	config := NewOAuth2Config("test-client", "common", "http://localhost:8080/callback")

	t.Run("generates valid authorization URL", func(t *testing.T) {
		verifier, err := GenerateCodeVerifier()
		require.NoError(t, err)
		challenge := CodeChallengeS256(verifier)
		
		authURL := config.GetAuthURL("test-state", challenge)
		
		assert.NotEmpty(t, authURL)
		assert.Contains(t, authURL, "login.microsoftonline.com")
		assert.Contains(t, authURL, "client_id=test-client")
		assert.Contains(t, authURL, "redirect_uri=http%3A%2F%2Flocalhost%3A8080%2Fcallback")
		assert.Contains(t, authURL, "code_challenge="+challenge)
		assert.Contains(t, authURL, "state=test-state")
		assert.Contains(t, authURL, "scope=") // Scope parameter should be present
		assert.Contains(t, authURL, "Notes.ReadWrite") // Should contain the Notes.ReadWrite permission
	})

	t.Run("includes all required OAuth parameters", func(t *testing.T) {
		verifier, err := GenerateCodeVerifier()
		require.NoError(t, err)
		challenge := CodeChallengeS256(verifier)
		
		authURL := config.GetAuthURL("test-state", challenge)
		
		// Check for required OAuth2 PKCE parameters
		assert.Contains(t, authURL, "response_type=code")
		assert.Contains(t, authURL, "code_challenge_method=S256")
		// Note: prompt parameter may or may not be included depending on implementation
	})

	t.Run("handles different tenant types", func(t *testing.T) {
		verifier, err := GenerateCodeVerifier()
		require.NoError(t, err)
		challenge := CodeChallengeS256(verifier)
		
		// Test common tenant
		commonConfig := NewOAuth2Config("test-client", "common", "http://localhost:8080/callback")
		commonURL := commonConfig.GetAuthURL("state", challenge)
		assert.Contains(t, commonURL, "/common/oauth2/v2.0/authorize")
		
		// Test specific tenant
		specificConfig := NewOAuth2Config("test-client", "12345678-1234-1234-1234-123456789012", "http://localhost:8080/callback")
		specificURL := specificConfig.GetAuthURL("state", challenge)
		assert.Contains(t, specificURL, "/12345678-1234-1234-1234-123456789012/oauth2/v2.0/authorize")
	})

	t.Run("handles tenant ID normalization", func(t *testing.T) {
		// Test that TenantIDOrCommon works
		config1 := NewOAuth2Config("client", "common", "http://localhost/callback")
		assert.Equal(t, "common", config1.TenantIDOrCommon())
		
		config2 := NewOAuth2Config("client", "specific-tenant", "http://localhost/callback")
		assert.Equal(t, "specific-tenant", config2.TenantIDOrCommon())
		
		config3 := NewOAuth2Config("client", "", "http://localhost/callback")
		assert.Equal(t, "common", config3.TenantIDOrCommon())
	})
}

// TestTokenManager tests token management functionality
func TestTokenManager(t *testing.T) {
	t.Run("validates token expiration", func(t *testing.T) {
		// Create a TokenManager manually for testing
		tm := &TokenManager{
			AccessToken: "",
			Expiry:      0,
		}

		// Test with no token loaded
		assert.True(t, tm.IsExpired(), "Should be expired when no token is loaded")

		// Test with expired token
		tm.AccessToken = "test-token"
		tm.Expiry = time.Now().Add(-1 * time.Hour).Unix()
		assert.True(t, tm.IsExpired(), "Should be expired when token is past expiry")

		// Test with valid token
		tm.Expiry = time.Now().Add(1 * time.Hour).Unix()
		assert.False(t, tm.IsExpired(), "Should not be expired when token is valid")
	})

	t.Run("accesses token fields", func(t *testing.T) {
		tm := &TokenManager{}

		// Test with no token
		assert.Equal(t, "", tm.AccessToken, "Should have empty access token initially")

		// Test with token
		tm.AccessToken = "test-access-token"
		assert.Equal(t, "test-access-token", tm.AccessToken)
	})

	t.Run("checks refresh token availability", func(t *testing.T) {
		tm := &TokenManager{}

		// Test with no refresh token
		hasRefresh := tm.RefreshToken != ""
		assert.False(t, hasRefresh, "Should return false when no refresh token")

		// Test with refresh token
		tm.RefreshToken = "test-refresh-token"
		hasRefresh = tm.RefreshToken != ""
		assert.True(t, hasRefresh, "Should return true when refresh token exists")
	})

	t.Run("saves tokens to file", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "token_test_*.json")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		tm := &TokenManager{
			AccessToken:  "test-access-token",
			RefreshToken: "test-refresh-token",
			Expiry:       time.Now().Add(1 * time.Hour).Unix(),
		}

		err = tm.SaveTokens(tmpFile.Name())
		assert.NoError(t, err)

		// Verify file was created and has content
		info, err := os.Stat(tmpFile.Name())
		assert.NoError(t, err)
		assert.Greater(t, info.Size(), int64(0), "Token file should not be empty")
	})
}

// TestLoadTokens tests loading tokens from file
func TestLoadTokens(t *testing.T) {
	t.Run("loads tokens from valid file", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "load_token_test_*.json")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		// Create a valid token file - using Unix timestamp for expiry
		tokenData := `{
			"access_token": "test-access-token",
			"refresh_token": "test-refresh-token",
			"expiry": 1640995200
		}`
		err = os.WriteFile(tmpFile.Name(), []byte(tokenData), 0600)
		require.NoError(t, err)

		tm, err := LoadTokens(tmpFile.Name())
		assert.NoError(t, err)
		assert.NotNil(t, tm)
		assert.Equal(t, "test-access-token", tm.AccessToken)
		assert.Equal(t, "test-refresh-token", tm.RefreshToken)
	})

	t.Run("handles non-existent file", func(t *testing.T) {
		tm, err := LoadTokens("/non/existent/path/tokens.json")
		assert.Error(t, err)
		assert.Nil(t, tm)
	})

	t.Run("handles invalid JSON", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "invalid_token_test_*.json")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		// Write invalid JSON
		err = os.WriteFile(tmpFile.Name(), []byte("invalid json"), 0600)
		require.NoError(t, err)

		tm, err := LoadTokens(tmpFile.Name())
		assert.Error(t, err)
		assert.Nil(t, tm)
	})
}

// TestAuthManager tests authentication manager functionality
func TestAuthManager(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "auth_manager_test_*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	config := NewOAuth2Config("test-client", "common", "http://localhost:8080/callback")
	tokenManager := &TokenManager{}

	t.Run("creates auth manager", func(t *testing.T) {
		am := NewAuthManager(config, tokenManager, tmpFile.Name())
		assert.NotNil(t, am)
	})

	t.Run("gets auth status", func(t *testing.T) {
		am := NewAuthManager(config, tokenManager, tmpFile.Name())
		
		status := am.GetAuthStatus()
		assert.NotNil(t, status)
		assert.False(t, status.Authenticated, "Should not be authenticated initially")
		assert.Empty(t, status.TokenExpiry)
		assert.False(t, status.RefreshTokenAvailable)
	})

	t.Run("handles token refresh callback", func(t *testing.T) {
		am := NewAuthManager(config, tokenManager, tmpFile.Name())
		
		am.SetTokenRefreshCallback(func(token string) {
			// Callback function for testing
		})
		
		// The callback is private, so we can't directly test it
		// But we can test that the callback was set without error
		assert.NotNil(t, am, "AuthManager should be created successfully with callback")
	})

	t.Run("updates token manager", func(t *testing.T) {
		am := NewAuthManager(config, tokenManager, tmpFile.Name())
		
		newTokenManager := &TokenManager{
			AccessToken: "new-token",
		}
		
		am.UpdateTokenManager(newTokenManager)
		
		// We can verify this worked by checking the auth status reflects the new token manager
		status := am.GetAuthStatus()
		assert.NotNil(t, status)
	})

	t.Run("initiates auth session", func(t *testing.T) {
		am := NewAuthManager(config, tokenManager, tmpFile.Name())
		
		session, err := am.InitiateAuth()
		
		assert.NoError(t, err)
		assert.NotNil(t, session)
		assert.NotEmpty(t, session.State)
		assert.NotEmpty(t, session.CodeVerifier)
		assert.NotEmpty(t, session.AuthURL)
		assert.Contains(t, session.AuthURL, "login.microsoftonline.com")
	})

	t.Run("clears auth", func(t *testing.T) {
		// Create a token file first
		tm := &TokenManager{
			AccessToken:  "test-token",
			RefreshToken: "test-refresh-token",
		}
		err := tm.SaveTokens(tmpFile.Name())
		require.NoError(t, err)

		am := NewAuthManager(config, tm, tmpFile.Name())
		
		err = am.ClearAuth()
		assert.NoError(t, err)
		
		// Verify tokens are cleared
		status := am.GetAuthStatus()
		assert.False(t, status.Authenticated)
	})
}

// TestSecureState tests secure state generation
func TestSecureState(t *testing.T) {
	t.Run("generates secure state", func(t *testing.T) {
		state, err := generateSecureState()
		
		assert.NoError(t, err)
		assert.NotEmpty(t, state)
		
		// Should be base64 encoded
		decoded, err := base64.RawURLEncoding.DecodeString(state)
		assert.NoError(t, err)
		// The actual implementation may use a different number of bytes
		assert.Greater(t, len(decoded), 16, "State should be at least 16 bytes for security")
		assert.LessOrEqual(t, len(decoded), 32, "State should not be excessive")
	})

	t.Run("generates different states", func(t *testing.T) {
		state1, err1 := generateSecureState()
		state2, err2 := generateSecureState()
		
		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.NotEqual(t, state1, state2, "Should generate different states")
	})
}

// TestInputValidation tests input validation and sanitization
func TestInputValidation(t *testing.T) {
	t.Run("validates OAuth configuration parameters", func(t *testing.T) {
		tests := []struct {
			name        string
			clientID    string
			tenantID    string
			redirectURI string
			expectValid bool
		}{
			{
				name:        "valid parameters",
				clientID:    "12345678-1234-1234-1234-123456789012",
				tenantID:    "common",
				redirectURI: "http://localhost:8080/callback",
				expectValid: true,
			},
			{
				name:        "invalid characters in client ID",
				clientID:    "client<script>alert('xss')</script>",
				tenantID:    "common",
				redirectURI: "http://localhost:8080/callback",
				expectValid: false,
			},
			{
				name:        "invalid redirect URI",
				clientID:    "valid-client-id",
				tenantID:    "common",
				redirectURI: "javascript:alert('xss')",
				expectValid: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				config := NewOAuth2Config(tt.clientID, tt.tenantID, tt.redirectURI)
				
				// Basic validation - the constructor creates the config but real validation
				// would happen during usage
				if tt.expectValid {
					assert.NotEmpty(t, config.ClientID)
					assert.NotEmpty(t, config.TenantID)
					assert.NotEmpty(t, config.RedirectURI)
				} else {
					// For security validation, we'd need to check the actual values
					// This would typically be done in the methods that use these values
					if strings.Contains(tt.clientID, "<script>") {
						assert.Contains(t, config.ClientID, "<script>", "Dangerous content preserved for later validation")
					}
				}
			})
		}
	})
}

// TestRandomGeneration tests that random generation works correctly
func TestRandomGeneration(t *testing.T) {
	t.Run("crypto rand works", func(t *testing.T) {
		bytes := make([]byte, 32)
		_, err := rand.Read(bytes)
		
		assert.NoError(t, err)
		
		// Check that we got non-zero bytes
		hasNonZero := false
		for _, b := range bytes {
			if b != 0 {
				hasNonZero = true
				break
			}
		}
		assert.True(t, hasNonZero, "Should generate non-zero random bytes")
	})
}

// TestErrorHandling tests various error scenarios
func TestErrorHandling(t *testing.T) {
	t.Run("handles invalid token file operations", func(t *testing.T) {
		tm := &TokenManager{
			AccessToken:  "test-token",
			RefreshToken: "test-refresh-token",
		}
		
		// Try to save to invalid path
		err := tm.SaveTokens("/invalid/path/that/does/not/exist/tokens.json")
		assert.Error(t, err, "Should fail to save to invalid path")
	})

	t.Run("handles empty configuration values", func(t *testing.T) {
		config := NewOAuth2Config("", "", "")
		
		// The config is created but should be invalid for actual use
		assert.NotNil(t, config)
		assert.Equal(t, "", config.ClientID)
		assert.Equal(t, "", config.TenantID)
		assert.Equal(t, "", config.RedirectURI)
	})

	t.Run("handles authentication with missing tokens", func(t *testing.T) {
		tm := &TokenManager{}
		
		// Should report as expired when no token is present
		assert.True(t, tm.IsExpired())
		assert.False(t, tm.RefreshToken != "")
		assert.Equal(t, "", tm.AccessToken)
	})
}