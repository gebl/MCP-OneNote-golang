package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/gebl/onenote-mcp-server/internal/graph"
	"github.com/gebl/onenote-mcp-server/internal/logging"
	"github.com/gebl/onenote-mcp-server/internal/pages"
	"github.com/gebl/onenote-mcp-server/internal/resources"
	"github.com/gebl/onenote-mcp-server/internal/utils"
)

// registerPageTools registers all page-related MCP tools
func registerPageTools(s *server.MCPServer, pageClient *pages.PageClient, graphClient *graph.Client, notebookCache *NotebookCache) {
	// listPages: List all pages in a section
	listPagesTool := mcp.NewTool(
		"listPages",
		mcp.WithDescription(resources.MustGetToolDescription("listPages")),
		mcp.WithString("sectionID", mcp.Required(), mcp.Description("Section ID - MUST be actual ID, NOT a section name. You must obtain the section ID through other means.")),
	)
	s.AddTool(listPagesTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		startTime := time.Now()
		logging.ToolsLogger.Info("Starting page enumeration", "operation", "listPages", "type", "tool_invocation")

		sectionID, err := req.RequireString("sectionID")
		if err != nil {
			logging.ToolsLogger.Error("listPages missing sectionID", "error", err)
			return mcp.NewToolResultError("sectionID is required"), nil
		}
		logging.ToolsLogger.Debug("listPages parameter", "sectionID", sectionID)

		// Check cache first
		if notebookCache.IsPagesCached(sectionID) {
			cachedPages, cached := notebookCache.GetPagesCache(sectionID)
			if cached {
				elapsed := time.Since(startTime)
				logging.ToolsLogger.Debug("listPages using cached data",
					"section_id", sectionID,
					"duration", elapsed,
					"pages_count", len(cachedPages),
					"cache_hit", true)

				// Handle empty cached results gracefully
				if len(cachedPages) == 0 {
					return mcp.NewToolResultText("No pages found in the specified section. The section may be empty or you may need to create pages first."), nil
				}

				// Create response with cache status
				cacheResponse := map[string]interface{}{
					"pages":       cachedPages,
					"cached":      true,
					"cache_hit":   true,
					"pages_count": len(cachedPages),
					"duration":    elapsed.String(),
				}

				// Convert cached response to JSON for proper formatting
				jsonResult, errMarshal := json.Marshal(cacheResponse)
				if errMarshal != nil {
					logging.ToolsLogger.Error("Failed to marshal cached pages response to JSON", "error", errMarshal)
					// Fall through to fetch from API if JSON marshaling fails
				} else {
					return mcp.NewToolResultText(string(jsonResult)), nil
				}
			}
		}

		// Cache miss or expired - fetch from API
		logging.ToolsLogger.Debug("listPages cache miss or expired, fetching from API", "section_id", sectionID)
		pages, err := pageClient.ListPages(sectionID)
		if err != nil {
			logging.ToolsLogger.Error("listPages operation failed", "section_id", sectionID, "error", err, "operation", "listPages")
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list pages: %v", err)), nil
		}

		// Cache the results
		notebookCache.SetPagesCache(sectionID, pages)
		logging.ToolsLogger.Debug("listPages cached fresh data", "section_id", sectionID, "pages_count", len(pages))

		elapsed := time.Since(startTime)
		logging.ToolsLogger.Info("listPages operation completed", "duration", elapsed, "pages_count", len(pages), "success", true)

		// Handle empty results gracefully
		if len(pages) == 0 {
			return mcp.NewToolResultText("No pages found in the specified section. The section may be empty or you may need to create pages first."), nil
		}

		// Create response with cache status
		apiResponse := map[string]interface{}{
			"pages":       pages,
			"cached":      false,
			"cache_hit":   false,
			"pages_count": len(pages),
			"duration":    elapsed.String(),
		}

		// Convert to JSON for proper formatting
		jsonResult, err := json.Marshal(apiResponse)
		if err != nil {
			logging.ToolsLogger.Error("Failed to marshal pages response to JSON", "error", err)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to format pages: %v", err)), nil
		}
		return mcp.NewToolResultText(string(jsonResult)), nil
	})

	// createPage: Create a new page in a section
	createPageTool := mcp.NewTool(
		"createPage",
		mcp.WithDescription(resources.MustGetToolDescription("createPage")),
		mcp.WithString("sectionID", mcp.Required(), mcp.Description("Section ID to create page in")),
		mcp.WithString("title", mcp.Required(), mcp.Description("Page title (cannot contain: ?*\\/:<>|&#''%%~)")),
		mcp.WithString("content", mcp.Required(), mcp.Description("HTML content for the page")),
	)
	s.AddTool(createPageTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		startTime := time.Now()
		logging.ToolsLogger.Info("Creating new OneNote page", "operation", "createPage", "type", "tool_invocation")

		sectionID, err := req.RequireString("sectionID")
		if err != nil {
			logging.ToolsLogger.Error("createPage missing sectionID", "error", err)
			return mcp.NewToolResultError("sectionID is required"), nil
		}

		title, err := req.RequireString("title")
		if err != nil {
			logging.ToolsLogger.Error("createPage missing title", "error", err)
			return mcp.NewToolResultError("title is required"), nil
		}

		content, err := req.RequireString("content")
		if err != nil {
			logging.ToolsLogger.Error("createPage missing content", "error", err)
			return mcp.NewToolResultError("content is required"), nil
		}

		logging.ToolsLogger.Debug("createPage parameters", "sectionID", sectionID, "title", title, "content_length", len(content))

		// Validate title for illegal characters
		illegalChars := []string{"?", "*", "\\", "/", ":", "<", ">", "|", "&", "#", "'", "'", "%", "~"}
		for _, char := range illegalChars {
			if strings.Contains(title, char) {
				logging.ToolsLogger.Error("createPage title contains illegal character", "character", char, "title", title)
				suggestedName := utils.SuggestValidName(title, char)
				return mcp.NewToolResultError(fmt.Sprintf("title contains illegal character '%s'. Illegal characters are: ?*\\/:<>|&#''%%%%~\n\nSuggestion: Try using '%s' instead of '%s'.\n\nSuggested valid title: '%s'", char, utils.GetReplacementChar(char), char, suggestedName)), nil
			}
		}
		logging.ToolsLogger.Debug("createPage title validation passed")

		result, err := pageClient.CreatePage(sectionID, title, content)
		if err != nil {
			logging.ToolsLogger.Error("createPage operation failed", "section_id", sectionID, "error", err, "operation", "createPage")
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create page: %v", err)), nil
		}

		// Clear pages cache for this section since we added a new page
		notebookCache.ClearPagesCache(sectionID)
		logging.ToolsLogger.Debug("createPage cleared pages cache", "section_id", sectionID)

		// Extract only the essential information: success status and page ID
		pageID, exists := result["id"].(string)
		if !exists {
			logging.ToolsLogger.Error("createPage result missing ID field", "result", result)
			return mcp.NewToolResultError("Page creation succeeded but no ID was returned"), nil
		}

		response := map[string]interface{}{
			"success": true,
			"pageID":  pageID,
		}

		jsonBytes, err := json.Marshal(response)
		if err != nil {
			logging.ToolsLogger.Error("createPage failed to marshal response", "error", err)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal response: %v", err)), nil
		}

		elapsed := time.Since(startTime)
		logging.ToolsLogger.Debug("createPage operation completed", "duration", elapsed, "page_id", pageID)
		return mcp.NewToolResultText(string(jsonBytes)), nil
	})

	// updatePageContent: Update the HTML content of a page
	updatePageContentTool := mcp.NewTool(
		"updatePageContent",
		mcp.WithDescription(resources.MustGetToolDescription("updatePageContent")),
		mcp.WithString("pageID", mcp.Required(), mcp.Description("Page ID to update")),
		mcp.WithString("content", mcp.Required(), mcp.Description("New HTML content for the page")),
	)
	s.AddTool(updatePageContentTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		startTime := time.Now()
		logging.ToolsLogger.Info("Updating OneNote page content", "operation", "updatePageContent", "type", "tool_invocation")

		pageID, err := req.RequireString("pageID")
		if err != nil {
			logging.ToolsLogger.Error("updatePageContent missing pageID", "error", err)
			return mcp.NewToolResultError("pageID is required"), nil
		}

		content, err := req.RequireString("content")
		if err != nil {
			logging.ToolsLogger.Error("updatePageContent missing content", "error", err)
			return mcp.NewToolResultError("content is required"), nil
		}

		logging.ToolsLogger.Debug("updatePageContent parameters", "pageID", pageID, "content_length", len(content))

		err = pageClient.UpdatePageContentSimple(pageID, content)
		if err != nil {
			logging.ToolsLogger.Error("updatePageContent operation failed", "page_id", pageID, "error", err, "operation", "updatePageContent")
			return mcp.NewToolResultError(fmt.Sprintf("Failed to update page content: %v", err)), nil
		}

		elapsed := time.Since(startTime)
		logging.ToolsLogger.Debug("updatePageContent operation completed", "duration", elapsed)
		return mcp.NewToolResultText("Page content updated successfully"), nil
	})

	// updatePageContentAdvanced: Update page content with advanced commands
	updatePageContentAdvancedTool := mcp.NewTool(
		"updatePageContentAdvanced",
		mcp.WithDescription(resources.MustGetToolDescription("updatePageContentAdvanced")),
		mcp.WithString("pageID", mcp.Required(), mcp.Description("Page ID to update")),
		mcp.WithString("commands", mcp.Required(), mcp.Description("JSON STRING containing an array of command objects. MUST be a string, not an array. Example: \"[{\\\"target\\\": \\\"body\\\", \\\"action\\\": \\\"append\\\", \\\"content\\\": \\\"<p>Hello</p>\\\"}]\"")),
	)
	s.AddTool(updatePageContentAdvancedTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		startTime := time.Now()
		logging.ToolsLogger.Info("MCP Tool: updatePageContentAdvanced", "operation", "updatePageContentAdvanced", "type", "tool_invocation")

		pageID, err := req.RequireString("pageID")
		if err != nil {
			logging.ToolsLogger.Error("updatePageContentAdvanced missing pageID", "error", err)
			return mcp.NewToolResultError("pageID is required"), nil
		}

		commandsJSON, err := req.RequireString("commands")
		if err != nil {
			logging.ToolsLogger.Error("updatePageContentAdvanced missing commands", "error", err)
			return mcp.NewToolResultError("commands is required"), nil
		}

		logging.ToolsLogger.Debug("updatePageContentAdvanced parameters", "pageID", pageID, "commands_length", len(commandsJSON))

		// Parse the commands JSON
		var commands []pages.UpdateCommand
		if errUnmarshal := json.Unmarshal([]byte(commandsJSON), &commands); errUnmarshal != nil {
			logging.ToolsLogger.Error("updatePageContentAdvanced failed to parse commands JSON", "error", errUnmarshal)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to parse commands JSON: %v", errUnmarshal)), nil
		}

		logging.ToolsLogger.Debug("updatePageContentAdvanced commands parsed", "command_count", len(commands))

		err = pageClient.UpdatePageContent(pageID, commands)
		if err != nil {
			logging.ToolsLogger.Error("updatePageContentAdvanced operation failed", "page_id", pageID, "error", err, "operation", "updatePageContentAdvanced")
			return mcp.NewToolResultError(fmt.Sprintf("Failed to update page content: %v", err)), nil
		}

		elapsed := time.Since(startTime)
		logging.ToolsLogger.Debug("updatePageContentAdvanced operation completed", "duration", elapsed)
		return mcp.NewToolResultText("Page content updated successfully with advanced commands"), nil
	})

	// deletePage: Delete a page by ID
	deletePageTool := mcp.NewTool(
		"deletePage",
		mcp.WithDescription(resources.MustGetToolDescription("deletePage")),
		mcp.WithString("pageID", mcp.Required(), mcp.Description("Page ID to delete")),
	)
	s.AddTool(deletePageTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		startTime := time.Now()
		logging.ToolsLogger.Info("Deleting OneNote page", "operation", "deletePage", "type", "tool_invocation")

		pageID, err := req.RequireString("pageID")
		if err != nil {
			logging.ToolsLogger.Error("deletePage missing pageID", "error", err)
			return mcp.NewToolResultError("pageID is required"), nil
		}

		logging.ToolsLogger.Debug("deletePage parameter", "pageID", pageID)

		err = pageClient.DeletePage(pageID)
		if err != nil {
			logging.ToolsLogger.Error("deletePage operation failed", "page_id", pageID, "error", err, "operation", "deletePage")
			return mcp.NewToolResultError(fmt.Sprintf("Failed to delete page: %v", err)), nil
		}

		// Clear all pages cache since we don't know which section the deleted page was in
		notebookCache.ClearAllPagesCache()
		logging.ToolsLogger.Debug("deletePage cleared all pages cache", "page_id", pageID)

		elapsed := time.Since(startTime)
		logging.ToolsLogger.Debug("deletePage operation completed", "duration", elapsed)
		return mcp.NewToolResultText("Page deleted successfully"), nil
	})

	// getPageContent: Get the HTML content of a page by ID
	getPageContentTool := mcp.NewTool(
		"getPageContent",
		mcp.WithDescription(resources.MustGetToolDescription("getPageContent")),
		mcp.WithString("pageID", mcp.Required(), mcp.Description("Page ID to fetch content for")),
		mcp.WithString("forUpdate", mcp.Description("Optional: set to 'true' to include includeIDs=true parameter for update operations")),
	)
	s.AddTool(getPageContentTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		startTime := time.Now()
		logging.ToolsLogger.Info("Retrieving OneNote page content", "operation", "getPageContent", "type", "tool_invocation")

		pageID, err := req.RequireString("pageID")
		if err != nil {
			logging.ToolsLogger.Error("getPageContent missing pageID", "error", err)
			return mcp.NewToolResultError("pageID is required"), nil
		}

		forUpdateStr := req.GetString("forUpdate", "")
		forUpdate := forUpdateStr == "true"

		logging.ToolsLogger.Debug("getPageContent parameters", "pageID", pageID, "forUpdate", forUpdate)

		content, err := pageClient.GetPageContent(pageID, forUpdate)
		if err != nil {
			logging.ToolsLogger.Error("getPageContent operation failed", "page_id", pageID, "error", err, "operation", "getPageContent")
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get page content: %v", err)), nil
		}

		elapsed := time.Since(startTime)
		logging.ToolsLogger.Debug("getPageContent operation completed", "duration", elapsed, "content_length", len(content))
		return mcp.NewToolResultText(content), nil
	})

	// getPageItemContent: Get a OneNote page item (e.g., image) by page item ID, returns binary data with proper MIME type
	getPageItemContentTool := mcp.NewTool(
		"getPageItemContent",
		mcp.WithDescription(resources.MustGetToolDescription("getPageItemContent")),
		mcp.WithString("pageID", mcp.Required(), mcp.Description("Page ID to fetch item from")),
		mcp.WithString("pageItemID", mcp.Required(), mcp.Description("Page resource item ID to fetch")),
		mcp.WithString("filename", mcp.Description("Custom filename for binary download")),
		mcp.WithBoolean("fullSize", mcp.Description("Skip image scaling and return original size (default: false = scale images to 1024x768 max)")),
	)
	s.AddTool(getPageItemContentTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		startTime := time.Now()
		logging.ToolsLogger.Info("MCP Tool: getPageItemContent", "operation", "getPageItemContent", "type", "tool_invocation")

		pageID := req.GetString("pageID", "")
		pageItemID := req.GetString("pageItemID", "")
		customFilename := req.GetString("filename", "")
		fullSizeStr := req.GetString("fullSize", "false")
		fullSize := fullSizeStr == "true"

		logging.ToolsLogger.Debug("getPageItemContent parameters", "pageID", pageID, "pageItemID", pageItemID, "filename", customFilename, "fullSize", fullSize)

		// Use the enhanced GetPageItem with fullSize parameter
		pageItemData, err := pageClient.GetPageItem(pageID, pageItemID, fullSize)
		if err != nil {
			logging.ToolsLogger.Error("getPageItemContent operation failed", "page_id", pageID, "page_item_id", pageItemID, "error", err)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get page item: %v", err)), nil
		}

		logging.ToolsLogger.Debug("getPageItemContent retrieved page item", "filename", pageItemData.Filename, "content_type", pageItemData.ContentType, "size_bytes", pageItemData.Size, "tag_name", pageItemData.TagName, "full_size", fullSize)

		// Use custom filename if provided, otherwise use the one from page item data
		filename := pageItemData.Filename
		if customFilename != "" {
			filename = customFilename
			logging.ToolsLogger.Debug("Using custom filename", "filename", filename)
		}

		// Return binary data with proper MIME type using mcp.NewToolResultImage
		// This should provide the raw binary data with the correct content type
		encoded := base64.StdEncoding.EncodeToString(pageItemData.Content)
		logging.ToolsLogger.Debug("Prepared content for return", "encoded_length", len(encoded), "content_type", pageItemData.ContentType, "filename", filename)

		elapsed := time.Since(startTime)
		logging.ToolsLogger.Debug("getPageItemContent operation completed", "duration", elapsed, "scaled", !fullSize && strings.HasPrefix(pageItemData.ContentType, "image/"))
		return mcp.NewToolResultImage(filename, encoded, pageItemData.ContentType), nil
	})

	// listPageItems: List all OneNote page items (images, files, etc.) for a specific page
	listPageItemsTool := mcp.NewTool(
		"listPageItems",
		mcp.WithDescription(resources.MustGetToolDescription("listPageItems")),
		mcp.WithString("pageID", mcp.Required(), mcp.Description("Page ID to list items for")),
	)
	s.AddTool(listPageItemsTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		startTime := time.Now()
		logging.ToolsLogger.Info("MCP Tool: listPageItems", "operation", "listPageItems", "type", "tool_invocation")

		pageID, err := req.RequireString("pageID")
		if err != nil {
			logging.ToolsLogger.Error("listPageItems missing pageID", "error", err)
			return mcp.NewToolResultError("pageID is required"), nil
		}

		logging.ToolsLogger.Debug("listPageItems parameter", "pageID", pageID)

		pageItems, err := pageClient.ListPageItems(pageID)
		if err != nil {
			logging.ToolsLogger.Error("listPageItems operation failed", "page_id", pageID, "error", err, "operation", "listPageItems")
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list page items: %v", err)), nil
		}

		jsonBytes, err := json.Marshal(pageItems)
		if err != nil {
			logging.ToolsLogger.Error("listPageItems failed to marshal page items", "error", err)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal page items: %v", err)), nil
		}

		elapsed := time.Since(startTime)
		logging.ToolsLogger.Info("listPageItems operation completed", "duration", elapsed, "items_count", len(pageItems), "success", true)
		return mcp.NewToolResultText(string(jsonBytes)), nil
	})

	// copyPage: Copy a page from one section to another
	copyPageTool := mcp.NewTool(
		"copyPage",
		mcp.WithDescription(resources.MustGetToolDescription("copyPage")),
		mcp.WithString("pageID", mcp.Required(), mcp.Description("Page ID to copy")),
		mcp.WithString("targetSectionID", mcp.Required(), mcp.Description("Target section ID to copy the page to")),
	)
	s.AddTool(copyPageTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		startTime := time.Now()
		logging.ToolsLogger.Info("Copying OneNote page", "operation", "copyPage", "type", "tool_invocation")

		pageID, err := req.RequireString("pageID")
		if err != nil {
			logging.ToolsLogger.Error("copyPage missing pageID", "error", err)
			return mcp.NewToolResultError("pageID is required"), nil
		}

		targetSectionID, err := req.RequireString("targetSectionID")
		if err != nil {
			logging.ToolsLogger.Error("copyPage missing targetSectionID", "error", err)
			return mcp.NewToolResultError("targetSectionID is required"), nil
		}

		logging.ToolsLogger.Debug("copyPage parameters", "pageID", pageID, "targetSectionID", targetSectionID)

		result, err := pageClient.CopyPage(pageID, targetSectionID)
		if err != nil {
			logging.ToolsLogger.Error("copyPage operation failed", "page_id", pageID, "error", err, "operation", "copyPage")
			return mcp.NewToolResultError(fmt.Sprintf("Failed to copy page: %v", err)), nil
		}

		// Clear pages cache for target section since we added a new page
		notebookCache.ClearPagesCache(targetSectionID)
		logging.ToolsLogger.Debug("copyPage cleared pages cache for target section", "target_section_id", targetSectionID)

		jsonBytes, err := json.Marshal(result)
		if err != nil {
			logging.ToolsLogger.Error("copyPage failed to marshal copy result", "error", err)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal copy result: %v", err)), nil
		}

		elapsed := time.Since(startTime)
		logging.ToolsLogger.Debug("copyPage operation completed", "duration", elapsed)
		return mcp.NewToolResultText(string(jsonBytes)), nil
	})

	// movePage: Move a page from one section to another (copy then delete)
	movePageTool := mcp.NewTool(
		"movePage",
		mcp.WithDescription(resources.MustGetToolDescription("movePage")),
		mcp.WithString("pageID", mcp.Required(), mcp.Description("Page ID to move")),
		mcp.WithString("targetSectionID", mcp.Required(), mcp.Description("Target section ID to move the page to")),
	)
	s.AddTool(movePageTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		startTime := time.Now()
		logging.ToolsLogger.Info("Moving OneNote page", "operation", "movePage", "type", "tool_invocation")

		pageID, err := req.RequireString("pageID")
		if err != nil {
			logging.ToolsLogger.Error("movePage missing pageID", "error", err)
			return mcp.NewToolResultError("pageID is required"), nil
		}

		targetSectionID, err := req.RequireString("targetSectionID")
		if err != nil {
			logging.ToolsLogger.Error("movePage missing targetSectionID", "error", err)
			return mcp.NewToolResultError("targetSectionID is required"), nil
		}

		logging.ToolsLogger.Debug("movePage parameters", "pageID", pageID, "targetSectionID", targetSectionID)

		result, err := pageClient.MovePage(pageID, targetSectionID)
		if err != nil {
			logging.ToolsLogger.Error("movePage operation failed", "page_id", pageID, "error", err, "operation", "movePage")
			return mcp.NewToolResultError(fmt.Sprintf("Failed to move page: %v", err)), nil
		}

		// Clear all pages cache since movePage affects both source and target sections
		// (we don't know the source section ID)
		notebookCache.ClearAllPagesCache()
		logging.ToolsLogger.Debug("movePage cleared all pages cache", "page_id", pageID, "target_section_id", targetSectionID)

		jsonBytes, err := json.Marshal(result)
		if err != nil {
			logging.ToolsLogger.Error("movePage failed to marshal move result", "error", err)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal move result: %v", err)), nil
		}

		elapsed := time.Since(startTime)
		logging.ToolsLogger.Debug("movePage operation completed", "duration", elapsed)
		return mcp.NewToolResultText(string(jsonBytes)), nil
	})

	// NOTE: getOnenoteOperation tool is currently commented out in the original code
	// Uncomment this section if you want to enable it
	/*
		// getOnenoteOperation: Get the status of an asynchronous OneNote operation
		getOnenoteOperationTool := mcp.NewTool(
			"getOnenoteOperation",
			mcp.WithDescription("Check the status of an asynchronous OneNote operation (e.g., copy/move operations)."),
			mcp.WithString("operationId", mcp.Required(), mcp.Description("Operation ID to check status for")),
		)
		s.AddTool(getOnenoteOperationTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			startTime := time.Now()
			logging.ToolsLogger.Info("MCP Tool: getOnenoteOperation", "operation", "getOnenoteOperation", "type", "tool_invocation")

			operationId, err := req.RequireString("operationId")
			if err != nil {
				logging.ToolsLogger.Error("getOnenoteOperation missing operationId", "error", err)
				return mcp.NewToolResultError("operationId is required"), nil
			}

			logging.ToolsLogger.Debug("getOnenoteOperation parameters", "operationId", operationId)

			result, err := graphClient.GetOnenoteOperation(operationId)
			if err != nil {
				logging.ToolsLogger.Error("getOnenoteOperation operation failed", "operation_id", operationId, "error", err, "operation", "getOnenoteOperation")
				return mcp.NewToolResultError(fmt.Sprintf("Failed to get operation status: %v", err)), nil
			}

			jsonBytes, err := json.Marshal(result)
			if err != nil {
				logging.ToolsLogger.Error("getOnenoteOperation failed to marshal operation status", "error", err)
				return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal operation status: %v", err)), nil
			}

			elapsed := time.Since(startTime)
			logging.ToolsLogger.Debug("getOnenoteOperation completed", "duration", elapsed)
			return mcp.NewToolResultText(string(jsonBytes)), nil
		})
	*/

	logging.ToolsLogger.Debug("Page tools registered successfully")
}

// Helper functions are shared from NotebookTools.go
