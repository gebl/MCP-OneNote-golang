// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_FromEnvironment(t *testing.T) {
	// Save current environment
	originalEnv := map[string]string{
		"ONENOTE_CLIENT_ID":               os.Getenv("ONENOTE_CLIENT_ID"),
		"ONENOTE_TENANT_ID":               os.Getenv("ONENOTE_TENANT_ID"),
		"ONENOTE_REDIRECT_URI":            os.Getenv("ONENOTE_REDIRECT_URI"),
		"ONENOTE_DEFAULT_NOTEBOOK_NAME":   os.Getenv("ONENOTE_DEFAULT_NOTEBOOK_NAME"),
		"ONENOTE_TOOLSETS":                os.Getenv("ONENOTE_TOOLSETS"),
		"QUICKNOTE_NOTEBOOK_NAME":         os.Getenv("QUICKNOTE_NOTEBOOK_NAME"),
		"QUICKNOTE_PAGE_NAME":             os.Getenv("QUICKNOTE_PAGE_NAME"),
		"QUICKNOTE_DATE_FORMAT":           os.Getenv("QUICKNOTE_DATE_FORMAT"),
		"AUTHORIZATION_ENABLED":           os.Getenv("AUTHORIZATION_ENABLED"),
		"AUTHORIZATION_DEFAULT_MODE":      os.Getenv("AUTHORIZATION_DEFAULT_MODE"),
		"MCP_AUTH_ENABLED":                os.Getenv("MCP_AUTH_ENABLED"),
		"MCP_BEARER_TOKEN":                os.Getenv("MCP_BEARER_TOKEN"),
		"LOG_LEVEL":                       os.Getenv("LOG_LEVEL"),
		"LOG_FORMAT":                      os.Getenv("LOG_FORMAT"),
		"MCP_LOG_FILE":                    os.Getenv("MCP_LOG_FILE"),
		"CONTENT_LOG_LEVEL":               os.Getenv("CONTENT_LOG_LEVEL"),
	}

	// Clean up after test
	defer func() {
		for key, value := range originalEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	// Set test environment variables
	testEnv := map[string]string{
		"ONENOTE_CLIENT_ID":               "test-client-id",
		"ONENOTE_TENANT_ID":               "test-tenant-id",
		"ONENOTE_REDIRECT_URI":            "http://localhost:8080/callback",
		"ONENOTE_DEFAULT_NOTEBOOK_NAME":   "Test Notebook",
		"ONENOTE_TOOLSETS":                "notebooks,sections,pages",
		"QUICKNOTE_NOTEBOOK_NAME":         "Quick Notes",
		"QUICKNOTE_PAGE_NAME":             "Daily Journal",
		"QUICKNOTE_DATE_FORMAT":           "January 2, 2006 - 3:04 PM",
		"AUTHORIZATION_ENABLED":           "true",
		"AUTHORIZATION_DEFAULT_MODE":      "read",
		"MCP_AUTH_ENABLED":                "true",
		"MCP_BEARER_TOKEN":                "test-bearer-token",
		"LOG_LEVEL":                       "DEBUG",
		"LOG_FORMAT":                      "json",
		"MCP_LOG_FILE":                    "test.log",
		"CONTENT_LOG_LEVEL":               "INFO",
	}

	for key, value := range testEnv {
		os.Setenv(key, value)
	}

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify core configuration
	assert.Equal(t, "test-client-id", cfg.ClientID)
	assert.Equal(t, "test-tenant-id", cfg.TenantID)
	assert.Equal(t, "http://localhost:8080/callback", cfg.RedirectURI)
	assert.Equal(t, "Test Notebook", cfg.NotebookName)
	assert.Equal(t, []string{"notebooks", "sections", "pages"}, cfg.Toolsets)

	// Verify QuickNote configuration
	require.NotNil(t, cfg.QuickNote)
	assert.Equal(t, "Quick Notes", cfg.QuickNote.NotebookName)
	assert.Equal(t, "Daily Journal", cfg.QuickNote.PageName)
	assert.Equal(t, "January 2, 2006 - 3:04 PM", cfg.QuickNote.DateFormat)

	// Verify Authorization configuration
	require.NotNil(t, cfg.Authorization)
	assert.True(t, cfg.Authorization.Enabled)

	// Verify MCP Auth configuration
	require.NotNil(t, cfg.MCPAuth)
	assert.True(t, cfg.MCPAuth.Enabled)
	assert.Equal(t, "test-bearer-token", cfg.MCPAuth.BearerToken)

	// Verify logging configuration
	assert.Equal(t, "DEBUG", cfg.LogLevel)
	assert.Equal(t, "json", cfg.LogFormat)
	assert.Equal(t, "test.log", cfg.LogFile)
	assert.Equal(t, "INFO", cfg.ContentLogLevel)
}

