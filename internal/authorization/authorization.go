// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package authorization

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/gebl/onenote-mcp-server/internal/logging"
)

// PermissionLevel defines the level of access allowed
type PermissionLevel string

const (
	PermissionNone  PermissionLevel = "none"  // Block all access
	PermissionRead  PermissionLevel = "read"  // Allow read-only operations
	PermissionWrite PermissionLevel = "write" // Allow read + write operations
	PermissionFull  PermissionLevel = "full"  // Allow all operations (same as write for most tools)
)


// ToolOperation represents whether a tool performs read or write operations
type ToolOperation string

const (
	OperationRead  ToolOperation = "read"
	OperationWrite ToolOperation = "write"
)

// AuthorizationConfig represents the simplified notebook-scoped authorization configuration
type AuthorizationConfig struct {
	Enabled                     bool                          `json:"enabled"`
	DefaultNotebookPermissions  PermissionLevel               `json:"default_notebook_permissions"`  // Default for any notebook
	NotebookPermissions         map[string]PermissionLevel    `json:"notebook_permissions"`          // Specific notebook permissions (exact match only)
	
	// Current selected notebook for operation scoping
	currentNotebook     string          `json:"-"`
	currentNotebookPerm PermissionLevel `json:"-"`
}


// ResourceContext contains information about the resource being accessed (simplified)
type ResourceContext struct {
	NotebookName string
	Operation    ToolOperation
}

// String returns a human-readable representation of the resource context
func (rc ResourceContext) String() string {
	parts := []string{}
	if rc.NotebookName != "" {
		parts = append(parts, fmt.Sprintf("notebook=%s", rc.NotebookName))
	}
	parts = append(parts, fmt.Sprintf("operation=%s", rc.Operation))
	return strings.Join(parts, ", ")
}

// ToolRegistry maps tool names to their categories and operations
// AuthToolNames defines which tools are always allowed (authentication tools + discovery tools)
var AuthToolNames = map[string]bool{
	"getAuthStatus":  true,
	"refreshToken":   true,
	"initiateAuth":   true,
	"clearAuth":      true,
	"listNotebooks":  true, // Always allow notebook discovery (results are filtered)
	"selectNotebook": true, // Always allow notebook selection (but selection itself is validated)
}

// NewAuthorizationConfig creates a new simplified authorization configuration
func NewAuthorizationConfig() *AuthorizationConfig {
	return &AuthorizationConfig{
		Enabled:                     false,
		DefaultNotebookPermissions:  PermissionRead,
		NotebookPermissions:         make(map[string]PermissionLevel),
		currentNotebook:             "",
		currentNotebookPerm:         PermissionNone,
	}
}

// ValidateConfig validates the authorization configuration (no pattern compilation needed)
func (ac *AuthorizationConfig) ValidateConfig() error {
	logging.AuthorizationLogger.Debug("Authorization configuration validated",
		"enabled", ac.Enabled,
		"notebook_permissions_count", len(ac.NotebookPermissions),
		"default_permissions", ac.DefaultNotebookPermissions)
	
	return nil
}


// SetCurrentNotebook sets the currently selected notebook and validates permission
func (ac *AuthorizationConfig) SetCurrentNotebook(notebookName string) error {
	if !ac.Enabled {
		ac.currentNotebook = notebookName
		ac.currentNotebookPerm = PermissionWrite // Allow everything when disabled
		logging.AuthorizationLogger.Debug("Authorization disabled, allowing notebook selection", "notebook", notebookName)
		return nil
	}
	
	permission := ac.GetNotebookPermission(notebookName)
	
	if permission == PermissionNone {
		logging.AuthorizationLogger.Info("Notebook selection denied",
			"notebook", notebookName,
			"permission", permission)
		return fmt.Errorf("access denied: cannot select notebook '%s' - permission is '%s'", notebookName, permission)
	}
	
	ac.currentNotebook = notebookName
	ac.currentNotebookPerm = permission
	logging.AuthorizationLogger.Debug("Notebook selected",
		"notebook", notebookName,
		"permission", permission)
	return nil
}

// GetCurrentNotebook returns the currently selected notebook name
func (ac *AuthorizationConfig) GetCurrentNotebook() string {
	return ac.currentNotebook
}

