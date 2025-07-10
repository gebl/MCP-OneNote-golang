package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/gebl/onenote-mcp-server/internal/graph"
	"github.com/gebl/onenote-mcp-server/internal/logging"
	"github.com/gebl/onenote-mcp-server/internal/notebooks"
	"github.com/gebl/onenote-mcp-server/internal/resources"
	"github.com/gebl/onenote-mcp-server/internal/sections"
	"github.com/gebl/onenote-mcp-server/internal/utils"
)

// Context keys for progress notification system
type contextKey string

const (
	mcpServerKey     contextKey = "mcpServer"
	progressTokenKey contextKey = "progressToken"
)

// SectionItem represents a section or section group in the hierarchical tree
type SectionItem struct {
	Type     string        `json:"type"`     // "section" or "sectionGroup"
	ID       string        `json:"id"`       // Unique identifier
	Name     string        `json:"name"`     // Display name
	Children []SectionItem `json:"children"` // Child items (nil for sections, populated for section groups)
}

// registerNotebookTools registers notebook and section-related MCP tools
func registerNotebookTools(s *server.MCPServer, graphClient *graph.Client, notebookCache *NotebookCache) {
	// Create specialized clients for notebook and section operations
	notebookClient := notebooks.NewNotebookClient(graphClient)
	sectionClient := sections.NewSectionClient(graphClient)

	// listNotebooks: List all OneNote notebooks for the user
	listNotebooksTool := mcp.NewTool(
		"listNotebooks",
		mcp.WithDescription(resources.MustGetToolDescription("listNotebooks")),
	)
	s.AddTool(listNotebooksTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		startTime := time.Now()
		logging.ToolsLogger.Debug("listNotebooks called with no parameters")

		notebooks, err := notebookClient.ListNotebooks()
		if err != nil {
			logging.ToolsLogger.Error("listNotebooks failed", "error", err)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list notebooks: %v", err)), nil
		}

		elapsed := time.Since(startTime)
		logging.ToolsLogger.Debug("listNotebooks completed", "duration", elapsed, "count", len(notebooks))

		// Handle empty results gracefully
		if len(notebooks) == 0 {
			return mcp.NewToolResultText("[]"), nil
		}

		// Create a simplified JSON array with just id and name for each notebook
		type NotebookInfo struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}

		var notebookList []NotebookInfo
		for _, notebook := range notebooks {
			id, _ := notebook["id"].(string)
			displayName, _ := notebook["displayName"].(string)

			notebookList = append(notebookList, NotebookInfo{
				ID:   id,
				Name: displayName,
			})
		}

		// Marshal to JSON
		jsonBytes, err := json.Marshal(notebookList)
		if err != nil {
			logging.ToolsLogger.Error("Failed to marshal notebooks to JSON", "error", err)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to format notebooks: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonBytes)), nil
	})

	// createSection: Create a new section in a notebook or section group
	createSectionTool := mcp.NewTool(
		"createSection",
		mcp.WithDescription(resources.MustGetToolDescription("createSection")),
		mcp.WithString("containerID", mcp.Description("Notebook ID or Section Group ID to create the section in. Optional - if left blank, automatically uses the server's configured default notebook.")),
		mcp.WithString("displayName", mcp.Required(), mcp.Description("Display name for the new section (cannot contain: ?*\\/:<>|&#''%%~)")),
	)
	s.AddTool(createSectionTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		startTime := time.Now()
		logging.ToolsLogger.Info("MCP Tool: createSection", "operation", "createSection", "type", "tool_invocation")

		containerID := req.GetString("containerID", "")
		if containerID == "" {
			logging.ToolsLogger.Debug("createSection no containerID provided, using default notebook")
			var err error
			containerID, err = notebooks.GetDefaultNotebookID(graphClient, graphClient.Config)
			if err != nil {
				logging.ToolsLogger.Error("createSection failed to get default notebook", "error", err)
				return mcp.NewToolResultError(fmt.Sprintf("No containerID provided and failed to get default notebook: %v", err)), nil
			}
		}

		displayName, err := req.RequireString("displayName")
		if err != nil {
			logging.ToolsLogger.Error("createSection missing displayName", "error", err)
			return mcp.NewToolResultError("displayName is required"), nil
		}

		logging.ToolsLogger.Debug("createSection parameters", "containerID", containerID, "displayName", displayName)

		// Validate display name for illegal characters
		illegalChars := []string{"?", "*", "\\", "/", ":", "<", ">", "|", "&", "#", "'", "'", "%", "~"}
		for _, char := range illegalChars {
			if strings.Contains(displayName, char) {
				logging.ToolsLogger.Error("createSection displayName contains illegal character", "character", char, "display_name", displayName)
				suggestedName := utils.SuggestValidName(displayName, char)
				return mcp.NewToolResultError(fmt.Sprintf("displayName contains illegal character '%s'. Illegal characters are: ?*\\/:<>|&#''%%%%~\n\nSuggestion: Try using '%s' instead of '%s'.\n\nSuggested valid name: '%s'", char, utils.GetReplacementChar(char), char, suggestedName)), nil
			}
		}
		logging.ToolsLogger.Debug("createSection displayName validation passed")

		result, err := sectionClient.CreateSection(containerID, displayName)
		if err != nil {
			logging.ToolsLogger.Error("createSection operation failed", "container_id", containerID, "error", err, "operation", "createSection")
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create section: %v", err)), nil
		}

		// Extract only the essential information: success status and section ID
		sectionID, exists := result["id"].(string)
		if !exists {
			logging.ToolsLogger.Error("createSection result missing ID field", "result", result)
			return mcp.NewToolResultError("Section creation succeeded but no ID was returned"), nil
		}

		response := map[string]interface{}{
			"success":   true,
			"sectionID": sectionID,
		}

		jsonBytes, err := json.Marshal(response)
		if err != nil {
			logging.ToolsLogger.Error("createSection failed to marshal response", "error", err)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal response: %v", err)), nil
		}

		elapsed := time.Since(startTime)
		logging.ToolsLogger.Debug("createSection operation completed", "duration", elapsed, "section_id", sectionID)
		return mcp.NewToolResultText(string(jsonBytes)), nil
	})

	// createSectionGroup: Create a new section group in a notebook or another section group
	createSectionGroupTool := mcp.NewTool(
		"createSectionGroup",
		mcp.WithDescription(resources.MustGetToolDescription("createSectionGroup")),
		mcp.WithString("containerID", mcp.Description("Notebook ID or Section Group ID to create the section group in. Optional - if left blank, automatically uses the server's configured default notebook.")),
		mcp.WithString("displayName", mcp.Required(), mcp.Description("Display name for the new section group (cannot contain: ?*\\/:<>|&#''%%~)")),
	)
	s.AddTool(createSectionGroupTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		startTime := time.Now()
		logging.ToolsLogger.Info("MCP Tool: createSectionGroup", "operation", "createSectionGroup", "type", "tool_invocation")

		containerID := req.GetString("containerID", "")
		if containerID == "" {
			logging.ToolsLogger.Debug("createSectionGroup no containerID provided, using default notebook")
			var err error
			containerID, err = notebooks.GetDefaultNotebookID(graphClient, graphClient.Config)
			if err != nil {
				logging.ToolsLogger.Error("createSectionGroup failed to get default notebook", "error", err)
				return mcp.NewToolResultError(fmt.Sprintf("No containerID provided and failed to get default notebook: %v", err)), nil
			}
		}

		displayName, err := req.RequireString("displayName")
		if err != nil {
			logging.ToolsLogger.Error("createSectionGroup missing displayName", "error", err)
			return mcp.NewToolResultError("displayName is required"), nil
		}

		logging.ToolsLogger.Debug("createSectionGroup parameters", "containerID", containerID, "displayName", displayName)

		// Validate display name for illegal characters
		illegalChars := []string{"?", "*", "\\", "/", ":", "<", ">", "|", "&", "#", "'", "'", "%", "~"}
		for _, char := range illegalChars {
			if strings.Contains(displayName, char) {
				logging.ToolsLogger.Error("createSectionGroup displayName contains illegal character", "character", char, "display_name", displayName)
				suggestedName := utils.SuggestValidName(displayName, char)
				return mcp.NewToolResultError(fmt.Sprintf("displayName contains illegal character '%s'. Illegal characters are: ?*\\/:<>|&#''%%%%~\n\nSuggestion: Try using '%s' instead of '%s'.\n\nSuggested valid name: '%s'", char, utils.GetReplacementChar(char), char, suggestedName)), nil
			}
		}
		logging.ToolsLogger.Debug("createSectionGroup displayName validation passed")

		result, err := sectionClient.CreateSectionGroup(containerID, displayName)
		if err != nil {
			logging.ToolsLogger.Error("createSectionGroup operation failed", "container_id", containerID, "error", err, "operation", "createSectionGroup")
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create section group: %v", err)), nil
		}

		// Extract only the essential information: success status and section group ID
		sectionGroupID, exists := result["id"].(string)
		if !exists {
			logging.ToolsLogger.Error("createSectionGroup result missing ID field", "result", result)
			return mcp.NewToolResultError("Section group creation succeeded but no ID was returned"), nil
		}

		response := map[string]interface{}{
			"success":        true,
			"sectionGroupID": sectionGroupID,
		}

		jsonBytes, err := json.Marshal(response)
		if err != nil {
			logging.ToolsLogger.Error("createSectionGroup failed to marshal response", "error", err)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal response: %v", err)), nil
		}

		elapsed := time.Since(startTime)
		logging.ToolsLogger.Debug("createSectionGroup operation completed", "duration", elapsed, "section_group_id", sectionGroupID)
		return mcp.NewToolResultText(string(jsonBytes)), nil
	})

	// getSelectedNotebook: Get currently selected notebook metadata from cache
	getSelectedNotebookTool := mcp.NewTool(
		"getSelectedNotebook",
		mcp.WithDescription(resources.MustGetToolDescription("getSelectedNotebook")),
	)
	s.AddTool(getSelectedNotebookTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		startTime := time.Now()
		logging.ToolsLogger.Debug("getSelectedNotebook called")

		notebook, isSet := notebookCache.GetNotebook()
		if !isSet {
			logging.ToolsLogger.Debug("No notebook currently selected")
			return mcp.NewToolResultError("No notebook is currently selected. Use the 'selectNotebook' tool to select a notebook first."), nil
		}

		jsonBytes, err := json.Marshal(notebook)
		if err != nil {
			logging.ToolsLogger.Error("getSelectedNotebook failed to marshal notebook", "error", err)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal selected notebook: %v", err)), nil
		}

		elapsed := time.Since(startTime)
		if displayName, ok := notebookCache.GetDisplayName(); ok {
			logging.ToolsLogger.Debug("getSelectedNotebook completed successfully",
				"duration", elapsed,
				"notebook_name", displayName)
		}

		return mcp.NewToolResultText(string(jsonBytes)), nil
	})

	// selectNotebook: Select a notebook by name or ID to use as the active notebook
	selectNotebookTool := mcp.NewTool(
		"selectNotebook",
		mcp.WithDescription(resources.MustGetToolDescription("selectNotebook")),
		mcp.WithString("identifier", mcp.Description("Notebook name or ID to select as active")),
	)
	s.AddTool(selectNotebookTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		startTime := time.Now()
		args := req.GetArguments()

		identifier, ok := args["identifier"].(string)
		if !ok || identifier == "" {
			logging.ToolsLogger.Error("selectNotebook missing identifier parameter")
			return mcp.NewToolResultError("Missing required parameter: identifier (notebook name or ID)"), nil
		}

		logging.ToolsLogger.Debug("selectNotebook called", "identifier", identifier)

		// Try to get notebook by name first, then by ID
		var notebook map[string]interface{}
		var err error

		// First try as name
		notebook, err = notebookClient.GetDetailedNotebookByName(identifier)
		if err != nil {
			// Try as ID - get all detailed notebooks and find matching ID
			logging.ToolsLogger.Debug("Failed to find notebook by name, trying as ID",
				"identifier", identifier, "name_error", err)

			detailedNotebooks, errList := notebookClient.ListNotebooksDetailed()
			if errList != nil {
				logging.ToolsLogger.Error("selectNotebook failed to list detailed notebooks",
					"identifier", identifier, "error", errList)
				return mcp.NewToolResultError(fmt.Sprintf("Failed to search for notebook '%s': %v", identifier, errList)), nil
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
				logging.ToolsLogger.Error("selectNotebook failed to find notebook by name or ID",
					"identifier", identifier)
				return mcp.NewToolResultError(fmt.Sprintf("Failed to find notebook '%s' by name or ID", identifier)), nil
			}
		}

		// Set the notebook in cache
		notebookCache.SetNotebook(notebook)

		// Prepare response
		response := map[string]interface{}{
			"success":  true,
			"message":  fmt.Sprintf("Successfully selected notebook: %s", identifier),
			"notebook": notebook,
		}

		jsonBytes, err := json.Marshal(response)
		if err != nil {
			logging.ToolsLogger.Error("selectNotebook failed to marshal response", "error", err)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal select response: %v", err)), nil
		}

		elapsed := time.Since(startTime)
		if displayName, ok := notebook["displayName"].(string); ok {
			logging.ToolsLogger.Debug("selectNotebook completed successfully",
				"duration", elapsed,
				"notebook_name", displayName,
				"notebook_id", notebook["id"])
		}

		return mcp.NewToolResultText(string(jsonBytes)), nil
	})

	// getNotebookSections: Get sections and section groups from selected notebook with caching
	getNotebookSectionsTool := mcp.NewTool(
		"getNotebookSections",
		mcp.WithDescription(resources.MustGetToolDescription("getNotebookSections")),
	)
	s.AddTool(getNotebookSectionsTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		startTime := time.Now()
		logging.ToolsLogger.Debug("getNotebookSections called")

		// Check if notebook is selected
		notebookID, isSet := notebookCache.GetNotebookID()
		if !isSet {
			logging.ToolsLogger.Debug("No notebook currently selected for getNotebookSections")
			return mcp.NewToolResultError("No notebook is currently selected. Use the 'selectNotebook' tool to select a notebook first."), nil
		}

		// Send initial progress notification
		// First check the _meta field for progressToken
		var progressToken string

		// Debug the entire request structure
		if reqBytes, err := json.Marshal(req); err == nil {
			logging.ToolsLogger.Debug("Full request structure", "request_json", string(reqBytes))
		}

		// Debug just the Meta field
		if req.Params.Meta != nil {
			if metaBytes, err := json.Marshal(req.Params.Meta); err == nil {
				logging.ToolsLogger.Debug("Meta field structure", "meta_json", string(metaBytes))
			}
		}

		logging.ToolsLogger.Debug("Starting progress token extraction",
			"has_meta", req.Params.Meta != nil,
			"has_progress_token_field", req.Params.Meta != nil && req.Params.Meta.ProgressToken != nil)

		if req.Params.Meta != nil && req.Params.Meta.ProgressToken != nil {
			rawToken := req.Params.Meta.ProgressToken
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
			logging.ToolsLogger.Debug("No progress token in _meta field")
		}

		// Fallback to checking arguments for backwards compatibility
		if progressToken == "" {
			progressToken = req.GetString("_progressToken", "")
			if progressToken != "" {
				logging.ToolsLogger.Debug("Progress token found in arguments fallback", "progress_token", progressToken)
			}
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

				elapsed := time.Since(startTime)
				logging.ToolsLogger.Debug("getNotebookSections completed from cache",
					"duration", elapsed,
					"notebook_id", notebookID,
					"cache_hit", true)

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
					"children":    cachedArray,
				}

				// Create response with cache status
				cacheResponse := map[string]interface{}{
					"notebook":       notebookRoot,
					"cached":         true,
					"cache_hit":      true,
					"sections_count": len(cachedArray),
					"duration":       elapsed.String(),
				}

				jsonBytes, err := json.Marshal(cacheResponse)
				if err != nil {
					logging.ToolsLogger.Error("getNotebookSections failed to marshal cached sections", "error", err)
					return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal cached sections: %v", err)), nil
				}

				// Send final 100% progress notification before returning result (MCP spec requirement)
				sendProgressNotification(s, ctx, progressToken, 100, 100, "Completed - returning cached notebook sections")

				return mcp.NewToolResultText(string(jsonBytes)), nil
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
			logging.ToolsLogger.Error("getNotebookSections failed to fetch all content", "notebook_id", notebookID, "error", err)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to fetch sections: %v", err)), nil
		}

		// Cache the complete structure
		cacheStructure := map[string]interface{}{
			"sections": sectionItems,
		}
		notebookCache.SetSectionsTree(cacheStructure)

		elapsed := time.Since(startTime)
		logging.ToolsLogger.Debug("getNotebookSections completed from API",
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

		// Create response with cache status
		apiResponse := map[string]interface{}{
			"notebook":       notebookRoot,
			"cached":         false,
			"cache_hit":      false,
			"sections_count": len(sectionItems),
			"duration":       elapsed.String(),
		}

		jsonBytes, err := json.Marshal(apiResponse)
		if err != nil {
			logging.ToolsLogger.Error("getNotebookSections failed to marshal sections", "error", err)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal sections: %v", err)), nil
		}

		// Send final progress notification
		sendProgressNotification(s, ctx, progressToken, 100, 100, "Completed fetching all sections and section groups")

		return mcp.NewToolResultText(string(jsonBytes)), nil
	})

	// clearCache: Clear all cached data (notebook sections and pages)
	clearCacheTool := mcp.NewTool(
		"clearCache",
		mcp.WithDescription(resources.MustGetToolDescription("clearCache")),
	)
	s.AddTool(clearCacheTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		startTime := time.Now()
		logging.ToolsLogger.Info("MCP Tool: clearCache", "operation", "clearCache", "type", "tool_invocation")

		// Clear all cache data
		notebookCache.ClearAllCache()

		elapsed := time.Since(startTime)
		logging.ToolsLogger.Debug("clearCache operation completed", "duration", elapsed)

		response := map[string]interface{}{
			"success": true,
			"message": "All cache data cleared successfully. Next requests will fetch fresh data from the API.",
			"cleared": []string{
				"notebook sections cache",
				"pages cache for all sections",
			},
			"duration": elapsed.String(),
		}

		jsonBytes, err := json.Marshal(response)
		if err != nil {
			logging.ToolsLogger.Error("clearCache failed to marshal response", "error", err)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonBytes)), nil
	})

	logging.ToolsLogger.Debug("Notebook and section tools registered successfully")
}