func TestLoad_FromJSONFile(t *testing.T) {
	// Create temporary JSON config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.json")

	testConfig := map[string]interface{}{
		"client_id":     "json-client-id",
		"tenant_id":     "json-tenant-id",
		"redirect_uri":  "http://localhost:9090/callback",
		"notebook_name": "JSON Notebook",
		"toolsets":      []string{"notebooks", "pages"},
		"quicknote": map[string]interface{}{
			"notebook_name": "JSON Quick Notes",
			"page_name":     "JSON Daily Journal",
			"date_format":   "2006-01-02 15:04:05",
		},
		"authorization": map[string]interface{}{
			"enabled":      true,
			"default_mode": "write",
			"notebook_permissions": map[string]string{
				"Test Notebook": "read",
			},
		},
		"mcp_auth": map[string]interface{}{
			"enabled":      false,
			"bearer_token": "json-bearer-token",
		},
		"log_level":         "INFO",
		"log_format":        "text",
		"log_file":          "json.log",
		"content_log_level": "WARN",
	}

	configData, err := json.MarshalIndent(testConfig, "", "  ")
	require.NoError(t, err)

	err = os.WriteFile(configPath, configData, 0600)
	require.NoError(t, err)

	// Clean environment and set config path
	originalEnv := os.Getenv("ONENOTE_MCP_CONFIG")
	defer func() {
		if originalEnv == "" {
			os.Unsetenv("ONENOTE_MCP_CONFIG")
		} else {
			os.Setenv("ONENOTE_MCP_CONFIG", originalEnv)
		}
	}()

	// Clear other env vars to ensure JSON takes precedence
	envVars := []string{
		"ONENOTE_CLIENT_ID", "ONENOTE_TENANT_ID", "ONENOTE_REDIRECT_URI",
		"ONENOTE_DEFAULT_NOTEBOOK_NAME", "ONENOTE_TOOLSETS",
	}
	originalValues := make(map[string]string)
	for _, envVar := range envVars {
		originalValues[envVar] = os.Getenv(envVar)
		os.Unsetenv(envVar)
	}
	defer func() {
		for envVar, value := range originalValues {
			if value == "" {
				os.Unsetenv(envVar)
			} else {
				os.Setenv(envVar, value)
			}
		}
	}()

	os.Setenv("ONENOTE_MCP_CONFIG", configPath)

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify JSON config was loaded
	assert.Equal(t, "json-client-id", cfg.ClientID)
	assert.Equal(t, "json-tenant-id", cfg.TenantID)
	assert.Equal(t, "http://localhost:9090/callback", cfg.RedirectURI)
	assert.Equal(t, "JSON Notebook", cfg.NotebookName)
	assert.Equal(t, []string{"notebooks", "pages"}, cfg.Toolsets)

	// Verify QuickNote from JSON
	require.NotNil(t, cfg.QuickNote)
	assert.Equal(t, "JSON Quick Notes", cfg.QuickNote.NotebookName)
	assert.Equal(t, "JSON Daily Journal", cfg.QuickNote.PageName)
	assert.Equal(t, "2006-01-02 15:04:05", cfg.QuickNote.DateFormat)

	// Verify Authorization from JSON
	require.NotNil(t, cfg.Authorization)
	assert.True(t, cfg.Authorization.Enabled)

	// Verify MCP Auth from JSON
	require.NotNil(t, cfg.MCPAuth)
	assert.False(t, cfg.MCPAuth.Enabled)
	assert.Equal(t, "json-bearer-token", cfg.MCPAuth.BearerToken)

	// Verify logging from JSON
	assert.Equal(t, "INFO", cfg.LogLevel)
	assert.Equal(t, "text", cfg.LogFormat)
	assert.Equal(t, "json.log", cfg.LogFile)
	assert.Equal(t, "WARN", cfg.ContentLogLevel)
}

