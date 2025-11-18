// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/gebl/onenote-mcp-server/internal/authorization"
	"github.com/gebl/onenote-mcp-server/internal/graph"
	"github.com/gebl/onenote-mcp-server/internal/logging"
	"github.com/gebl/onenote-mcp-server/internal/notebooks"
	"github.com/gebl/onenote-mcp-server/internal/resources"
	"github.com/gebl/onenote-mcp-server/internal/sections"
	"github.com/gebl/onenote-mcp-server/internal/utils"
)

// Context keys are now imported from utils to avoid duplication
const (
	mcpServerKey     = utils.MCPServerKey
	progressTokenKey = utils.ProgressTokenKey
)

// SectionItem represents a section or section group in the hierarchical tree
type SectionItem struct {
	Type     string        `json:"type"`     // "section" or "sectionGroup"
	ID       string        `json:"id"`       // Unique identifier
	Name     string        `json:"name"`     // Display name
	Children []SectionItem `json:"children"` // Child items (nil for sections, populated for section groups)
}

// Input/Output structs for notebook tools
type NotebooksInput struct{}
type NotebooksOutput struct {
	Notebooks []map[string]interface{} `json:"notebooks"`
}

type SelectNotebookInput struct {
	Name string `json:"name" jsonschema:"The exact display name of the notebook to select"`
}
type SelectNotebookOutput struct {
	Success      bool                   `json:"success"`
	Notebook     map[string]interface{} `json:"notebook"`
	Message      string                 `json:"message"`
	NotebookName string                 `json:"notebookName"`
}

type SectionCreateInput struct {
	ContainerID string `json:"containerID" jsonschema:"Notebook ID or Section Group ID to create the section in. Optional - if left blank, automatically uses the server's configured default notebook."`
	DisplayName string `json:"displayName" jsonschema:"Display name for the new section (cannot contain: ?*\\/:<>|&#''%%~)"`
}
type SectionCreateOutput struct {
	Success   bool   `json:"success"`
	SectionID string `json:"sectionID"`
}

type SectionGroupCreateInput struct {
	ContainerID string `json:"containerID" jsonschema:"Notebook ID or Section Group ID to create the section group in. Optional - if left blank, automatically uses the server's configured default notebook."`
	DisplayName string `json:"displayName" jsonschema:"Display name for the new section group (cannot contain: ?*\\/:<>|&#''%%~)"`
}
type SectionGroupCreateOutput struct {
	Success        bool   `json:"success"`
	SectionGroupID string `json:"sectionGroupID"`
}

type NotebookCurrentInput struct{}
type NotebookCurrentOutput struct {
	Notebook map[string]interface{} `json:"notebook"`
}

type NotebookSelectInput struct {
	Identifier string `json:"identifier" jsonschema:"Notebook name or ID to select as active"`
}
type NotebookSelectOutput struct {
	Success  bool                   `json:"success"`
	Message  string                 `json:"message"`
	Notebook map[string]interface{} `json:"notebook"`
}

type SectionsInput struct{}
type SectionsOutput struct {
	Notebook      map[string]interface{} `json:"notebook"`
	Cached        bool                   `json:"cached"`
	CacheHit      bool                   `json:"cache_hit"`
	SectionsCount int                    `json:"sections_count"`
	Duration      string                 `json:"duration"`
}

type CacheClearInput struct{}
type CacheClearOutput struct {
	Success  bool     `json:"success"`
	Message  string   `json:"message"`
	Cleared  []string `json:"cleared"`
	Duration string   `json:"duration"`
}

