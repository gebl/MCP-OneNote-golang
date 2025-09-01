// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package authorization

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/gebl/onenote-mcp-server/internal/logging"
)

// NotebookCache interface defines the methods needed from the notebook cache
type NotebookCache interface {
	GetNotebook() (map[string]interface{}, bool)
	GetDisplayName() (string, bool)
}

// ExtractResourceContext extracts simplified resource context from tool call
func ExtractResourceContext(ctx context.Context, toolName string, req mcp.CallToolRequest, cache NotebookCache) ResourceContext {
	resourceContext := ResourceContext{
		Operation: getToolOperation(toolName),
	}

	logging.AuthorizationLogger.Debug("Extracting simplified resource context",
		"tool", toolName,
		"operation", resourceContext.Operation)

	// Get current notebook name from cache if available
	if displayName, hasName := cache.GetDisplayName(); hasName {
		resourceContext.NotebookName = displayName
		logging.AuthorizationLogger.Debug("Got notebook from cache",
			"notebook_name", resourceContext.NotebookName)
	}

	logging.AuthorizationLogger.Debug("Resource context extracted",
		"tool", toolName,
		"notebook_name", resourceContext.NotebookName,
		"operation", resourceContext.Operation)

	return resourceContext
}

// getToolOperation determines if a tool performs read or write operations
func getToolOperation(toolName string) ToolOperation {
	switch toolName {
	// Read-only operations
	case "getAuthStatus", "listNotebooks", "listSections", "listPages", 
		 "getPageContent", "getPageMetadata", "searchPages":
		return OperationRead
	
	// Write operations
	case "selectNotebook", "createSection", "updateSection", "deleteSection",
		 "createPage", "updatePageContent", "updatePageContentAdvanced", 
		 "deletePage", "quickNote":
		return OperationWrite
	
	// Default to read for unknown tools (safe default)
	default:
		logging.AuthorizationLogger.Debug("Unknown tool, defaulting to read operation", "tool", toolName)
		return OperationRead
	}
}