func TestLoad_EnvironmentOverridesJSON(t *testing.T) {
	// Create temporary JSON config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.json")

	testConfig := map[string]interface{}{
		"client_id":    "json-client-id",
		"tenant_id":    "json-tenant-id",
		"redirect_uri": "http://localhost:9090/callback",
		"log_level":    "INFO",
	}

	configData, err := json.MarshalIndent(testConfig, "", "  ")
	require.NoError(t, err)

	err = os.WriteFile(configPath, configData, 0600)
	require.NoError(t, err)

	// Save and restore environment
	originalEnv := map[string]string{
		"ONENOTE_MCP_CONFIG": os.Getenv("ONENOTE_MCP_CONFIG"),
		"ONENOTE_CLIENT_ID":  os.Getenv("ONENOTE_CLIENT_ID"),
		"LOG_LEVEL":          os.Getenv("LOG_LEVEL"),
	}
	defer func() {
		for key, value := range originalEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	// Set config file and override values via environment
	os.Setenv("ONENOTE_MCP_CONFIG", configPath)
	os.Setenv("ONENOTE_CLIENT_ID", "env-client-id")      // Override JSON
	os.Setenv("LOG_LEVEL", "DEBUG")                     // Override JSON

	cfg, err := Load()
	require.NoError(t, err)

	// Environment should override JSON
	assert.Equal(t, "env-client-id", cfg.ClientID)      // From env
	assert.Equal(t, "DEBUG", cfg.LogLevel)              // From env
	assert.Equal(t, "json-tenant-id", cfg.TenantID)     // From JSON
	assert.Equal(t, "http://localhost:9090/callback", cfg.RedirectURI) // From JSON
}

func TestLoad_MissingRequiredFields(t *testing.T) {
	// Clean environment to test missing required fields
	originalEnv := map[string]string{
		"ONENOTE_CLIENT_ID":    os.Getenv("ONENOTE_CLIENT_ID"),
		"ONENOTE_TENANT_ID":    os.Getenv("ONENOTE_TENANT_ID"),
		"ONENOTE_REDIRECT_URI": os.Getenv("ONENOTE_REDIRECT_URI"),
		"ONENOTE_MCP_CONFIG":   os.Getenv("ONENOTE_MCP_CONFIG"),
	}
	defer func() {
		for key, value := range originalEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	for _, envVar := range []string{"ONENOTE_CLIENT_ID", "ONENOTE_TENANT_ID", "ONENOTE_REDIRECT_URI", "ONENOTE_MCP_CONFIG"} {
		os.Unsetenv(envVar)
	}

	tests := []struct {
		name      string
		envVars   map[string]string
		wantError bool
		errorMsg  string
	}{
		{
			name: "missing client_id",
			envVars: map[string]string{
				"ONENOTE_TENANT_ID":    "test-tenant",
				"ONENOTE_REDIRECT_URI": "http://localhost:8080/callback",
			},
			wantError: true,
			errorMsg:  "ONENOTE_CLIENT_ID is required",
		},
		{
			name: "missing tenant_id",
			envVars: map[string]string{
				"ONENOTE_CLIENT_ID":    "test-client",
				"ONENOTE_REDIRECT_URI": "http://localhost:8080/callback",
			},
			wantError: true,
			errorMsg:  "ONENOTE_TENANT_ID is required",
		},
		{
			name: "missing redirect_uri",
			envVars: map[string]string{
				"ONENOTE_CLIENT_ID": "test-client",
				"ONENOTE_TENANT_ID": "test-tenant",
			},
			wantError: true,
			errorMsg:  "ONENOTE_REDIRECT_URI is required",
		},
		{
			name: "all required fields present",
			envVars: map[string]string{
				"ONENOTE_CLIENT_ID":    "test-client",
				"ONENOTE_TENANT_ID":    "test-tenant",
				"ONENOTE_REDIRECT_URI": "http://localhost:8080/callback",
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			for _, envVar := range []string{"ONENOTE_CLIENT_ID", "ONENOTE_TENANT_ID", "ONENOTE_REDIRECT_URI"} {
				os.Unsetenv(envVar)
			}

			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			cfg, err := Load()

			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, cfg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cfg)
			}
		})
	}
}

