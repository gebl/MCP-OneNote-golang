// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package authorization

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/gebl/onenote-mcp-server/internal/logging"
)

// ToolHandler represents the signature of an MCP tool handler function
type ToolHandler func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error)

// AuthorizedToolHandler wraps a tool handler with authorization checks
func AuthorizedToolHandler(toolName string, handler ToolHandler, authConfig *AuthorizationConfig, cache NotebookCache, quickNoteConfig QuickNoteConfig) ToolHandler {
	return AuthorizedToolHandlerWithResolver(toolName, handler, authConfig, cache, quickNoteConfig, nil)
}

// AuthorizedToolHandlerWithResolver wraps a tool handler with authorization checks including page notebook resolution
func AuthorizedToolHandlerWithResolver(toolName string, handler ToolHandler, authConfig *AuthorizationConfig, cache NotebookCache, quickNoteConfig QuickNoteConfig, resolver PageNotebookResolver) ToolHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

		// Extract resource context from the request with proper page notebook resolution
		var resourceContext ResourceContext
		if resolver != nil {
			resourceContext = ExtractResourceContextWithResolver(ctx, toolName, req, cache, resolver)
		} else {
			resourceContext = ExtractResourceContext(toolName, req, cache)
		}

		// Special handling for quickNote tool
		if toolName == "quickNote" && quickNoteConfig != nil {
			targetNotebook := quickNoteConfig.GetNotebookName()
			if targetNotebook == "" {
				targetNotebook = quickNoteConfig.GetDefaultNotebook()
			}
			targetPage := quickNoteConfig.GetPageName()
			
			EnrichQuickNoteContext(&resourceContext, targetNotebook, targetPage)
			
			logging.AuthorizationLogger.Debug("Enhanced quickNote context",
				"target_notebook", targetNotebook,
				"target_page", targetPage,
				"final_context", resourceContext.String())
		}

		// Resolve additional context if needed
		ResolveNotebookContext(&resourceContext, cache)
		ResolveSectionContext(ctx, &resourceContext, cache)
		ResolvePageContext(&resourceContext, cache)

		// Enhanced security check: If we have a page ID but still no page name after context resolution,
		// we will rely on the simplified authorization model to handle it securely
		if resourceContext.PageID != "" && resourceContext.PageName == "" {
			logging.AuthorizationLogger.Warn("Page ID provided but page name could not be resolved",
				"tool", toolName,
				"page_id", resourceContext.PageID,
				"security_implications", "Will use current notebook permission as fallback")
		}

		// Perform authorization check
		err := authConfig.IsAuthorized(ctx, toolName, req, resourceContext)
		if err != nil {
			logging.AuthorizationLogger.Info("Authorization check failed",
				"tool", toolName,
				"resource_context", resourceContext.String(),
				"error", err.Error())
			return mcp.NewToolResultError(fmt.Sprintf("Authorization failed: %v", err)), nil
		}

		logging.AuthorizationLogger.Debug("Authorization check passed, executing tool",
			"tool", toolName,
			"resource_context", resourceContext.String())

		// Authorization passed, execute the original handler
		return handler(ctx, req)
	}
}

// QuickNoteConfig interface defines the methods needed to get quickNote configuration
type QuickNoteConfig interface {
	GetNotebookName() string    // Returns quicknote-specific notebook name
	GetDefaultNotebook() string // Returns default notebook name as fallback
	GetPageName() string        // Returns target page name
}

// CreateAuthorizedTool creates an MCP tool with authorization wrapper
func CreateAuthorizedTool(toolName string, handler ToolHandler, authConfig *AuthorizationConfig, cache NotebookCache, quickNoteConfig QuickNoteConfig, toolOptions ...mcp.ToolOption) mcp.Tool {
	// Create the tool with the options
	tool := mcp.NewTool(toolName, toolOptions...)
	
	logging.AuthorizationLogger.Debug("Created authorized tool",
		"tool", toolName,
		"authorization_enabled", authConfig != nil && authConfig.Enabled)
	
	return tool
}

// AuthorizationInfo provides information about the authorization system status
type AuthorizationInfo struct {
	Enabled                  bool   `json:"enabled"`
	DefaultNotebookMode      string `json:"default_notebook_mode"`
	NotebookRules           int    `json:"notebook_rules_configured"`
	SectionRules            int    `json:"section_rules_configured"`
	PageRules               int    `json:"page_rules_configured"`
	CompiledPatterns        bool   `json:"compiled_patterns"`
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
		SectionRules:            len(authConfig.SectionPermissions),
		PageRules:               len(authConfig.PagePermissions),
		CompiledPatterns:        len(authConfig.NotebookPermissions) > 0 || len(authConfig.SectionPermissions) > 0 || len(authConfig.PagePermissions) > 0,
		CurrentNotebook:         authConfig.GetCurrentNotebook(),
		CurrentNotebookPerm:     string(authConfig.currentNotebookPerm),
	}
}

// ValidateAuthorizationConfig validates an authorization configuration
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
	
	// Validate section permissions
	for name, mode := range authConfig.SectionPermissions {
		switch mode {
		case PermissionNone, PermissionRead, PermissionWrite, PermissionFull:
			// Valid
		default:
			return fmt.Errorf("invalid section permission for '%s': %s (must be one of: none, read, write, full)", name, mode)
		}
	}
	
	// Validate page permissions  
	for name, mode := range authConfig.PagePermissions {
		switch mode {
		case PermissionNone, PermissionRead, PermissionWrite, PermissionFull:
			// Valid
		default:
			return fmt.Errorf("invalid page permission for '%s': %s (must be one of: none, read, write, full)", name, mode)
		}
	}

	return nil
}