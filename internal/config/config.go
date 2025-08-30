// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

// config.go - Configuration management for the OneNote MCP server.
//
// This file provides flexible configuration loading from multiple sources including
// environment variables, JSON configuration files, and command-line arguments.
// It centralizes all configuration management for the OneNote MCP server.
//
// Key Features:
// - Multi-source configuration loading (env vars, JSON files, defaults)
// - Environment variable support with automatic mapping
// - JSON configuration file support with validation
// - Configuration validation and error reporting
// - Flexible toolset configuration
// - Comprehensive logging of configuration values
//
// Configuration Sources (in order of precedence):
// 1. Environment variables (highest priority)
// 2. JSON configuration file (if ONENOTE_MCP_CONFIG is set)
// 3. Default values (lowest priority)
//
// Environment Variables:
// - ONENOTE_CLIENT_ID: Azure App Registration Client ID
// - ONENOTE_TENANT_ID: Azure Tenant ID (use "common" for multi-tenant)
// - ONENOTE_REDIRECT_URI: OAuth2 redirect URI for authentication
// - ONENOTE_DEFAULT_NOTEBOOK_NAME: Default notebook name (optional)
// - ONENOTE_TOOLSETS: Comma-separated list of enabled toolsets
// - ONENOTE_MCP_CONFIG: Path to JSON configuration file (optional)
// - LOG_LEVEL: General logging level (DEBUG, INFO, WARN, ERROR)
// - LOG_FORMAT: Log format ("text" or "json")
// - MCP_LOG_FILE: Optional log file path
// - CONTENT_LOG_LEVEL: Content logging verbosity (DEBUG, INFO, WARN, ERROR, OFF)
//
// JSON Configuration File Format:
// {
//   "client_id": "your-azure-app-client-id",
//   "tenant_id": "your-azure-tenant-id",
//   "redirect_uri": "http://localhost:8080/callback",
//   "notebook_name": "My Default Notebook",
//   "toolsets": ["notebooks", "sections", "pages", "content"],
//   "log_level": "DEBUG",
//   "log_format": "text",
//   "log_file": "mcp-server.log",
//   "content_log_level": "DEBUG"
// }
//
// Available Toolsets:
// - "notebooks": Notebook listing and management
// - "sections": Section operations within notebooks
// - "pages": Page CRUD operations
// - "content": Content extraction and manipulation
//
// Configuration Validation:
// - Required fields validation
// - URL format validation for redirect URI
// - Azure app registration validation
// - Toolset availability checking
//
// Error Handling:
// - Missing required configuration values
// - Invalid JSON configuration files
// - File permission issues
// - Environment variable parsing errors
//
// Usage Example:
//   cfg, err := config.Load()
//   if err != nil {
//       log.Fatalf("Failed to load config: %v", err)
//   }
//   fmt.Printf("Client ID: %s\n", cfg.ClientID)
//
// For detailed configuration options, see README.md and docs/setup.md

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gebl/onenote-mcp-server/internal/authorization"
	"github.com/gebl/onenote-mcp-server/internal/logging"
)

// Config holds all configuration values for the OneNote MCP server.
type Config struct {
	ClientID     string   `json:"client_id"`     // Application (client) ID
	TenantID     string   `json:"tenant_id"`     // Directory (tenant) ID
	RedirectURI  string   `json:"redirect_uri"`  // Redirect URI for OAuth2 callback
	Toolsets     []string `json:"toolsets"`      // Enabled toolsets (e.g., notebooks, sections, pages)
	NotebookName string   `json:"notebook_name"` // Default notebook name (optional) - maps to ONENOTE_DEFAULT_NOTEBOOK_NAME

	// Quicknote configuration
	QuickNote *QuickNoteConfig `json:"quicknote"` // Quicknote settings for rapid note-taking

	// MCP Authentication configuration
	MCPAuth *MCPAuthConfig `json:"mcp_auth"` // MCP server authentication settings

	// Authorization configuration
	Authorization *authorization.AuthorizationConfig `json:"authorization"` // Tool and resource authorization settings

	// Server configuration
	Stateless *bool `json:"stateless"` // Enable stateless mode for HTTP server

	// Logging configuration
	LogLevel        string `json:"log_level"`         // General logging level: DEBUG, INFO, WARN, ERROR
	LogFormat       string `json:"log_format"`        // Log format: "text" or "json"
	LogFile         string `json:"log_file"`          // Optional log file path
	ContentLogLevel string `json:"content_log_level"` // Content logging verbosity: DEBUG, INFO, WARN, ERROR, OFF
}

