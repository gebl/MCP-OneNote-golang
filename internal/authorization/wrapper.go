// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package authorization

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/gebl/onenote-mcp-server/internal/logging"
)

// ToolHandler represents the signature of an MCP tool handler function
type ToolHandler func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error)

// ResourceHandler represents the signature of an MCP resource handler function
type ResourceHandler func(ctx context.Context, req *mcp.ReadResourceRequest) ([]mcp.ResourceContents, error)

// AuthorizedToolHandler wraps a tool handler with simplified authorization checks
func AuthorizedToolHandler(toolName string, handler ToolHandler, authConfig *AuthorizationConfig, cache NotebookCache, quickNoteConfig QuickNoteConfig) ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Skip authorization if not enabled
		if authConfig == nil || !authConfig.Enabled {
			logging.AuthorizationLogger.Debug("Authorization wrapper bypassed",
				"tool", toolName,
				"reason", "authorization_disabled")
			return handler(ctx, req)
		}

		logging.AuthorizationLogger.Debug("Authorization wrapper invoked",
			"tool", toolName,
			"authorization_enabled", authConfig.Enabled)

		// Extract simplified resource context
		resourceContext := ExtractResourceContext(ctx, toolName, req, cache)

		// Special handling for quickNote tool - use quicknote notebook if specified
		if toolName == "quickNote" && quickNoteConfig != nil {
			targetNotebook := quickNoteConfig.GetNotebookName()
			if targetNotebook == "" {
				targetNotebook = quickNoteConfig.GetDefaultNotebook()
			}
			
			if targetNotebook != "" {
				resourceContext.NotebookName = targetNotebook
				logging.AuthorizationLogger.Debug("Using quickNote target notebook",
					"target_notebook", targetNotebook)
			}
		}

		// Perform authorization check
		err := authConfig.IsAuthorized(ctx, toolName, req, resourceContext)
		if err != nil {
			logging.AuthorizationLogger.Info("Authorization check failed",
				"tool", toolName,
				"resource_context", resourceContext.String(),
				"error", err.Error())
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("Authorization failed: %v", err)},
				},
				IsError: true,
			}, nil
		}

		logging.AuthorizationLogger.Debug("Authorization check passed, executing tool",
			"tool", toolName,
			"resource_context", resourceContext.String())

		// Authorization passed, execute the original handler
		return handler(ctx, req)
	}
}

// AuthorizedResourceHandler wraps a resource handler with simplified authorization checks
func AuthorizedResourceHandler(resourceName string, handler ResourceHandler, authConfig *AuthorizationConfig, cache NotebookCache) ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		// Skip authorization if not enabled
		if authConfig == nil || !authConfig.Enabled {
			logging.AuthorizationLogger.Debug("Resource authorization wrapper bypassed",
				"resource", resourceName,
				"reason", "authorization_disabled")
			return handler(ctx, req)
		}

		logging.AuthorizationLogger.Debug("Resource authorization wrapper invoked",
			"resource", resourceName,
			"resource_uri", req.Params.URI,
			"authorization_enabled", authConfig.Enabled)

		// For resources, we need to check if the current notebook allows read access
		// Resources are inherently read-only operations
		currentNotebook, hasNotebook := cache.GetDisplayName()
		if !hasNotebook || currentNotebook == "" {
			logging.AuthorizationLogger.Error("AUTHORIZATION DENIED: Resource access requires notebook selection",
				"resource", resourceName,
				"resource_uri", req.Params.URI,
				"reason", "no_notebook_selected")
			return nil, fmt.Errorf("access denied: no notebook selected. Use notebook_select tool first")
		}

		// Check read permission for current notebook
		permission := authConfig.GetNotebookPermission(currentNotebook)
		if permission == PermissionNone {
			logging.AuthorizationLogger.Error("AUTHORIZATION DENIED: Resource access blocked for notebook",
				"resource", resourceName,
				"resource_uri", req.Params.URI,
				"notebook", currentNotebook,
				"permission", permission)
			return nil, fmt.Errorf("access denied: insufficient permissions for notebook '%s'", currentNotebook)
		}

		logging.AuthorizationLogger.Debug("Resource authorization passed",
			"resource", resourceName,
			"resource_uri", req.Params.URI,
			"notebook", currentNotebook,
			"permission", permission)

		// Call the original handler
		return handler(ctx, req)
	}
}

// QuickNoteConfig interface defines the methods needed to get quickNote configuration
type QuickNoteConfig interface {
	GetNotebookName() string    // Returns quicknote-specific notebook name
	GetDefaultNotebook() string // Returns default notebook name as fallback
	GetPageName() string        // Returns target page name
}

// AuthorizationInfo provides information about the simplified authorization system status
type AuthorizationInfo struct {
	Enabled                  bool   `json:"enabled"`
	DefaultNotebookMode      string `json:"default_notebook_mode"`
	NotebookRules           int    `json:"notebook_rules_configured"`
	CurrentNotebook         string `json:"current_notebook"`
	CurrentNotebookPerm     string `json:"current_notebook_permission"`
}

// GetAuthorizationInfo returns information about the current authorization configuration
func GetAuthorizationInfo(authConfig *AuthorizationConfig) AuthorizationInfo {
	if authConfig == nil {
		return AuthorizationInfo{
			Enabled: false,
		}
	}

	return AuthorizationInfo{
		Enabled:                  authConfig.Enabled,
		DefaultNotebookMode:      string(authConfig.DefaultNotebookPermissions),
		NotebookRules:           len(authConfig.NotebookPermissions),
		CurrentNotebook:         authConfig.GetCurrentNotebook(),
		CurrentNotebookPerm:     string(authConfig.currentNotebookPerm),
	}
}

// ValidateAuthorizationConfig validates the simplified authorization configuration
func ValidateAuthorizationConfig(authConfig *AuthorizationConfig) error {
	if authConfig == nil {
		return nil // nil config is valid (authorization disabled)
	}

	// Validate default notebook permissions
	switch authConfig.DefaultNotebookPermissions {
	case PermissionNone, PermissionRead, PermissionWrite, PermissionFull:
		// Valid
	default:
		return fmt.Errorf("invalid default_notebook_permissions: %s (must be one of: none, read, write, full)", authConfig.DefaultNotebookPermissions)
	}
	
	// Validate notebook permissions
	for name, mode := range authConfig.NotebookPermissions {
		switch mode {
		case PermissionNone, PermissionRead, PermissionWrite, PermissionFull:
			// Valid
		default:
			return fmt.Errorf("invalid notebook permission for '%s': %s (must be one of: none, read, write, full)", name, mode)
		}
	}

	return nil
}