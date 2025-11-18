// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

// OneNote MCP Server Section Resources
//
// This file implements MCP (Model Context Protocol) section-related resources for accessing
// Microsoft OneNote sections and section groups through a hierarchical REST-like URI structure.
//
// ## Section Resource URIs
//
// ### Available Section Resource URIs
//
// #### 1. List Sections in a Notebook
// **URI:** `onenote://notebooks/{NotebookDisplayName}/sections`
// **Purpose:** Get hierarchical view of all sections and section groups within a notebook
// **Parameters:**
//   - `{NotebookDisplayName}`: URL-encoded display name of the notebook
// **Returns:** JSON object with hierarchical structure showing sections and section groups
//
// #### 2. List All Sections Across All Notebooks
// **URI:** `onenote://sections`
// **Purpose:** Get all sections across all notebooks using Microsoft Graph global sections endpoint
// **Parameters:** None
// **API Equivalent:** `https://graph.microsoft.com/v1.0/me/onenote/sections?$select=displayName,id`
// **Returns:** JSON object with flat list of all sections with displayName and id fields
//

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/gebl/onenote-mcp-server/internal/authorization"
	"github.com/gebl/onenote-mcp-server/internal/config"
	"github.com/gebl/onenote-mcp-server/internal/graph"
	"github.com/gebl/onenote-mcp-server/internal/logging"
	"github.com/gebl/onenote-mcp-server/internal/notebooks"
	"github.com/gebl/onenote-mcp-server/internal/sections"
)

