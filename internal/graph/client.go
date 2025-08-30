// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

// client.go - Core Microsoft Graph API client for OneNote operations.
//
// This file defines the Client struct and core infrastructure for interacting with OneNote
// via the Microsoft Graph API. It provides the foundation for OneNote operations including
// authentication, token management, and client initialization.
//
// Key Features:
// - Microsoft Graph SDK integration with custom authentication provider
// - Automatic token refresh with retry logic on authentication failures
// - Client struct definition and constructors
// - Token management and authentication provider implementation
//
// Usage Example:
//   graphClient := graph.NewClientWithTokenRefresh(accessToken, oauthConfig, tokenManager, tokenPath, config)
//
//   // Use the client for OneNote operations
//   notebooks, err := graphClient.ListNotebooks()
//   if err != nil {
//       logging.GraphLogger.Error("Failed to list notebooks", "error", err)
//   }

package graph

import (
	"context"
	"fmt"
	"path/filepath"

	abstractions "github.com/microsoft/kiota-abstractions-go"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	msgraphsdkcore "github.com/microsoftgraph/msgraph-sdk-go-core"

	"github.com/gebl/onenote-mcp-server/internal/auth"
	"github.com/gebl/onenote-mcp-server/internal/logging"
)

const unknownPath = "unknown"

// Client handles Microsoft Graph API requests for OneNote.
// It provides methods to list, search, and manipulate notebooks, sections, pages, and page items.
type Client struct {
	GraphClient  *msgraphsdk.GraphServiceClient // Microsoft Graph SDK client
	AuthProvider *StaticTokenProvider           // Authentication provider for token updates
	AccessToken  string                         // OAuth2 access token
	OAuthConfig  *auth.OAuth2Config             // OAuth2 configuration for token refresh
	TokenManager *auth.TokenManager             // Token manager for refresh operations
	TokenPath    string                         // Path to save refreshed tokens
	Config       *Config                        // Configuration including default notebook name
}

// Config holds configuration values needed by the graph client
type Config struct {
	NotebookName string // Default notebook name from configuration
}

