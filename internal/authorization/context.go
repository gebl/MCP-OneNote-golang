// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package authorization

import (
	"context"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/gebl/onenote-mcp-server/internal/logging"
	"github.com/gebl/onenote-mcp-server/internal/utils"
)

// NotebookCache interface defines the methods needed from the notebook cache
type NotebookCache interface {
	GetNotebook() (map[string]interface{}, bool)
	GetDisplayName() (string, bool)
	GetNotebookID() (string, bool)
	GetSectionName(sectionID string) (string, bool)
	GetSectionNameWithProgress(ctx context.Context, sectionID string, mcpServer interface{}, progressToken string, graphClient interface{}) (string, bool)
	GetPageName(pageID string) (string, bool)
	GetPageNameWithProgress(ctx context.Context, pageID string, mcpServer interface{}, progressToken string, graphClient interface{}) (string, bool)
}

// ExtractResourceContextEnhanced extracts resource context with API fallback for missing cache data
// This function can make API calls to resolve missing section/page names when needed
func ExtractResourceContextEnhanced(ctx context.Context, toolName string, req mcp.CallToolRequest, cache NotebookCache, mcpServer interface{}, graphClient interface{}) ResourceContext {
	progressToken := extractProgressToken(req)
	
	resourceContext := ResourceContext{
		Operation: getToolOperation(toolName),
	}

	logging.AuthorizationLogger.Debug("Extracting resource context with API fallback support",
		"tool", toolName,
		"operation", resourceContext.Operation,
		"has_progress_token", progressToken != "")

	// Get current notebook from cache if available
	if notebook, isSet := cache.GetNotebook(); isSet {
		if notebookID, ok := notebook["id"].(string); ok {
			resourceContext.NotebookID = notebookID
		}
		if displayName, ok := notebook["displayName"].(string); ok {
			resourceContext.NotebookName = displayName
		}

		logging.AuthorizationLogger.Debug("Got notebook from cache",
			"notebook_name", resourceContext.NotebookName,
			"notebook_id", resourceContext.NotebookID)
	} else {
		// Fallback to display name from cache
		if displayName, ok := cache.GetDisplayName(); ok {
			resourceContext.NotebookName = displayName
		}
		if notebookID, ok := cache.GetNotebookID(); ok {
			resourceContext.NotebookID = notebookID
		}
	}

	// Extract resource identifiers from tool parameters with API fallback
	extractParameterContextEnhanced(&resourceContext, ctx, req, cache, toolName, mcpServer, progressToken, graphClient)

	logging.AuthorizationLogger.Debug("Resource context extracted with API fallback",
		"tool", toolName,
		"notebook_name", resourceContext.NotebookName,
		"notebook_id", resourceContext.NotebookID,
		"section_name", resourceContext.SectionName,
		"section_id", resourceContext.SectionID,
		"page_id", resourceContext.PageID,
		"page_name", resourceContext.PageName,
		"operation", resourceContext.Operation)

	return resourceContext
}

// ExtractResourceContext extracts resource context from an MCP tool request
// This function uses cached data whenever possible to avoid expensive API calls
func ExtractResourceContext(toolName string, req mcp.CallToolRequest, cache NotebookCache) ResourceContext {
	context := ResourceContext{
		Operation: getToolOperation(toolName),
	}

	logging.AuthorizationLogger.Debug("Extracting resource context",
		"tool", toolName,
		"operation", context.Operation)

	// Get current notebook from cache if available
	if notebook, isSet := cache.GetNotebook(); isSet {
		if notebookID, ok := notebook["id"].(string); ok {
			context.NotebookID = notebookID
		}
		if displayName, ok := notebook["displayName"].(string); ok {
			context.NotebookName = displayName
		}

		logging.AuthorizationLogger.Debug("Got notebook from cache",
			"notebook_name", context.NotebookName,
			"notebook_id", context.NotebookID)
	} else {
		// Fallback to display name from cache
		if displayName, ok := cache.GetDisplayName(); ok {
			context.NotebookName = displayName
		}
		if notebookID, ok := cache.GetNotebookID(); ok {
			context.NotebookID = notebookID
		}
	}

	// Extract resource identifiers from tool parameters
	extractParameterContext(&context, req, cache, toolName)

	logging.AuthorizationLogger.Debug("Resource context extracted",
		"tool", toolName,
		"notebook_name", context.NotebookName,
		"notebook_id", context.NotebookID,
		"section_name", context.SectionName,
		"section_id", context.SectionID,
		"page_id", context.PageID,
		"page_name", context.PageName,
		"operation", context.Operation)

	return context
}

// extractProgressToken extracts the progress token from the MCP request metadata
func extractProgressToken(req mcp.CallToolRequest) string {
	// Use the centralized utility function
	return utils.ExtractProgressToken(req)
}