// registerSectionResources registers all section-related MCP resources
func registerSectionResources(s *mcp.Server, graphClient *graph.Client, cfg *config.Config, authConfig *authorization.AuthorizationConfig, cache authorization.NotebookCache) {
	logging.MainLogger.Debug("Starting section resource registration process")

	// Register sections by notebook name resource template
	logging.MainLogger.Debug("Creating sections by notebook resource template",
		"template_pattern", "onenote://notebooks/{name}/sections",
		"resource_type", "template_resource")
	sectionsTemplate := &mcp.ResourceTemplate{
		URITemplate: "onenote://notebooks/{name}/sections",
		Name:        "OneNote Sections for Notebook",
		Description: "Hierarchical view of sections and section groups within a specific notebook to understand its organizational structure",
		MIMEType:    "application/json",
	}

	sectionsHandler := func(ctx context.Context, request *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		logging.MainLogger.Debug("Resource template handler invoked",
			"template_pattern", "onenote://notebooks/{name}/sections",
			"request_uri", request.Params.URI,
			"handler_type", "notebook_sections")

		// Extract notebook name from URI
		notebookName := extractNotebookNameFromSectionsURI(request.Params.URI)
		if notebookName == "" {
			logging.MainLogger.Error("Invalid notebook name in sections URI",
				"request_uri", request.Params.URI,
				"extracted_name", notebookName)
			return nil, fmt.Errorf("invalid notebook name in URI: %s", request.Params.URI)
		}

		logging.MainLogger.Debug("Extracted notebook name from sections URI",
			"notebook_name", notebookName,
			"request_uri", request.Params.URI)

		// Check authorization for this specific notebook
		if authConfig != nil && authConfig.Enabled {
			if err := authConfig.CheckNotebookPermission(notebookName); err != nil {
				logging.MainLogger.Error("Authorization denied for notebook sections resource",
					"notebook", notebookName,
					"request_uri", request.Params.URI,
					"error", err)
				return nil, err
			}
			logging.MainLogger.Debug("Authorization granted for notebook sections resource",
				"notebook", notebookName,
				"request_uri", request.Params.URI)
		}

		// Call the same logic as getNotebookSections tool with progress support
		jsonData, err := getNotebookSectionsForResource(ctx, s, graphClient, notebookName, cfg)
		if err != nil {
			logging.MainLogger.Error("Failed to get notebook sections for resource",
				"notebook_name", notebookName,
				"error", err)
			return nil, err
		}

		responseSize := len(jsonData)
		logging.MainLogger.Debug("Successfully prepared sections resource response",
			"notebook_name", notebookName,
			"request_uri", request.Params.URI,
			"response_size_bytes", responseSize)

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      request.Params.URI,
					MIMEType: "application/json",
					Text:     string(jsonData),
				},
			},
		}, nil
	}

	s.AddResourceTemplate(sectionsTemplate, sectionsHandler)
	logging.MainLogger.Debug("Registered sections by notebook resource template successfully",
		"template_pattern", "onenote://notebooks/{name}/sections")

	// Register global sections resource - static resource without parameters
	logging.MainLogger.Debug("Creating global sections resource",
		"resource_uri", "onenote://sections",
		"resource_type", "static_resource")
	globalSectionsResource := &mcp.Resource{
		URI:         "onenote://sections",
		Name:        "OneNote All Sections",
		Description: "Get all sections across all notebooks using Microsoft Graph global sections endpoint (/me/onenote/sections?$select=displayName,id)",
		MIMEType:    "application/json",
	}

	globalSectionsHandler := func(ctx context.Context, request *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		logging.MainLogger.Debug("Global sections resource handler invoked",
			"resource_uri", "onenote://sections",
			"request_uri", request.Params.URI,
			"handler_type", "global_sections")

		// Check if authorization is enabled - global sections requires notebook selection
		if authConfig != nil && authConfig.Enabled {
			if cache == nil {
				return nil, fmt.Errorf("access denied: authorization is enabled but no notebook cache available")
			}
			
			currentNotebook, hasNotebook := cache.GetDisplayName()
			if !hasNotebook || currentNotebook == "" {
				logging.MainLogger.Info("Global sections resource requires notebook selection",
					"resource_uri", "onenote://sections",
					"reason", "no_notebook_selected")
				return nil, fmt.Errorf("access denied: no notebook selected. Use selectNotebook tool first, then access sections within that notebook")
			}
			
			logging.MainLogger.Debug("Global sections resource authorized for selected notebook",
				"resource_uri", "onenote://sections",
				"selected_notebook", currentNotebook)
		}

		// If authorization is enabled, get sections from selected notebook instead of global
		var jsonData []byte
		var err error
		if authConfig != nil && authConfig.Enabled && cache != nil {
			currentNotebook, _ := cache.GetDisplayName()
			logging.MainLogger.Debug("Getting sections for selected notebook instead of global",
				"selected_notebook", currentNotebook)
			jsonData, err = getNotebookSectionsForResource(ctx, s, graphClient, currentNotebook, cfg)
		} else {
			// Call the global sections API with progress support
			jsonData, err = getAllSectionsForResource(ctx, s, graphClient, cfg, authConfig)
		}
		if err != nil {
			logging.MainLogger.Error("Failed to get all sections for resource",
				"error", err)
			return nil, err
		}

		responseSize := len(jsonData)
		logging.MainLogger.Debug("Successfully prepared global sections resource response",
			"request_uri", request.Params.URI,
			"response_size_bytes", responseSize)

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      request.Params.URI,
					MIMEType: "application/json",
					Text:     string(jsonData),
				},
			},
		}, nil
	}

	s.AddResource(globalSectionsResource, globalSectionsHandler)
	logging.MainLogger.Debug("Registered global sections resource successfully",
		"resource_uri", "onenote://sections")

	logging.MainLogger.Debug("Section resource registration completed successfully",
		"total_resources", 2,
		"template_resources", 1,
		"static_resources", 1)
}

// Section-specific URI extraction functions

// extractNotebookNameFromSectionsURI extracts the notebook name from a URI like "onenote://notebooks/{name}/sections"
func extractNotebookNameFromSectionsURI(uri string) string {
	// URI format: onenote://notebooks/{name}/sections
	parts := strings.Split(uri, "/")
	if len(parts) >= 5 && parts[0] == "onenote:" && parts[2] == "notebooks" && parts[4] == "sections" {
		// URL decode the notebook name since it's part of a URI
		decodedName, err := url.QueryUnescape(parts[3])
		if err != nil {
			logging.MainLogger.Warn("Failed to URL decode notebook name from sections URI, using raw value",
				"raw_name", parts[3],
				"error", err,
				"uri", uri)
			return parts[3] // fallback to raw value
		}
		logging.MainLogger.Debug("URL decoded notebook name from sections URI",
			"raw_name", parts[3],
			"decoded_name", decodedName,
			"uri", uri)
		return decodedName
	}
	return ""
}

