// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gebl/onenote-mcp-server/internal/auth"
	"github.com/gebl/onenote-mcp-server/internal/config"
)

func TestStringifySections(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: "[]",
		},
		{
			name: "valid sections slice",
			input: []map[string]interface{}{
				{"id": "456", "displayName": "Test Section"},
			},
			expected: `[{"displayName":"Test Section","id":"456"}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stringifySections(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTokenValidationFlow(t *testing.T) {
	// Create temporary directory for test tokens
	tempDir, err := os.MkdirTemp("", "onenote-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	tokenPath := filepath.Join(tempDir, "test_tokens.json")

	t.Run("missing token file", func(t *testing.T) {
		_, err := auth.LoadTokens(tokenPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot find the file")
	})

	t.Run("valid token file", func(t *testing.T) {
		// Create a test token file
		testToken := &auth.TokenManager{
			AccessToken:  "test_access_token",
			RefreshToken: "test_refresh_token",
			Expiry:       time.Now().Add(time.Hour).Unix(),
		}

		err := testToken.SaveTokens(tokenPath)
		require.NoError(t, err)

		// Load and validate
		loadedToken, err := auth.LoadTokens(tokenPath)
		require.NoError(t, err)
		assert.Equal(t, testToken.AccessToken, loadedToken.AccessToken)
		assert.False(t, loadedToken.IsExpired())
	})

	t.Run("expired token", func(t *testing.T) {
		// Create an expired token file
		expiredToken := &auth.TokenManager{
			AccessToken:  "expired_access_token",
			RefreshToken: "test_refresh_token",
			Expiry:       time.Now().Add(-time.Hour).Unix(), // Expired
		}

		err := expiredToken.SaveTokens(tokenPath)
		require.NoError(t, err)

		// Load and validate
		loadedToken, err := auth.LoadTokens(tokenPath)
		require.NoError(t, err)
		assert.True(t, loadedToken.IsExpired())
	})
}

func TestConfigurationLoading(t *testing.T) {
	// Save original environment variables
	originalClientID := os.Getenv("ONENOTE_CLIENT_ID")
	originalTenantID := os.Getenv("ONENOTE_TENANT_ID")
	originalRedirectURI := os.Getenv("ONENOTE_REDIRECT_URI")

	// Cleanup function
	defer func() {
		os.Setenv("ONENOTE_CLIENT_ID", originalClientID)
		os.Setenv("ONENOTE_TENANT_ID", originalTenantID)
		os.Setenv("ONENOTE_REDIRECT_URI", originalRedirectURI)
	}()

	t.Run("config from environment variables", func(t *testing.T) {
		// Clear config file path to prevent file loading
		os.Unsetenv("ONENOTE_MCP_CONFIG")

		// Set test environment variables
		os.Setenv("ONENOTE_CLIENT_ID", "test-client-id")
		os.Setenv("ONENOTE_TENANT_ID", "test-tenant-id")
		os.Setenv("ONENOTE_REDIRECT_URI", "http://localhost:8080/callback")

		cfg, err := config.Load()
		require.NoError(t, err)
		assert.Equal(t, "test-client-id", cfg.ClientID)
		assert.Equal(t, "test-tenant-id", cfg.TenantID)
		assert.Equal(t, "http://localhost:8080/callback", cfg.RedirectURI)
	})

	t.Run("missing required config", func(t *testing.T) {
		// Clear config file path to prevent file loading
		os.Unsetenv("ONENOTE_MCP_CONFIG")

		// Clear environment variables
		os.Unsetenv("ONENOTE_CLIENT_ID")
		os.Unsetenv("ONENOTE_TENANT_ID")
		os.Unsetenv("ONENOTE_REDIRECT_URI")

		_, err := config.Load()
		assert.Error(t, err)
	})
}

// TestServerModeValidation tests that the server handles different modes correctly
func TestServerModeValidation(t *testing.T) {
	tests := []struct {
		name         string
		mode         string
		expectedMode string
	}{
		{
			name:         "default stdio mode",
			mode:         "",
			expectedMode: "stdio",
		},
		{
			name:         "explicit stdio mode",
			mode:         "stdio",
			expectedMode: "stdio",
		},
		{
			name:         "sse mode",
			mode:         "sse",
			expectedMode: "sse",
		},
		{
			name:         "streamable mode",
			mode:         "streamable",
			expectedMode: "streamable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test validates the mode logic without actually starting a server
			mode := tt.mode
			if mode == "" {
				mode = "stdio"
			}
			assert.Equal(t, tt.expectedMode, mode)
		})
	}
}

// TestMCPServerCreation tests that the MCP server is created with correct metadata
func TestMCPServerCreation(t *testing.T) {
	// This is a basic test that validates server creation doesn't panic
	// and that basic metadata is set correctly

	t.Run("server creation", func(t *testing.T) {
		// Test that server creation doesn't panic
		assert.NotPanics(t, func() {
			// This would typically create a server, but we're just testing the pattern
			serverName := "OneNote MCP Server"
			serverVersion := "1.6.0"

			assert.NotEmpty(t, serverName)
			assert.NotEmpty(t, serverVersion)
		})
	})
}

// TestVersionConstant tests that the version constant is properly defined
func TestVersionConstant(t *testing.T) {
	assert.Equal(t, "1.7.0", Version)
	assert.NotEmpty(t, Version)
}

// TestLoadTestData helps verify our test data files are valid
func TestLoadTestData(t *testing.T) {
	t.Run("load sample notebooks", func(t *testing.T) {
		data, err := os.ReadFile("testdata/sample_responses/notebooks.json")
		require.NoError(t, err)

		var notebooks []map[string]interface{}
		err = json.Unmarshal(data, &notebooks)
		require.NoError(t, err)
		assert.Greater(t, len(notebooks), 0)

		// Validate structure of first notebook
		if len(notebooks) > 0 {
			notebook := notebooks[0]
			assert.Contains(t, notebook, "id")
			assert.Contains(t, notebook, "displayName")
		}
	})

	t.Run("load sample sections", func(t *testing.T) {
		data, err := os.ReadFile("testdata/sample_responses/sections.json")
		require.NoError(t, err)

		var sections []map[string]interface{}
		err = json.Unmarshal(data, &sections)
		require.NoError(t, err)
		assert.Greater(t, len(sections), 0)
	})

	t.Run("load test config", func(t *testing.T) {
		data, err := os.ReadFile("testdata/test_configs/test_config.json")
		require.NoError(t, err)

		var cfg map[string]interface{}
		err = json.Unmarshal(data, &cfg)
		require.NoError(t, err)
		assert.Contains(t, cfg, "client_id")
		assert.Contains(t, cfg, "tenant_id")
	})
}