// extractParameterContextEnhanced extracts context from tool parameters with API fallback
func extractParameterContextEnhanced(context *ResourceContext, ctx context.Context, req mcp.CallToolRequest, cache NotebookCache, toolName string, mcpServer interface{}, progressToken string, graphClient interface{}) {
	args := req.GetArguments()

	// Handle different parameter types that can identify resources
	if sectionID := getStringParam(args, "sectionID"); sectionID != "" {
		context.SectionID = sectionID
		
		// Try to get section name from cache first
		if sectionName, found := cache.GetSectionName(sectionID); found {
			context.SectionName = sectionName
			logging.AuthorizationLogger.Debug("Extracted section from sectionID parameter (cache hit)",
				"section_id", sectionID,
				"section_name", context.SectionName)
		} else {
			// Cache miss - try API fallback if available
			logging.AuthorizationLogger.Debug("Section name cache miss, attempting API fallback",
				"section_id", sectionID)
			
			if sectionName, found := cache.GetSectionNameWithProgress(ctx, sectionID, mcpServer, progressToken, graphClient); found {
				context.SectionName = sectionName
				logging.AuthorizationLogger.Info("Section name resolved via API fallback",
					"section_id", sectionID,
					"section_name", context.SectionName)
			} else {
				logging.AuthorizationLogger.Warn("Section name could not be resolved via cache or API",
					"section_id", sectionID)
			}
		}
		
		logging.AuthorizationLogger.Debug("Extracted section from sectionID parameter",
			"section_id", sectionID,
			"section_name", context.SectionName)
	}

	if containerID := getStringParam(args, "containerID"); containerID != "" {
		// containerID could be a notebook ID or section group ID
		// If we don't have a notebook context yet, assume it's a notebook
		if context.NotebookID == "" {
			context.NotebookID = containerID
			// Try to resolve notebook name from containerID (as notebook ID)
			if displayName, ok := cache.GetDisplayName(); ok {
				// Check if the cached notebook ID matches
				if notebookID, idOk := cache.GetNotebookID(); idOk && notebookID == containerID {
					context.NotebookName = displayName
					logging.AuthorizationLogger.Debug("Extracted notebook from containerID parameter and resolved name",
						"container_id", containerID,
						"notebook_name", displayName)
				} else {
					logging.AuthorizationLogger.Debug("Extracted notebook from containerID parameter but could not resolve name",
						"container_id", containerID)
				}
			} else {
				logging.AuthorizationLogger.Debug("Extracted notebook from containerID parameter but no cached name available",
					"container_id", containerID)
			}
		} else {
			// If we already have a notebook, this might be a section group
			context.SectionID = containerID
			// Try to resolve section name from containerID (as section ID) with API fallback
			if sectionName, found := cache.GetSectionName(containerID); found {
				context.SectionName = sectionName
				logging.AuthorizationLogger.Debug("Extracted section from containerID parameter and resolved name (cache hit)",
					"container_id", containerID,
					"section_name", sectionName)
			} else {
				// Cache miss - try API fallback
				logging.AuthorizationLogger.Debug("Section name cache miss for containerID, attempting API fallback",
					"container_id", containerID)
				
				if sectionName, found := cache.GetSectionNameWithProgress(ctx, containerID, mcpServer, progressToken, graphClient); found {
					context.SectionName = sectionName
					logging.AuthorizationLogger.Info("Section name resolved via API fallback for containerID",
						"container_id", containerID,
						"section_name", context.SectionName)
				} else {
					logging.AuthorizationLogger.Debug("Extracted section from containerID parameter but could not resolve name",
						"container_id", containerID)
				}
			}
		}
	}

	if pageID := getStringParam(args, "pageID"); pageID != "" {
		context.PageID = pageID
		
		// Try to get page name from cache first
		if pageName, found := cache.GetPageName(pageID); found {
			context.PageName = pageName
			logging.AuthorizationLogger.Debug("Extracted page from pageID parameter (cache hit)",
				"page_id", pageID,
				"page_name", context.PageName)
		} else {
			// Cache miss - try API fallback if available
			logging.AuthorizationLogger.Debug("Page name cache miss, attempting API fallback",
				"page_id", pageID)
			
			if pageName, found := cache.GetPageNameWithProgress(ctx, pageID, mcpServer, progressToken, graphClient); found {
				context.PageName = pageName
				logging.AuthorizationLogger.Info("Page name resolved via API fallback",
					"page_id", pageID,
					"page_name", context.PageName)
			} else {
				logging.AuthorizationLogger.Debug("Extracted page from pageID parameter but could not resolve name from cache",
					"page_id", pageID)
			}
		}
	}

	// Handle other page identification parameters...
	if title := getStringParam(args, "title"); title != "" && context.PageName == "" {
		context.PageName = title
		logging.AuthorizationLogger.Debug("Extracted page name from title parameter",
			"title", title)
	}
}