// Shared helper functions used by both notebook and section resources

// getMapKeys returns a slice of all keys in a map[string]interface{} for debugging purposes
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// getNotebookSectionsForResource calls the same logic as getNotebookSections tool with progress support
// This function reuses the logic from NotebookTools.go to get notebook sections with caching and progress notifications
func getNotebookSectionsForResource(ctx context.Context, s *mcp.Server, graphClient *graph.Client, notebookName string, cfg *config.Config) ([]byte, error) {
	logging.MainLogger.Debug("getNotebookSectionsForResource called", "notebook_name", notebookName)

	// Create specialized clients
	notebookClient := notebooks.NewNotebookClient(graphClient)
	sectionClient := sections.NewSectionClient(graphClient)

	// Get the specific notebook by name
	logging.MainLogger.Debug("Getting notebook details for resource",
		"notebook_name", notebookName)
	notebook, err := notebookClient.GetDetailedNotebookByName(notebookName)
	if err != nil {
		logging.MainLogger.Error("Failed to get notebook for resource",
			"notebook_name", notebookName,
			"error", err)
		return nil, fmt.Errorf("failed to get notebook '%s': %v", notebookName, err)
	}

	notebookID, exists := notebook["notebookId"].(string)
	if !exists {
		logging.MainLogger.Error("Notebook ID not found in notebook data",
			"notebook_name", notebookName,
			"notebook_data_keys", getMapKeys(notebook))
		return nil, fmt.Errorf("notebook ID not found for notebook '%s'", notebookName)
	}

	logging.MainLogger.Debug("Retrieved notebook ID for resource",
		"notebook_name", notebookName,
		"notebook_id", notebookID)

	// Extract progress token from request metadata (MCP spec for resources)
	var progressToken string
	// For resources, progress tokens might be provided differently than tools
	// We'll check if there's a way to extract it from the context or request
	logging.MainLogger.Debug("Resource progress support - checking for progress token",
		"notebook_name", notebookName)

	// Import the shared types and functions from NotebookTools.go
	// We need to call the same fetchAllNotebookContentWithProgress function
	// Create progress context using the same keys as NotebookTools.go
	progressCtx := context.WithValue(ctx, mcpServerKey, s)
	progressCtx = context.WithValue(progressCtx, progressTokenKey, progressToken)

	// Use the existing fetchAllNotebookContentWithProgress function from NotebookTools.go
	sectionItems, err := fetchAllNotebookContentWithProgress(sectionClient, notebookID, progressCtx)
	if err != nil {
		logging.MainLogger.Error("Failed to fetch all content for resource",
			"notebook_id", notebookID,
			"notebook_name", notebookName,
			"error", err)
		return nil, fmt.Errorf("failed to fetch sections: %v", err)
	}

	// Get notebook display name from the retrieved notebook data
	var notebookDisplayName string
	if displayName, ok := notebook["displayName"].(string); ok {
		notebookDisplayName = displayName
	} else {
		notebookDisplayName = notebookName
	}

	// Apply authorization filtering if enabled
	if cfg != nil && cfg.Authorization != nil && cfg.Authorization.Enabled {
		logging.MainLogger.Debug("Applying authorization filtering to resource sections",
			"notebook_name", notebookDisplayName,
			"sections_count_before_filtering", len(sectionItems))
		
		// Convert SectionItem slice to []map[string]interface{} for filtering
		var sectionsForFiltering []map[string]interface{}
		for _, item := range sectionItems {
			sectionMap := map[string]interface{}{
				"displayName": item.Name,
				"id":          item.ID,
				"type":        item.Type,
			}
			sectionsForFiltering = append(sectionsForFiltering, sectionMap)
		}
		
		// Apply filtering
		// Note: Section filtering removed - all sections within selected notebook are now accessible
		filteredSections := sectionsForFiltering
		
		// Convert back to SectionItem slice
		var filteredSectionItems []SectionItem
		for _, filteredSection := range filteredSections {
			for _, originalItem := range sectionItems {
				if originalItem.ID == filteredSection["id"].(string) {
					filteredSectionItems = append(filteredSectionItems, originalItem)
					break
				}
			}
		}
		
		sectionItems = filteredSectionItems
		
		logging.MainLogger.Debug("Authorization filtering completed for resource sections",
			"notebook_name", notebookDisplayName,
			"sections_count_after_filtering", len(sectionItems))
	}

	// Build the same response format as getNotebookSections tool
	response := map[string]interface{}{
		"notebook_name":  notebookDisplayName,
		"notebook_id":    notebookID,
		"sections":       sectionItems,
		"sections_count": len(sectionItems),
		"cached":         false, // Resources typically don't use cache
		"source":         "resource_api",
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		logging.MainLogger.Error("Failed to marshal sections for resource",
			"error", err,
			"notebook_name", notebookName)
		return nil, fmt.Errorf("failed to marshal sections: %v", err)
	}

	logging.MainLogger.Debug("Successfully prepared notebook sections for resource",
		"notebook_name", notebookName,
		"notebook_id", notebookID,
		"sections_count", len(sectionItems))

	return jsonData, nil
}

