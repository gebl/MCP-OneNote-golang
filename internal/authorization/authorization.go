// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package authorization

import (
	"context"
	"fmt"
	"regexp"
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

// ToolCategory represents different categories of tools for authorization
type ToolCategory string

const (
	CategoryAuthTools     ToolCategory = "auth_tools"
	CategoryNotebookRead  ToolCategory = "notebook_read"
	CategoryNotebookWrite ToolCategory = "notebook_write"
	CategoryPageRead      ToolCategory = "page_read"
	CategoryPageWrite     ToolCategory = "page_write"
	CategoryTestTools     ToolCategory = "test_tools"
)

// ToolOperation represents whether a tool performs read or write operations
type ToolOperation string

const (
	OperationRead  ToolOperation = "read"
	OperationWrite ToolOperation = "write"
)

// AuthorizationConfig represents the authorization configuration
type AuthorizationConfig struct {
	Enabled             bool                           `json:"enabled"`
	DefaultMode         PermissionLevel                `json:"default_mode"`          // Global fallback
	DefaultToolMode     PermissionLevel                `json:"default_tool_mode"`     // Default for tool categories
	DefaultNotebookMode PermissionLevel                `json:"default_notebook_mode"` // Default for notebooks
	DefaultSectionMode  PermissionLevel                `json:"default_section_mode"`  // Default for sections
	DefaultPageMode     PermissionLevel                `json:"default_page_mode"`     // Default for pages
	ToolPermissions     map[ToolCategory]PermissionLevel `json:"tool_permissions"`
	NotebookPermissions map[string]PermissionLevel    `json:"notebook_permissions"`
	SectionPermissions  map[string]PermissionLevel    `json:"section_permissions"`
	PagePermissions     map[string]PermissionLevel    `json:"page_permissions"`
	
	// Compiled matchers for performance (not serialized)
	notebookMatchers []PermissionMatcher `json:"-"`
	sectionMatchers  []PermissionMatcher `json:"-"`
	pageMatchers     []PermissionMatcher `json:"-"`
}

// PermissionMatcher represents a compiled permission pattern
type PermissionMatcher struct {
	Pattern    string
	Permission PermissionLevel
	IsExact    bool
	IsPrefix   bool
	IsPath     bool
	Regex      *regexp.Regexp
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
// ToolInfo contains metadata about an MCP tool
type ToolInfo struct {
	Category     ToolCategory
	Operation    ToolOperation
	IsFilterTool bool // True for tools that filter results instead of requiring specific resource access
}

var ToolRegistry = map[string]ToolInfo{
	// Authentication tools
	"getAuthStatus": {CategoryAuthTools, OperationRead, false},
	"refreshToken":  {CategoryAuthTools, OperationWrite, false},
	"initiateAuth":  {CategoryAuthTools, OperationWrite, false},
	"clearAuth":     {CategoryAuthTools, OperationWrite, false},

	// Notebook read tools
	"listNotebooks":       {CategoryNotebookRead, OperationRead, true},  // Filter tool
	"getSelectedNotebook": {CategoryNotebookRead, OperationRead, false},
	"getNotebookSections": {CategoryNotebookRead, OperationRead, true},  // Filter tool

	// Notebook write tools
	"selectNotebook":     {CategoryNotebookWrite, OperationWrite, false},
	"createSection":      {CategoryNotebookWrite, OperationWrite, false},
	"createSectionGroup": {CategoryNotebookWrite, OperationWrite, false},
	"clearCache":         {CategoryNotebookWrite, OperationWrite, false},

	// Page read tools
	"listPages":           {CategoryPageRead, OperationRead, true},  // Filter tool
	"getPageContent":      {CategoryPageRead, OperationRead, false},
	"getPageItemContent":  {CategoryPageRead, OperationRead, false},
	"listPageItems":       {CategoryPageRead, OperationRead, false},

	// Page write tools
	"createPage":                 {CategoryPageWrite, OperationWrite, false},
	"updatePageContent":          {CategoryPageWrite, OperationWrite, false},
	"updatePageContentAdvanced":  {CategoryPageWrite, OperationWrite, false},
	"deletePage":                 {CategoryPageWrite, OperationWrite, false},
	"copyPage":                   {CategoryPageWrite, OperationWrite, false},
	"movePage":                   {CategoryPageWrite, OperationWrite, false},
	"quickNote":                  {CategoryPageWrite, OperationWrite, false},

	// Test tools
	"testProgress": {CategoryTestTools, OperationRead, false},
}

// NewAuthorizationConfig creates a new authorization configuration with compiled matchers
func NewAuthorizationConfig() *AuthorizationConfig {
	return &AuthorizationConfig{
		Enabled:             false,
		DefaultMode:         PermissionRead,    // Global fallback
		DefaultToolMode:     PermissionRead,    // Default for tool categories
		DefaultNotebookMode: PermissionRead,    // Default for notebooks
		DefaultSectionMode:  PermissionRead,    // Default for sections
		DefaultPageMode:     PermissionRead,    // Default for pages
		ToolPermissions:     make(map[ToolCategory]PermissionLevel),
		NotebookPermissions: make(map[string]PermissionLevel),
		SectionPermissions:  make(map[string]PermissionLevel),
		PagePermissions:     make(map[string]PermissionLevel),
		notebookMatchers:    []PermissionMatcher{},
		sectionMatchers:     []PermissionMatcher{},
		pageMatchers:        []PermissionMatcher{},
	}
}

// CompileMatchers compiles the permission patterns for efficient matching
func (ac *AuthorizationConfig) CompileMatchers() error {
	var err error
	
	// Compile notebook matchers
	ac.notebookMatchers, err = compilePermissionMatchers(ac.NotebookPermissions)
	if err != nil {
		return fmt.Errorf("failed to compile notebook matchers: %v", err)
	}
	
	// Compile section matchers
	ac.sectionMatchers, err = compilePermissionMatchers(ac.SectionPermissions)
	if err != nil {
		return fmt.Errorf("failed to compile section matchers: %v", err)
	}
	
	// Compile page matchers
	ac.pageMatchers, err = compilePermissionMatchers(ac.PagePermissions)
	if err != nil {
		return fmt.Errorf("failed to compile page matchers: %v", err)
	}
	
	logging.AuthorizationLogger.Debug("Authorization matchers compiled successfully",
		"notebook_matchers", len(ac.notebookMatchers),
		"section_matchers", len(ac.sectionMatchers),
		"page_matchers", len(ac.pageMatchers))
	
	return nil
}

// compilePermissionMatchers compiles a map of patterns to matchers
func compilePermissionMatchers(permissions map[string]PermissionLevel) ([]PermissionMatcher, error) {
	var matchers []PermissionMatcher
	
	for pattern, permission := range permissions {
		matcher := PermissionMatcher{
			Pattern:    pattern,
			Permission: permission,
		}
		
		// Determine pattern type and compile if needed
		if strings.Contains(pattern, "*") {
			if strings.Contains(pattern, "/") {
				// Path pattern like "*/Draft*" or "Personal Notes/*"
				matcher.IsPath = true
				regexPattern := strings.ReplaceAll(regexp.QuoteMeta(pattern), "\\*", ".*")
				regex, err := regexp.Compile("^" + regexPattern + "$")
				if err != nil {
					return nil, fmt.Errorf("invalid pattern '%s': %v", pattern, err)
				}
				matcher.Regex = regex
			} else if strings.HasSuffix(pattern, "*") {
				// Prefix pattern like "Archive*"
				matcher.IsPrefix = true
			} else {
				// General wildcard pattern
				regexPattern := strings.ReplaceAll(regexp.QuoteMeta(pattern), "\\*", ".*")
				regex, err := regexp.Compile("^" + regexPattern + "$")
				if err != nil {
					return nil, fmt.Errorf("invalid pattern '%s': %v", pattern, err)
				}
				matcher.Regex = regex
			}
		} else {
			// Exact match
			matcher.IsExact = true
		}
		
		matchers = append(matchers, matcher)
	}
	
	return matchers, nil
}

// IsAuthorized checks if a tool call is authorized based on the configuration
func (ac *AuthorizationConfig) IsAuthorized(ctx context.Context, toolName string, req mcp.CallToolRequest, resourceContext ResourceContext) error {
	if !ac.Enabled {
		logging.AuthorizationLogger.Debug("Authorization disabled, allowing all operations", "tool", toolName)
		return nil
	}
	
	logging.AuthorizationLogger.Debug("Checking authorization", 
		"tool", toolName,
		"resource_context", resourceContext.String())
	
	// 1. Get tool info
	toolInfo, exists := ToolRegistry[toolName]
	if !exists {
		logging.AuthorizationLogger.Warn("Unknown tool requested", "tool", toolName)
		return fmt.Errorf("unknown tool: %s", toolName)
	}
	
	// 2. Check tool-level permissions first (fast)
	toolPermission := ac.getToolPermission(toolInfo.Category)
	if !ac.permissionAllowsOperation(toolPermission, toolInfo.Operation) {
		logging.AuthorizationLogger.Info("Tool category access denied",
			"tool", toolName,
			"category", toolInfo.Category,
			"required_operation", toolInfo.Operation,
			"allowed_permission", toolPermission)
		return fmt.Errorf("access denied: tool category '%s' requires '%s' permission but only '%s' is granted", 
			toolInfo.Category, toolInfo.Operation, toolPermission)
	}
	
	// 2.5. Filter tools bypass resource-level checks (they filter results instead)
	if toolInfo.IsFilterTool {
		logging.AuthorizationLogger.Debug("Filter tool bypassing resource-level authorization check",
			"tool", toolName,
			"category", toolInfo.Category,
			"operation", toolInfo.Operation)
		return nil
	}
	
	// 3. Check resource-level permissions (notebook and section)
	resourcePermission := ac.getResourcePermission(resourceContext)
	if !ac.permissionAllowsOperation(resourcePermission, toolInfo.Operation) {
		logging.AuthorizationLogger.Info("Resource access denied",
			"tool", toolName,
			"resource_context", resourceContext.String(),
			"required_operation", toolInfo.Operation,
			"allowed_permission", resourcePermission)
		return fmt.Errorf("access denied: resource requires '%s' permission but only '%s' is granted for %s", 
			toolInfo.Operation, resourcePermission, resourceContext.String())
	}
	
	logging.AuthorizationLogger.Debug("Authorization granted",
		"tool", toolName,
		"category", toolInfo.Category,
		"operation", toolInfo.Operation,
		"resource_context", resourceContext.String())
	
	return nil
}

// getToolPermission returns the permission level for a tool category
func (ac *AuthorizationConfig) getToolPermission(category ToolCategory) PermissionLevel {
	if permission, exists := ac.ToolPermissions[category]; exists {
		return permission
	}
	// Use tool-specific default if available, otherwise global default
	if ac.DefaultToolMode != "" {
		return ac.DefaultToolMode
	}
	return ac.DefaultMode
}

// getResourcePermission returns the permission level for a resource context
func (ac *AuthorizationConfig) getResourcePermission(resourceContext ResourceContext) PermissionLevel {
	// Check page permissions first (most specific)
	if resourceContext.PageName != "" {
		if permission := ac.matchPattern(resourceContext.PageName, ac.pageMatchers); permission != "" {
			logging.AuthorizationLogger.Debug("Page permission found via pattern match",
				"page_name", resourceContext.PageName,
				"page_id", resourceContext.PageID,
				"permission", permission)
			return permission
		}
		// No page-specific permission found, use page default
		if ac.DefaultPageMode != "" {
			logging.AuthorizationLogger.Debug("No page-specific permission found, applying default page mode",
				"page_name", resourceContext.PageName,
				"page_id", resourceContext.PageID,
				"default_page_mode", ac.DefaultPageMode)
			return ac.DefaultPageMode
		}
	} else if resourceContext.PageID != "" {
		// We have a page ID but no page name - this means we couldn't resolve the page name
		// Apply the default page mode to prevent authorization bypass
		logging.AuthorizationLogger.Info("Page ID present but page name could not be resolved, applying default page mode to prevent bypass",
			"page_id", resourceContext.PageID,
			"default_page_mode", ac.DefaultPageMode,
			"security_action", "preventing_page_id_authorization_bypass")
		if ac.DefaultPageMode != "" {
			return ac.DefaultPageMode
		}
	}
	
	// Check section permissions next (more specific than notebook)
	if resourceContext.SectionName != "" {
		sectionPath := resourceContext.NotebookName + "/" + resourceContext.SectionName
		if permission := ac.matchPattern(sectionPath, ac.sectionMatchers); permission != "" {
			return permission
		}
		
		// Also check section name alone
		if permission := ac.matchPattern(resourceContext.SectionName, ac.sectionMatchers); permission != "" {
			return permission
		}
		// No section-specific permission found, use section default
		if ac.DefaultSectionMode != "" {
			return ac.DefaultSectionMode
		}
	}
	
	// Check notebook permissions
	if resourceContext.NotebookName != "" {
		if permission := ac.matchPattern(resourceContext.NotebookName, ac.notebookMatchers); permission != "" {
			return permission
		}
		// No notebook-specific permission found, use notebook default
		if ac.DefaultNotebookMode != "" {
			return ac.DefaultNotebookMode
		}
	}
	
	// Fall back to global default
	return ac.DefaultMode
}

// matchPattern matches a string against compiled matchers
func (ac *AuthorizationConfig) matchPattern(value string, matchers []PermissionMatcher) PermissionLevel {
	for _, matcher := range matchers {
		var matches bool
		
		if matcher.IsExact {
			matches = value == matcher.Pattern
		} else if matcher.IsPrefix {
			prefix := strings.TrimSuffix(matcher.Pattern, "*")
			matches = strings.HasPrefix(value, prefix)
		} else if matcher.Regex != nil {
			matches = matcher.Regex.MatchString(value)
		}
		
		if matches {
			logging.AuthorizationLogger.Debug("Pattern matched",
				"value", value,
				"pattern", matcher.Pattern,
				"permission", matcher.Permission,
				"match_type", ac.getMatchType(matcher))
			return matcher.Permission
		}
	}
	
	return ""
}

// getMatchType returns a string describing the match type for logging
func (ac *AuthorizationConfig) getMatchType(matcher PermissionMatcher) string {
	if matcher.IsExact {
		return "exact"
	} else if matcher.IsPrefix {
		return "prefix"
	} else if matcher.IsPath {
		return "path"
	}
	return "regex"
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
		"default_mode", ac.DefaultMode,
		"notebook_permissions_configured", len(ac.NotebookPermissions) > 0)

	var filtered []map[string]interface{}
	var filteredOut []string
	var filterDecisions []map[string]interface{}

	for _, notebook := range notebooks {
		if displayName, ok := notebook["displayName"].(string); ok {
			notebookID, _ := notebook["id"].(string)
			
			// Get permission with detailed decision tracking
			permission := ac.getNotebookPermission(displayName)
			
			// Log detailed decision reasoning
			decision := map[string]interface{}{
				"notebook_name": displayName,
				"notebook_id":   notebookID,
				"permission":    permission,
				"allowed":       permission != PermissionNone && permission != "",
			}
			
			// Determine why this permission was assigned
			var reason string
			var matchedPattern string
			
			// Check exact match first
			if exactPermission, exists := ac.NotebookPermissions[displayName]; exists {
				reason = "exact_match"
				matchedPattern = displayName
				decision["reason"] = reason
				decision["matched_pattern"] = matchedPattern
				decision["exact_permission"] = exactPermission
			} else {
				// Check pattern matching
				patternPermission := ac.matchPattern(displayName, ac.notebookMatchers)
				if patternPermission != "" {
					reason = "pattern_match"
					// Find which pattern matched
					for _, matcher := range ac.notebookMatchers {
						var matches bool
						if matcher.IsExact {
							matches = displayName == matcher.Pattern
						} else if matcher.IsPrefix {
							prefix := strings.TrimSuffix(matcher.Pattern, "*")
							matches = strings.HasPrefix(displayName, prefix)
						} else if matcher.Regex != nil {
							matches = matcher.Regex.MatchString(displayName)
						}
						if matches {
							matchedPattern = matcher.Pattern
							break
						}
					}
					decision["reason"] = reason
					decision["matched_pattern"] = matchedPattern
					decision["pattern_permission"] = patternPermission
				} else {
					reason = "default_fallback"
					decision["reason"] = reason
					decision["default_permission"] = ac.DefaultMode
				}
			}
			
			filterDecisions = append(filterDecisions, decision)
			
			if permission != PermissionNone && permission != "" {
				filtered = append(filtered, notebook)
				logging.AuthorizationLogger.Debug("Notebook allowed by filter",
					"notebook_name", displayName,
					"notebook_id", notebookID,
					"permission", permission,
					"reason", reason,
					"matched_pattern", matchedPattern)
			} else {
				filteredOut = append(filteredOut, displayName)
				logging.AuthorizationLogger.Debug("Notebook blocked by filter",
					"notebook_name", displayName,
					"notebook_id", notebookID,
					"permission", permission,
					"reason", reason,
					"matched_pattern", matchedPattern,
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
		
		// Log detailed decisions for filtered out notebooks
		for _, decision := range filterDecisions {
			if !decision["allowed"].(bool) {
				logging.AuthorizationLogger.Info("Notebook filter decision (BLOCKED)",
					"notebook_name", decision["notebook_name"],
					"notebook_id", decision["notebook_id"],
					"permission", decision["permission"],
					"reason", decision["reason"],
					"matched_pattern", decision["matched_pattern"])
			}
		}
	}

	// Log decisions for allowed notebooks at debug level
	for _, decision := range filterDecisions {
		if decision["allowed"].(bool) {
			logging.AuthorizationLogger.Debug("Notebook filter decision (ALLOWED)",
				"notebook_name", decision["notebook_name"],
				"notebook_id", decision["notebook_id"],
				"permission", decision["permission"],
				"reason", decision["reason"],
				"matched_pattern", decision["matched_pattern"])
		}
	}

	logging.AuthorizationLogger.Debug("Notebook filtering completed",
		"original_count", len(notebooks),
		"filtered_count", len(filtered),
		"removed_count", len(filteredOut),
		"decisions_tracked", len(filterDecisions))

	return filtered
}

// FilterSections filters a list of sections based on authorization permissions
// Only sections with read, write, or full permissions are included
func (ac *AuthorizationConfig) FilterSections(sections []map[string]interface{}, notebookName string) []map[string]interface{} {
	if !ac.Enabled {
		logging.AuthorizationLogger.Debug("Section filtering skipped - authorization disabled",
			"section_count", len(sections),
			"notebook", notebookName)
		return sections // No filtering when authorization disabled
	}

	logging.AuthorizationLogger.Debug("Starting section filtering",
		"section_count", len(sections),
		"notebook", notebookName,
		"default_mode", ac.DefaultMode,
		"section_permissions_configured", len(ac.SectionPermissions) > 0,
		"notebook_permissions_configured", len(ac.NotebookPermissions) > 0)

	var filtered []map[string]interface{}
	var filteredOut []string
	var filterDecisions []map[string]interface{}

	// Get notebook-level permission for context
	notebookPermission := ac.getNotebookPermission(notebookName)
	logging.AuthorizationLogger.Debug("Notebook permission context for section filtering",
		"notebook", notebookName,
		"notebook_permission", notebookPermission)

	for _, section := range sections {
		if sectionName, ok := section["displayName"].(string); ok {
			sectionID, _ := section["id"].(string)
			sectionType, _ := section["type"].(string)
			
			// Check section-specific permission first
			sectionPath := notebookName + "/" + sectionName
			sectionPermission := ac.getSectionPermission(sectionPath)
			
			// If no section-specific rule, fall back to notebook permission
			var effectivePermission PermissionLevel
			var permissionSource string
			var matchedPattern string
			
			if sectionPermission != "" {
				effectivePermission = sectionPermission
				permissionSource = "section_specific"
				
				// Find which section pattern matched
				if exactPermission, exists := ac.SectionPermissions[sectionPath]; exists {
					matchedPattern = sectionPath + " (exact)"
					logging.AuthorizationLogger.Debug("Section exact match found",
						"section_path", sectionPath,
						"exact_permission", exactPermission)
				} else if exactPermission, exists := ac.SectionPermissions[sectionName]; exists {
					matchedPattern = sectionName + " (exact)"
					logging.AuthorizationLogger.Debug("Section name exact match found",
						"section_name", sectionName,
						"exact_permission", exactPermission)
				} else {
					// Check patterns
					for _, matcher := range ac.sectionMatchers {
						var matches bool
						var testValue string
						
						// Test both full path and section name
						for _, tv := range []string{sectionPath, sectionName} {
							if matcher.IsExact {
								matches = tv == matcher.Pattern
							} else if matcher.IsPrefix {
								prefix := strings.TrimSuffix(matcher.Pattern, "*")
								matches = strings.HasPrefix(tv, prefix)
							} else if matcher.Regex != nil {
								matches = matcher.Regex.MatchString(tv)
							}
							if matches {
								testValue = tv
								break
							}
						}
						
						if matches {
							matchedPattern = matcher.Pattern + " (pattern on " + testValue + ")"
							logging.AuthorizationLogger.Debug("Section pattern match found",
								"section_path", sectionPath,
								"section_name", sectionName,
								"matched_pattern", matcher.Pattern,
								"test_value", testValue,
								"pattern_permission", matcher.Permission)
							break
						}
					}
				}
			} else {
				effectivePermission = notebookPermission
				permissionSource = "notebook_fallback"
				matchedPattern = "inherited from notebook: " + notebookName
			}
			
			// Create detailed decision record
			decision := map[string]interface{}{
				"section_name":         sectionName,
				"section_id":          sectionID,
				"section_type":        sectionType,
				"section_path":        sectionPath,
				"notebook":            notebookName,
				"section_permission":  sectionPermission,
				"notebook_permission": notebookPermission,
				"effective_permission": effectivePermission,
				"permission_source":   permissionSource,
				"matched_pattern":     matchedPattern,
				"allowed":             effectivePermission != PermissionNone && effectivePermission != "",
			}
			
			filterDecisions = append(filterDecisions, decision)

			if effectivePermission != PermissionNone && effectivePermission != "" {
				filtered = append(filtered, section)
				logging.AuthorizationLogger.Debug("Section allowed by filter",
					"section_name", sectionName,
					"section_id", sectionID,
					"section_type", sectionType,
					"section_path", sectionPath,
					"notebook", notebookName,
					"effective_permission", effectivePermission,
					"permission_source", permissionSource,
					"matched_pattern", matchedPattern)
			} else {
				filteredOut = append(filteredOut, sectionName)
				logging.AuthorizationLogger.Debug("Section blocked by filter",
					"section_name", sectionName,
					"section_id", sectionID,
					"section_type", sectionType,
					"section_path", sectionPath,
					"notebook", notebookName,
					"effective_permission", effectivePermission,
					"permission_source", permissionSource,
					"matched_pattern", matchedPattern,
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
		
		// Log detailed decisions for filtered out sections
		for _, decision := range filterDecisions {
			if !decision["allowed"].(bool) {
				logging.AuthorizationLogger.Info("Section filter decision (BLOCKED)",
					"section_name", decision["section_name"],
					"section_id", decision["section_id"],
					"section_path", decision["section_path"],
					"notebook", decision["notebook"],
					"effective_permission", decision["effective_permission"],
					"permission_source", decision["permission_source"],
					"matched_pattern", decision["matched_pattern"])
			}
		}
	}

	// Log decisions for allowed sections at debug level
	for _, decision := range filterDecisions {
		if decision["allowed"].(bool) {
			logging.AuthorizationLogger.Debug("Section filter decision (ALLOWED)",
				"section_name", decision["section_name"],
				"section_id", decision["section_id"],
				"section_path", decision["section_path"],
				"notebook", decision["notebook"],
				"effective_permission", decision["effective_permission"],
				"permission_source", decision["permission_source"],
				"matched_pattern", decision["matched_pattern"])
		}
	}

	logging.AuthorizationLogger.Debug("Section filtering completed",
		"notebook", notebookName,
		"original_count", len(sections),
		"filtered_count", len(filtered),
		"removed_count", len(filteredOut),
		"decisions_tracked", len(filterDecisions))

	return filtered
}

// FilterPages filters a list of pages based on authorization permissions
// Uses the section permission for the containing section
func (ac *AuthorizationConfig) FilterPages(pages []map[string]interface{}, sectionID, sectionName, notebookName string) []map[string]interface{} {
	if !ac.Enabled {
		logging.AuthorizationLogger.Debug("Page filtering skipped - authorization disabled",
			"page_count", len(pages),
			"section_id", sectionID,
			"section_name", sectionName,
			"notebook", notebookName)
		return pages // No filtering when authorization disabled
	}

	logging.AuthorizationLogger.Debug("Starting page filtering",
		"page_count", len(pages),
		"section_id", sectionID,
		"section_name", sectionName,
		"notebook", notebookName,
		"default_mode", ac.DefaultMode,
		"page_permissions_configured", len(ac.PagePermissions) > 0,
		"section_permissions_configured", len(ac.SectionPermissions) > 0,
		"notebook_permissions_configured", len(ac.NotebookPermissions) > 0)

	// Determine effective permission for this section
	var effectivePermission PermissionLevel
	var permissionSource string
	var contextInfo string
	
	if sectionName != "" && notebookName != "" {
		sectionPath := notebookName + "/" + sectionName
		sectionPermission := ac.getSectionPermission(sectionPath)
		if sectionPermission != "" {
			effectivePermission = sectionPermission
			permissionSource = "section_specific"
			contextInfo = fmt.Sprintf("section permission for path '%s'", sectionPath)
		} else {
			effectivePermission = ac.getNotebookPermission(notebookName)
			permissionSource = "notebook_fallback"
			contextInfo = fmt.Sprintf("inherited from notebook '%s'", notebookName)
		}
	} else {
		// Fallback to default mode if we can't determine context
		effectivePermission = ac.DefaultMode
		permissionSource = "default_fallback"
		contextInfo = "missing section/notebook context, using default mode"
	}

	logging.AuthorizationLogger.Debug("Page filtering context determined",
		"section_id", sectionID,
		"section_name", sectionName,
		"notebook", notebookName,
		"effective_permission", effectivePermission,
		"permission_source", permissionSource,
		"context_info", contextInfo)

	// If no permission at section/notebook level, filter out all pages
	if effectivePermission == PermissionNone || effectivePermission == "" {
		if len(pages) > 0 {
			var pageNames []string
			var pageDecisions []map[string]interface{}
			
			for _, page := range pages {
				pageTitle := "Unknown"
				pageID := "Unknown"
				if title, ok := page["title"].(string); ok {
					pageTitle = title
					pageNames = append(pageNames, title)
				}
				// Check both possible field names for page ID
				if id, ok := page["id"].(string); ok {
					pageID = id
				} else if id, ok := page["pageId"].(string); ok {
					pageID = id
				}
				
				decision := map[string]interface{}{
					"page_title":           pageTitle,
					"page_id":             pageID,
					"section_id":          sectionID,
					"section_name":        sectionName,
					"notebook":            notebookName,
					"effective_permission": effectivePermission,
					"permission_source":   permissionSource,
					"allowed":             false,
					"block_reason":        "section/notebook permission is 'none' or empty",
				}
				pageDecisions = append(pageDecisions, decision)
				
				logging.AuthorizationLogger.Debug("Page blocked by section/notebook filter",
					"page_title", pageTitle,
					"page_id", pageID,
					"section_id", sectionID,
					"section_name", sectionName,
					"notebook", notebookName,
					"effective_permission", effectivePermission,
					"permission_source", permissionSource,
					"why_blocked", "section/notebook permission is 'none' or empty")
			}
			
			logging.AuthorizationLogger.Info("Filtered out all pages due to section/notebook authorization",
				"section_id", sectionID,
				"section_name", sectionName,
				"notebook", notebookName,
				"effective_permission", effectivePermission,
				"permission_source", permissionSource,
				"filtered_count", len(pages),
				"filtered_pages", pageNames)
			
			// Log each blocked page decision at info level
			for _, decision := range pageDecisions {
				logging.AuthorizationLogger.Info("Page filter decision (BLOCKED)",
					"page_title", decision["page_title"],
					"page_id", decision["page_id"],
					"section_id", decision["section_id"],
					"section_name", decision["section_name"],
					"notebook", decision["notebook"],
					"effective_permission", decision["effective_permission"],
					"permission_source", decision["permission_source"],
					"block_reason", decision["block_reason"])
			}
		}
		return []map[string]interface{}{}
	}

	// Check individual page permissions if page permissions are defined
	if len(ac.PagePermissions) > 0 {
		logging.AuthorizationLogger.Debug("Applying individual page-level permissions",
			"page_count", len(pages),
			"page_permissions_count", len(ac.PagePermissions))
		
		var filteredPages []map[string]interface{}
		var filteredOutPages []string
		var filterDecisions []map[string]interface{}
		
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
				logging.AuthorizationLogger.Warn("Page missing title field, skipping page-level authorization",
					"page", page,
					"section_id", sectionID,
					"section_name", sectionName,
					"notebook", notebookName)
				continue
			}
			
			// Check page-specific permission
			pagePermission := ac.getPagePermission(pageTitle)
			var pagePermissionSource string
			var matchedPagePattern string
			
			// Determine why this page permission was assigned
			if exactPermission, exists := ac.PagePermissions[pageTitle]; exists {
				pagePermissionSource = "exact_match"
				matchedPagePattern = pageTitle + " (exact)"
				logging.AuthorizationLogger.Debug("Page exact match found",
					"page_title", pageTitle,
					"exact_permission", exactPermission)
			} else {
				// Check patterns
				patternPermission := ac.matchPattern(pageTitle, ac.pageMatchers)
				if patternPermission != "" {
					pagePermissionSource = "pattern_match"
					// Find which pattern matched
					for _, matcher := range ac.pageMatchers {
						var matches bool
						if matcher.IsExact {
							matches = pageTitle == matcher.Pattern
						} else if matcher.IsPrefix {
							prefix := strings.TrimSuffix(matcher.Pattern, "*")
							matches = strings.HasPrefix(pageTitle, prefix)
						} else if matcher.Regex != nil {
							matches = matcher.Regex.MatchString(pageTitle)
						}
						if matches {
							matchedPagePattern = matcher.Pattern + " (pattern)"
							break
						}
					}
				} else {
					pagePermissionSource = "default_fallback"
					matchedPagePattern = "using default mode"
				}
			}
			
			decision := map[string]interface{}{
				"page_title":           pageTitle,
				"page_id":             pageID,
				"section_id":          sectionID,
				"section_name":        sectionName,
				"notebook":            notebookName,
				"page_permission":     pagePermission,
				"page_permission_source": pagePermissionSource,
				"matched_page_pattern": matchedPagePattern,
				"section_permission":  effectivePermission,
				"permission_source":   permissionSource,
				"allowed":             pagePermission != PermissionNone,
			}
			filterDecisions = append(filterDecisions, decision)
			
			if pagePermission != PermissionNone {
				filteredPages = append(filteredPages, page)
				logging.AuthorizationLogger.Debug("Page allowed by page permissions",
					"page_title", pageTitle,
					"page_id", pageID,
					"page_permission", pagePermission,
					"page_permission_source", pagePermissionSource,
					"matched_page_pattern", matchedPagePattern,
					"section_id", sectionID,
					"section_name", sectionName,
					"notebook", notebookName)
			} else {
				filteredOutPages = append(filteredOutPages, pageTitle)
				logging.AuthorizationLogger.Debug("Page blocked by page permissions",
					"page_title", pageTitle,
					"page_id", pageID,
					"page_permission", pagePermission,
					"page_permission_source", pagePermissionSource,
					"matched_page_pattern", matchedPagePattern,
					"section_id", sectionID,
					"section_name", sectionName,
					"notebook", notebookName,
					"why_blocked", "page permission is 'none'")
			}
		}
		
		if len(filteredOutPages) > 0 {
			logging.AuthorizationLogger.Info("Filtered out pages by page-level authorization",
				"section_id", sectionID,
				"section_name", sectionName,
				"notebook", notebookName,
				"original_count", len(pages),
				"filtered_count", len(filteredPages),
				"filtered_out_pages", filteredOutPages)
			
			// Log detailed decisions for filtered out pages
			for _, decision := range filterDecisions {
				if !decision["allowed"].(bool) {
					logging.AuthorizationLogger.Info("Page filter decision (BLOCKED)",
						"page_title", decision["page_title"],
						"page_id", decision["page_id"],
						"section_id", decision["section_id"],
						"section_name", decision["section_name"],
						"notebook", decision["notebook"],
						"page_permission", decision["page_permission"],
						"page_permission_source", decision["page_permission_source"],
						"matched_page_pattern", decision["matched_page_pattern"])
				}
			}
		}
		
		// Log decisions for allowed pages at debug level
		for _, decision := range filterDecisions {
			if decision["allowed"].(bool) {
				logging.AuthorizationLogger.Debug("Page filter decision (ALLOWED)",
					"page_title", decision["page_title"],
					"page_id", decision["page_id"],
					"section_id", decision["section_id"],
					"section_name", decision["section_name"],
					"notebook", decision["notebook"],
					"page_permission", decision["page_permission"],
					"page_permission_source", decision["page_permission_source"],
					"matched_page_pattern", decision["matched_page_pattern"])
			}
		}
		
		logging.AuthorizationLogger.Debug("Page filtering completed with individual page permissions",
			"section_id", sectionID,
			"section_name", sectionName,
			"notebook", notebookName,
			"original_count", len(pages),
			"filtered_count", len(filteredPages),
			"removed_count", len(filteredOutPages),
			"decisions_tracked", len(filterDecisions))
		
		return filteredPages
	}

	// No page permissions defined - allow all pages based on section/notebook permission
	var allowedDecisions []map[string]interface{}
	for _, page := range pages {
		pageTitle := "Unknown"
		pageID := "Unknown"
		if title, ok := page["title"].(string); ok {
			pageTitle = title
		}
		// Check both possible field names for page ID
		if id, ok := page["id"].(string); ok {
			pageID = id
		} else if id, ok := page["pageId"].(string); ok {
			pageID = id
		}
		
		decision := map[string]interface{}{
			"page_title":           pageTitle,
			"page_id":             pageID,
			"section_id":          sectionID,
			"section_name":        sectionName,
			"notebook":            notebookName,
			"effective_permission": effectivePermission,
			"permission_source":   permissionSource,
			"allowed":             true,
			"allow_reason":        "no page permissions defined, using section/notebook permission",
		}
		allowedDecisions = append(allowedDecisions, decision)
		
		logging.AuthorizationLogger.Debug("Page allowed by section/notebook permission",
			"page_title", pageTitle,
			"page_id", pageID,
			"section_id", sectionID,
			"section_name", sectionName,
			"notebook", notebookName,
			"effective_permission", effectivePermission,
			"permission_source", permissionSource,
			"reason", "no page permissions defined")
	}

	logging.AuthorizationLogger.Debug("Page filtering completed - all pages allowed (no page permissions defined)",
		"section_id", sectionID,
		"section_name", sectionName,
		"notebook", notebookName,
		"effective_permission", effectivePermission,
		"permission_source", permissionSource,
		"page_count", len(pages),
		"decisions_tracked", len(allowedDecisions))

	return pages
}

// getNotebookPermission gets the effective permission for a notebook
func (ac *AuthorizationConfig) getNotebookPermission(notebookName string) PermissionLevel {
	// Try exact match first
	if permission, exists := ac.NotebookPermissions[notebookName]; exists {
		return permission
	}

	// Try pattern matching
	permission := ac.matchPattern(notebookName, ac.notebookMatchers)
	if permission != "" {
		return permission
	}

	// Fall back to notebook default, then global default
	if ac.DefaultNotebookMode != "" {
		return ac.DefaultNotebookMode
	}
	return ac.DefaultMode
}

// getSectionPermission gets the effective permission for a section
func (ac *AuthorizationConfig) getSectionPermission(sectionPath string) PermissionLevel {
	// Try exact match first
	if permission, exists := ac.SectionPermissions[sectionPath]; exists {
		return permission
	}

	// Try pattern matching
	permission := ac.matchPattern(sectionPath, ac.sectionMatchers)
	if permission != "" {
		return permission
	}
	
	// Fall back to section default (but don't fall back to global default for sections,
	// let the caller handle notebook/global fallback)
	return ac.DefaultSectionMode
}

// getPagePermission gets the effective permission for a page
func (ac *AuthorizationConfig) getPagePermission(pageName string) PermissionLevel {
	// Try exact match first
	if permission, exists := ac.PagePermissions[pageName]; exists {
		return permission
	}

	// Try pattern matching
	permission := ac.matchPattern(pageName, ac.pageMatchers)
	if permission != "" {
		return permission
	}

	// Fall back to page default, then global default
	if ac.DefaultPageMode != "" {
		return ac.DefaultPageMode
	}
	return ac.DefaultMode
}