// registerNotebookTools registers notebook and section-related MCP tools
func registerNotebookTools(s *mcp.Server, graphClient *graph.Client, notebookCache *NotebookCache, authConfig *authorization.AuthorizationConfig, cache authorization.NotebookCache, quickNoteConfig authorization.QuickNoteConfig) {
	// Create specialized clients for notebook and section operations
	notebookClient := notebooks.NewNotebookClient(graphClient)
	sectionClient := sections.NewSectionClient(graphClient)

	// notebooks: List all OneNote notebooks for the user
	notebooksHandler := func(ctx context.Context, req *mcp.CallToolRequest, input NotebooksInput) (*mcp.CallToolResult, NotebooksOutput, error) {
		startTime := time.Now()
		logging.ToolsLogger.Debug("notebooks called with no parameters")

		notebooks, err := notebookClient.ListNotebooksDetailed()
		if err != nil {
			logging.ToolsLogger.Error("notebooks failed", "error", err)
			return utils.ToolResults.NewError("notebooks", err), NotebooksOutput{}, nil
		}

		// Apply authorization filtering
		originalCount := len(notebooks)
		if authConfig != nil && authConfig.Enabled {
			notebooks = authConfig.FilterNotebooks(notebooks)
		}

		elapsed := time.Since(startTime)
		logging.ToolsLogger.Debug("notebooks completed",
			"duration", elapsed,
			"original_count", originalCount,
			"filtered_count", len(notebooks))

		// Handle empty results gracefully
		if len(notebooks) == 0 {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: "No notebooks found"},
				},
			}, NotebooksOutput{Notebooks: []map[string]interface{}{}}, nil
		}

		// Create a JSON array with id, name, default status flags, and permission info for each notebook
		type NotebookInfo struct {
			ID                    string `json:"id"`
			Name                  string `json:"name"`
			IsAPIDefault          bool   `json:"isAPIDefault"`          // True if this is the default notebook according to Microsoft Graph API
			IsConfigDefault       bool   `json:"isConfigDefault"`       // True if this matches the configured default notebook name
			Permission            string `json:"permission"`            // Permission level for this notebook (none, read, write)
			PermissionSource      string `json:"permissionSource"`      // Source of permission (exact, pattern, default)
			CanSelect             bool   `json:"canSelect"`             // True if user can select this notebook
		}

		var notebookList []NotebookInfo
		for _, notebook := range notebooks {
			id, _ := notebook["notebookId"].(string)
			displayName, _ := notebook["displayName"].(string)
			
			// Check if this is the API default notebook
			isAPIDefault := false
			if apiDefaultValue, exists := notebook["isDefault"].(bool); exists {
				isAPIDefault = apiDefaultValue
			}
			
			// Check if this matches the configured default notebook name
			isConfigDefault := false
			if graphClient.Config != nil && graphClient.Config.NotebookName != "" {
				isConfigDefault = displayName == graphClient.Config.NotebookName
			}

			// Get permission information
			var permission string = "read"      // Default fallback
			var permissionSource string = "default"
			var canSelect bool = true
			
			if authConfig != nil && authConfig.Enabled {
				actualPermission, _, source := authConfig.GetNotebookPermissionWithSource(displayName)
				permission = string(actualPermission)
				permissionSource = source
				canSelect = actualPermission != authorization.PermissionNone && actualPermission != ""
			}

			notebookList = append(notebookList, NotebookInfo{
				ID:                    id,
				Name:                  displayName,
				IsAPIDefault:          isAPIDefault,
				IsConfigDefault:       isConfigDefault,
				Permission:            permission,
				PermissionSource:      permissionSource,
				CanSelect:             canSelect,
			})
		}

		// Convert to the required format for output
		notebooksData := make([]map[string]interface{}, len(notebookList))
		for i, nb := range notebookList {
			notebooksData[i] = map[string]interface{}{
				"id":                nb.ID,
				"name":              nb.Name,
				"isAPIDefault":      nb.IsAPIDefault,
				"isConfigDefault":   nb.IsConfigDefault,
				"permission":        nb.Permission,
				"permissionSource":  nb.PermissionSource,
				"canSelect":         nb.CanSelect,
			}
		}

		// Marshal for display in content
		jsonBytes, err := json.Marshal(notebooksData)
		if err != nil {
			logging.ToolsLogger.Error("Failed to marshal notebooks to JSON", "error", err)
			return utils.ToolResults.NewError("notebooks", err), NotebooksOutput{}, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: string(jsonBytes)},
			},
		}, NotebooksOutput{Notebooks: notebooksData}, nil
	}

	// Register notebooks tool (TODO: Add authorization wrapper compatibility)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "notebooks",
		Description: resources.MustGetToolDescription("notebooks"),
	}, notebooksHandler)

	// section_create: Create a new section in a notebook or section group
	sectionCreateHandler := func(ctx context.Context, req *mcp.CallToolRequest, input SectionCreateInput) (*mcp.CallToolResult, SectionCreateOutput, error) {
		logger := utils.NewToolLogger("section_create")

		containerID := input.ContainerID
		if containerID == "" {
			logger.LogDebug("No containerID provided, using default notebook")
			var err error
			containerID, err = notebooks.GetDefaultNotebookID(graphClient, graphClient.Config)
			if err != nil {
				logger.LogError(err)
				return utils.ToolResults.NewError("get default notebook", err), SectionCreateOutput{}, nil
			}
		}

		displayName := input.DisplayName
		if displayName == "" {
			logger.LogError(fmt.Errorf("displayName is required"))
			return utils.ToolResults.NewError("section_create", fmt.Errorf("displayName is required")), SectionCreateOutput{}, nil
		}

		logger.LogDebug("Parameters extracted", "containerID", containerID, "displayName", displayName)

		// Validate display name for illegal characters
		illegalChars := []string{"?", "*", "\\", "/", ":", "<", ">", "|", "&", "#", "'", "'", "%", "~"}
		for _, char := range illegalChars {
			if strings.Contains(displayName, char) {
				logger.LogError(fmt.Errorf("displayName contains illegal character: %s", char), "character", char, "display_name", displayName)
				suggestedName := utils.SuggestValidName(displayName, char)
				return utils.ToolResults.NewError("validate displayName", fmt.Errorf("displayName contains illegal character '%s'. Illegal characters are: ?*\\/:<>|&#''%%%%~\n\nSuggestion: Try using '%s' instead of '%s'.\n\nSuggested valid name: '%s'", char, utils.GetReplacementChar(char), char, suggestedName)), SectionCreateOutput{}, nil
			}
		}
		logger.LogDebug("DisplayName validation passed")

		result, err := sectionClient.CreateSection(containerID, displayName)
		if err != nil {
			logger.LogError(err, "container_id", containerID)
			return utils.ToolResults.NewError("create section", err), SectionCreateOutput{}, nil
		}

		// Extract only the essential information: success status and section ID
		sectionID, exists := result["id"].(string)
		if !exists {
			logger.LogError(fmt.Errorf("section creation succeeded but no ID was returned"), "result", result)
			return utils.ToolResults.NewError("extract section ID", fmt.Errorf("Section creation succeeded but no ID was returned")), SectionCreateOutput{}, nil
		}

		output := SectionCreateOutput{
			Success:   true,
			SectionID: sectionID,
		}

		logger.LogSuccess("section_id", sectionID)
		return utils.ToolResults.NewJSONResult("section_create", output), output, nil
	}

	// Register section_create tool
	mcp.AddTool(s, &mcp.Tool{
		Name:        "section_create",
		Description: resources.MustGetToolDescription("section_create"),
	}, sectionCreateHandler)

	// section_group_create: Create a new section group in a notebook or another section group
	sectionGroupCreateHandler := func(ctx context.Context, req *mcp.CallToolRequest, input SectionGroupCreateInput) (*mcp.CallToolResult, SectionGroupCreateOutput, error) {
		logger := utils.NewToolLogger("section_group_create")

		containerID := input.ContainerID
		if containerID == "" {
			logger.LogDebug("No containerID provided, using default notebook")
			var err error
			containerID, err = notebooks.GetDefaultNotebookID(graphClient, graphClient.Config)
			if err != nil {
				logger.LogError(err)
				return utils.ToolResults.NewError("get default notebook", err), SectionGroupCreateOutput{}, nil
			}
		}

		displayName := input.DisplayName
		if displayName == "" {
			logger.LogError(fmt.Errorf("displayName is required"))
			return utils.ToolResults.NewError("section_group_create", fmt.Errorf("displayName is required")), SectionGroupCreateOutput{}, nil
		}

		logger.LogDebug("Parameters extracted", "containerID", containerID, "displayName", displayName)

		// Validate display name for illegal characters
		illegalChars := []string{"?", "*", "\\", "/", ":", "<", ">", "|", "&", "#", "'", "'", "%", "~"}
		for _, char := range illegalChars {
			if strings.Contains(displayName, char) {
				logger.LogError(fmt.Errorf("displayName contains illegal character: %s", char), "character", char, "display_name", displayName)
				suggestedName := utils.SuggestValidName(displayName, char)
				return utils.ToolResults.NewError("validate displayName", fmt.Errorf("displayName contains illegal character '%s'. Illegal characters are: ?*\\/:<>|&#''%%%%~\n\nSuggestion: Try using '%s' instead of '%s'.\n\nSuggested valid name: '%s'", char, utils.GetReplacementChar(char), char, suggestedName)), SectionGroupCreateOutput{}, nil
			}
		}
		logger.LogDebug("DisplayName validation passed")

		result, err := sectionClient.CreateSectionGroup(containerID, displayName)
		if err != nil {
			logger.LogError(err, "container_id", containerID)
			return utils.ToolResults.NewError("create section group", err), SectionGroupCreateOutput{}, nil
		}

		// Extract only the essential information: success status and section group ID
		sectionGroupID, exists := result["id"].(string)
		if !exists {
			logger.LogError(fmt.Errorf("section group creation succeeded but no ID was returned"), "result", result)
			return utils.ToolResults.NewError("extract section group ID", fmt.Errorf("Section group creation succeeded but no ID was returned")), SectionGroupCreateOutput{}, nil
		}

		output := SectionGroupCreateOutput{
			Success:        true,
			SectionGroupID: sectionGroupID,
		}

		logger.LogSuccess("section_group_id", sectionGroupID)
		return utils.ToolResults.NewJSONResult("section_group_create", output), output, nil
	}

	// Register section_group_create tool
	mcp.AddTool(s, &mcp.Tool{
		Name:        "section_group_create",
		Description: resources.MustGetToolDescription("section_group_create"),
	}, sectionGroupCreateHandler)

	// notebook_current: Get currently selected notebook metadata from cache
	notebookCurrentHandler := func(ctx context.Context, req *mcp.CallToolRequest, input NotebookCurrentInput) (*mcp.CallToolResult, NotebookCurrentOutput, error) {
		logger := utils.NewToolLogger("notebook_current")
		logger.LogDebug("Getting current notebook from cache")

		notebook, isSet := notebookCache.GetNotebook()
		if !isSet {
			logger.LogDebug("No notebook currently selected")
			return utils.ToolResults.NewError("notebook_current", fmt.Errorf("No notebook is currently selected. Use the 'notebook_select' tool to select a notebook first")), NotebookCurrentOutput{}, nil
		}

		output := NotebookCurrentOutput{
			Notebook: notebook,
		}

		if displayName, ok := notebookCache.GetDisplayName(); ok {
			logger.LogSuccess("notebook_name", displayName)
		} else {
			logger.LogSuccess("notebook_id", notebook["id"])
		}

		return utils.ToolResults.NewJSONResult("notebook_current", output), output, nil
	}

	// Register notebook_current tool
	mcp.AddTool(s, &mcp.Tool{
		Name:        "notebook_current",
		Description: resources.MustGetToolDescription("notebook_current"),
	}, notebookCurrentHandler)

	// notebook_select: Select a notebook by name or ID to use as the active notebook
	notebookSelectHandler := func(ctx context.Context, req *mcp.CallToolRequest, input NotebookSelectInput) (*mcp.CallToolResult, NotebookSelectOutput, error) {
		logger := utils.NewToolLogger("notebook_select")

		identifier := input.Identifier
		if identifier == "" {
			logger.LogError(fmt.Errorf("identifier is required"))
			return utils.ToolResults.NewError("notebook_select", fmt.Errorf("Missing required parameter: identifier (notebook name or ID)")), NotebookSelectOutput{}, nil
		}

		logger.LogDebug("Selecting notebook", "identifier", identifier)

		// Try to get notebook by name first, then by ID
		var notebook map[string]interface{}
		var err error

		// First try as name
		notebook, err = notebookClient.GetDetailedNotebookByName(identifier)
		if err != nil {
			// Try as ID - get all detailed notebooks and find matching ID
			logger.LogDebug("Failed to find notebook by name, trying as ID", "identifier", identifier, "name_error", err)

			detailedNotebooks, errList := notebookClient.ListNotebooksDetailed()
			if errList != nil {
				logger.LogError(errList, "identifier", identifier)
				return utils.ToolResults.NewError("list notebooks", fmt.Errorf("Failed to search for notebook '%s': %v", identifier, errList)), NotebookSelectOutput{}, nil
			}

			// Find notebook by ID
			found := false
			for _, nb := range detailedNotebooks {
				if nbID, ok := nb["id"].(string); ok && nbID == identifier {
					notebook = nb
					found = true
					break
				}
			}

			if !found {
				logger.LogError(fmt.Errorf("notebook not found"), "identifier", identifier)
				return utils.ToolResults.NewError("find notebook", fmt.Errorf("Failed to find notebook '%s' by name or ID", identifier)), NotebookSelectOutput{}, nil
			}
		}

		// Validate authorization for this notebook selection
		notebookDisplayName, _ := notebook["displayName"].(string)
		if authConfig != nil && authConfig.Enabled {
			if err := authConfig.SetCurrentNotebook(notebookDisplayName); err != nil {
				logger.LogError(err, "notebook_name", notebookDisplayName)
				return utils.ToolResults.NewError("authorization check", fmt.Errorf("Authorization denied: %v", err)), NotebookSelectOutput{}, nil
			}
			logger.LogDebug("Authorization granted", "notebook_name", notebookDisplayName)
		}

		// Set the notebook in cache after authorization check
		notebookCache.SetNotebook(notebook)

		output := NotebookSelectOutput{
			Success:  true,
			Message:  fmt.Sprintf("Successfully selected notebook: %s", identifier),
			Notebook: notebook,
		}

		if displayName, ok := notebook["displayName"].(string); ok {
			logger.LogSuccess("notebook_name", displayName, "notebook_id", notebook["id"])
		} else {
			logger.LogSuccess("notebook_id", notebook["id"])
		}

		return utils.ToolResults.NewJSONResult("notebook_select", output), output, nil
	}

	// Register notebook_select tool
	mcp.AddTool(s, &mcp.Tool{
		Name:        "notebook_select",
		Description: resources.MustGetToolDescription("notebook_select"),
	}, notebookSelectHandler)

	// sections: Get sections and section groups from selected notebook with caching
	sectionsHandler := func(ctx context.Context, req *mcp.CallToolRequest, input SectionsInput) (*mcp.CallToolResult, SectionsOutput, error) {
		startTime := time.Now()
		logger := utils.NewToolLogger("sections")
		logger.LogDebug("Getting sections from selected notebook")

		// Check if notebook is selected
		notebookID, isSet := notebookCache.GetNotebookID()
		if !isSet {
			logger.LogDebug("No notebook currently selected")
			return utils.ToolResults.NewError("sections", fmt.Errorf("No notebook is currently selected. Use the 'notebook_select' tool to select a notebook first")), SectionsOutput{}, nil
		}

		// Send initial progress notification
		// First check the _meta field for progressToken
		var progressToken string

		// Extract progress token from request using the new SDK method
		if rawToken := req.Params.GetProgressToken(); rawToken != nil {
			logging.ToolsLogger.Debug("Raw progress token found",
				"token_type", fmt.Sprintf("%T", rawToken),
				"token_value", rawToken)

			// Handle both string and numeric progress tokens
			switch token := rawToken.(type) {
			case string:
				progressToken = token
				logging.ToolsLogger.Debug("Progress token extracted as string", "progress_token", progressToken)
			case int:
				progressToken = fmt.Sprintf("%d", token)
				logging.ToolsLogger.Debug("Progress token extracted as int", "progress_token", progressToken, "original_value", token)
			case int64:
				progressToken = fmt.Sprintf("%d", token)
				logging.ToolsLogger.Debug("Progress token extracted as int64", "progress_token", progressToken, "original_value", token)
			case float64:
				// Check if it's a whole number and convert appropriately
				if token == float64(int64(token)) {
					progressToken = fmt.Sprintf("%.0f", token)
				} else {
					progressToken = fmt.Sprintf("%g", token)
				}
				logging.ToolsLogger.Debug("Progress token extracted as float64",
					"progress_token", progressToken,
					"original_value", token,
					"is_whole_number", token == float64(int64(token)))
			default:
				progressToken = fmt.Sprintf("%v", token)
				logging.ToolsLogger.Debug("Progress token extracted with default conversion",
					"token_type", fmt.Sprintf("%T", token),
					"token_value", token,
					"progress_token", progressToken)
			}
		} else {
			logging.ToolsLogger.Debug("No progress token found")
		}

		logging.ToolsLogger.Debug("Final progress token extraction result",
			"progress_token", progressToken,
			"token_empty", progressToken == "")

		sendProgressNotification(s, ctx, progressToken, 0, 100, "Starting to fetch notebook sections...")

		// Check cache first
		if notebookCache.IsSectionsCached() {
			// Send progress notification for cache check
			sendProgressNotification(s, ctx, progressToken, 10, 100, "Checking cache for notebook sections...")

			cachedStructure, cached := notebookCache.GetSectionsTree()
			if cached {
				// Send progress notification for cache hit
				sendProgressNotification(s, ctx, progressToken, 80, 100, "Found cached notebook sections, preparing response...")
				// Extract the array from the cached wrapper - handle both old and new format
				var cachedArray []SectionItem
				if sectionsInterface, hasSections := cachedStructure["sections"]; hasSections {
					// Try to convert to []SectionItem (new format)
					if sections, ok := sectionsInterface.([]SectionItem); ok {
						cachedArray = sections
					} else if oldSections, ok := sectionsInterface.([]map[string]interface{}); ok {
						// Handle legacy format - convert to SectionItem
						for _, oldSection := range oldSections {
							sectionItem := convertMapToSectionItem(oldSection)
							cachedArray = append(cachedArray, sectionItem)
						}
					}
				}

				// Apply authorization filtering to cached sections
				originalCachedCount := len(cachedArray)
				if authConfig != nil && authConfig.Enabled {
					// Get notebook display name for filtering context
					notebookDisplayName, _ := notebookCache.GetDisplayName()
					if notebookDisplayName == "" {
						notebookDisplayName = "Unknown Notebook"
					}
					
					// Convert SectionItem slice to []map[string]interface{} for filtering
					var sectionsForFiltering []map[string]interface{}
					for _, item := range cachedArray {
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
					var filteredCachedArray []SectionItem
					for _, filteredSection := range filteredSections {
						for _, originalItem := range cachedArray {
							if originalItem.ID == filteredSection["id"].(string) {
								filteredCachedArray = append(filteredCachedArray, originalItem)
								break
							}
						}
					}
					
					cachedArray = filteredCachedArray
					logging.ToolsLogger.Debug("Applied authorization filtering to cached sections",
						"notebook", notebookDisplayName,
						"original_count", originalCachedCount,
						"filtered_count", len(cachedArray))
				}

				elapsed := time.Since(startTime)
				logging.ToolsLogger.Debug("sections completed from cache",
					"duration", elapsed,
					"notebook_id", notebookID,
					"original_cached_count", originalCachedCount,
					"filtered_cached_count", len(cachedArray),
					"cache_hit", true)

				// Get notebook display name (already retrieved above for filtering)
				notebookDisplayName, _ := notebookCache.GetDisplayName()
				if notebookDisplayName == "" {
					notebookDisplayName = "Unknown Notebook"
				}

				// Create notebook root structure with sections as children
				notebookRoot := map[string]interface{}{
					"type":        "notebook",
					"id":          notebookID,
					"displayName": notebookDisplayName,
					"children":    cachedArray,
				}



				// Send final 100% progress notification before returning result (MCP spec requirement)
				sendProgressNotification(s, ctx, progressToken, 100, 100, "Completed - returning cached notebook sections")

				output := SectionsOutput{
					Notebook:      notebookRoot,
					Cached:        true,
					CacheHit:      true,
					SectionsCount: len(cachedArray),
					Duration:      elapsed.String(),
				}

				return utils.ToolResults.NewJSONResult("sections", output), output, nil
			}
		}

		// Cache miss - need to fetch from API
		logging.ToolsLogger.Debug("Cache miss, fetching sections from Graph API", "notebook_id", notebookID)

		// Send progress notification for cache miss (continuing from cache check at 10%)
		sendProgressNotification(s, ctx, progressToken, 20, 100, "Cache miss, fetching sections from Microsoft Graph API...")

		// Fetch all sections and section groups recursively
		logging.ToolsLogger.Debug("Fetching all sections and section groups recursively", "notebook_id", notebookID)

		// Create a progress context to pass the server and progress token
		progressCtx := context.WithValue(ctx, mcpServerKey, s)
		progressCtx = context.WithValue(progressCtx, progressTokenKey, progressToken)

		sectionItems, err := fetchAllNotebookContentWithProgress(sectionClient, notebookID, progressCtx)
		if err != nil {
			logger.LogError(err, "notebook_id", notebookID)
			return utils.ToolResults.NewError("fetch sections", err), SectionsOutput{}, nil
		}

		// Apply authorization filtering to sections
		originalSectionCount := len(sectionItems)
		if authConfig != nil && authConfig.Enabled {
			// Get notebook display name for filtering context
			notebookDisplayName, _ := notebookCache.GetDisplayName()
			if notebookDisplayName == "" {
				notebookDisplayName = "Unknown Notebook"
			}
			
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
			logging.ToolsLogger.Debug("Applied authorization filtering to sections",
				"notebook", notebookDisplayName,
				"original_count", originalSectionCount,
				"filtered_count", len(sectionItems))
		}

		// Cache the complete structure
		cacheStructure := map[string]interface{}{
			"sections": sectionItems,
		}
		notebookCache.SetSectionsTree(cacheStructure)

		elapsed := time.Since(startTime)
		logging.ToolsLogger.Debug("sections completed from API",
			"duration", elapsed,
			"notebook_id", notebookID,
			"sections_count", len(sectionItems),
			"cache_hit", false)

		// Get notebook display name
		notebookDisplayName, _ := notebookCache.GetDisplayName()
		if notebookDisplayName == "" {
			notebookDisplayName = "Unknown Notebook"
		}

		// Create notebook root structure with sections as children
		notebookRoot := map[string]interface{}{
			"type":        "notebook",
			"id":          notebookID,
			"displayName": notebookDisplayName,
			"children":    sectionItems,
		}



		// Send final progress notification
		sendProgressNotification(s, ctx, progressToken, 100, 100, "Completed fetching all sections and section groups")

		output := SectionsOutput{
			Notebook:      notebookRoot,
			Cached:        false,
			CacheHit:      false,
			SectionsCount: len(sectionItems),
			Duration:      elapsed.String(),
		}

		logger.LogSuccess("sections_count", len(sectionItems), "duration", elapsed)
		return utils.ToolResults.NewJSONResult("sections", output), output, nil
	}

	// Register sections tool
	mcp.AddTool(s, &mcp.Tool{
		Name:        "sections",
		Description: resources.MustGetToolDescription("sections"),
	}, sectionsHandler)

	// cache_clear: Clear all cached data (notebook sections and pages)
	cacheClearHandler := func(ctx context.Context, req *mcp.CallToolRequest, input CacheClearInput) (*mcp.CallToolResult, CacheClearOutput, error) {
		startTime := time.Now()
		logger := utils.NewToolLogger("cache_clear")
		logger.LogDebug("Clearing all cache data")

		// Clear all cache data
		notebookCache.ClearAllCache()

		elapsed := time.Since(startTime)
		logger.LogDebug("Cache clear operation completed", "duration", elapsed)

		output := CacheClearOutput{
			Success: true,
			Message: "All cache data cleared successfully. Next requests will fetch fresh data from the API.",
			Cleared: []string{
				"notebook sections cache",
				"pages cache for all sections",
			},
			Duration: elapsed.String(),
		}

		logger.LogSuccess("cleared_items", len(output.Cleared), "duration", elapsed)
		return utils.ToolResults.NewJSONResult("cache_clear", output), output, nil
	}

	// Register cache_clear tool (doesn't require authorization since it's a system maintenance operation)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "cache_clear",
		Description: resources.MustGetToolDescription("cache_clear"),
	}, cacheClearHandler)

	logging.ToolsLogger.Debug("Notebook and section tools registered successfully")
}