// IsAuthorized checks if a tool call is authorized based on the simplified notebook-scoped configuration
func (ac *AuthorizationConfig) IsAuthorized(ctx context.Context, toolName string, req mcp.CallToolRequest, resourceContext ResourceContext) error {
	if !ac.Enabled {
		logging.AuthorizationLogger.Debug("Authorization disabled, allowing all operations", "tool", toolName)
		return nil
	}
	
	logging.AuthorizationLogger.Debug("Checking authorization", 
		"tool", toolName,
		"resource_context", resourceContext.String(),
		"current_notebook", ac.currentNotebook)
	
	// 1. Auth tools are always allowed
	if AuthToolNames[toolName] {
		logging.AuthorizationLogger.Debug("Auth tool allowed", "tool", toolName)
		return nil
	}
	
	// 2. For non-auth tools, ensure we have a selected notebook with permission
	if ac.currentNotebook == "" {
		logging.AuthorizationLogger.Info("No notebook selected for non-auth tool",
			"tool", toolName)
		return fmt.Errorf("access denied: no notebook selected - use selectNotebook tool first")
	}
	
	if ac.currentNotebookPerm == PermissionNone {
		logging.AuthorizationLogger.Info("Current notebook has no permission",
			"tool", toolName,
			"notebook", ac.currentNotebook,
			"permission", ac.currentNotebookPerm)
		return fmt.Errorf("access denied: current notebook '%s' has no permission", ac.currentNotebook)
	}
	
	// 3. SECURITY: All operations must be scoped to current notebook (if notebook context provided)
	if resourceContext.NotebookName != "" && resourceContext.NotebookName != ac.currentNotebook {
		logging.AuthorizationLogger.Error("SECURITY VIOLATION: Cross-notebook access attempt",
			"tool", toolName,
			"requested_notebook", resourceContext.NotebookName,
			"current_notebook", ac.currentNotebook,
			"security_action", "BLOCKING_CROSS_NOTEBOOK_ACCESS")
		return fmt.Errorf("access denied: cannot access notebook '%s' when '%s' is selected", resourceContext.NotebookName, ac.currentNotebook)
	}
	
	// 4. Check if current notebook permission allows the requested operation
	if !ac.permissionAllowsOperation(ac.currentNotebookPerm, resourceContext.Operation) {
		logging.AuthorizationLogger.Info("Operation not allowed by current notebook permission",
			"tool", toolName,
			"current_notebook", ac.currentNotebook,
			"required_operation", resourceContext.Operation,
			"notebook_permission", ac.currentNotebookPerm)
		return fmt.Errorf("access denied: operation '%s' requires '%s' permission but current notebook has '%s'", 
			resourceContext.Operation, resourceContext.Operation, ac.currentNotebookPerm)
	}
	
	logging.AuthorizationLogger.Debug("Authorization granted",
		"tool", toolName,
		"operation", resourceContext.Operation,
		"current_notebook", ac.currentNotebook,
		"notebook_permission", ac.currentNotebookPerm)
	
	return nil
}




// permissionAllowsOperation checks if a permission level allows a specific operation
func (ac *AuthorizationConfig) permissionAllowsOperation(permission PermissionLevel, operation ToolOperation) bool {
	switch permission {
	case PermissionNone:
		return false
	case PermissionRead:
		return operation == OperationRead
	case PermissionWrite, PermissionFull:
		return true // Write permission allows both read and write
	default:
		return false
	}
}

// FilterNotebooks filters a list of notebooks based on authorization permissions
// Only notebooks with read, write, or full permissions are included
func (ac *AuthorizationConfig) FilterNotebooks(notebooks []map[string]interface{}) []map[string]interface{} {
	if !ac.Enabled {
		logging.AuthorizationLogger.Debug("Notebook filtering skipped - authorization disabled",
			"notebook_count", len(notebooks))
		return notebooks // No filtering when authorization disabled
	}

	logging.AuthorizationLogger.Debug("Starting notebook filtering",
		"notebook_count", len(notebooks),
		"default_permissions", ac.DefaultNotebookPermissions,
		"notebook_permissions_configured", len(ac.NotebookPermissions) > 0)

	var filtered []map[string]interface{}
	var filteredOut []string

	for _, notebook := range notebooks {
		if displayName, ok := notebook["displayName"].(string); ok {
			notebookID, _ := notebook["id"].(string)
			
			// Get permission using exact match or default
			permission := ac.GetNotebookPermission(displayName)
			
			if permission != PermissionNone && permission != "" {
				filtered = append(filtered, notebook)
				logging.AuthorizationLogger.Debug("Notebook allowed by filter",
					"notebook_name", displayName,
					"notebook_id", notebookID,
					"permission", permission)
			} else {
				filteredOut = append(filteredOut, displayName)
				logging.AuthorizationLogger.Debug("Notebook blocked by filter",
					"notebook_name", displayName,
					"notebook_id", notebookID,
					"permission", permission,
					"why_blocked", "permission is 'none' or empty")
			}
		} else {
			logging.AuthorizationLogger.Warn("Notebook missing displayName field, skipping",
				"notebook", notebook)
		}
	}

	// Log comprehensive filtering summary
	if len(filteredOut) > 0 {
		logging.AuthorizationLogger.Info("Filtered out notebooks due to authorization",
			"filtered_count", len(filteredOut),
			"filtered_notebooks", filteredOut,
			"remaining_count", len(filtered))
	}

	logging.AuthorizationLogger.Debug("Notebook filtering completed",
		"original_count", len(notebooks),
		"filtered_count", len(filtered),
		"removed_count", len(filteredOut))

	return filtered
}


// GetNotebookPermission gets the effective permission for a notebook using exact matching
func (ac *AuthorizationConfig) GetNotebookPermission(notebookName string) PermissionLevel {
	// Try exact match first
	if permission, exists := ac.NotebookPermissions[notebookName]; exists {
		return permission
	}

	// Fall back to default
	return ac.DefaultNotebookPermissions
}

// GetNotebookPermissionWithSource gets the permission and source information for a notebook
func (ac *AuthorizationConfig) GetNotebookPermissionWithSource(notebookName string) (PermissionLevel, string, string) {
	// Try exact match first
	if permission, exists := ac.NotebookPermissions[notebookName]; exists {
		return permission, notebookName, "exact"
	}

	// Fall back to default
	return ac.DefaultNotebookPermissions, "", "default"
}