// extractParameterContext extracts context from tool parameters
func extractParameterContext(context *ResourceContext, req mcp.CallToolRequest, cache NotebookCache, toolName string) {
	args := req.GetArguments()

	// Handle different parameter types that can identify resources
	if sectionID := getStringParam(args, "sectionID"); sectionID != "" {
		context.SectionID = sectionID
		// Try to get section name from cache
		if sectionName, found := cache.GetSectionName(sectionID); found {
			context.SectionName = sectionName
		}
		logging.AuthorizationLogger.Debug("Extracted section from sectionID parameter",
			"section_id", sectionID,
			"section_name", context.SectionName)
	}

	if containerID := getStringParam(args, "containerID"); containerID != "" {
		// containerID could be a notebook ID or section group ID
		// If we don't have a notebook context yet, assume it's a notebook
		if context.NotebookID == "" {
			context.NotebookID = containerID
			// Try to resolve notebook name from containerID (as notebook ID)
			if displayName, ok := cache.GetDisplayName(); ok {
				// Check if the cached notebook ID matches
				if notebookID, idOk := cache.GetNotebookID(); idOk && notebookID == containerID {
					context.NotebookName = displayName
					logging.AuthorizationLogger.Debug("Extracted notebook from containerID parameter and resolved name",
						"container_id", containerID,
						"notebook_name", displayName)
				} else {
					logging.AuthorizationLogger.Debug("Extracted notebook from containerID parameter but could not resolve name",
						"container_id", containerID)
				}
			} else {
				logging.AuthorizationLogger.Debug("Extracted notebook from containerID parameter but no cached name available",
					"container_id", containerID)
			}
		} else {
			// If we already have a notebook, this might be a section group
			context.SectionID = containerID
			// Try to resolve section name from containerID (as section ID)
			if sectionName, found := cache.GetSectionName(containerID); found {
				context.SectionName = sectionName
				logging.AuthorizationLogger.Debug("Extracted section from containerID parameter and resolved name",
					"container_id", containerID,
					"section_name", sectionName)
			} else {
				logging.AuthorizationLogger.Debug("Extracted section from containerID parameter but could not resolve name",
					"container_id", containerID)
			}
		}
	}

	if pageID := getStringParam(args, "pageID"); pageID != "" {
		context.PageID = pageID
		// Try to get page name from cache for authorization checking
		if pageName, found := cache.GetPageName(pageID); found {
			context.PageName = pageName
			logging.AuthorizationLogger.Debug("Extracted page from pageID parameter and resolved name from cache",
				"page_id", pageID,
				"page_name", pageName)
		} else {
			logging.AuthorizationLogger.Debug("Extracted page from pageID parameter but could not resolve name from cache",
				"page_id", pageID)
		}
	}

	if targetSectionID := getStringParam(args, "targetSectionID"); targetSectionID != "" {
		// For operations like copyPage, movePage
		context.SectionID = targetSectionID
		if sectionName, found := cache.GetSectionName(targetSectionID); found {
			context.SectionName = sectionName
		}
		logging.AuthorizationLogger.Debug("Extracted target section from targetSectionID parameter",
			"target_section_id", targetSectionID,
			"section_name", context.SectionName)
	}

	// Handle notebook selection tools
	if identifier := getStringParam(args, "identifier"); identifier != "" && 
		(strings.Contains(toolName, "selectNotebook") || strings.Contains(toolName, "getNotebook")) {
		// This could be a notebook name or ID
		// First assume it's a name
		context.NotebookName = identifier
		
		// But also check if it might be an ID by checking if the current cached notebook ID matches
		if notebookID, ok := cache.GetNotebookID(); ok && notebookID == identifier {
			// The identifier is actually the notebook ID of the currently cached notebook
			if displayName, nameOk := cache.GetDisplayName(); nameOk {
				context.NotebookName = displayName
				context.NotebookID = identifier
				logging.AuthorizationLogger.Debug("Extracted notebook from identifier parameter (detected as ID) and resolved name",
					"identifier", identifier,
					"notebook_name", displayName)
			} else {
				// Keep the identifier as both name and ID
				context.NotebookID = identifier
				logging.AuthorizationLogger.Debug("Extracted notebook from identifier parameter (detected as ID) but could not resolve name",
					"identifier", identifier)
			}
		} else {
			// Assume it's a name, but warn that it could be an ID
			logging.AuthorizationLogger.Debug("Extracted notebook from identifier parameter (assumed as name)",
				"identifier", identifier)
		}
	}

	// Handle quickNote configuration
	if toolName == "quickNote" {
		// quickNote uses configured notebook and page names
		// These will be resolved during the actual operation
		logging.AuthorizationLogger.Debug("QuickNote tool detected, context will be resolved from configuration")
	}
}