// sendProgressNotification sends a progress notification using the centralized utility
func sendProgressNotification(s *mcp.Server, ctx context.Context, progressToken string, progress int, total int, message string) {
	utils.SendProgressNotification(s, ctx, progressToken, progress, total, message)
}

// Helper functions for notebook and section operations

// fetchAllNotebookContentWithProgress fetches all sections and section groups with progress notifications
func fetchAllNotebookContentWithProgress(sectionClient *sections.SectionClient, notebookID string, ctx context.Context) ([]SectionItem, error) {
	logging.ToolsLogger.Debug("Starting fetchAllNotebookContentWithProgress", "notebook_id", notebookID)

	// Extract progress info from context
	var mcpServer *mcp.Server
	var progressToken string
	if serverVal := ctx.Value(mcpServerKey); serverVal != nil {
		mcpServer, _ = serverVal.(*mcp.Server)
	}
	if tokenVal := ctx.Value(progressTokenKey); tokenVal != nil {
		progressToken, _ = tokenVal.(string)
	}

	logging.ToolsLogger.Debug("Progress context extracted",
		"notebook_id", notebookID,
		"has_mcp_server", mcpServer != nil,
		"has_progress_token", progressToken != "",
		"progress_token", progressToken)

	// Send progress notification - fetching top-level items (continuing from main function at 20%)
	if mcpServer != nil {
		logging.ToolsLogger.Debug("Sending progress notification for API fetch",
			"notebook_id", notebookID,
			"progress_token", progressToken,
			"progress", 30,
			"message", "Fetching top-level sections and section groups...")
		sendProgressNotification(mcpServer, ctx, progressToken, 30, 100, "Fetching top-level sections and section groups...")
	} else {
		logging.ToolsLogger.Debug("No MCP server available for progress notifications", "notebook_id", notebookID)
	}

	// Send progress notification before the critical API call
	if mcpServer != nil {
		logging.ToolsLogger.Debug("Sending progress notification for main API call",
			"notebook_id", notebookID,
			"progress_token", progressToken,
			"progress", 32,
			"message", "Making main API call to get sections and section groups...")
		sendProgressNotification(mcpServer, ctx, progressToken, 32, 100, "Making main API call to get sections and section groups...")
	}

	// Get immediate sections and section groups from the notebook using ListSections
	items, err := sectionClient.ListSectionsWithContext(ctx, notebookID)
	if err != nil {
		logging.ToolsLogger.Error("Failed to get sections and section groups from notebook", "notebook_id", notebookID, "error", err)
		return nil, fmt.Errorf("failed to get sections and section groups from notebook: %v", err)
	}

	// Send progress notification after API call completes
	if mcpServer != nil {
		logging.ToolsLogger.Debug("Sending progress notification after API fetch",
			"notebook_id", notebookID,
			"progress_token", progressToken,
			"progress", 35,
			"total_items", len(items),
			"message", fmt.Sprintf("Retrieved %d items, starting to process...", len(items)))
		sendProgressNotification(mcpServer, ctx, progressToken, 35, 100, fmt.Sprintf("Retrieved %d items, starting to process...", len(items)))
	}

	var sectionItems []SectionItem
	totalItems := len(items)
	processedItems := 0

	logging.ToolsLogger.Debug("Starting item processing loop",
		"notebook_id", notebookID,
		"total_items", totalItems)

	// Process each item returned by ListSections
	for i, item := range items {
		itemID := getStringValue(item, "id")
		itemName := getStringValue(item, "displayName")
		if itemName == "" {
			itemName = itemID
		}

		logging.ToolsLogger.Debug("Processing individual item",
			"notebook_id", notebookID,
			"item_index", i+1,
			"total_items", totalItems,
			"item_id", itemID,
			"item_name", itemName)

		// Send progress for each item being processed - more frequent updates to prevent timeouts
		if mcpServer != nil {
			progress := 35 + int(float64(i)/float64(totalItems)*45) // Progress from 35 to 80 (leave room for completion)
			logging.ToolsLogger.Debug("Sending item progress notification",
				"notebook_id", notebookID,
				"item_index", i+1,
				"item_name", itemName,
				"progress", progress,
				"message", fmt.Sprintf("Processing item %d/%d: %s", i+1, totalItems, itemName))
			sendProgressNotification(mcpServer, ctx, progressToken, progress, 100, fmt.Sprintf("Processing item %d/%d: %s", i+1, totalItems, itemName))
		}

		// Build the section item - this may involve recursive API calls for section groups
		sectionItem, err := buildSectionItemWithProgress(item, sectionClient, ctx, i, totalItems)
		if err != nil {
			logging.ToolsLogger.Warn("Failed to build section item, skipping",
				"notebook_id", notebookID,
				"item_id", itemID,
				"item_name", itemName,
				"error", err)
			continue // Skip this item but continue with others
		}

		// Send progress after each item is completed
		if mcpServer != nil {
			progressAfterCompletion := 35 + int(float64(i+1)/float64(totalItems)*45) // Updated progress after completion
			logging.ToolsLogger.Debug("Sending item completion progress notification",
				"notebook_id", notebookID,
				"item_index", i+1,
				"item_name", itemName,
				"progress", progressAfterCompletion,
				"message", fmt.Sprintf("Completed item %d/%d: %s", i+1, totalItems, itemName))
			sendProgressNotification(mcpServer, ctx, progressToken, progressAfterCompletion, 100, fmt.Sprintf("Completed item %d/%d: %s", i+1, totalItems, itemName))
		}

		logging.ToolsLogger.Debug("Successfully built section item",
			"notebook_id", notebookID,
			"item_id", itemID,
			"item_name", itemName,
			"item_type", sectionItem.Type)

		sectionItems = append(sectionItems, sectionItem)
		processedItems++
	}

	// Send completion progress
	if mcpServer != nil {
		logging.ToolsLogger.Debug("Sending completion progress notification",
			"notebook_id", notebookID,
			"processed_items", processedItems,
			"total_items", totalItems,
			"progress", 85)
		sendProgressNotification(mcpServer, ctx, progressToken, 85, 100, fmt.Sprintf("Completed processing all %d items", processedItems))
	} else {
		logging.ToolsLogger.Debug("No MCP server for completion notification",
			"notebook_id", notebookID,
			"processed_items", processedItems)
	}

	logging.ToolsLogger.Debug("fetchAllNotebookContentWithProgress completed",
		"notebook_id", notebookID,
		"result_items", len(sectionItems))

	return sectionItems, nil
}