// QuickNoteConfig holds configuration for the quicknote tool.
type QuickNoteConfig struct {
	NotebookName string `json:"notebook_name"` // Target notebook name for quicknotes
	PageName     string `json:"page_name"`     // Target page name for quicknotes
	DateFormat   string `json:"date_format"`   // Date format string for timestamps (Go time format)
}

// MCPAuthConfig holds MCP server authentication configuration.
type MCPAuthConfig struct {
	Enabled     bool   `json:"enabled"`      // Enable MCP authentication for HTTP/SSE modes
	BearerToken string `json:"bearer_token"` // Bearer token for authentication
}

// GetNotebookName returns the quicknote-specific notebook name
func (qc *QuickNoteConfig) GetNotebookName() string {
	if qc == nil {
		return ""
	}
	return qc.NotebookName
}

// GetDefaultNotebook returns the default notebook name from the main config
// This method should be called from the main Config struct
func (c *Config) GetDefaultNotebook() string {
	return c.NotebookName
}

// GetPageName returns the target page name for quicknote
func (qc *QuickNoteConfig) GetPageName() string {
	if qc == nil {
		return ""
	}
	return qc.PageName
}

// Load reads configuration from environment variables and optionally from a JSON config file.
// Returns a pointer to a Config and an error, if any.
//
// IMPORTANT: This function assumes logging is initialized with DEBUG level to capture
// all configuration loading details. After config loading, logging will be reconfigured
// based on the loaded configuration values.
func Load() (*Config, error) {
	startTime := time.Now()
	logger := logging.ConfigLogger

	logger.Debug("Starting configuration loading")

	// Initialize stateless with default value (false)
	statelessDefault := false
	statelessPtr := &statelessDefault
	if statelessEnv := os.Getenv("MCP_STATELESS"); statelessEnv == "true" {
		statelessTrue := true
		statelessPtr = &statelessTrue
	}

	cfg := &Config{
		ClientID:     os.Getenv("ONENOTE_CLIENT_ID"),
		TenantID:     os.Getenv("ONENOTE_TENANT_ID"),
		RedirectURI:  os.Getenv("ONENOTE_REDIRECT_URI"),
		NotebookName: os.Getenv("ONENOTE_DEFAULT_NOTEBOOK_NAME"),

		// Quicknote configuration from environment variables
		QuickNote: &QuickNoteConfig{
			NotebookName: os.Getenv("QUICKNOTE_NOTEBOOK_NAME"),
			PageName:     os.Getenv("QUICKNOTE_PAGE_NAME"),
			DateFormat:   os.Getenv("QUICKNOTE_DATE_FORMAT"),
		},

		// MCP Authentication configuration from environment variables
		MCPAuth: &MCPAuthConfig{
			Enabled:     os.Getenv("MCP_AUTH_ENABLED") == "true",
			BearerToken: os.Getenv("MCP_BEARER_TOKEN"),
		},

		// Authorization configuration - will be initialized after JSON loading
		Authorization: authorization.NewAuthorizationConfig(),

		// Server configuration from environment variables
		Stateless: statelessPtr,

		// Logging configuration from environment variables
		LogLevel:        os.Getenv("LOG_LEVEL"),
		LogFormat:       os.Getenv("LOG_FORMAT"),
		LogFile:         os.Getenv("MCP_LOG_FILE"),
		ContentLogLevel: os.Getenv("CONTENT_LOG_LEVEL"),
	}

	logger.Debug("Loaded from environment variables",
		"client_id", maskSensitiveData(cfg.ClientID),
		"tenant_id", cfg.TenantID,
		"redirect_uri", cfg.RedirectURI,
		"notebook_name", cfg.NotebookName,
		"quicknote_notebook", cfg.QuickNote.NotebookName,
		"quicknote_page", cfg.QuickNote.PageName,
		"quicknote_date_format", cfg.QuickNote.DateFormat,
		"mcp_auth_enabled", cfg.MCPAuth.Enabled,
		"mcp_bearer_token", maskSensitiveData(cfg.MCPAuth.BearerToken),
		"stateless", cfg.Stateless != nil && *cfg.Stateless,
		"log_level", cfg.LogLevel,
		"log_format", cfg.LogFormat,
		"log_file", cfg.LogFile,
		"content_log_level", cfg.ContentLogLevel)

	// Log all environment variables that could affect configuration
	logger.Debug("Environment variables scan",
		"ONENOTE_CLIENT_ID", maskEnvironmentValue("ONENOTE_CLIENT_ID"),
		"ONENOTE_TENANT_ID", maskEnvironmentValue("ONENOTE_TENANT_ID"),
		"ONENOTE_REDIRECT_URI", maskEnvironmentValue("ONENOTE_REDIRECT_URI"),
		"ONENOTE_DEFAULT_NOTEBOOK_NAME", maskEnvironmentValue("ONENOTE_DEFAULT_NOTEBOOK_NAME"),
		"ONENOTE_TOOLSETS", maskEnvironmentValue("ONENOTE_TOOLSETS"),
		"ONENOTE_MCP_CONFIG", maskEnvironmentValue("ONENOTE_MCP_CONFIG"),
		"QUICKNOTE_NOTEBOOK_NAME", maskEnvironmentValue("QUICKNOTE_NOTEBOOK_NAME"),
		"QUICKNOTE_PAGE_NAME", maskEnvironmentValue("QUICKNOTE_PAGE_NAME"),
		"QUICKNOTE_DATE_FORMAT", maskEnvironmentValue("QUICKNOTE_DATE_FORMAT"),
		"MCP_AUTH_ENABLED", maskEnvironmentValue("MCP_AUTH_ENABLED"),
		"MCP_BEARER_TOKEN", maskEnvironmentValue("MCP_BEARER_TOKEN"),
		"MCP_STATELESS", maskEnvironmentValue("MCP_STATELESS"),
		"LOG_LEVEL", maskEnvironmentValue("LOG_LEVEL"),
		"LOG_FORMAT", maskEnvironmentValue("LOG_FORMAT"),
		"MCP_LOG_FILE", maskEnvironmentValue("MCP_LOG_FILE"),
		"CONTENT_LOG_LEVEL", maskEnvironmentValue("CONTENT_LOG_LEVEL"))

	// Toolsets from env (comma-separated string)
	var envHasToolsets bool
	if toolsets := os.Getenv("ONENOTE_TOOLSETS"); toolsets != "" {
		cfg.Toolsets = strings.Split(toolsets, ",")
		envHasToolsets = true
		logger.Debug("Loaded toolsets from environment", "toolsets", cfg.Toolsets)
	} else {
		logger.Debug("No toolsets specified in environment - using defaults")
		cfg.Toolsets = []string{"notebooks", "sections", "pages", "content"}
	}


	// Optionally load from config file if ONENOTE_MCP_CONFIG is set
	if path := os.Getenv("ONENOTE_MCP_CONFIG"); path != "" {
		logger.Debug("Loading from config file", "path", path)

		// Save environment variable values to restore after JSON loading
		// (environment variables have higher precedence than JSON)
		envClientID := cfg.ClientID
		envTenantID := cfg.TenantID
		envRedirectURI := cfg.RedirectURI
		envNotebookName := cfg.NotebookName
		var envToolsets []string
		if envHasToolsets {
			envToolsets = make([]string, len(cfg.Toolsets))
			copy(envToolsets, cfg.Toolsets)
		}
		envQuickNote := QuickNoteConfig{
			NotebookName: cfg.QuickNote.NotebookName,
			PageName:     cfg.QuickNote.PageName,
			DateFormat:   cfg.QuickNote.DateFormat,
		}
		envMCPAuth := MCPAuthConfig{
			Enabled:     cfg.MCPAuth.Enabled,
			BearerToken: cfg.MCPAuth.BearerToken,
		}
		envStateless := cfg.Stateless
		envLogLevel := cfg.LogLevel
		envLogFormat := cfg.LogFormat
		envLogFile := cfg.LogFile
		envContentLogLevel := cfg.ContentLogLevel

		fileInfo, err := os.Stat(path)
		if err != nil {
			logger.Error("Failed to stat config file", "path", path, "error", err)
			return nil, fmt.Errorf("config file not accessible: %v", err)
		}
		logger.Debug("Config file info", "size", fileInfo.Size(), "mod_time", fileInfo.ModTime())

		f, err := os.Open(path)
		if err != nil {
			logger.Error("Failed to open config file", "path", path, "error", err)
			return nil, fmt.Errorf("failed to open config file: %v", err)
		}
		defer f.Close()

		dec := json.NewDecoder(f)
		if err := dec.Decode(cfg); err != nil {
			logger.Error("Failed to decode JSON from config file", "path", path, "error", err)
			return nil, fmt.Errorf("failed to parse config file JSON: %v", err)
		}

		// Restore environment variable values (they have higher precedence)
		if envClientID != "" {
			cfg.ClientID = envClientID
		}
		if envTenantID != "" {
			cfg.TenantID = envTenantID
		}
		if envRedirectURI != "" {
			cfg.RedirectURI = envRedirectURI
		}
		if envNotebookName != "" {
			cfg.NotebookName = envNotebookName
		}
		if len(envToolsets) > 0 {
			cfg.Toolsets = envToolsets
		}
		if envQuickNote.NotebookName != "" {
			cfg.QuickNote.NotebookName = envQuickNote.NotebookName
		}
		if envQuickNote.PageName != "" {
			cfg.QuickNote.PageName = envQuickNote.PageName
		}
		if envQuickNote.DateFormat != "" {
			cfg.QuickNote.DateFormat = envQuickNote.DateFormat
		}
		if cfg.MCPAuth == nil {
			cfg.MCPAuth = &MCPAuthConfig{}
		}
		if envMCPAuth.Enabled || envMCPAuth.BearerToken != "" {
			cfg.MCPAuth.Enabled = envMCPAuth.Enabled
			cfg.MCPAuth.BearerToken = envMCPAuth.BearerToken
		}
		if envStateless != nil {
			cfg.Stateless = envStateless
		}
		if envLogLevel != "" {
			cfg.LogLevel = envLogLevel
		}
		if envLogFormat != "" {
			cfg.LogFormat = envLogFormat
		}
		if envLogFile != "" {
			cfg.LogFile = envLogFile
		}
		if envContentLogLevel != "" {
			cfg.ContentLogLevel = envContentLogLevel
		}

		// Ensure stateless has a default value if not set in JSON
		if cfg.Stateless == nil {
			defaultStateless := false
			cfg.Stateless = &defaultStateless
		}

		// Ensure QuickNote has a default value if not set in JSON
		if cfg.QuickNote == nil {
			cfg.QuickNote = &QuickNoteConfig{}
		}

		// Set default date format if not provided
		if cfg.QuickNote.DateFormat == "" {
			cfg.QuickNote.DateFormat = "January 2, 2006 - 3:04 PM"
		}

		// Ensure Authorization has a default value if not set in JSON
		if cfg.Authorization == nil {
			cfg.Authorization = authorization.NewAuthorizationConfig()
		}

		// Ensure DefaultNotebookPermissions has a valid default if not set in JSON
		if cfg.Authorization.DefaultNotebookPermissions == "" {
			cfg.Authorization.DefaultNotebookPermissions = authorization.PermissionRead
		}

		// Set default toolsets if not specified in JSON
		if len(cfg.Toolsets) == 0 {
			cfg.Toolsets = []string{"notebooks", "sections", "pages", "content"}
		}

		logger.Debug("Successfully loaded from config file",
			"client_id", maskSensitiveData(cfg.ClientID),
			"tenant_id", cfg.TenantID,
			"redirect_uri", cfg.RedirectURI,
			"toolsets", cfg.Toolsets,
			"notebook_name", cfg.NotebookName,
			"quicknote_notebook", cfg.QuickNote.NotebookName,
			"quicknote_page", cfg.QuickNote.PageName,
			"quicknote_date_format", cfg.QuickNote.DateFormat,
			"mcp_auth_enabled", cfg.MCPAuth.Enabled,
			"mcp_bearer_token", maskSensitiveData(cfg.MCPAuth.BearerToken),
			"stateless", cfg.Stateless != nil && *cfg.Stateless,
			"log_level", cfg.LogLevel,
			"log_format", cfg.LogFormat,
			"log_file", cfg.LogFile,
			"content_log_level", cfg.ContentLogLevel)

		logger.Info("Configuration loaded from JSON file",
			"config_file", path,
			"file_size", fileInfo.Size(),
			"contains_client_id", cfg.ClientID != "",
			"contains_tenant_id", cfg.TenantID != "",
			"contains_redirect_uri", cfg.RedirectURI != "",
			"contains_notebook_name", cfg.NotebookName != "",
			"toolsets_count", len(cfg.Toolsets),
			"mcp_auth_configured", cfg.MCPAuth != nil && cfg.MCPAuth.Enabled,
			"logging_configured", cfg.LogLevel != "" || cfg.LogFormat != "" || cfg.LogFile != "" || cfg.ContentLogLevel != "")
	} else {
		logger.Debug("No config file specified (ONENOTE_MCP_CONFIG not set)")
		
		// Set default date format if not provided from environment
		if cfg.QuickNote.DateFormat == "" {
			cfg.QuickNote.DateFormat = "January 2, 2006 - 3:04 PM"
		}
	}

	// Apply environment variable overrides for authorization configuration
	if cfg.Authorization != nil {
		if authEnabled := os.Getenv("AUTHORIZATION_ENABLED"); authEnabled != "" {
			cfg.Authorization.Enabled = authEnabled == "true"
		}
		if defaultMode := os.Getenv("AUTHORIZATION_DEFAULT_MODE"); defaultMode != "" {
			cfg.Authorization.DefaultNotebookPermissions = authorization.PermissionLevel(defaultMode)
		}
		// Parse authorization permissions from environment variables if present
		if notebookPerms := os.Getenv("AUTHORIZATION_NOTEBOOK_PERMISSIONS"); notebookPerms != "" {
			// Parse JSON format: {"pattern":"permission",...}
			var perms map[string]string
			if err := json.Unmarshal([]byte(notebookPerms), &perms); err == nil {
				for pattern, perm := range perms {
					cfg.Authorization.NotebookPermissions[pattern] = authorization.PermissionLevel(perm)
				}
			}
		}
		if sectionPerms := os.Getenv("AUTHORIZATION_SECTION_PERMISSIONS"); sectionPerms != "" {
			var perms map[string]string
			if err := json.Unmarshal([]byte(sectionPerms), &perms); err == nil {
				for pattern, perm := range perms {
					cfg.Authorization.SectionPermissions[pattern] = authorization.PermissionLevel(perm)
				}
			}
		}
		if pagePerms := os.Getenv("AUTHORIZATION_PAGE_PERMISSIONS"); pagePerms != "" {
			var perms map[string]string
			if err := json.Unmarshal([]byte(pagePerms), &perms); err == nil {
				for pattern, perm := range perms {
					cfg.Authorization.PagePermissions[pattern] = authorization.PermissionLevel(perm)
				}
			}
		}
	}

	// Apply defaults for empty fields
	if cfg.LogLevel == "" {
		cfg.LogLevel = "INFO"
	}
	if cfg.LogFormat == "" {
		cfg.LogFormat = "text"
	}
	if cfg.ContentLogLevel == "" {
		cfg.ContentLogLevel = "INFO"
	}

	// Validate configuration
	logger.Debug("Validating configuration")
	if err := validateConfig(cfg); err != nil {
		logger.Error("Configuration validation failed", "error", err)
		return nil, err
	}
	logger.Debug("Configuration validation passed")

	// Compile authorization matchers if authorization is configured
	if cfg.Authorization != nil {
		logger.Debug("Compiling authorization matchers")
		if err := cfg.Authorization.CompilePatterns(); err != nil {
			logger.Error("Authorization matcher compilation failed", "error", err)
			return nil, fmt.Errorf("failed to compile authorization matchers: %v", err)
		}
		logger.Debug("Authorization matchers compiled successfully")
	}

	elapsed := time.Since(startTime)

	// Comprehensive final configuration summary
	logger.Info("Configuration loaded successfully",
		"duration", elapsed,
		"client_id_configured", cfg.ClientID != "",
		"tenant_id_configured", cfg.TenantID != "",
		"redirect_uri_configured", cfg.RedirectURI != "",
		"notebook_name_configured", cfg.NotebookName != "",
		"toolsets_configured", len(cfg.Toolsets) > 0,
		"mcp_auth_configured", cfg.MCPAuth != nil && cfg.MCPAuth.Enabled,
		"logging_level_set", cfg.LogLevel != "",
		"config_source", getConfigSource())

	logger.Debug("Final configuration details",
		"client_id", maskSensitiveData(cfg.ClientID),
		"tenant_id", cfg.TenantID,
		"redirect_uri", cfg.RedirectURI,
		"notebook_name", cfg.NotebookName,
		"quicknote_notebook", cfg.QuickNote.NotebookName,
		"quicknote_page", cfg.QuickNote.PageName,
		"quicknote_date_format", cfg.QuickNote.DateFormat,
		"toolsets", cfg.Toolsets,
		"toolsets_count", len(cfg.Toolsets),
		"mcp_auth_enabled", cfg.MCPAuth.Enabled,
		"mcp_bearer_token", maskSensitiveData(cfg.MCPAuth.BearerToken),
		"stateless", cfg.Stateless != nil && *cfg.Stateless,
		"log_level", cfg.LogLevel,
		"log_format", cfg.LogFormat,
		"log_file", cfg.LogFile,
		"content_log_level", cfg.ContentLogLevel)

	// Log effective configuration values (after all sources processed)
	logger.Debug("Effective configuration values",
		"client_id_length", len(cfg.ClientID),
		"tenant_id_set", cfg.TenantID != "",
		"redirect_uri_set", cfg.RedirectURI != "",
		"default_notebook_name", cfg.NotebookName,
		"enabled_toolsets", cfg.Toolsets,
		"mcp_auth_config", map[string]interface{}{
			"enabled":      cfg.MCPAuth.Enabled,
			"token_length": len(cfg.MCPAuth.BearerToken),
		},
		"logging_config", map[string]string{
			"level":         cfg.LogLevel,
			"format":        cfg.LogFormat,
			"file":          cfg.LogFile,
			"content_level": cfg.ContentLogLevel,
		})

	return cfg, nil
}