// sendProgressNotification sends a progress notification with consistent logging and immediate flushing
func sendProgressNotification(s *server.MCPServer, ctx context.Context, progressToken string, progress int, total int, message string) {
	if progressToken == "" {
		logging.ToolsLogger.Debug("Skipping progress notification - no progress token provided",
			"progress", progress,
			"total", total,
			"message", message)
		return
	}

	// Calculate percentage for enhanced logging
	percentage := float64(progress) / float64(total) * 100

	logging.ToolsLogger.Debug("Preparing to send progress notification to client",
		"progressToken", progressToken,
		"progress", progress,
		"total", total,
		"percentage", fmt.Sprintf("%.1f%%", percentage),
		"message", message,
		"has_server", s != nil)

	if s == nil {
		logging.ToolsLogger.Warn("Cannot send progress notification - MCP server is nil",
			"progressToken", progressToken,
			"progress", progress,
			"message", message)
		return
	}

	err := s.SendNotificationToClient(ctx, "notifications/progress", map[string]any{
		"progressToken": progressToken,
		"progress":      progress,
		"total":         total,
		"message":       message,
	})

	if err != nil {
		logging.ToolsLogger.Warn("Failed to send progress notification to client",
			"error", err,
			"progressToken", progressToken,
			"progress", progress,
			"total", total,
			"percentage", fmt.Sprintf("%.1f%%", percentage),
			"message", message)
	} else {
		logging.ToolsLogger.Debug("Successfully sent progress notification to client",
			"progressToken", progressToken,
			"progress", progress,
			"total", total,
			"percentage", fmt.Sprintf("%.1f%%", percentage),
			"message", message)
	}
}

