// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

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

	"github.com/gebl/onenote-mcp-server/internal/authorization"
	"github.com/gebl/onenote-mcp-server/internal/config"
	"github.com/gebl/onenote-mcp-server/internal/graph"
	"github.com/gebl/onenote-mcp-server/internal/logging"
	"github.com/gebl/onenote-mcp-server/internal/notebooks"
	"github.com/gebl/onenote-mcp-server/internal/pages"
	"github.com/gebl/onenote-mcp-server/internal/resources"
	"github.com/gebl/onenote-mcp-server/internal/sections"
	"github.com/gebl/onenote-mcp-server/internal/utils"
)

// registerPageTools registers all page-related MCP tools
func registerPageTools(s *server.MCPServer, pageClient *pages.PageClient, graphClient *graph.Client, notebookCache *NotebookCache, cfg *config.Config, authConfig *authorization.AuthorizationConfig, cache authorization.NotebookCache, quickNoteConfig authorization.QuickNoteConfig) {
	// listPages: List all pages in a section
	listPagesTool := mcp.NewTool(
		"listPages",
		mcp.WithDescription(resources.MustGetToolDescription("listPages")),
		mcp.WithString("sectionID", mcp.Required(), mcp.Description("Section ID - MUST be actual ID, NOT a section name. You must obtain the section ID through other means.")),
	)
	listPagesHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		startTime := time.Now()
		logging.ToolsLogger.Info("Starting page enumeration", "operation", "listPages", "type", "tool_invocation")

		sectionID, err := req.RequireString("sectionID")
		if err != nil {
			logging.ToolsLogger.Error("listPages missing sectionID", "error", err)
			return mcp.NewToolResultError("sectionID is required"), nil
		}
		logging.ToolsLogger.Debug("listPages parameter", "sectionID", sectionID)

		// Extract progress token early for use throughout the function
		progressToken := utils.ExtractProgressToken(req)
		
		// Check cache first
		if progressToken != "" {
			sendProgressNotification(s, ctx, progressToken, 5, 100, "Checking page cache...")
		}
		
		if notebookCache.IsPagesCached(sectionID) {
			cachedPages, cached := notebookCache.GetPagesCache(sectionID)
			if cached {
				// Apply authorization filtering to cached pages
				originalCachedCount := len(cachedPages)
				if authConfig != nil && authConfig.Enabled {
					// Try to get section and notebook context for filtering
					var sectionName, notebookName string
					if cache != nil {
						// If sections aren't cached yet, try to fetch them for proper authorization context
						if !notebookCache.IsSectionsCached() {
							logging.ToolsLogger.Debug("Sections not cached, fetching for authorization context", 
								"section_id", sectionID)
							
							// Progress token already extracted above
							if progressToken != "" {
								sendProgressNotification(s, ctx, progressToken, 10, 100, "Fetching sections for authorization context...")
							}
							
							// Fetch sections to populate cache for authorization
							if err := populateSectionsForAuthorization(s, ctx, graphClient, notebookCache, progressToken); err != nil {
								logging.ToolsLogger.Warn("Failed to populate sections for authorization context", "error", err)
							}
							
							if progressToken != "" {
								sendProgressNotification(s, ctx, progressToken, 30, 100, "Authorization context updated, applying filters...")
							}
						}
						// Try to get section name with progress-aware API lookup if cache misses
					if progressToken != "" {
						sendProgressNotification(s, ctx, progressToken, 32, 100, "Resolving section name for authorization...")
					}
					sectionName, _ = cache.GetSectionNameWithProgress(ctx, sectionID, s, progressToken, graphClient)
					notebookName, _ = cache.GetDisplayName()
					}
					
					cachedPages = authConfig.FilterPages(cachedPages, sectionID, sectionName, notebookName)
					logging.ToolsLogger.Debug("Applied authorization filtering to cached pages",
						"section_id", sectionID,
						"section_name", sectionName,
						"notebook", notebookName,
						"original_count", originalCachedCount,
						"filtered_count", len(cachedPages))
				}

				elapsed := time.Since(startTime)
				logging.ToolsLogger.Debug("listPages using cached data",
					"section_id", sectionID,
					"duration", elapsed,
					"original_cached_count", originalCachedCount,
					"filtered_cached_count", len(cachedPages),
					"cache_hit", true)

				// If cache returns empty results, fall back to fresh API call to be sure
				if len(cachedPages) == 0 {
					logging.ToolsLogger.Debug("Cache returned 0 pages, falling back to fresh API call to verify",
						"section_id", sectionID)
					// Continue to fresh API call below instead of returning immediately
				} else {
					// Cache has pages, use them
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
		}

		// Cache miss or expired - fetch from API
		logging.ToolsLogger.Debug("listPages cache miss or expired, fetching from API", "section_id", sectionID)
		
		if progressToken != "" {
			sendProgressNotification(s, ctx, progressToken, 40, 100, "Fetching pages from OneNote API...")
		}
		
		pages, err := pageClient.ListPages(sectionID)
		
		if progressToken != "" {
			sendProgressNotification(s, ctx, progressToken, 55, 100, "Pages retrieved, processing authorization...")
		}
		if err != nil {
			logging.ToolsLogger.Error("listPages operation failed", "section_id", sectionID, "error", err, "operation", "listPages")
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list pages: %v", err)), nil
		}

		// Apply authorization filtering to fresh pages
		originalApiCount := len(pages)
		if authConfig != nil && authConfig.Enabled {
			// Try to get section and notebook context for filtering
			var sectionName, notebookName string
			if cache != nil {
				// If sections aren't cached yet, try to fetch them for proper authorization context
				if !notebookCache.IsSectionsCached() {
					logging.ToolsLogger.Debug("Sections not cached, fetching for authorization context", 
						"section_id", sectionID)
					
					// Progress token already extracted above
					if progressToken != "" {
						sendProgressNotification(s, ctx, progressToken, 60, 100, "Fetching sections for authorization context...")
					}
					
					// Fetch sections to populate cache for authorization
					if err := populateSectionsForAuthorization(s, ctx, graphClient, notebookCache, progressToken); err != nil {
						logging.ToolsLogger.Warn("Failed to populate sections for authorization context", "error", err)
					}
					
					if progressToken != "" {
						sendProgressNotification(s, ctx, progressToken, 80, 100, "Authorization context updated, applying filters...")
					}
				}
				// Try to get section name with progress-aware API lookup if cache misses
			if progressToken != "" {
				sendProgressNotification(s, ctx, progressToken, 82, 100, "Resolving section name for authorization...")
			}
			sectionName, _ = cache.GetSectionNameWithProgress(ctx, sectionID, s, progressToken, graphClient)
			notebookName, _ = cache.GetDisplayName()
			}
			
			pages = authConfig.FilterPages(pages, sectionID, sectionName, notebookName)
			logging.ToolsLogger.Debug("Applied authorization filtering to fresh pages",
				"section_id", sectionID,
				"section_name", sectionName,
				"notebook", notebookName,
				"original_count", originalApiCount,
				"filtered_count", len(pages))
			
			if progressToken != "" {
				sendProgressNotification(s, ctx, progressToken, 85, 100, "Authorization filtering completed, caching results...")
			}
		}

		// Cache the filtered results
		notebookCache.SetPagesCache(sectionID, pages)
		
		if progressToken != "" {
			sendProgressNotification(s, ctx, progressToken, 90, 100, "Caching completed, preparing response...")
		}
		logging.ToolsLogger.Debug("listPages cached fresh filtered data", 
			"section_id", sectionID, 
			"original_count", originalApiCount,
			"filtered_count", len(pages))

		elapsed := time.Since(startTime)
		logging.ToolsLogger.Info("listPages operation completed", "duration", elapsed, "pages_count", len(pages), "success", true)

		// Handle empty results gracefully
		if len(pages) == 0 {
			return mcp.NewToolResultText("No pages found in the specified section. The section may be empty or you may need to create pages first."), nil
		}

		// Create response with cache status
		if progressToken != "" {
			sendProgressNotification(s, ctx, progressToken, 95, 100, "Formatting response...")
		}
		
		apiResponse := map[string]interface{}{
			"pages":       pages,
			"cached":      false,
			"cache_hit":   false,
			"pages_count": len(pages),
			"duration":    elapsed.String(),
		}

		// Convert to JSON for proper formatting
		jsonResult, err := json.Marshal(apiResponse)
		
		if progressToken != "" {
			sendProgressNotification(s, ctx, progressToken, 100, 100, "Complete!")
		}
		if err != nil {
			logging.ToolsLogger.Error("Failed to marshal pages response to JSON", "error", err)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to format pages: %v", err)), nil
		}
		return mcp.NewToolResultText(string(jsonResult)), nil
	}
	s.AddTool(listPagesTool, server.ToolHandlerFunc(authorization.AuthorizedToolHandler("listPages", listPagesHandler, authConfig, cache, quickNoteConfig)))

	// createPage: Create a new page in a section
	createPageTool := mcp.NewTool(
		"createPage",
		mcp.WithDescription(resources.MustGetToolDescription("createPage")),
		mcp.WithString("sectionID", mcp.Required(), mcp.Description("Section ID to create page in")),
		mcp.WithString("title", mcp.Required(), mcp.Description("Page title (cannot contain: ?*\\/:<>|&#''%%~)")),
		mcp.WithString("content", mcp.Required(), mcp.Description("Content for the page (HTML, Markdown, or plain text - automatically detected and converted)")),
	)
	createPageHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

		// Detect format and convert content to HTML
		convertedHTML, detectedFormat := utils.ConvertToHTML(content)
		logging.ToolsLogger.Debug("createPage content format detection",
			"detected_format", detectedFormat.String(),
			"original_length", len(content),
			"converted_length", len(convertedHTML))

		result, err := pageClient.CreatePage(sectionID, title, convertedHTML)
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
			"success":         true,
			"pageID":          pageID,
			"detected_format": detectedFormat.String(),
			"content_length":  len(content),
			"html_length":     len(convertedHTML),
		}

		jsonBytes, err := json.Marshal(response)
		if err != nil {
			logging.ToolsLogger.Error("createPage failed to marshal response", "error", err)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal response: %v", err)), nil
		}

		elapsed := time.Since(startTime)
		logging.ToolsLogger.Debug("createPage operation completed", "duration", elapsed, "page_id", pageID)
		return mcp.NewToolResultText(string(jsonBytes)), nil
	}
	s.AddTool(createPageTool, server.ToolHandlerFunc(authorization.AuthorizedToolHandler("createPage", createPageHandler, authConfig, cache, quickNoteConfig)))

	// updatePageContent: Update the HTML content of a page
	updatePageContentTool := mcp.NewTool(
		"updatePageContent",
		mcp.WithDescription(resources.MustGetToolDescription("updatePageContent")),
		mcp.WithString("pageID", mcp.Required(), mcp.Description("Page ID to update")),
		mcp.WithString("content", mcp.Required(), mcp.Description("New content for the page (HTML, Markdown, or plain text - automatically detected and converted)")),
	)
	updatePageContentHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

		// Detect format and convert content to HTML
		convertedHTML, detectedFormat := utils.ConvertToHTML(content)
		logging.ToolsLogger.Debug("updatePageContent content format detection",
			"detected_format", detectedFormat.String(),
			"original_length", len(content),
			"converted_length", len(convertedHTML))

		err = pageClient.UpdatePageContentSimple(pageID, convertedHTML)
		if err != nil {
			logging.ToolsLogger.Error("updatePageContent operation failed", "page_id", pageID, "error", err, "operation", "updatePageContent")
			return mcp.NewToolResultError(fmt.Sprintf("Failed to update page content: %v", err)), nil
		}

		elapsed := time.Since(startTime)
		logging.ToolsLogger.Debug("updatePageContent operation completed", "duration", elapsed)
		
		response := map[string]interface{}{
			"success":         true,
			"message":         "Page content updated successfully",
			"detected_format": detectedFormat.String(),
			"content_length":  len(content),
			"html_length":     len(convertedHTML),
		}

		jsonBytes, err := json.Marshal(response)
		if err != nil {
			logging.ToolsLogger.Error("updatePageContent failed to marshal response", "error", err)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonBytes)), nil
	}
	s.AddTool(updatePageContentTool, server.ToolHandlerFunc(authorization.AuthorizedToolHandlerWithResolver("updatePageContent", updatePageContentHandler, authConfig, cache, quickNoteConfig, pageClient)))

	// updatePageContentAdvanced: Update page content with advanced commands
	updatePageContentAdvancedTool := mcp.NewTool(
		"updatePageContentAdvanced",
		mcp.WithDescription(resources.MustGetToolDescription("updatePageContentAdvanced")),
		mcp.WithString("pageID", mcp.Required(), mcp.Description("Page ID to update")),
		mcp.WithString("commands", mcp.Required(), mcp.Description("JSON STRING containing an array of command objects. MUST be a string, not an array. Content in commands supports HTML, Markdown, or plain text (automatically detected and converted). Example: \"[{\\\"target\\\": \\\"body\\\", \\\"action\\\": \\\"append\\\", \\\"content\\\": \\\"# Header\\n- Item 1\\\"}]\"")),
	)
	updatePageContentAdvancedHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

		// Apply format detection and conversion to each command's content
		var formatDetectionResults []map[string]interface{}
		for i, command := range commands {
			if command.Content != "" {
				originalContent := command.Content
				convertedHTML, detectedFormat := utils.ConvertToHTML(command.Content)
				commands[i].Content = convertedHTML

				// Track format detection results
				formatDetectionResults = append(formatDetectionResults, map[string]interface{}{
					"command_index":    i,
					"target":          command.Target,
					"action":          command.Action,
					"detected_format": detectedFormat.String(),
					"original_length": len(originalContent),
					"html_length":     len(convertedHTML),
				})

				logging.ToolsLogger.Debug("updatePageContentAdvanced command content format detection",
					"command_index", i,
					"target", command.Target,
					"action", command.Action,
					"detected_format", detectedFormat.String(),
					"original_length", len(originalContent),
					"converted_length", len(convertedHTML))
			}
		}

		err = pageClient.UpdatePageContent(pageID, commands)
		if err != nil {
			logging.ToolsLogger.Error("updatePageContentAdvanced operation failed", "page_id", pageID, "error", err, "operation", "updatePageContentAdvanced")
			return mcp.NewToolResultError(fmt.Sprintf("Failed to update page content: %v", err)), nil
		}

		elapsed := time.Since(startTime)
		logging.ToolsLogger.Debug("updatePageContentAdvanced operation completed", "duration", elapsed)
		
		response := map[string]interface{}{
			"success":                true,
			"message":               "Page content updated successfully with advanced commands",
			"commands_processed":    len(commands),
			"format_detection":      formatDetectionResults,
		}

		jsonBytes, err := json.Marshal(response)
		if err != nil {
			logging.ToolsLogger.Error("updatePageContentAdvanced failed to marshal response", "error", err)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonBytes)), nil
	}
	s.AddTool(updatePageContentAdvancedTool, server.ToolHandlerFunc(authorization.AuthorizedToolHandlerWithResolver("updatePageContentAdvanced", updatePageContentAdvancedHandler, authConfig, cache, quickNoteConfig, pageClient)))

	// deletePage: Delete a page by ID
	deletePageTool := mcp.NewTool(
		"deletePage",
		mcp.WithDescription(resources.MustGetToolDescription("deletePage")),
		mcp.WithString("pageID", mcp.Required(), mcp.Description("Page ID to delete")),
	)
	deletePageHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	}
	s.AddTool(deletePageTool, server.ToolHandlerFunc(authorization.AuthorizedToolHandlerWithResolver("deletePage", deletePageHandler, authConfig, cache, quickNoteConfig, pageClient)))

	// getPageContent: Get the HTML content of a page by ID
	getPageContentTool := mcp.NewTool(
		"getPageContent",
		mcp.WithDescription(resources.MustGetToolDescription("getPageContent")),
		mcp.WithString("pageID", mcp.Required(), mcp.Description("Page ID to fetch content for")),
		mcp.WithString("forUpdate", mcp.Description("Optional: set to 'true' to include includeIDs=true parameter for update operations")),
		mcp.WithString("format", mcp.Description("Optional: output format - 'HTML' (default), 'Markdown', or 'Text'. Note: forUpdate only works with HTML format.")),
	)
	getPageContentHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		startTime := time.Now()
		logging.ToolsLogger.Info("Retrieving OneNote page content", "operation", "getPageContent", "type", "tool_invocation")

		pageID, err := req.RequireString("pageID")
		if err != nil {
			logging.ToolsLogger.Error("getPageContent missing pageID", "error", err)
			return mcp.NewToolResultError("pageID is required"), nil
		}

		forUpdateStr := req.GetString("forUpdate", "")
		forUpdate := forUpdateStr == "true"
		
		formatStr := req.GetString("format", "HTML")
		format := strings.ToUpper(strings.TrimSpace(formatStr))
		
		// Validate format parameter
		validFormats := []string{"HTML", "MARKDOWN", "TEXT"}
		isValidFormat := false
		for _, validFormat := range validFormats {
			if format == validFormat {
				isValidFormat = true
				break
			}
		}
		if !isValidFormat {
			logging.ToolsLogger.Error("getPageContent invalid format", "format", format, "valid_formats", validFormats)
			return mcp.NewToolResultError(fmt.Sprintf("Invalid format '%s'. Valid formats are: HTML, Markdown, Text", formatStr)), nil
		}
		
		// Check for incompatible parameter combinations
		if forUpdate && format != "HTML" {
			logging.ToolsLogger.Error("getPageContent forUpdate incompatible with non-HTML format", "forUpdate", forUpdate, "format", format)
			return mcp.NewToolResultError("forUpdate parameter can only be used with HTML format (for page updates)"), nil
		}

		logging.ToolsLogger.Debug("getPageContent parameters", "pageID", pageID, "forUpdate", forUpdate, "format", format)

		// Always get HTML content first
		htmlContent, err := pageClient.GetPageContent(pageID, forUpdate)
		if err != nil {
			logging.ToolsLogger.Error("getPageContent operation failed", "page_id", pageID, "error", err, "operation", "getPageContent")
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get page content: %v", err)), nil
		}

		var finalContent string
		var conversionError error

		// Convert content based on requested format
		switch format {
		case "HTML":
			finalContent = htmlContent
			logging.ToolsLogger.Debug("getPageContent returning HTML content as-is")
		case "MARKDOWN":
			logging.ToolsLogger.Debug("getPageContent converting HTML to Markdown")
			finalContent, conversionError = utils.ConvertHTMLToMarkdown(htmlContent)
			if conversionError != nil {
				logging.ToolsLogger.Error("getPageContent HTML to Markdown conversion failed", "error", conversionError)
				return mcp.NewToolResultError(fmt.Sprintf("Failed to convert content to Markdown: %v", conversionError)), nil
			}
		case "TEXT":
			logging.ToolsLogger.Debug("getPageContent converting HTML to plain text")
			finalContent, conversionError = utils.ConvertHTMLToText(htmlContent)
			if conversionError != nil {
				logging.ToolsLogger.Error("getPageContent HTML to plain text conversion failed", "error", conversionError)
				return mcp.NewToolResultError(fmt.Sprintf("Failed to convert content to plain text: %v", conversionError)), nil
			}
		}

		elapsed := time.Since(startTime)
		logging.ToolsLogger.Debug("getPageContent operation completed", 
			"duration", elapsed, 
			"original_html_length", len(htmlContent), 
			"final_content_length", len(finalContent),
			"output_format", format)
		
		// Create response with format information
		response := map[string]interface{}{
			"content":            finalContent,
			"format":            format,
			"original_html_length": len(htmlContent),
			"final_content_length": len(finalContent),
		}
		
		// Add forUpdate info if applicable
		if forUpdate {
			response["for_update"] = true
		}

		jsonBytes, err := json.Marshal(response)
		if err != nil {
			logging.ToolsLogger.Error("getPageContent failed to marshal response", "error", err)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonBytes)), nil
	}
	s.AddTool(getPageContentTool, server.ToolHandlerFunc(authorization.AuthorizedToolHandlerWithResolver("getPageContent", getPageContentHandler, authConfig, cache, quickNoteConfig, pageClient)))

	// getPageItemContent: Get a OneNote page item (e.g., image) by page item ID, returns binary data with proper MIME type
	getPageItemContentTool := mcp.NewTool(
		"getPageItemContent",
		mcp.WithDescription(resources.MustGetToolDescription("getPageItemContent")),
		mcp.WithString("pageID", mcp.Required(), mcp.Description("Page ID to fetch item from")),
		mcp.WithString("pageItemID", mcp.Required(), mcp.Description("Page resource item ID to fetch")),
		mcp.WithString("filename", mcp.Description("Custom filename for binary download")),
		mcp.WithBoolean("fullSize", mcp.Description("Skip image scaling and return original size (default: false = scale images to 1024x768 max)")),
	)
	getPageItemContentHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	}
	s.AddTool(getPageItemContentTool, server.ToolHandlerFunc(authorization.AuthorizedToolHandlerWithResolver("getPageItemContent", getPageItemContentHandler, authConfig, cache, quickNoteConfig, pageClient)))

	// listPageItems: List all OneNote page items (images, files, etc.) for a specific page
	listPageItemsTool := mcp.NewTool(
		"listPageItems",
		mcp.WithDescription(resources.MustGetToolDescription("listPageItems")),
		mcp.WithString("pageID", mcp.Required(), mcp.Description("Page ID to list items for")),
	)
	listPageItemsHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	}
	s.AddTool(listPageItemsTool, server.ToolHandlerFunc(authorization.AuthorizedToolHandlerWithResolver("listPageItems", listPageItemsHandler, authConfig, cache, quickNoteConfig, pageClient)))

	// copyPage: Copy a page from one section to another
	copyPageTool := mcp.NewTool(
		"copyPage",
		mcp.WithDescription(resources.MustGetToolDescription("copyPage")),
		mcp.WithString("pageID", mcp.Required(), mcp.Description("Page ID to copy")),
		mcp.WithString("targetSectionID", mcp.Required(), mcp.Description("Target section ID to copy the page to")),
	)
	copyPageHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	}
	s.AddTool(copyPageTool, server.ToolHandlerFunc(authorization.AuthorizedToolHandlerWithResolver("copyPage", copyPageHandler, authConfig, cache, quickNoteConfig, pageClient)))

	// movePage: Move a page from one section to another (copy then delete)
	movePageTool := mcp.NewTool(
		"movePage",
		mcp.WithDescription(resources.MustGetToolDescription("movePage")),
		mcp.WithString("pageID", mcp.Required(), mcp.Description("Page ID to move")),
		mcp.WithString("targetSectionID", mcp.Required(), mcp.Description("Target section ID to move the page to")),
	)
	movePageHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	}
	s.AddTool(movePageTool, server.ToolHandlerFunc(authorization.AuthorizedToolHandlerWithResolver("movePage", movePageHandler, authConfig, cache, quickNoteConfig, pageClient)))

	// quickNote: Add a timestamped note to a configured page
	quickNoteTool := mcp.NewTool(
		"quickNote",
		mcp.WithDescription("Add a timestamped note to a configured page. Uses quicknote settings from configuration file to determine target notebook, page, and date format. Appends content to the page with current timestamp."),
		mcp.WithString("content", mcp.Required(), mcp.Description("Content to add to the quicknote page")),
	)
	quickNoteHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		startTime := time.Now()
		logging.ToolsLogger.Info("Adding quicknote entry", "operation", "quickNote", "type", "tool_invocation")

		// Extract progress token from request metadata for progress notifications
		var progressToken string
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
					progressToken = fmt.Sprintf("%d", int64(token))
					logging.ToolsLogger.Debug("Progress token extracted as whole number from float64", "progress_token", progressToken, "original_value", token)
				} else {
					progressToken = fmt.Sprintf("%.0f", token)
					logging.ToolsLogger.Debug("Progress token extracted as rounded float64", "progress_token", progressToken, "original_value", token)
				}
			default:
				logging.ToolsLogger.Debug("Progress token has unsupported type, converting to string", "type", fmt.Sprintf("%T", token), "value", token)
				progressToken = fmt.Sprintf("%v", token)
			}
		} else {
			logging.ToolsLogger.Debug("No progress token found in request metadata")
		}

		content, err := req.RequireString("content")
		if err != nil {
			logging.ToolsLogger.Error("quickNote missing content", "error", err)
			return mcp.NewToolResultError("content is required"), nil
		}

		// Check if quicknote is configured
		if cfg.QuickNote == nil {
			logging.ToolsLogger.Error("quickNote not configured")
			return mcp.NewToolResultError("QuickNote is not configured. Please set page_name and optionally notebook_name and date_format in your configuration."), nil
		}

		if cfg.QuickNote.PageName == "" {
			logging.ToolsLogger.Error("quickNote page name not configured", "page", cfg.QuickNote.PageName)
			return mcp.NewToolResultError("QuickNote page_name is required in quicknote configuration."), nil
		}

		// Determine target notebook name - use quicknote-specific notebook or fall back to default
		targetNotebookName := cfg.QuickNote.NotebookName
		if targetNotebookName == "" {
			targetNotebookName = cfg.NotebookName
			logging.ToolsLogger.Debug("quickNote using default notebook name", "default_notebook", targetNotebookName)
		}

		if targetNotebookName == "" {
			logging.ToolsLogger.Error("quickNote no notebook name available", "quicknote_notebook", cfg.QuickNote.NotebookName, "default_notebook", cfg.NotebookName)
			return mcp.NewToolResultError("No notebook name configured. Please set either quicknote.notebook_name or notebook_name in your configuration."), nil
		}

		logging.ToolsLogger.Debug("quickNote parameters",
			"content_length", len(content),
			"target_notebook", targetNotebookName,
			"quicknote_notebook_config", cfg.QuickNote.NotebookName,
			"default_notebook_config", cfg.NotebookName,
			"target_page", cfg.QuickNote.PageName,
			"date_format", cfg.QuickNote.DateFormat)

		// Send initial progress notification
		sendProgressNotification(s, ctx, progressToken, 10, 100, "Starting quicknote operation...")

		// Find the notebook by name using cached lookup
		notebookClient := notebooks.NewNotebookClient(graphClient)
		targetNotebook, fromCache, err := getDetailedNotebookByNameCached(notebookClient, notebookCache, targetNotebookName)
		if err != nil {
			logging.ToolsLogger.Error("quickNote failed to find target notebook", "notebook_name", targetNotebookName, "error", err)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to find notebook '%s': %v", targetNotebookName, err)), nil
		}
		
		// Only send progress notification if we actually made an API call
		if fromCache {
			sendProgressNotification(s, ctx, progressToken, 25, 100, fmt.Sprintf("Found notebook in cache: %s", targetNotebookName))
		} else {
			sendProgressNotification(s, ctx, progressToken, 25, 100, fmt.Sprintf("Found notebook via API: %s", targetNotebookName))
		}

		// Find the page within the notebook
		sectionClient := sections.NewSectionClient(graphClient)
		notebookID, ok := targetNotebook["id"].(string)
		if !ok {
			logging.ToolsLogger.Error("quickNote notebook ID not found in response")
			return mcp.NewToolResultError("Notebook ID not found in response"), nil
		}

		// Progress notification will be sent based on cache status inside findPageInNotebookWithCache
		targetPage, targetSectionID, pageFromCache, err := findPageInNotebookWithCache(pageClient, sectionClient, notebookCache, s, notebookID, cfg.QuickNote.PageName, progressToken, ctx)
		if err != nil {
			logging.ToolsLogger.Error("quickNote failed to find target page", "page_name", cfg.QuickNote.PageName, "notebook_id", notebookID, "error", err)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to find page '%s' in notebook '%s': %v", cfg.QuickNote.PageName, targetNotebookName, err)), nil
		}
		
		// Adjust progress based on cache usage
		if pageFromCache {
			sendProgressNotification(s, ctx, progressToken, 75, 100, "Found page in cache, preparing content...")
		} else {
			sendProgressNotification(s, ctx, progressToken, 75, 100, "Found page via search, preparing content...")
		}

		// Format the current time using the configured date format
		currentTime := time.Now()
		formattedDate := currentTime.Format(cfg.QuickNote.DateFormat)

		// Detect format and convert content to HTML
		convertedHTML, detectedFormat := utils.ConvertToHTML(content)
		logging.ToolsLogger.Debug("quickNote content format detection",
			"detected_format", detectedFormat.String(),
			"original_length", len(content),
			"converted_length", len(convertedHTML))

		// Create HTML content with timestamp header and converted content
		htmlContent := fmt.Sprintf(`<h3>%s</h3>%s`, formattedDate, convertedHTML)

		// Create update command to append to the body
		commands := []pages.UpdateCommand{
			{
				Target:  "body",
				Action:  "append",
				Content: htmlContent,
			},
		}

		// Update the page content
		sendProgressNotification(s, ctx, progressToken, 80, 100, "Adding timestamped content to page...")
		pageID, ok := targetPage["pageId"].(string)
		if !ok {
			logging.ToolsLogger.Error("quickNote page ID not found in response")
			return mcp.NewToolResultError("Page ID not found in response"), nil
		}

		err = pageClient.UpdatePageContent(pageID, commands)
		if err != nil {
			logging.ToolsLogger.Error("quickNote failed to update page content", "page_id", pageID, "error", err)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to add quicknote to page: %v", err)), nil
		}

		// Clear pages cache for this section and page search cache
		sendProgressNotification(s, ctx, progressToken, 90, 100, "Clearing cache and finalizing...")
		notebookCache.ClearPagesCache(targetSectionID)
		notebookCache.ClearAllPageSearchCache() // Clear page search cache since page content changed
		// Note: We don't clear notebook lookup cache as notebook metadata shouldn't change from page updates
		logging.ToolsLogger.Debug("quickNote cleared pages cache and page search cache", "section_id", targetSectionID)

		elapsed := time.Since(startTime)
		logging.ToolsLogger.Info("quickNote operation completed",
			"duration", elapsed,
			"notebook", targetNotebookName,
			"page", cfg.QuickNote.PageName,
			"timestamp", formattedDate,
			"content_length", len(content))

		response := map[string]interface{}{
			"success":         true,
			"timestamp":       formattedDate,
			"notebook":        targetNotebookName,
			"page":            cfg.QuickNote.PageName,
			"message":         "Quicknote added successfully",
			"detected_format": detectedFormat.String(),
			"content_length":  len(content),
			"html_length":     len(convertedHTML),
		}

		jsonBytes, err := json.Marshal(response)
		if err != nil {
			logging.ToolsLogger.Error("quickNote failed to marshal response", "error", err)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal response: %v", err)), nil
		}

		// Send final progress notification
		sendProgressNotification(s, ctx, progressToken, 100, 100, "Quicknote added successfully!")

		return mcp.NewToolResultText(string(jsonBytes)), nil
	}
	s.AddTool(quickNoteTool, server.ToolHandlerFunc(authorization.AuthorizedToolHandler("quickNote", quickNoteHandler, authConfig, cache, quickNoteConfig)))

	// NOTE: getOnenoteOperation tool is currently commented out in the original code
	// Uncomment this section if you want to enable it
	/*
		// getOnenoteOperation: Get the status of an asynchronous OneNote operation
		getOnenoteOperationTool := mcp.NewTool(
			"getOnenoteOperation",
			mcp.WithDescription("Check the status of an asynchronous OneNote operation (e.g., copy/move operations)."),
			mcp.WithString("operationId", mcp.Required(), mcp.Description("Operation ID to check status for")),
		)
		getOnenoteOperationHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
		}
		s.AddTool(getOnenoteOperationTool, server.ToolHandlerFunc(authorization.AuthorizedToolHandler("getOnenoteOperation", getOnenoteOperationHandler, authConfig, cache, quickNoteConfig)))
	*/

	logging.ToolsLogger.Debug("Page tools registered successfully")
}