// getStringField safely extracts a string field from a map
func getStringField(item map[string]interface{}, field string) string {
	if value, ok := item[field].(string); ok {
		return value
	}
	return ""
}

// getStringValue safely extracts a string value from a map (alias for getStringField)
func getStringValue(item map[string]interface{}, field string) string {
	return getStringField(item, field)
}

// buildSectionItemWithProgress builds a SectionItem with progress notifications
func buildSectionItemWithProgress(item map[string]interface{}, sectionClient *sections.SectionClient, ctx context.Context, itemIndex, totalItems int) (SectionItem, error) {
	id := getStringValue(item, "id")
	name := getStringValue(item, "displayName")

	logging.ToolsLogger.Debug("Starting buildSectionItemWithProgress",
		"item_id", id,
		"item_name", name,
		"item_index", itemIndex,
		"total_items", totalItems)

	// Extract progress info from context
	var mcpServer *mcp.Server
	var progressToken string
	if serverVal := ctx.Value(mcpServerKey); serverVal != nil {
		mcpServer, _ = serverVal.(*mcp.Server)
	}
	if tokenVal := ctx.Value(progressTokenKey); tokenVal != nil {
		progressToken, _ = tokenVal.(string)
	}

	logging.ToolsLogger.Debug("Progress context in buildSectionItemWithProgress",
		"item_id", id,
		"has_mcp_server", mcpServer != nil,
		"has_progress_token", progressToken != "",
		"progress_token", progressToken)

	// Determine if this is a section or section group by checking for specific fields
	itemType := determineSectionItemType(item)

	logging.ToolsLogger.Debug("Determined item type",
		"item_id", id,
		"item_name", name,
		"item_type", itemType)

	sectionItem := SectionItem{
		Type: itemType,
		ID:   id,
		Name: name,
	}

	// If this is a section group, populate its children by calling ListSections recursively
	if itemType == "sectionGroup" {
		logging.ToolsLogger.Debug("Processing section group - building children",
			"section_group_id", id,
			"section_group_name", name,
			"item_index", itemIndex,
			"total_items", totalItems)

		// Send progress notification for section group - early notification to prevent gaps
		if mcpServer != nil {
			progress := 35 + int(float64(itemIndex)/float64(totalItems)*45) + 2
			logging.ToolsLogger.Debug("Sending section group progress notification",
				"section_group_id", id,
				"section_group_name", name,
				"progress", progress,
				"message", fmt.Sprintf("Starting to fetch children for section group: %s", name))
			sendProgressNotification(mcpServer, ctx, progressToken, progress, 100, fmt.Sprintf("Starting to fetch children for section group: %s", name))
		} else {
			logging.ToolsLogger.Debug("No MCP server for section group progress notification",
				"section_group_id", id)
		}

		// Send progress notification before the actual API call that was causing long delays
		if mcpServer != nil {
			progress := 35 + int(float64(itemIndex)/float64(totalItems)*45) + 3
			logging.ToolsLogger.Debug("Sending API call progress notification",
				"section_group_id", id,
				"section_group_name", name,
				"progress", progress,
				"message", fmt.Sprintf("Making API call to get children for: %s", name))
			sendProgressNotification(mcpServer, ctx, progressToken, progress, 100, fmt.Sprintf("Making API call to get children for: %s", name))
		}

		childItems, err := sectionClient.ListSectionsWithContext(ctx, id)
		if err != nil {
			logging.ToolsLogger.Warn("Failed to get children for section group, returning empty children",
				"section_group_id", id,
				"section_group_name", name,
				"error", err)
			// Return the section group without children rather than failing completely
			sectionItem.Children = []SectionItem{}
			return sectionItem, nil
		}

		// Send progress notification after API call completes
		if mcpServer != nil {
			progress := 35 + int(float64(itemIndex)/float64(totalItems)*45) + 4 // Add small increment after API call
			logging.ToolsLogger.Debug("Sending section group API completion progress notification",
				"section_group_id", id,
				"section_group_name", name,
				"progress", progress,
				"child_count", len(childItems),
				"message", fmt.Sprintf("Retrieved %d children for section group: %s", len(childItems), name))
			sendProgressNotification(mcpServer, ctx, progressToken, progress, 100, fmt.Sprintf("Retrieved %d children for section group: %s", len(childItems), name))
		}

		logging.ToolsLogger.Debug("Retrieved child items for section group",
			"section_group_id", id,
			"section_group_name", name,
			"child_count", len(childItems))

		var children []SectionItem
		for childIndex, childItem := range childItems {
			childID := getStringValue(childItem, "id")
			childName := getStringValue(childItem, "displayName")
			if childName == "" {
				childName = childID
			}

			logging.ToolsLogger.Debug("Processing child item in section group",
				"parent_section_group_id", id,
				"parent_section_group_name", name,
				"child_id", childID,
				"child_name", childName,
				"child_index", childIndex+1,
				"total_children", len(childItems))

			// Send progress for each child - frequent progress tracking to prevent timeouts
			if mcpServer != nil && len(childItems) > 0 {
				// Calculate progress within the section group processing - smaller increments for frequent updates
				baseProgress := 35 + int(float64(itemIndex)/float64(totalItems)*45) + 4
				childProgress := baseProgress + int(float64(childIndex)/float64(len(childItems))*5) // Add up to 5% progress within section group
				progressMessage := fmt.Sprintf("Processing child %d/%d in %s: %s", childIndex+1, len(childItems), name, childName)
				logging.ToolsLogger.Debug("Sending child progress notification",
					"parent_section_group_id", id,
					"child_id", childID,
					"child_name", childName,
					"progress", childProgress,
					"message", progressMessage)
				sendProgressNotification(mcpServer, ctx, progressToken, childProgress, 100, progressMessage)
			}

			// Recursively build child items with progress
			// Create a progress context for child items to pass the MCP server and progress token
			childProgressCtx := context.WithValue(ctx, mcpServerKey, mcpServer)
			childProgressCtx = context.WithValue(childProgressCtx, progressTokenKey, progressToken)
			childSectionItem, err := buildSectionItemWithProgress(childItem, sectionClient, childProgressCtx, childIndex, len(childItems))
			if err != nil {
				logging.ToolsLogger.Warn("Failed to build child section item, skipping",
					"parent_section_group_id", id,
					"child_id", childID,
					"child_name", childName,
					"error", err)
				continue // Skip this child but continue with others
			}

			// Send progress after each child is completed - quick update to show completion
			if mcpServer != nil && len(childItems) > 0 {
				baseProgress := 35 + int(float64(itemIndex)/float64(totalItems)*45) + 4
				childCompletionProgress := baseProgress + int(float64(childIndex+1)/float64(len(childItems))*5)
				progressMessage := fmt.Sprintf("Completed child %d/%d in %s: %s", childIndex+1, len(childItems), name, childName)
				logging.ToolsLogger.Debug("Sending child completion progress notification",
					"parent_section_group_id", id,
					"child_id", childID,
					"child_name", childName,
					"progress", childCompletionProgress,
					"message", progressMessage)
				sendProgressNotification(mcpServer, ctx, progressToken, childCompletionProgress, 100, progressMessage)
			}

			logging.ToolsLogger.Debug("Successfully built child section item",
				"parent_section_group_id", id,
				"child_id", childID,
				"child_name", childName,
				"child_type", childSectionItem.Type)

			children = append(children, childSectionItem)
		}

		sectionItem.Children = children

		// Send progress after completing all children in section group
		if mcpServer != nil {
			finalProgress := 35 + int(float64(itemIndex)/float64(totalItems)*45) + 10 // Complete the section group processing
			progressMessage := fmt.Sprintf("Completed section group %s with %d children", name, len(children))
			logging.ToolsLogger.Debug("Sending section group completion progress notification",
				"section_group_id", id,
				"section_group_name", name,
				"progress", finalProgress,
				"children_count", len(children),
				"message", progressMessage)
			sendProgressNotification(mcpServer, ctx, progressToken, finalProgress, 100, progressMessage)
		}

		logging.ToolsLogger.Debug("Completed building section group with all children",
			"section_group_id", id,
			"section_group_name", name,
			"children_count", len(children),
			"successful_children", len(children))
	} else {
		// Sections don't have children
		logging.ToolsLogger.Debug("Item is a section, no children to process",
			"section_id", id,
			"section_name", name)
		sectionItem.Children = nil

		// Send progress for simple section
		if mcpServer != nil {
			progress := 35 + int(float64(itemIndex)/float64(totalItems)*45) + 8 // Quick progress for simple section
			progressMessage := fmt.Sprintf("Processed section: %s", name)
			logging.ToolsLogger.Debug("Sending section progress notification",
				"section_id", id,
				"section_name", name,
				"progress", progress,
				"message", progressMessage)
			sendProgressNotification(mcpServer, ctx, progressToken, progress, 100, progressMessage)
		}
	}

	return sectionItem, nil
}