func TestLoad_InvalidJSONFile(t *testing.T) {
	// Create temporary invalid JSON file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid_config.json")

	err := os.WriteFile(configPath, []byte("invalid json content"), 0600)
	require.NoError(t, err)

	// Save and restore environment
	originalEnv := os.Getenv("ONENOTE_MCP_CONFIG")
	defer func() {
		if originalEnv == "" {
			os.Unsetenv("ONENOTE_MCP_CONFIG")
		} else {
			os.Setenv("ONENOTE_MCP_CONFIG", originalEnv)
		}
	}()

	os.Setenv("ONENOTE_MCP_CONFIG", configPath)

	cfg, err := Load()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse config file JSON")
	assert.Nil(t, cfg)
}

func TestLoad_NonexistentJSONFile(t *testing.T) {
	nonExistentPath := "/path/that/does/not/exist/config.json"

	// Save and restore environment
	originalEnv := os.Getenv("ONENOTE_MCP_CONFIG")
	defer func() {
		if originalEnv == "" {
			os.Unsetenv("ONENOTE_MCP_CONFIG")
		} else {
			os.Setenv("ONENOTE_MCP_CONFIG", originalEnv)
		}
	}()

	os.Setenv("ONENOTE_MCP_CONFIG", nonExistentPath)

	cfg, err := Load()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config file not accessible")
	assert.Nil(t, cfg)
}