// findPageInNotebookWithCache searches for a page by name using cached listPages calls
// This leverages the existing page cache and progress notification system
// Returns the page data, section ID where the page was found, whether result was from cache, and any error
func findPageInNotebookWithCache(pageClient *pages.PageClient, sectionClient *sections.SectionClient, notebookCache *NotebookCache, s *server.MCPServer, notebookID string, pageName string, progressToken string, ctx context.Context) (map[string]interface{}, string, bool, error) {
	logging.ToolsLogger.Debug("Searching for page in notebook using cached listPages", "notebook_id", notebookID, "page_name", pageName)

	// Check if page search results are already cached
	if cachedResult, cached := notebookCache.GetPageSearchCache(notebookID, pageName); cached {
		logging.ToolsLogger.Debug("Found cached page search result", 
			"notebook_id", notebookID, 
			"page_name", pageName,
			"found", cachedResult.Found)
		
		if cachedResult.Found {
			sendProgressNotification(s, ctx, progressToken, 30, 100, "Page found in search cache")
			logging.ToolsLogger.Info("Found target page from search cache",
				"page_title", pageName,
				"section_id", cachedResult.SectionID)
			return cachedResult.Page, cachedResult.SectionID, true, nil
		} else {
			// Cached result indicates page was not found
			sendProgressNotification(s, ctx, progressToken, 30, 100, "Page not found (from cache)")
			return nil, "", true, fmt.Errorf("page '%s' not found in notebook (from cache)", pageName)
		}
	}

	logging.ToolsLogger.Debug("No cached search result found, performing fresh search")
	sendProgressNotification(s, ctx, progressToken, 30, 100, "Getting notebook sections for page search...")

	// Create progress callback for section listing (30-35%)
	sectionProgressCallback := func(progress int, message string) {
		adjustedProgress := 30 + (progress * 5 / 100) // Map 0-100% to 30-35%
		sendProgressNotification(s, ctx, progressToken, adjustedProgress, 100, fmt.Sprintf("Sections: %s", message))
	}

	// Get all sections in the notebook using progress-aware method
	sections, err := sectionClient.ListSectionsWithProgress(notebookID, sectionProgressCallback)
	if err != nil {
		return nil, "", false, fmt.Errorf("failed to list sections in notebook: %v", err)
	}

	totalSections := len(sections)
	logging.ToolsLogger.Debug("Found sections in notebook for cached search", "notebook_id", notebookID, "sections_count", totalSections)

	sendProgressNotification(s, ctx, progressToken, 35, 100, fmt.Sprintf("Found %d sections, starting page search...", totalSections))

	// Search through each section using cached listPages calls with progress
	for i, section := range sections {
		sectionID, ok := section["id"].(string)
		if !ok {
			logging.ToolsLogger.Debug("Skipping section with invalid ID in cached search", "section_index", i+1)
			continue // Skip if ID is not available
		}

		sectionName, ok := section["displayName"].(string)
		if !ok {
			sectionName = "unknown"
		}

		// Calculate progress for this section (35-70% range)
		baseProgress := 35 + int(float64(i)/float64(totalSections)*35) // Distribute 35% across sections
		sendProgressNotification(s, ctx, progressToken, baseProgress, 100, fmt.Sprintf("Searching section %d of %d: %s", i+1, totalSections, sectionName))

		logging.ToolsLogger.Debug("Searching in section using cached listPages",
			"section_index", i+1,
			"total_sections", totalSections,
			"section_id", sectionID,
			"section_name", sectionName)

		// Use the same cache logic as the listPages tool
		var pages []map[string]interface{}
		var fromCache bool

		if notebookCache.IsPagesCached(sectionID) {
			cachedPages, cached := notebookCache.GetPagesCache(sectionID)
			if cached {
				pages = cachedPages
				fromCache = true
				sendProgressNotification(s, ctx, progressToken, baseProgress+1, 100, fmt.Sprintf("Using cached pages for section: %s", sectionName))
				logging.ToolsLogger.Debug("Using cached pages for section",
					"section_id", sectionID,
					"section_name", sectionName,
					"pages_count", len(pages))
			}
		}

		if !fromCache {
			// Cache miss - fetch from API with progress
			logging.ToolsLogger.Debug("Cache miss, fetching pages from API with progress",
				"section_id", sectionID,
				"section_name", sectionName)

			// Create progress callback for page listing within section
			pageProgressCallback := func(progress int, message string) {
				// Map page progress to a small portion of overall progress
				adjustedProgress := baseProgress + (progress * 2 / 100) // Allow 2% per section for page loading
				if adjustedProgress > baseProgress + 2 {
					adjustedProgress = baseProgress + 2
				}
				sendProgressNotification(s, ctx, progressToken, adjustedProgress, 100, fmt.Sprintf("Section %s: %s", sectionName, message))
			}

			pagesFromAPI, err := pageClient.ListPagesWithProgress(sectionID, pageProgressCallback)
			if err != nil {
				logging.ToolsLogger.Warn("Failed to list pages in section during cached search",
					"section_id", sectionID,
					"section_name", sectionName,
					"error", err)
				continue // Continue searching other sections
			}
			pages = pagesFromAPI
			// Cache the results
			notebookCache.SetPagesCache(sectionID, pages)
			logging.ToolsLogger.Debug("Cached fresh pages data",
				"section_id", sectionID,
				"section_name", sectionName,
				"pages_count", len(pages))
		}

		// Extract page names for logging
		var pageNames []string
		for _, page := range pages {
			if pageTitle, ok := page["title"].(string); ok {
				pageNames = append(pageNames, pageTitle)
			} else {
				pageNames = append(pageNames, "<invalid title>")
			}
		}

		logging.ToolsLogger.Debug("Found pages in section (cached search)",
			"section_id", sectionID,
			"section_name", sectionName,
			"pages_count", len(pages),
			"page_names", pageNames,
			"from_cache", fromCache)

		// Look for the page by name
		for j, page := range pages {
			pageTitle, ok := page["title"].(string)
			if !ok {
				logging.ToolsLogger.Debug("Skipping page with invalid title in cached search",
					"section_name", sectionName,
					"page_index", j+1)
				continue // Skip if title is not available
			}

			logging.ToolsLogger.Debug("Examining page in cached search",
				"section_name", sectionName,
				"page_index", j+1,
				"total_pages", len(pages),
				"page_title", pageTitle,
				"target_page", pageName,
				"title_match", pageTitle == pageName,
				"from_cache", fromCache)

			if pageTitle == pageName {
				pageID, ok := page["pageId"].(string)
				if !ok {
					logging.ToolsLogger.Debug("Found matching page but invalid ID in cached search",
						"page_title", pageTitle,
						"section_name", sectionName)
					continue // Skip if ID is not available
				}

				// Cache the successful search result
				searchResult := PageSearchResult{
					Page:      page,
					SectionID: sectionID,
					Found:     true,
				}
				notebookCache.SetPageSearchCache(notebookID, pageName, searchResult)
				logging.ToolsLogger.Debug("Cached successful page search result",
					"notebook_id", notebookID,
					"page_name", pageName,
					"section_id", sectionID)

				// Send completion notification if available
				sendProgressNotification(s, ctx, progressToken, 75, 100, fmt.Sprintf("Page found in section: %s", sectionName))

				logging.ToolsLogger.Info("Found target page using cached search",
					"page_id", pageID,
					"page_title", pageTitle,
					"section_id", sectionID,
					"section_name", sectionName,
					"from_cache", fromCache)
				return page, sectionID, false, nil
			}
		}

		logging.ToolsLogger.Debug("Page not found in section (cached search)",
			"section_name", sectionName,
			"target_page", pageName,
			"searched_pages", len(pages),
			"from_cache", fromCache)
	}

	// Cache the failed search result
	searchResult := PageSearchResult{
		Page:      nil,
		SectionID: "",
		Found:     false,
	}
	notebookCache.SetPageSearchCache(notebookID, pageName, searchResult)
	logging.ToolsLogger.Debug("Cached failed page search result",
		"notebook_id", notebookID,
		"page_name", pageName)

	return nil, "", false, fmt.Errorf("page '%s' not found in notebook", pageName)
}