// determineSectionItemType determines if an item is a section or section group
func determineSectionItemType(item map[string]interface{}) string {
	// Check if this item has the structure of a section group
	// Section groups typically have different metadata structure than sections

	// One way to distinguish: check if the item has specific fields that are unique to sections vs section groups
	// Looking at the Microsoft Graph API documentation:
	// - Sections have pagesUrl field
	// - Section groups have sectionsUrl and sectionGroupsUrl fields

	if _, hasPagesURL := item["pagesUrl"]; hasPagesURL {
		return "section"
	}

	if _, hasSectionsURL := item["sectionsUrl"]; hasSectionsURL {
		return "sectionGroup"
	}

	if _, hasSectionGroupsURL := item["sectionGroupsUrl"]; hasSectionGroupsURL {
		return "sectionGroup"
	}

	// Fallback: if we can't determine, assume it's a section
	// This is safer since sections are leaf nodes and won't cause infinite recursion
	logging.ToolsLogger.Debug("Could not determine item type, defaulting to section", "item_id", getStringValue(item, "id"))
	return "section"
}

// convertMapToSectionItem converts a legacy map format to SectionItem (for cache compatibility)
func convertMapToSectionItem(item map[string]interface{}) SectionItem {
	sectionItem := SectionItem{
		Type: getStringValue(item, "type"),
		ID:   getStringValue(item, "id"),
		Name: getStringValue(item, "name"),
	}

	// Handle children if they exist
	if childrenInterface, hasChildren := item["children"]; hasChildren {
		if childrenArray, ok := childrenInterface.([]interface{}); ok {
			var children []SectionItem
			for _, childInterface := range childrenArray {
				if childMap, ok := childInterface.(map[string]interface{}); ok {
					child := convertMapToSectionItem(childMap)
					children = append(children, child)
				}
			}
			sectionItem.Children = children
		}
	}

	return sectionItem
}