// getStringParam safely extracts a string parameter from arguments
func getStringParam(args map[string]interface{}, key string) string {
	if value, exists := args[key]; exists {
		if strValue, ok := value.(string); ok {
			return strValue
		}
	}
	return ""
}

// getToolOperation returns the operation type for a given tool
func getToolOperation(toolName string) ToolOperation {
	if toolInfo, exists := ToolRegistry[toolName]; exists {
		return toolInfo.Operation
	}
	return OperationRead // Default to read operation for unknown tools
}

// ResolveNotebookContext tries to resolve notebook context when only ID is available
// This is a helper function that could be extended to make API calls if needed
func ResolveNotebookContext(context *ResourceContext, cache NotebookCache) {
	if context.NotebookName == "" && context.NotebookID != "" {
		// Try to get notebook name from cache
		if displayName, ok := cache.GetDisplayName(); ok {
			// Check if the cached notebook ID matches
			if notebookID, idOk := cache.GetNotebookID(); idOk && notebookID == context.NotebookID {
				context.NotebookName = displayName
				logging.AuthorizationLogger.Debug("Resolved notebook name from cache",
					"notebook_id", context.NotebookID,
					"notebook_name", context.NotebookName)
			}
		}
	}
}

// ResolveSectionContext tries to resolve section context when only ID is available
// This function attempts to resolve section IDs to section names for proper authorization
// It uses API fallback if the cache lookup fails
func ResolveSectionContext(ctx context.Context, context *ResourceContext, cache NotebookCache) {
	if context.SectionName == "" && context.SectionID != "" {
		// Try to get section name from cache first
		if sectionName, found := cache.GetSectionName(context.SectionID); found {
			context.SectionName = sectionName
			logging.AuthorizationLogger.Debug("Resolved section name from cache",
				"section_id", context.SectionID,
				"section_name", context.SectionName)
		} else {
			// Cache miss - try API fallback if references are available
			logging.AuthorizationLogger.Debug("Section name not found in cache, attempting API fallback for authorization",
				"section_id", context.SectionID)
			
			// Try to get API references from cache if available
			if cacheWithRefs, ok := cache.(interface {
				GetAPIReferences() (interface{}, interface{})
			}); ok {
				graphClient, mcpServer := cacheWithRefs.GetAPIReferences()
				if graphClient != nil {
					progressToken := "" // No progress token in authorization context resolution
					if sectionName, found := cache.GetSectionNameWithProgress(ctx, context.SectionID, mcpServer, progressToken, graphClient); found {
						context.SectionName = sectionName
						logging.AuthorizationLogger.Info("Section name resolved via API fallback for authorization",
							"section_id", context.SectionID,
							"section_name", context.SectionName)
					} else {
						logging.AuthorizationLogger.Warn("Section name could not be resolved via API for authorization",
							"section_id", context.SectionID,
							"authorization_impact", "will fall back to notebook-level permissions")
					}
				} else {
					logging.AuthorizationLogger.Debug("No graph client available for API fallback in authorization",
						"section_id", context.SectionID)
				}
			} else {
				logging.AuthorizationLogger.Debug("Cache does not support API references for authorization fallback",
					"section_id", context.SectionID)
			}
		}
	}
}

// ResolvePageContext tries to resolve page context when only ID is available
// This function attempts to resolve page IDs to page names for proper authorization
func ResolvePageContext(context *ResourceContext, cache NotebookCache) {
	if context.PageName == "" && context.PageID != "" {
		// Try to get page name from cache
		if pageName, found := cache.GetPageName(context.PageID); found {
			context.PageName = pageName
			logging.AuthorizationLogger.Debug("Resolved page name from cache",
				"page_id", context.PageID,
				"page_name", context.PageName)
		} else {
			logging.AuthorizationLogger.Debug("Page name not found in cache for authorization context",
				"page_id", context.PageID)
		}
	}
}

// EnrichQuickNoteContext enriches the context for quickNote operations
// This should be called with the actual configuration values during authorization
func EnrichQuickNoteContext(context *ResourceContext, notebookName, pageName string) {
	if context.NotebookName == "" {
		context.NotebookName = notebookName
	}
	// Set the configured page name for authorization checking
	// Note: We don't set PageID here since quickNote resolves it at runtime
	// But we can set PageName for authorization purposes
	if pageName != "" {
		context.PageName = pageName
	}
	
	logging.AuthorizationLogger.Debug("Enriched quickNote context",
		"configured_notebook", notebookName,
		"configured_page", pageName,
		"context_notebook", context.NotebookName,
		"context_page", context.PageName)
}