// getDetailedNotebookByNameCached retrieves comprehensive notebook information by display name with caching
// This is a cached wrapper around notebooks.GetDetailedNotebookByName
// Returns the notebook data and a boolean indicating if it was from cache
func getDetailedNotebookByNameCached(notebookClient *notebooks.NotebookClient, notebookCache *NotebookCache, notebookName string) (map[string]interface{}, bool, error) {
	logging.ToolsLogger.Debug("Getting detailed notebook by name with caching", "notebook_name", notebookName)

	// Check if notebook lookup is already cached
	if cachedNotebook, cached := notebookCache.GetNotebookLookupCache(notebookName); cached {
		logging.ToolsLogger.Debug("Found cached notebook lookup result", 
			"notebook_name", notebookName,
			"cached_notebook_id", cachedNotebook["id"],
			"cached_notebook_data", cachedNotebook)
		logging.ToolsLogger.Info("Found notebook from lookup cache",
			"notebook_name", notebookName,
			"notebook_id", cachedNotebook["id"])
		return cachedNotebook, true, nil
	}

	logging.ToolsLogger.Debug("No cached notebook lookup found, performing fresh lookup")

	// Cache miss - fetch from API
	notebook, err := notebookClient.GetDetailedNotebookByName(notebookName)
	if err != nil {
		logging.ToolsLogger.Debug("Failed to get notebook by name", "notebook_name", notebookName, "error", err)
		return nil, false, err
	}

	// Log the fresh notebook data before caching
	logging.ToolsLogger.Debug("Fresh notebook lookup result", 
		"notebook_name", notebookName,
		"fresh_notebook_id", notebook["id"],
		"fresh_notebook_data", notebook)

	// Cache the successful result
	notebookCache.SetNotebookLookupCache(notebookName, notebook)
	logging.ToolsLogger.Debug("Cached notebook lookup result", "notebook_name", notebookName, "notebook_id", notebook["id"])

	return notebook, false, nil
}