// Helper functions for notebook and section operations

// fetchAllNotebookContentWithProgress fetches all sections and section groups with progress notifications
func fetchAllNotebookContentWithProgress(sectionClient *sections.SectionClient, notebookID string, ctx context.Context) ([]SectionItem, error) {
	logging.ToolsLogger.Debug("Starting fetchAllNotebookContentWithProgress", "notebook_id", notebookID)

	// Extract progress info from context
	var mcpServer *server.MCPServer
	var progressToken string
	if serverVal := ctx.Value(mcpServerKey); serverVal != nil {
		mcpServer, _ = serverVal.(*server.MCPServer)
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

	// Get immediate sections and section groups from the notebook using ListSections
	items, err := sectionClient.ListSectionsWithContext(ctx, notebookID)
	if err != nil {
		logging.ToolsLogger.Error("Failed to get sections and section groups from notebook", "notebook_id", notebookID, "error", err)
		return nil, fmt.Errorf("failed to get sections and section groups from notebook: %v", err)
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

		// Send progress for each item being processed
		if mcpServer != nil {
			progress := 30 + int(float64(i)/float64(totalItems)*50) // Progress from 30 to 80
			logging.ToolsLogger.Debug("Sending item progress notification",
				"notebook_id", notebookID,
				"item_index", i+1,
				"item_name", itemName,
				"progress", progress,
				"message", fmt.Sprintf("Processing: %s", itemName))
			sendProgressNotification(mcpServer, ctx, progressToken, progress, 100, fmt.Sprintf("Processing: %s", itemName))
		}

		sectionItem, err := buildSectionItemWithProgress(item, sectionClient, ctx, i, totalItems)
		if err != nil {
			logging.ToolsLogger.Warn("Failed to build section item, skipping",
				"notebook_id", notebookID,
				"item_id", itemID,
				"item_name", itemName,
				"error", err)
			continue // Skip this item but continue with others
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
			"progress", 90)
		sendProgressNotification(mcpServer, ctx, progressToken, 90, 100, fmt.Sprintf("Completed processing %d items", processedItems))
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
	var mcpServer *server.MCPServer
	var progressToken string
	if serverVal := ctx.Value(mcpServerKey); serverVal != nil {
		mcpServer, _ = serverVal.(*server.MCPServer)
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

		// Send progress notification for section group
		if mcpServer != nil {
			progress := 30 + int(float64(itemIndex)/float64(totalItems)*50)
			logging.ToolsLogger.Debug("Sending section group progress notification",
				"section_group_id", id,
				"section_group_name", name,
				"progress", progress,
				"message", fmt.Sprintf("Fetching children for section group: %s", name))
			sendProgressNotification(mcpServer, ctx, progressToken, progress, 100, fmt.Sprintf("Fetching children for section group: %s", name))
		} else {
			logging.ToolsLogger.Debug("No MCP server for section group progress notification",
				"section_group_id", id)
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

			// Send progress for each child
			if mcpServer != nil && len(childItems) > 0 {
				progress := 30 + int(float64(itemIndex)/float64(totalItems)*50)
				progressMessage := fmt.Sprintf("Processing child %d/%d in %s: %s", childIndex+1, len(childItems), name, childName)
				logging.ToolsLogger.Debug("Sending child progress notification",
					"parent_section_group_id", id,
					"child_id", childID,
					"child_name", childName,
					"progress", progress,
					"message", progressMessage)
				sendProgressNotification(mcpServer, ctx, progressToken, progress, 100, progressMessage)
			}

			// Recursively build child items with progress
			childSectionItem, err := buildSectionItemWithProgress(childItem, sectionClient, ctx, childIndex, len(childItems))
			if err != nil {
				logging.ToolsLogger.Warn("Failed to build child section item, skipping",
					"parent_section_group_id", id,
					"child_id", childID,
					"child_name", childName,
					"error", err)
				continue // Skip this child but continue with others
			}

			logging.ToolsLogger.Debug("Successfully built child section item",
				"parent_section_group_id", id,
				"child_id", childID,
				"child_name", childName,
				"child_type", childSectionItem.Type)

			children = append(children, childSectionItem)
		}

		sectionItem.Children = children
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