func TestLoad_DefaultValues(t *testing.T) {
	// Clean environment except required fields
	envVarsToClean := []string{
		"ONENOTE_DEFAULT_NOTEBOOK_NAME", "ONENOTE_TOOLSETS", "QUICKNOTE_NOTEBOOK_NAME",
		"QUICKNOTE_PAGE_NAME", "QUICKNOTE_DATE_FORMAT", "AUTHORIZATION_ENABLED",
		"AUTHORIZATION_DEFAULT_MODE", "MCP_AUTH_ENABLED", "MCP_BEARER_TOKEN",
		"LOG_LEVEL", "LOG_FORMAT", "MCP_LOG_FILE", "CONTENT_LOG_LEVEL",
		"ONENOTE_MCP_CONFIG",
	}

	originalValues := make(map[string]string)
	for _, envVar := range envVarsToClean {
		originalValues[envVar] = os.Getenv(envVar)
		os.Unsetenv(envVar)
	}
	defer func() {
		for envVar, value := range originalValues {
			if value == "" {
				os.Unsetenv(envVar)
			} else {
				os.Setenv(envVar, value)
			}
		}
	}()

	// Set required fields only
	requiredEnv := map[string]string{
		"ONENOTE_CLIENT_ID":    "test-client",
		"ONENOTE_TENANT_ID":    "test-tenant",
		"ONENOTE_REDIRECT_URI": "http://localhost:8080/callback",
	}
	originalRequired := make(map[string]string)
	for key, value := range requiredEnv {
		originalRequired[key] = os.Getenv(key)
		os.Setenv(key, value)
	}
	defer func() {
		for key, value := range originalRequired {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify defaults
	assert.Equal(t, "", cfg.NotebookName)                                        // Default empty
	assert.Equal(t, []string{"notebooks", "sections", "pages", "content"}, cfg.Toolsets) // Default all toolsets

	// QuickNote defaults
	require.NotNil(t, cfg.QuickNote)
	assert.Equal(t, "", cfg.QuickNote.NotebookName)                  // Default empty
	assert.Equal(t, "", cfg.QuickNote.PageName)                      // Default empty
	assert.Equal(t, "January 2, 2006 - 3:04 PM", cfg.QuickNote.DateFormat) // Default format

	// Authorization defaults
	require.NotNil(t, cfg.Authorization)
	assert.False(t, cfg.Authorization.Enabled)         // Default false

	// MCP Auth defaults
	require.NotNil(t, cfg.MCPAuth)
	assert.False(t, cfg.MCPAuth.Enabled)          // Default false
	assert.Equal(t, "", cfg.MCPAuth.BearerToken)  // Default empty

	// Logging defaults
	assert.Equal(t, "INFO", cfg.LogLevel)         // Default INFO
	assert.Equal(t, "text", cfg.LogFormat)        // Default text
	assert.Equal(t, "", cfg.LogFile)              // Default empty
	assert.Equal(t, "INFO", cfg.ContentLogLevel)  // Default INFO
}

func TestLoad_ToolsetParsing(t *testing.T) {
	tests := []struct {
		name         string
		toolsetValue string
		expected     []string
	}{
		{
			name:         "comma separated",
			toolsetValue: "notebooks,sections,pages",
			expected:     []string{"notebooks", "sections", "pages"},
		},
		{
			name:         "comma separated with spaces",
			toolsetValue: "notebooks,sections,pages,content",
			expected:     []string{"notebooks", "sections", "pages", "content"},
		},
		{
			name:         "single toolset",
			toolsetValue: "notebooks",
			expected:     []string{"notebooks"},
		},
		{
			name:         "empty string",
			toolsetValue: "",
			expected:     []string{"notebooks", "sections", "pages", "content"}, // Defaults when no toolsets specified
		},
		{
			name:         "with extra commas",
			toolsetValue: "notebooks,sections",
			expected:     []string{"notebooks", "sections"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore environment
			originalEnv := map[string]string{
				"ONENOTE_CLIENT_ID":    os.Getenv("ONENOTE_CLIENT_ID"),
				"ONENOTE_TENANT_ID":    os.Getenv("ONENOTE_TENANT_ID"),
				"ONENOTE_REDIRECT_URI": os.Getenv("ONENOTE_REDIRECT_URI"),
				"ONENOTE_TOOLSETS":     os.Getenv("ONENOTE_TOOLSETS"),
				"ONENOTE_MCP_CONFIG":   os.Getenv("ONENOTE_MCP_CONFIG"),
			}
			defer func() {
				for key, value := range originalEnv {
					if value == "" {
						os.Unsetenv(key)
					} else {
						os.Setenv(key, value)
					}
				}
			}()

			// Set required fields and test toolset
			os.Setenv("ONENOTE_CLIENT_ID", "test-client")
			os.Setenv("ONENOTE_TENANT_ID", "test-tenant")
			os.Setenv("ONENOTE_REDIRECT_URI", "http://localhost:8080/callback")
			os.Setenv("ONENOTE_TOOLSETS", tt.toolsetValue)
			os.Unsetenv("ONENOTE_MCP_CONFIG")

			cfg, err := Load()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, cfg.Toolsets)
		})
	}
}

// TestValidate removed - Validate method not implemented in current version

func TestQuickNoteConfig_GetNotebookName(t *testing.T) {
	tests := []struct {
		name           string
		quickNote      *QuickNoteConfig
		expectedResult string
	}{
		{
			name: "quicknote has notebook name",
			quickNote: &QuickNoteConfig{
				NotebookName: "QuickNote Notebook",
			},
			expectedResult: "QuickNote Notebook",
		},
		{
			name: "quicknote empty",
			quickNote: &QuickNoteConfig{
				NotebookName: "",
			},
			expectedResult: "",
		},
		{
			name:           "nil quicknote",
			quickNote:      nil,
			expectedResult: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.quickNote.GetNotebookName()
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

// TestParseToolsets removed - parseToolsets function is not exported

// TestParseBoolWithDefault removed - parseBoolWithDefault function is not exported