// Helper functions for progress notifications and section population

// extractProgressToken extracts progress token from MCP request for notifications

// populateSectionsForAuthorization fetches sections to populate cache for authorization context
func populateSectionsForAuthorization(s *server.MCPServer, ctx context.Context, graphClient *graph.Client, notebookCache *NotebookCache, progressToken string) error {
	// Check if notebook is selected
	notebookID, isSet := notebookCache.GetNotebookID()
	if !isSet {
		return fmt.Errorf("no notebook selected")
	}
	
	logging.ToolsLogger.Debug("Populating sections for authorization context", 
		"notebook_id", notebookID,
		"has_progress_token", progressToken != "")
	
	// Create section client for fetching sections
	sectionClient := sections.NewSectionClient(graphClient)
	
	// Get sections with progress notifications
	if progressToken != "" {
		sendProgressNotification(s, ctx, progressToken, 15, 100, "Fetching notebook sections for authorization...")
	}
	
	// Create a progress context with typed keys (compatible with fetchAllNotebookContentWithProgress)
	progressCtx := context.WithValue(ctx, mcpServerKey, s)
	progressCtx = context.WithValue(progressCtx, progressTokenKey, progressToken)
	
	if progressToken != "" {
		sendProgressNotification(s, ctx, progressToken, 18, 100, "Calling sections API...")
	}
	
	// Fetch all sections and section groups recursively using the same function as getNotebookSections
	sectionItems, err := fetchAllNotebookContentWithProgress(sectionClient, notebookID, progressCtx)
	if err != nil {
		logging.ToolsLogger.Warn("Failed to fetch sections for authorization context", "error", err)
		return err
	}
	
	if progressToken != "" {
		sendProgressNotification(s, ctx, progressToken, 22, 100, "Processing section tree structure...")
	}
	
	// Create sections tree structure compatible with cache format
	sectionsTreeStructure := map[string]interface{}{
		"sections": sectionItems,
	}
	
	if progressToken != "" {
		sendProgressNotification(s, ctx, progressToken, 24, 100, "Caching section information...")
	}
	
	// Cache the sections tree
	notebookCache.SetSectionsTree(sectionsTreeStructure)
	
	if progressToken != "" {
		sendProgressNotification(s, ctx, progressToken, 25, 100, "Sections cached for authorization context")
	}
	
	logging.ToolsLogger.Debug("Successfully populated sections for authorization context", "sections_count", len(sectionItems))
	return nil
}