// NewClient creates a new Graph API client using the SDK's adapter pattern.
// accessToken: OAuth2 access token for Microsoft Graph API.
// Returns a pointer to a Client instance.
func NewClient(accessToken string) *Client {
	authProvider := &StaticTokenProvider{AccessToken: accessToken}
	adapter, err := msgraphsdkcore.NewGraphRequestAdapterBase(authProvider, msgraphsdkcore.GraphClientOptions{
		GraphServiceVersion: "v1.0",
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to create GraphRequestAdapter: %v", err))
	}
	client := msgraphsdk.NewGraphServiceClient(adapter)
	return &Client{
		GraphClient: client,
		AccessToken: accessToken,
	}
}

// NewClientWithTokenRefresh creates a new Graph API client with token refresh capabilities.
// accessToken: OAuth2 access token for Microsoft Graph API.
// oauthConfig: OAuth2 configuration for token refresh.
// tokenManager: Token manager for refresh operations.
// tokenPath: Path to save refreshed tokens.
// config: Configuration including default notebook name.
// Returns a pointer to a Client instance.
func NewClientWithTokenRefresh(accessToken string, oauthConfig *auth.OAuth2Config, tokenManager *auth.TokenManager, tokenPath string, config *Config) *Client {
	// Get absolute path for better debugging
	absTokenPath, err := filepath.Abs(tokenPath)
	if err != nil {
		logging.GraphLogger.Debug("Could not get absolute path for token file", "path", tokenPath, "error", err)
		absTokenPath = unknownPath
	}

	logging.GraphLogger.Debug("Creating Graph client with token refresh capability",
		"token_path", tokenPath,
		"abs_token_path", absTokenPath,
		"oauth_config_available", oauthConfig != nil,
		"token_manager_available", tokenManager != nil)

	authProvider := &StaticTokenProvider{AccessToken: accessToken}
	
	logging.GraphLogger.Debug("Creating GraphRequestAdapter", 
		"auth_provider_nil", authProvider == nil,
		"access_token_empty", accessToken == "",
		"access_token_length", len(accessToken))
	
	adapter, err := msgraphsdkcore.NewGraphRequestAdapterBase(authProvider, msgraphsdkcore.GraphClientOptions{
		GraphServiceVersion: "v1.0",
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to create GraphRequestAdapter: %v", err))
	}
	
	if adapter == nil {
		panic("GraphRequestAdapter is nil after creation")
	}
	
	client := msgraphsdk.NewGraphServiceClient(adapter)
	if client == nil {
		panic("GraphServiceClient is nil after creation")
	}
	
	logging.GraphLogger.Debug("Graph client created successfully", 
		"adapter_nil", adapter == nil,
		"client_nil", client == nil)
	return &Client{
		GraphClient:  client,
		AuthProvider: authProvider,
		AccessToken:  accessToken,
		OAuthConfig:  oauthConfig,
		TokenManager: tokenManager,
		TokenPath:    tokenPath,
		Config:       config,
	}
}

// StaticTokenProvider implements the AuthenticationProvider interface for a static token.
// Used with the Microsoft Graph Go SDK.
type StaticTokenProvider struct {
	AccessToken string
}

// AuthenticateRequest adds the Authorization header to the request.
func (s *StaticTokenProvider) AuthenticateRequest(ctx context.Context, request *abstractions.RequestInformation, additionalAuthenticationContext map[string]interface{}) error {
	if request == nil {
		logging.GraphLogger.Error("Request is nil in AuthenticateRequest")
		return fmt.Errorf("request cannot be nil")
	}
	if request.Headers == nil {
		logging.GraphLogger.Error("Request headers is nil in AuthenticateRequest")
		return fmt.Errorf("request headers cannot be nil")
	}
	if s.AccessToken == "" {
		logging.GraphLogger.Error("Access token is empty in AuthenticateRequest")
		return fmt.Errorf("access token cannot be empty")
	}
	
	logging.GraphLogger.Debug("Adding authorization header", "token_length", len(s.AccessToken))
	request.Headers.Add("Authorization", "Bearer "+s.AccessToken)
	return nil
}

// GetAuthorizationToken returns the static access token (for compatibility).
func (s *StaticTokenProvider) GetAuthorizationToken(ctx context.Context, scopes []string) (string, error) {
	return s.AccessToken, nil
}

// UpdateAccessToken updates the access token in the StaticTokenProvider.
func (s *StaticTokenProvider) UpdateAccessToken(newToken string) {
	s.AccessToken = newToken
}

// UpdateToken updates the access token in both the client and the auth provider.
func (c *Client) UpdateToken(newToken string) {
	c.AccessToken = newToken
	if c.AuthProvider != nil {
		c.AuthProvider.UpdateAccessToken(newToken)
	}

	// If token is empty, clear the TokenManager as well
	if newToken == "" && c.TokenManager != nil {
		logging.GraphLogger.Info("Clearing TokenManager due to empty access token")
		c.TokenManager.AccessToken = ""
		c.TokenManager.RefreshToken = ""
		c.TokenManager.Expiry = 0
	}

	logging.GraphLogger.Debug("Updated access token in Graph client", "empty", newToken == "")
}

// GetDefaultNotebookID returns the ID of the default notebook specified in the config.
// If no default notebook is configured, it returns an error.
// NOTE: This method has been moved to the notebooks package to avoid circular dependencies.
// Use notebooks.GetDefaultNotebookID(client, config) instead.
func (c *Client) GetDefaultNotebookID() (string, error) {
	return "", fmt.Errorf("GetDefaultNotebookID has been moved to the notebooks package. Use notebooks.GetDefaultNotebookID(client, config) instead")
}

// RefreshTokenIfNeeded checks if the token is expired and refreshes it if necessary.
func (c *Client) RefreshTokenIfNeeded() error {
	if c.TokenManager == nil || c.OAuthConfig == nil {
		return fmt.Errorf("token manager or OAuth config not available")
	}

	if !c.TokenManager.IsExpired() {
		logging.GraphLogger.Debug("Token is not expired, no refresh needed")
		return nil
	}

	logging.GraphLogger.Debug("Token is expired, refreshing")

	// Refresh the token
	newTokenManager, err := c.OAuthConfig.RefreshToken(context.Background(), c.TokenManager.RefreshToken)
	if err != nil {
		return fmt.Errorf("failed to refresh token: %v", err)
	}

	// Update the token manager
	c.TokenManager.AccessToken = newTokenManager.AccessToken
	c.TokenManager.RefreshToken = newTokenManager.RefreshToken
	c.TokenManager.Expiry = newTokenManager.Expiry

	// Update the client's access token
	c.AccessToken = newTokenManager.AccessToken
	if c.AuthProvider != nil {
		c.AuthProvider.UpdateAccessToken(newTokenManager.AccessToken)
	}

	// Save the new tokens
	if c.TokenPath != "" {
		// Get absolute path for better debugging
		absTokenPath, err := filepath.Abs(c.TokenPath)
		if err != nil {
			logging.GraphLogger.Debug("Could not get absolute path for token file", "path", c.TokenPath, "error", err)
			absTokenPath = unknownPath
		}

		logging.GraphLogger.Debug("Saving refreshed tokens", "path", c.TokenPath, "abs_path", absTokenPath)
		if err := newTokenManager.SaveTokens(c.TokenPath); err != nil {
			logging.GraphLogger.Warn("Failed to save refreshed tokens", "path", c.TokenPath, "abs_path", absTokenPath, "error", err)
		} else {
			logging.GraphLogger.Debug("Refreshed tokens saved successfully", "path", c.TokenPath, "abs_path", absTokenPath)
		}
	} else {
		logging.GraphLogger.Debug("No token path configured, skipping token save")
	}

	logging.GraphLogger.Debug("Token refreshed successfully")
	return nil
}
