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

// AuthorizationConfig represents the simplified authorization configuration
type AuthorizationConfig struct {
	Enabled                     bool                          `json:"enabled"`
	DefaultNotebookPermissions  PermissionLevel               `json:"default_notebook_permissions"`  // Default for any notebook
	NotebookPermissions         map[string]PermissionLevel    `json:"notebook_permissions"`          // Specific notebook permissions
	SectionPermissions          map[string]PermissionLevel    `json:"section_permissions"`           // Section permissions with pattern support
	PagePermissions             map[string]PermissionLevel    `json:"page_permissions"`              // Page permissions with pattern support
	
	// Pattern engines for efficient matching (not serialized)
	notebookEngine *PatternEngine `json:"-"`
	sectionEngine  *PatternEngine `json:"-"`
	pageEngine     *PatternEngine `json:"-"`
	
	// Current selected notebook for operation scoping
	currentNotebook     string          `json:"-"`
	currentNotebookPerm PermissionLevel `json:"-"`
}


// ResourceContext contains information about the resource being accessed
type ResourceContext struct {
	NotebookName string
	NotebookID   string
	SectionName  string
	SectionID    string
	PageID       string
	PageName     string
	Operation    ToolOperation
}

// String returns a human-readable representation of the resource context
func (rc ResourceContext) String() string {
	parts := []string{}
	if rc.NotebookName != "" {
		parts = append(parts, fmt.Sprintf("notebook=%s", rc.NotebookName))
	}
	if rc.SectionName != "" {
		parts = append(parts, fmt.Sprintf("section=%s", rc.SectionName))
	}
	if rc.PageName != "" {
		parts = append(parts, fmt.Sprintf("page=%s", rc.PageName))
	} else if rc.PageID != "" {
		parts = append(parts, fmt.Sprintf("pageID=%s", rc.PageID))
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

// NewAuthorizationConfig creates a new authorization configuration with pattern engines
func NewAuthorizationConfig() *AuthorizationConfig {
	return &AuthorizationConfig{
		Enabled:                     false,
		DefaultNotebookPermissions:  PermissionRead,
		NotebookPermissions:         make(map[string]PermissionLevel),
		SectionPermissions:          make(map[string]PermissionLevel),
		PagePermissions:             make(map[string]PermissionLevel),
		notebookEngine:              NewPatternEngine(),
		sectionEngine:               NewPatternEngine(),
		pageEngine:                  NewPatternEngine(),
		currentNotebook:             "",
		currentNotebookPerm:         PermissionNone,
	}
}

// CompilePatterns compiles the permission patterns for efficient matching
func (ac *AuthorizationConfig) CompilePatterns() error {
	// Compile notebook patterns
	if err := ac.notebookEngine.CompilePatterns(ac.NotebookPermissions); err != nil {
		return fmt.Errorf("failed to compile notebook patterns: %v", err)
	}
	
	// Compile section patterns
	if err := ac.sectionEngine.CompilePatterns(ac.SectionPermissions); err != nil {
		return fmt.Errorf("failed to compile section patterns: %v", err)
	}
	
	// Compile page patterns
	if err := ac.pageEngine.CompilePatterns(ac.PagePermissions); err != nil {
		return fmt.Errorf("failed to compile page patterns: %v", err)
	}
	
	logging.AuthorizationLogger.Debug("Authorization patterns compiled successfully",
		"notebook_patterns", len(ac.NotebookPermissions),
		"section_patterns", len(ac.SectionPermissions),
		"page_patterns", len(ac.PagePermissions))
	
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
	
	// Handle empty permission string by defaulting to read permission
	// This can happen if authorization config isn't properly initialized
	if permission == "" {
		logging.AuthorizationLogger.Warn("Empty permission returned, defaulting to read",
			"notebook", notebookName,
			"default_permissions", ac.DefaultNotebookPermissions)
		permission = PermissionRead
	}
	
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

// IsAuthorized checks if a tool call is authorized based on the simplified configuration
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
	
	// 3. SECURITY: All operations must be scoped to current notebook
	if resourceContext.NotebookName != "" && resourceContext.NotebookName != ac.currentNotebook {
		logging.AuthorizationLogger.Error("SECURITY VIOLATION: Cross-notebook access attempt",
			"tool", toolName,
			"requested_notebook", resourceContext.NotebookName,
			"current_notebook", ac.currentNotebook,
			"security_action", "BLOCKING_CROSS_NOTEBOOK_ACCESS")
		return fmt.Errorf("access denied: cannot access notebook '%s' when '%s' is selected", resourceContext.NotebookName, ac.currentNotebook)
	}
	
	// 4. Check resource-level permissions using simplified hierarchy
	resourcePermission := ac.getResourcePermission(resourceContext)
	if !ac.permissionAllowsOperation(resourcePermission, resourceContext.Operation) {
		logging.AuthorizationLogger.Info("Resource access denied",
			"tool", toolName,
			"resource_context", resourceContext.String(),
			"required_operation", resourceContext.Operation,
			"allowed_permission", resourcePermission)
		return fmt.Errorf("access denied: resource requires '%s' permission but only '%s' is granted for %s", 
			resourceContext.Operation, resourcePermission, resourceContext.String())
	}
	
	logging.AuthorizationLogger.Debug("Authorization granted",
		"tool", toolName,
		"operation", resourceContext.Operation,
		"resource_context", resourceContext.String(),
		"resource_permission", resourcePermission)
	
	return nil
}


// getResourcePermission returns the permission level using simplified hierarchical resolution
func (ac *AuthorizationConfig) getResourcePermission(resourceContext ResourceContext) PermissionLevel {
	// SECURITY: For page operations, if we have a page ID but no notebook context,
	// this indicates we couldn't resolve which notebook the page belongs to.
	// This is a security risk, so we deny access (fail-closed security model).
	if resourceContext.PageID != "" && resourceContext.NotebookName == "" {
		logging.AuthorizationLogger.Error("SECURITY: Denying page access - notebook ownership could not be determined",
			"page_id", resourceContext.PageID,
			"page_name", resourceContext.PageName,
			"security_action", "FAIL_CLOSED_ON_UNRESOLVED_OWNERSHIP")
		return PermissionNone
	}
	
	// Use current notebook as fallback if no specific notebook in context
	// (but only for non-page operations or when page notebook was successfully resolved)
	notebookName := resourceContext.NotebookName
	if notebookName == "" {
		notebookName = ac.currentNotebook
	}
	
	// For pages: check page permission -> section permission -> notebook permission
	if resourceContext.PageName != "" {
		// 1. Check page-specific permission
		if pagePermission, _, found := ac.pageEngine.Match(resourceContext.PageName); found {
			return pagePermission
		}
		
		// 2. Check parent section permission
		if resourceContext.SectionName != "" {
			return ac.getSectionPermissionWithFallback(resourceContext.SectionName, notebookName)
		}
		
		// 3. Fall back to notebook permission
		return ac.currentNotebookPerm
	}
	
	// For sections: check section permission -> notebook permission
	if resourceContext.SectionName != "" {
		return ac.getSectionPermissionWithFallback(resourceContext.SectionName, notebookName)
	}
	
	// SECURITY: For section operations, if we have a section ID but no section name,
	// this indicates we couldn't resolve the section name for proper authorization.
	// This is a security risk, so we deny access (fail-closed security model).
	if resourceContext.SectionID != "" && resourceContext.SectionName == "" {
		logging.AuthorizationLogger.Error("SECURITY: Denying section access - section name could not be resolved for authorization",
			"section_id", resourceContext.SectionID,
			"notebook_name", notebookName,
			"security_action", "FAIL_CLOSED_ON_UNRESOLVED_SECTION_NAME")
		return PermissionNone
	}
	
	// For notebook-level operations: use current notebook permission
	return ac.currentNotebookPerm
}

// getSectionPermissionWithFallback gets section permission or falls back to notebook permission
func (ac *AuthorizationConfig) getSectionPermissionWithFallback(sectionName, notebookName string) PermissionLevel {
	// Try direct section name match
	if sectionPermission, _, found := ac.sectionEngine.Match(sectionName); found {
		return sectionPermission
	}
	
	// Try full path match (notebook/section)
	fullPath := notebookName + "/" + sectionName
	if sectionPermission, _, found := ac.sectionEngine.Match(fullPath); found {
		return sectionPermission
	}
	
	// Fall back to notebook permission
	return ac.currentNotebookPerm
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
			
			// Get permission using pattern engine
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

// FilterSections filters a list of sections based on authorization permissions
// Only sections with read, write, or full permissions are included
// Note: Filtering is only performed when operating within the current notebook scope
func (ac *AuthorizationConfig) FilterSections(sections []map[string]interface{}, notebookName string) []map[string]interface{} {
	if !ac.Enabled {
		logging.AuthorizationLogger.Debug("Section filtering skipped - authorization disabled",
			"section_count", len(sections),
			"notebook", notebookName)
		return sections // No filtering when authorization disabled
	}

	// Only filter sections if we're operating within the current notebook
	if notebookName != ac.currentNotebook {
		logging.AuthorizationLogger.Debug("Section filtering skipped - not current notebook",
			"section_count", len(sections),
			"notebook", notebookName,
			"current_notebook", ac.currentNotebook)
		return sections
	}

	logging.AuthorizationLogger.Debug("Starting section filtering",
		"section_count", len(sections),
		"notebook", notebookName,
		"current_notebook_permission", ac.currentNotebookPerm,
		"section_permissions_configured", len(ac.SectionPermissions) > 0)

	var filtered []map[string]interface{}
	var filteredOut []string

	for _, section := range sections {
		if sectionName, ok := section["displayName"].(string); ok {
			sectionID, _ := section["id"].(string)
			sectionType, _ := section["type"].(string)
			
			// Get effective permission using simplified hierarchy
			effectivePermission := ac.getSectionPermissionWithFallback(sectionName, notebookName)

			if effectivePermission != PermissionNone && effectivePermission != "" {
				filtered = append(filtered, section)
				logging.AuthorizationLogger.Debug("Section allowed by filter",
					"section_name", sectionName,
					"section_id", sectionID,
					"section_type", sectionType,
					"notebook", notebookName,
					"effective_permission", effectivePermission)
			} else {
				filteredOut = append(filteredOut, sectionName)
				logging.AuthorizationLogger.Debug("Section blocked by filter",
					"section_name", sectionName,
					"section_id", sectionID,
					"section_type", sectionType,
					"notebook", notebookName,
					"effective_permission", effectivePermission,
					"why_blocked", "effective permission is 'none' or empty")
			}
		} else {
			logging.AuthorizationLogger.Warn("Section missing displayName field, skipping",
				"section", section,
				"notebook", notebookName)
		}
	}

	// Log comprehensive filtering summary
	if len(filteredOut) > 0 {
		logging.AuthorizationLogger.Info("Filtered out sections due to authorization",
			"notebook", notebookName,
			"filtered_count", len(filteredOut),
			"filtered_sections", filteredOut,
			"remaining_count", len(filtered))
	}

	logging.AuthorizationLogger.Debug("Section filtering completed",
		"notebook", notebookName,
		"original_count", len(sections),
		"filtered_count", len(filtered),
		"removed_count", len(filteredOut))

	return filtered
}

// FilterPages filters a list of pages based on authorization permissions
// Only performed for pages within the current notebook scope
func (ac *AuthorizationConfig) FilterPages(pages []map[string]interface{}, sectionID, sectionName, notebookName string) []map[string]interface{} {
	if !ac.Enabled {
		logging.AuthorizationLogger.Debug("Page filtering skipped - authorization disabled",
			"page_count", len(pages),
			"section_id", sectionID,
			"section_name", sectionName,
			"notebook", notebookName)
		return pages // No filtering when authorization disabled
	}

	// Only filter pages if we're operating within the current notebook
	if notebookName != ac.currentNotebook {
		logging.AuthorizationLogger.Debug("Page filtering skipped - not current notebook",
			"page_count", len(pages),
			"section_id", sectionID,
			"section_name", sectionName,
			"notebook", notebookName,
			"current_notebook", ac.currentNotebook)
		return pages
	}

	logging.AuthorizationLogger.Debug("Starting page filtering",
		"page_count", len(pages),
		"section_id", sectionID,
		"section_name", sectionName,
		"notebook", notebookName,
		"page_permissions_configured", len(ac.PagePermissions) > 0)

	// Get section permission as baseline
	sectionPermission := ac.getSectionPermissionWithFallback(sectionName, notebookName)
	if sectionPermission == PermissionNone || sectionPermission == "" {
		// If section is blocked, all pages are blocked
		logging.AuthorizationLogger.Info("All pages blocked due to section permission",
			"section_id", sectionID,
			"section_name", sectionName,
			"notebook", notebookName,
			"section_permission", sectionPermission,
			"filtered_count", len(pages))
		return []map[string]interface{}{}
	}

	// If no page-specific permissions are configured, allow all pages
	if len(ac.PagePermissions) == 0 {
		logging.AuthorizationLogger.Debug("Page filtering completed - all pages allowed (no page permissions defined)",
			"section_id", sectionID,
			"section_name", sectionName,
			"notebook", notebookName,
			"section_permission", sectionPermission,
			"page_count", len(pages))
		return pages
	}

	// Apply page-specific filtering
	var filtered []map[string]interface{}
	var filteredOut []string

	for _, page := range pages {
		pageTitle, hasTitle := page["title"].(string)
		pageID := ""
		// Check both possible field names for page ID
		if id, ok := page["id"].(string); ok {
			pageID = id
		} else if id, ok := page["pageId"].(string); ok {
			pageID = id
		}

		if !hasTitle {
			// Skip pages without titles, but log for debugging
			logging.AuthorizationLogger.Warn("Page missing title field, allowing based on section permission",
				"page", page,
				"section_id", sectionID,
				"section_name", sectionName,
				"notebook", notebookName)
			filtered = append(filtered, page)
			continue
		}

		// Check page-specific permission using pattern engine
		if pagePermission, pattern, found := ac.pageEngine.Match(pageTitle); found {
			if pagePermission != PermissionNone {
				filtered = append(filtered, page)
				logging.AuthorizationLogger.Debug("Page allowed by page pattern",
					"page_title", pageTitle,
					"page_id", pageID,
					"matched_pattern", pattern,
					"page_permission", pagePermission)
			} else {
				filteredOut = append(filteredOut, pageTitle)
				logging.AuthorizationLogger.Debug("Page blocked by page pattern",
					"page_title", pageTitle,
					"page_id", pageID,
					"matched_pattern", pattern,
					"page_permission", pagePermission)
			}
		} else {
			// No page-specific permission, use section permission
			filtered = append(filtered, page)
			logging.AuthorizationLogger.Debug("Page allowed by section fallback",
				"page_title", pageTitle,
				"page_id", pageID,
				"section_permission", sectionPermission)
		}
	}

	// Log comprehensive filtering summary
	if len(filteredOut) > 0 {
		logging.AuthorizationLogger.Info("Filtered out pages by page-level authorization",
			"section_id", sectionID,
			"section_name", sectionName,
			"notebook", notebookName,
			"original_count", len(pages),
			"filtered_count", len(filtered),
			"filtered_out_pages", filteredOut)
	}

	logging.AuthorizationLogger.Debug("Page filtering completed",
		"section_id", sectionID,
		"section_name", sectionName,
		"notebook", notebookName,
		"original_count", len(pages),
		"filtered_count", len(filtered),
		"removed_count", len(filteredOut))

	return filtered
}

// GetNotebookPermission gets the effective permission for a notebook
func (ac *AuthorizationConfig) GetNotebookPermission(notebookName string) PermissionLevel {
	// Try pattern matching with the engine
	if permission, _, found := ac.notebookEngine.Match(notebookName); found {
		return permission
	}

	// Fall back to default
	return ac.DefaultNotebookPermissions
}

// GetNotebookPermissionWithSource gets the permission and source information for a notebook
func (ac *AuthorizationConfig) GetNotebookPermissionWithSource(notebookName string) (PermissionLevel, string, string) {
	// Try pattern matching with the engine
	if permission, pattern, found := ac.notebookEngine.Match(notebookName); found {
		var source string
		if pattern == notebookName {
			source = "exact"
		} else {
			source = "pattern"
		}
		return permission, pattern, source
	}

	// Fall back to default
	return ac.DefaultNotebookPermissions, "", "default"
}