// getAllSectionsForResource fetches all sections across all notebooks using the global sections endpoint
// This function calls the Microsoft Graph API equivalent of https://graph.microsoft.com/v1.0/me/onenote/sections?$select=displayName,id
func getAllSectionsForResource(ctx context.Context, s *mcp.Server, graphClient *graph.Client, cfg *config.Config, authConfig *authorization.AuthorizationConfig) ([]byte, error) {
	logging.MainLogger.Debug("getAllSectionsForResource called")

	// Extract progress token from request metadata (MCP spec for resources)
	var progressToken string
	// For resources, progress tokens might be provided differently than tools
	logging.MainLogger.Debug("Resource progress support - checking for progress token")

	// Send initial progress notification
	if s != nil {
		sendProgressNotification(s, ctx, progressToken, 0, 100, "Starting to fetch all sections...")
	}

	// Create sections client
	sectionClient := sections.NewSectionClient(graphClient)

	// Create progress context
	progressCtx := context.WithValue(ctx, mcpServerKey, s)
	progressCtx = context.WithValue(progressCtx, progressTokenKey, progressToken)

	// Send progress for API call
	if s != nil {
		sendProgressNotification(s, ctx, progressToken, 20, 100, "Calling Microsoft Graph sections API...")
	}

	// Use the new global sections method with context
	sectionsData, err := sectionClient.ListAllSectionsWithContext(progressCtx)
	if err != nil {
		logging.MainLogger.Error("Failed to fetch all sections for resource",
			"error", err)
		return nil, fmt.Errorf("failed to fetch all sections: %v", err)
	}

	// Send progress for processing
	if s != nil {
		sendProgressNotification(s, ctx, progressToken, 70, 100, "Processing sections data...")
	}

	// Apply authorization filtering if enabled
	// Note: For global sections, we can only apply section name-based permissions
	// since we don't have notebook context for each section
	if cfg != nil && cfg.Authorization != nil && cfg.Authorization.Enabled {
		logging.MainLogger.Debug("Applying authorization filtering to global sections resource",
			"sections_count_before_filtering", len(sectionsData))
		
		var filteredSections []map[string]interface{}
		for _, section := range sectionsData {
			// Note: Section filtering removed - all sections are now accessible
			filteredSections = append(filteredSections, section)
		}
		sectionsData = filteredSections
		
		logging.MainLogger.Debug("Authorization filtering completed for global sections resource",
			"sections_count_after_filtering", len(sectionsData))
	}

	// Build response in the same format as the original API
	response := map[string]interface{}{
		"sections":       sectionsData,
		"sections_count": len(sectionsData),
		"source":         "global_sections_api",
		"api_endpoint":   "/me/onenote/sections?$select=displayName,id",
	}

	// Send progress for marshaling
	if s != nil {
		sendProgressNotification(s, ctx, progressToken, 90, 100, "Preparing response...")
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		logging.MainLogger.Error("Failed to marshal all sections for resource",
			"error", err)
		return nil, fmt.Errorf("failed to marshal sections: %v", err)
	}

	// Send final progress notification
	if s != nil {
		sendProgressNotification(s, ctx, progressToken, 100, 100, "Completed fetching all sections")
	}

	logging.MainLogger.Debug("Successfully prepared all sections for resource",
		"sections_count", len(sectionsData))

	return jsonData, nil
}
