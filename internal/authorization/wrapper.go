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

		// Extract resource context from the request
		resourceContext := ExtractResourceContext(toolName, req, cache)

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
		ResolvePageContext(&resourceContext, cache)

		// Enhanced security check: If we have a page ID but still no page name after context resolution,
		// and page permissions are configured, we need to be more cautious
		if resourceContext.PageID != "" && resourceContext.PageName == "" && authConfig.DefaultPageMode == PermissionNone {
			logging.AuthorizationLogger.Warn("Page ID provided but page name could not be resolved with default_page_mode=none",
				"tool", toolName,
				"page_id", resourceContext.PageID,
				"default_page_mode", authConfig.DefaultPageMode,
				"security_implications", "This will apply the restrictive default page mode to prevent authorization bypass")
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
	Enabled         bool   `json:"enabled"`
	DefaultMode     string `json:"default_mode"`
	ToolCategories  int    `json:"tool_categories_configured"`
	NotebookRules   int    `json:"notebook_rules_configured"`
	SectionRules    int    `json:"section_rules_configured"`
	CompiledMatchers bool   `json:"compiled_matchers"`
}

// GetAuthorizationInfo returns information about the current authorization configuration
func GetAuthorizationInfo(authConfig *AuthorizationConfig) AuthorizationInfo {
	if authConfig == nil {
		return AuthorizationInfo{
			Enabled: false,
		}
	}

	return AuthorizationInfo{
		Enabled:         authConfig.Enabled,
		DefaultMode:     string(authConfig.DefaultMode),
		ToolCategories:  len(authConfig.ToolPermissions),
		NotebookRules:   len(authConfig.NotebookPermissions),
		SectionRules:    len(authConfig.SectionPermissions),
		CompiledMatchers: len(authConfig.notebookMatchers) > 0 || len(authConfig.sectionMatchers) > 0,
	}
}

// ValidateAuthorizationConfig validates an authorization configuration
func ValidateAuthorizationConfig(authConfig *AuthorizationConfig) error {
	if authConfig == nil {
		return nil // nil config is valid (authorization disabled)
	}

	// Validate default modes
	switch authConfig.DefaultMode {
	case PermissionNone, PermissionRead, PermissionWrite, PermissionFull:
		// Valid
	default:
		return fmt.Errorf("invalid default_mode: %s (must be one of: none, read, write, full)", authConfig.DefaultMode)
	}
	
	// Validate specific default modes (optional fields)
	for name, mode := range map[string]PermissionLevel{
		"default_tool_mode":     authConfig.DefaultToolMode,
		"default_notebook_mode": authConfig.DefaultNotebookMode,
		"default_section_mode":  authConfig.DefaultSectionMode,
		"default_page_mode":     authConfig.DefaultPageMode,
	} {
		if mode != "" {
			switch mode {
			case PermissionNone, PermissionRead, PermissionWrite, PermissionFull:
				// Valid
			default:
				return fmt.Errorf("invalid %s: %s (must be one of: none, read, write, full)", name, mode)
			}
		}
	}

	// Validate tool permissions
	for category, permission := range authConfig.ToolPermissions {
		switch permission {
		case PermissionNone, PermissionRead, PermissionWrite, PermissionFull:
			// Valid
		default:
			return fmt.Errorf("invalid permission '%s' for tool category '%s' (must be one of: none, read, write, full)", permission, category)
		}
		
		// Validate that category exists
		validCategory := false
		switch category {
		case CategoryAuthTools, CategoryNotebookRead, CategoryNotebookWrite, CategoryPageRead, CategoryPageWrite, CategoryTestTools:
			validCategory = true
		}
		if !validCategory {
			return fmt.Errorf("invalid tool category: %s", category)
		}
	}

	// Validate notebook permissions
	for notebook, permission := range authConfig.NotebookPermissions {
		switch permission {
		case PermissionNone, PermissionRead, PermissionWrite, PermissionFull:
			// Valid
		default:
			return fmt.Errorf("invalid permission '%s' for notebook '%s' (must be one of: none, read, write, full)", permission, notebook)
		}
	}

	// Validate section permissions
	for section, permission := range authConfig.SectionPermissions {
		switch permission {
		case PermissionNone, PermissionRead, PermissionWrite, PermissionFull:
			// Valid
		default:
			return fmt.Errorf("invalid permission '%s' for section '%s' (must be one of: none, read, write, full)", permission, section)
		}
	}

	// Validate page permissions
	for page, permission := range authConfig.PagePermissions {
		switch permission {
		case PermissionNone, PermissionRead, PermissionWrite, PermissionFull:
			// Valid
		default:
			return fmt.Errorf("invalid permission '%s' for page '%s' (must be one of: none, read, write, full)", permission, page)
		}
	}

	logging.AuthorizationLogger.Debug("Authorization configuration validated successfully",
		"enabled", authConfig.Enabled,
		"default_mode", authConfig.DefaultMode,
		"tool_permissions", len(authConfig.ToolPermissions),
		"notebook_permissions", len(authConfig.NotebookPermissions),
		"section_permissions", len(authConfig.SectionPermissions),
		"page_permissions", len(authConfig.PagePermissions))

	return nil
}