// GetLogLevel returns the configured logging level.
func (c *Config) GetLogLevel() string {
	return c.LogLevel
}

// GetLogFormat returns the configured log format (text or json).
func (c *Config) GetLogFormat() string {
	return c.LogFormat
}

// GetLogFile returns the configured log file path.
func (c *Config) GetLogFile() string {
	return c.LogFile
}

// GetContentLogLevel returns the configured content logging level.
func (c *Config) GetContentLogLevel() string {
	return c.ContentLogLevel
}

// validateConfig performs validation on the loaded configuration
func validateConfig(cfg *Config) error {
	logger := logging.ConfigLogger

	logger.Debug("Starting configuration validation",
		"client_id_present", cfg.ClientID != "",
		"tenant_id_present", cfg.TenantID != "",
		"redirect_uri_present", cfg.RedirectURI != "",
		"toolsets_count", len(cfg.Toolsets))

	logger.Debug("Validating ClientID")
	if cfg.ClientID == "" {
		logger.Error("ClientID validation failed - required field missing")
		return fmt.Errorf("ONENOTE_CLIENT_ID is required")
	}
	logger.Debug("ClientID validation passed", "client_id_length", len(cfg.ClientID))

	logger.Debug("Validating TenantID")
	if cfg.TenantID == "" {
		logger.Error("TenantID validation failed - required field missing")
		return fmt.Errorf("ONENOTE_TENANT_ID is required")
	}
	logger.Debug("TenantID validation passed", "tenant_id", cfg.TenantID)

	logger.Debug("Validating RedirectURI")
	if cfg.RedirectURI == "" {
		logger.Error("RedirectURI validation failed - required field missing")
		return fmt.Errorf("ONENOTE_REDIRECT_URI is required")
	}

	// Basic URL validation
	if !strings.HasPrefix(cfg.RedirectURI, "http://") && !strings.HasPrefix(cfg.RedirectURI, "https://") {
		logger.Error("RedirectURI validation failed - invalid URL format", "redirect_uri", cfg.RedirectURI)
		return fmt.Errorf("ONENOTE_REDIRECT_URI must be a valid HTTP/HTTPS URL")
	}
	logger.Debug("RedirectURI validation passed", "redirect_uri", cfg.RedirectURI)

	logger.Debug("Validating toolsets", "toolsets", cfg.Toolsets)
	if len(cfg.Toolsets) > 0 {
		validToolsets := map[string]bool{
			"notebooks": true,
			"sections":  true,
			"pages":     true,
			"content":   true,
		}

		for i, toolset := range cfg.Toolsets {
			cleanToolset := strings.TrimSpace(toolset)
			if !validToolsets[cleanToolset] {
				logger.Error("Toolset validation failed",
					"invalid_toolset", toolset,
					"index", i,
					"valid_options", []string{"notebooks", "sections", "pages", "content"})
				return fmt.Errorf("invalid toolset: %s (valid options: notebooks, sections, pages, content)", toolset)
			}
			logger.Debug("Toolset validated", "toolset", cleanToolset, "index", i)
		}
		logger.Debug("All toolsets validation passed", "valid_toolsets", cfg.Toolsets)
	} else {
		logger.Debug("No toolsets specified - using default behavior")
	}

	// Validate logging configuration values if present
	if cfg.LogLevel != "" {
		validLogLevels := []string{"DEBUG", "INFO", "WARN", "WARNING", "ERROR"}
		validLevel := false
		for _, level := range validLogLevels {
			if strings.ToUpper(cfg.LogLevel) == level {
				validLevel = true
				break
			}
		}
		if !validLevel {
			logger.Warn("Invalid log level specified, will use default",
				"specified_level", cfg.LogLevel,
				"valid_levels", validLogLevels)
		} else {
			logger.Debug("Log level validation passed", "log_level", cfg.LogLevel)
		}
	}

	if cfg.ContentLogLevel != "" {
		validContentLevels := []string{"DEBUG", "INFO", "WARN", "WARNING", "ERROR", "OFF"}
		validLevel := false
		for _, level := range validContentLevels {
			if strings.ToUpper(cfg.ContentLogLevel) == level {
				validLevel = true
				break
			}
		}
		if !validLevel {
			logger.Warn("Invalid content log level specified, will use default",
				"specified_level", cfg.ContentLogLevel,
				"valid_levels", validContentLevels)
		} else {
			logger.Debug("Content log level validation passed", "content_log_level", cfg.ContentLogLevel)
		}
	}

	// Validate authorization configuration if present
	if cfg.Authorization != nil {
		logger.Debug("Validating authorization configuration")
		if err := authorization.ValidateAuthorizationConfig(cfg.Authorization); err != nil {
			logger.Error("Authorization configuration validation failed", "error", err)
			return fmt.Errorf("authorization configuration is invalid: %v", err)
		}
		logger.Debug("Authorization configuration validation passed")
	}

	logger.Debug("Configuration validation completed successfully")
	return nil
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

// maskEnvironmentValue gets and masks an environment variable value for logging
func maskEnvironmentValue(envVar string) string {
	value := os.Getenv(envVar)
	if value == "" {
		return "<not set>"
	}

	// Identify sensitive environment variables
	sensitiveVars := map[string]bool{
		"ONENOTE_CLIENT_ID":    true,
		"ONENOTE_TENANT_ID":    false, // Tenant IDs are not as sensitive
		"ONENOTE_REDIRECT_URI": false, // Redirect URIs are not sensitive
		"MCP_BEARER_TOKEN":     true,  // Bearer tokens are sensitive
	}

	if sensitive, exists := sensitiveVars[envVar]; exists && sensitive {
		return maskSensitiveData(value)
	}

	return value
}

// getConfigSource determines the primary source of configuration
func getConfigSource() string {
	if os.Getenv("ONENOTE_MCP_CONFIG") != "" {
		return "json_file_with_env_overrides"
	}
	return "environment_variables_only"
}
