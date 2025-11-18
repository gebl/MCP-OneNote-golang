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

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/gebl/onenote-mcp-server/internal/authorization"
	"github.com/gebl/onenote-mcp-server/internal/config"
	"github.com/gebl/onenote-mcp-server/internal/graph"
	"github.com/gebl/onenote-mcp-server/internal/notebooks"
	"github.com/gebl/onenote-mcp-server/internal/pages"
	"github.com/gebl/onenote-mcp-server/internal/resources"
	"github.com/gebl/onenote-mcp-server/internal/sections"
	"github.com/gebl/onenote-mcp-server/internal/utils"
)

// Input/Output structs for page tools
type PagesInput struct {
	SectionID string `json:"sectionID" jsonschema:"Section ID - MUST be actual ID, NOT a section name. You must obtain the section ID through other means."`
}
type PagesOutput struct {
	Pages       []map[string]interface{} `json:"pages"`
	Cached      bool                     `json:"cached"`
	CacheHit    bool                     `json:"cache_hit"`
	PagesCount  int                      `json:"pages_count"`
	Duration    string                   `json:"duration"`
}

type PageCreateInput struct {
	SectionID   string `json:"sectionID" jsonschema:"Section ID to create page in"`
	Title       string `json:"title" jsonschema:"Page title (cannot contain: ?*\\/:<>|&#''%%~)"`
	Content     string `json:"content" jsonschema:"Content for the page (HTML, Markdown, or plain text - automatically detected and converted)"`
	ContentType string `json:"contentType,omitempty" jsonschema:"Optional: content type - 'text', 'html', or 'markdown'. If specified, automatic format detection is bypassed and content is converted accordingly"`
}
type PageCreateOutput struct {
	Success        bool   `json:"success"`
	PageID         string `json:"pageID"`
	Format         string `json:"format"`
	FormatSource   string `json:"format_source"`
	ContentLength  int    `json:"content_length"`
	HTMLLength     int    `json:"html_length"`
	SpecifiedType  string `json:"specified_type,omitempty"`
}

type PageUpdateInput struct {
	PageID      string `json:"pageID" jsonschema:"Page ID to update"`
	Content     string `json:"content" jsonschema:"New content for the page (HTML, Markdown, or plain text - automatically detected and converted)"`
	ContentType string `json:"contentType,omitempty" jsonschema:"Optional: content type - 'text', 'html', or 'markdown'. If specified, automatic format detection is bypassed and content is converted accordingly"`
}
type PageUpdateOutput struct {
	Success        bool   `json:"success"`
	Message        string `json:"message"`
	Format         string `json:"format"`
	FormatSource   string `json:"format_source"`
	ContentLength  int    `json:"content_length"`
	HTMLLength     int    `json:"html_length"`
	SpecifiedType  string `json:"specified_type,omitempty"`
}

type PageUpdateAdvancedInput struct {
	PageID      string `json:"pageID" jsonschema:"Page ID to update"`
	Commands    string `json:"commands" jsonschema:"JSON STRING containing an array of command objects. MUST be a string, not an array. Content in commands supports HTML, Markdown, or plain text (automatically detected and converted). Example: \\\"[{\\\\\\\"target\\\\\\\": \\\\\\\"body\\\\\\\", \\\\\\\"action\\\\\\\": \\\\\\\"append\\\\\\\", \\\\\\\"content\\\\\\\": \\\\\\\"# Header\\\\n- Item 1\\\\\\\"}]\\\\\\\""`
	ContentType string `json:"contentType,omitempty" jsonschema:"Optional: content type for all commands - 'text', 'html', or 'markdown'. If specified, automatic format detection is bypassed for all command content and converted accordingly"`
}
type PageUpdateAdvancedOutput struct {
	Success             bool                     `json:"success"`
	Message             string                   `json:"message"`
	CommandsProcessed   int                      `json:"commands_processed"`
	FormatDetection     []map[string]interface{} `json:"format_detection"`
}

type PageDeleteInput struct {
	PageID string `json:"pageID" jsonschema:"Page ID to delete"`
}
type PageDeleteOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type PageContentInput struct {
	PageID    string `json:"pageID" jsonschema:"Page ID to fetch content for"`
	ForUpdate string `json:"forUpdate,omitempty" jsonschema:"Optional: set to 'true' to include includeIDs=true parameter for update operations"`
	Format    string `json:"format,omitempty" jsonschema:"Optional: output format - 'HTML' (default), 'Markdown', or 'Text'. Note: forUpdate only works with HTML format."`
}
type PageContentOutput struct {
	Content             string `json:"content"`
	Format              string `json:"format"`
	OriginalHTMLLength  int    `json:"original_html_length"`
	FinalContentLength  int    `json:"final_content_length"`
	ForUpdate           bool   `json:"for_update,omitempty"`
}

type PageItemContentInput struct {
	PageID     string `json:"pageID" jsonschema:"Page ID to fetch item from"`
	PageItemID string `json:"pageItemID" jsonschema:"Page resource item ID to fetch"`
	Filename   string `json:"filename,omitempty" jsonschema:"Custom filename for binary download"`
	FullSize   bool   `json:"fullSize,omitempty" jsonschema:"Skip image scaling and return original size (default: false = scale images to 1024x768 max)"`
}

type PageItemsInput struct {
	PageID string `json:"pageID" jsonschema:"Page ID to list items for"`
}
type PageItemsOutput struct {
	Items []map[string]interface{} `json:"items"`
}

type PageCopyInput struct {
	PageID          string `json:"pageID" jsonschema:"Page ID to copy"`
	TargetSectionID string `json:"targetSectionID" jsonschema:"Target section ID to copy the page to"`
}
type PageCopyOutput struct {
	Success bool                   `json:"success"`
	Result  map[string]interface{} `json:"result"`
}

type PageMoveInput struct {
	PageID          string `json:"pageID" jsonschema:"Page ID to move"`
	TargetSectionID string `json:"targetSectionID" jsonschema:"Target section ID to move the page to"`
}
type PageMoveOutput struct {
	Success bool                   `json:"success"`
	Result  map[string]interface{} `json:"result"`
}

type QuickNoteInput struct {
	Content string `json:"content" jsonschema:"Content to add to the quicknote page"`
}
type QuickNoteOutput struct {
	Success         bool   `json:"success"`
	Timestamp       string `json:"timestamp"`
	Notebook        string `json:"notebook"`
	Page            string `json:"page"`
	Message         string `json:"message"`
	DetectedFormat  string `json:"detected_format"`
	ContentLength   int    `json:"content_length"`
	HTMLLength      int    `json:"html_length"`
}

// registerPageTools registers all page-related MCP tools
// verifySectionNotebookOwnership checks that a section belongs to the currently selected notebook
func verifySectionNotebookOwnership(ctx context.Context, sectionID string, sectionClient *sections.SectionClient, notebookCache *NotebookCache, operationName string) error {
	// SECURITY: Verify the section belongs to the currently selected notebook
	currentNotebook, hasNotebook := notebookCache.GetDisplayName()
	if !hasNotebook || currentNotebook == "" {
		logger := utils.NewToolLogger(operationName)
		logger.LogError(fmt.Errorf("no notebook selected"), "section_id", sectionID)
		return fmt.Errorf("no notebook selected. Use selectNotebook tool first")
	}

	// Resolve which notebook owns this section
	resolvedNotebookID, resolvedNotebookName, err := sectionClient.ResolveSectionNotebook(ctx, sectionID)
	if err != nil {
		logger := utils.NewToolLogger(operationName)
		logger.LogError(fmt.Errorf("failed to resolve section notebook ownership: %v", err),
			"section_id", sectionID,
			"security_action", "BLOCKING_UNVERIFIED_SECTION_ACCESS")
		return fmt.Errorf("could not verify section ownership: %v", err)
	}

	// Check if the resolved notebook matches the currently selected notebook
	if resolvedNotebookName != currentNotebook {
		logger := utils.NewToolLogger(operationName)
		logger.LogError(fmt.Errorf("SECURITY VIOLATION: Section belongs to different notebook than selected"),
			"section_id", sectionID,
			"resolved_notebook", resolvedNotebookName,
			"current_notebook", currentNotebook,
			"security_action", "BLOCKING_CROSS_NOTEBOOK_SECTION_ACCESS")
		return fmt.Errorf("access denied: section belongs to notebook '%s' but '%s' is selected", resolvedNotebookName, currentNotebook)
	}

	logger := utils.NewToolLogger(operationName)
	logger.LogDebug("Section notebook ownership verified",
		"section_id", sectionID,
		"notebook", resolvedNotebookName,
		"notebook_id", resolvedNotebookID)

	return nil
}

// verifyPageNotebookOwnership checks that a page belongs to the currently selected notebook
func verifyPageNotebookOwnership(ctx context.Context, pageID string, pageClient *pages.PageClient, notebookCache *NotebookCache, operationName string) error {
	// SECURITY: Verify the page belongs to the currently selected notebook
	currentNotebook, hasNotebook := notebookCache.GetDisplayName()
	if !hasNotebook || currentNotebook == "" {
		logger := utils.NewToolLogger(operationName)
		logger.LogError(fmt.Errorf("no notebook selected"), "page_id", pageID)
		return fmt.Errorf("no notebook selected. Use selectNotebook tool first")
	}

	// Resolve which notebook owns this page
	resolvedNotebookID, resolvedNotebookName, resolvedSectionID, resolvedSectionName, err := pageClient.ResolvePageNotebook(ctx, pageID)
	if err != nil {
		logger := utils.NewToolLogger(operationName)
		logger.LogError(fmt.Errorf("failed to resolve page notebook ownership: %v", err),
			"page_id", pageID,
			"security_action", "BLOCKING_UNVERIFIED_PAGE_ACCESS")
		return fmt.Errorf("could not verify page ownership: %v", err)
	}

	// Check if the resolved notebook matches the currently selected notebook
	if resolvedNotebookName != currentNotebook {
		logger := utils.NewToolLogger(operationName)
		logger.LogError(fmt.Errorf("SECURITY VIOLATION: Page belongs to different notebook than selected"),
			"page_id", pageID,
			"resolved_notebook", resolvedNotebookName,
			"current_notebook", currentNotebook,
			"resolved_section", resolvedSectionName,
			"security_action", "BLOCKING_CROSS_NOTEBOOK_PAGE_ACCESS")
		return fmt.Errorf("access denied: page belongs to notebook '%s' but '%s' is selected", resolvedNotebookName, currentNotebook)
	}

	logger := utils.NewToolLogger(operationName)
	logger.LogDebug("Page notebook ownership verified",
		"page_id", pageID,
		"notebook", resolvedNotebookName,
		"section", resolvedSectionName,
		"section_id", resolvedSectionID,
		"notebook_id", resolvedNotebookID)

	return nil
}

func registerPageTools(s *mcp.Server, pageClient *pages.PageClient, graphClient *graph.Client, notebookCache *NotebookCache, cfg *config.Config, authConfig *authorization.AuthorizationConfig, cache authorization.NotebookCache, quickNoteConfig authorization.QuickNoteConfig) {
	// pages: List all pages in a section
	pagesHandler := func(ctx context.Context, req *mcp.CallToolRequest, input PagesInput) (*mcp.CallToolResult, PagesOutput, error) {
		startTime := time.Now()
		logger := utils.NewToolLogger("pages")
		sectionID := input.SectionID
		logger.LogDebug("Page enumeration parameters", "sectionID", sectionID)

		// SECURITY: Verify the section belongs to the currently selected notebook
		// We need to create a section client to resolve ownership
		sectionClient := sections.NewSectionClient(graphClient)
		if err := verifySectionNotebookOwnership(ctx, sectionID, sectionClient, notebookCache, "pages"); err != nil {
			return utils.ToolResults.NewError("pages", err), PagesOutput{}, nil
		}

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
							logger.LogDebug("Sections not cached, fetching for authorization context",
								"section_id", sectionID)

							// Progress token already extracted above
							if progressToken != "" {
								sendProgressNotification(s, ctx, progressToken, 10, 100, "Fetching sections for authorization context...")
							}

							// Fetch sections to populate cache for authorization
							if err := populateSectionsForAuthorization(s, ctx, graphClient, notebookCache, progressToken); err != nil {
								logger.LogError(fmt.Errorf("failed to populate sections for authorization context: %v", err))
							}

							if progressToken != "" {
								sendProgressNotification(s, ctx, progressToken, 30, 100, "Authorization context updated, applying filters...")
							}
						}
						// Try to get section name with progress-aware API lookup if cache misses
					if progressToken != "" {
						sendProgressNotification(s, ctx, progressToken, 32, 100, "Resolving section name for authorization...")
					}
					// Note: Section name resolution simplified for authorization
					sectionName, _ = cache.GetDisplayName()
					notebookName, _ = cache.GetDisplayName()
					}

					// Note: Page filtering removed - all pages within selected notebook are now accessible
					logger.LogDebug("Applied authorization filtering to cached pages",
						"section_id", sectionID,
						"section_name", sectionName,
						"notebook", notebookName,
						"original_count", originalCachedCount,
						"filtered_count", len(cachedPages))
				}

				elapsed := time.Since(startTime)
				logger.LogDebug("Pages using cached data",
					"section_id", sectionID,
					"duration", elapsed,
					"original_cached_count", originalCachedCount,
					"filtered_cached_count", len(cachedPages),
					"cache_hit", true)

				// If cache returns empty results, fall back to fresh API call to be sure
				if len(cachedPages) == 0 {
					logger.LogDebug("Cache returned 0 pages, falling back to fresh API call to verify",
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

					return utils.ToolResults.NewJSONResult("pages", cacheResponse), PagesOutput{
						Pages:       cachedPages,
						Cached:      true,
						CacheHit:    true,
						PagesCount:  len(cachedPages),
						Duration:    elapsed.String(),
					}, nil
				}
			}
		}

		// Cache miss or expired - fetch from API
		logger.LogDebug("Pages cache miss or expired, fetching from API", "section_id", sectionID)

		if progressToken != "" {
			sendProgressNotification(s, ctx, progressToken, 40, 100, "Fetching pages from OneNote API...")
		}

		pages, err := pageClient.ListPages(sectionID)

		if progressToken != "" {
			sendProgressNotification(s, ctx, progressToken, 55, 100, "Pages retrieved, processing authorization...")
		}
		if err != nil {
			logger.LogError(err, "section_id", sectionID)
			return utils.ToolResults.NewError("pages", fmt.Errorf("failed to list pages: %v", err)), PagesOutput{}, nil
		}

		// Apply authorization filtering to fresh pages
		originalApiCount := len(pages)
		if authConfig != nil && authConfig.Enabled {
			// Try to get section and notebook context for filtering
			var sectionName, notebookName string
			if cache != nil {
				// If sections aren't cached yet, try to fetch them for proper authorization context
				if !notebookCache.IsSectionsCached() {
					logger.LogDebug("Sections not cached, fetching for authorization context",
						"section_id", sectionID)

					// Progress token already extracted above
					if progressToken != "" {
						sendProgressNotification(s, ctx, progressToken, 60, 100, "Fetching sections for authorization context...")
					}

					// Fetch sections to populate cache for authorization
					if err := populateSectionsForAuthorization(s, ctx, graphClient, notebookCache, progressToken); err != nil {
						logger.LogError(fmt.Errorf("failed to populate sections for authorization context: %v", err))
					}

					if progressToken != "" {
						sendProgressNotification(s, ctx, progressToken, 80, 100, "Authorization context updated, applying filters...")
					}
				}
				// Try to get section name with progress-aware API lookup if cache misses
			if progressToken != "" {
				sendProgressNotification(s, ctx, progressToken, 82, 100, "Resolving section name for authorization...")
			}
			// Note: Section name resolution simplified for authorization
			sectionName, _ = cache.GetDisplayName()
			notebookName, _ = cache.GetDisplayName()
			}

			// Note: Page filtering removed - all pages within selected notebook are now accessible
			logger.LogDebug("Applied authorization filtering to fresh pages",
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
		logger.LogDebug("Pages cached fresh filtered data",
			"section_id", sectionID,
			"original_count", originalApiCount,
			"filtered_count", len(pages))

		elapsed := time.Since(startTime)
		logger.LogSuccess("pages_count", len(pages))

		// Handle empty results gracefully
		if len(pages) == 0 {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: "No pages found in the specified section. The section may be empty or you may need to create pages first."},
				},
			}, PagesOutput{
				Pages:       []map[string]interface{}{},
				Cached:      false,
				CacheHit:    false,
				PagesCount:  0,
				Duration:    elapsed.String(),
			}, nil
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

		if progressToken != "" {
			sendProgressNotification(s, ctx, progressToken, 100, 100, "Complete!")
		}

		return utils.ToolResults.NewJSONResult("pages", apiResponse), PagesOutput{
			Pages:       pages,
			Cached:      false,
			CacheHit:    false,
			PagesCount:  len(pages),
			Duration:    elapsed.String(),
		}, nil
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "pages",
		Description: resources.MustGetToolDescription("pages"),
	}, pagesHandler)

	// page_create: Create a new page in a section
	pageCreateHandler := func(ctx context.Context, req *mcp.CallToolRequest, input PageCreateInput) (*mcp.CallToolResult, PageCreateOutput, error) {
		logger := utils.NewToolLogger("page_create")
		sectionID := input.SectionID
		title := input.Title
		content := input.Content
		contentType := input.ContentType

		logger.LogDebug("Page creation parameters", "sectionID", sectionID, "title", title, "content_length", len(content), "contentType", contentType)

		// SECURITY: Verify the section belongs to the currently selected notebook
		sectionClient := sections.NewSectionClient(graphClient)
		if err := verifySectionNotebookOwnership(ctx, sectionID, sectionClient, notebookCache, "page_create"); err != nil {
			return utils.ToolResults.NewError("page_create", err), PageCreateOutput{}, nil
		}

		// Validate title for illegal characters
		illegalChars := []string{"?", "*", "\\", "/", ":", "<", ">", "|", "&", "#", "'", "'", "%", "~"}
		for _, char := range illegalChars {
			if strings.Contains(title, char) {
				logger.LogError(fmt.Errorf("title contains illegal character: %s", char), "character", char, "title", title)
				suggestedName := utils.SuggestValidName(title, char)
				return utils.ToolResults.NewError("page_create", fmt.Errorf("title contains illegal character '%s'. Illegal characters are: ?*\\/:<>|&#''%%%%~\n\nSuggestion: Try using '%s' instead of '%s'.\n\nSuggested valid title: '%s'", char, utils.GetReplacementChar(char), char, suggestedName)), PageCreateOutput{}, nil
			}
		}
		logger.LogDebug("Title validation passed")

		// Convert content to HTML - use explicit type if provided, otherwise auto-detect
		var convertedHTML string
		var detectedFormat utils.TextFormat
		var formatSource string

		if contentType != "" {
			// Use explicit content type
			var err error
			convertedHTML, detectedFormat, err = utils.ConvertToHTMLWithType(content, contentType)
			if err != nil {
				logger.LogError(err, "contentType", contentType)
				return utils.ToolResults.NewError("page_create", fmt.Errorf("invalid content type: %v", err)), PageCreateOutput{}, nil
			}
			formatSource = "explicit"
			logger.LogDebug("Content converted with explicit type",
				"specified_type", contentType,
				"final_format", detectedFormat.String(),
				"original_length", len(content),
				"converted_length", len(convertedHTML))
		} else {
			// Auto-detect format
			convertedHTML, detectedFormat = utils.ConvertToHTML(content)
			formatSource = "detected"
			logger.LogDebug("Content format detection",
				"detected_format", detectedFormat.String(),
				"original_length", len(content),
				"converted_length", len(convertedHTML))
		}

		result, err := pageClient.CreatePage(sectionID, title, convertedHTML)
		if err != nil {
			logger.LogError(err, "section_id", sectionID)
			return utils.ToolResults.NewError("page_create", fmt.Errorf("failed to create page: %v", err)), PageCreateOutput{}, nil
		}

		// Clear pages cache for this section since we added a new page
		notebookCache.ClearPagesCache(sectionID)
		logger.LogDebug("Cleared pages cache", "section_id", sectionID)

		// Extract only the essential information: success status and page ID
		pageID, exists := result["id"].(string)
		if !exists {
			logger.LogError(fmt.Errorf("result missing ID field"), "result", result)
			return utils.ToolResults.NewError("page_create", fmt.Errorf("page creation succeeded but no ID was returned")), PageCreateOutput{}, nil
		}

		output := PageCreateOutput{
			Success:        true,
			PageID:         pageID,
			Format:         detectedFormat.String(),
			FormatSource:   formatSource,
			ContentLength:  len(content),
			HTMLLength:     len(convertedHTML),
		}

		if contentType != "" {
			output.SpecifiedType = contentType
		}

		logger.LogSuccess("page_id", pageID)
		return utils.ToolResults.NewJSONResult("page_create", output), output, nil
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "page_create",
		Description: resources.MustGetToolDescription("page_create"),
	}, pageCreateHandler)

	// page_update: Update the HTML content of a page
	pageUpdateHandler := func(ctx context.Context, req *mcp.CallToolRequest, input PageUpdateInput) (*mcp.CallToolResult, PageUpdateOutput, error) {
		logger := utils.NewToolLogger("page_update")
		pageID := input.PageID
		content := input.Content
		contentType := input.ContentType

		logger.LogDebug("Page update parameters", "pageID", pageID, "content_length", len(content), "contentType", contentType)

		// SECURITY: Verify the page belongs to the currently selected notebook
		if err := verifyPageNotebookOwnership(ctx, pageID, pageClient, notebookCache, "page_update"); err != nil {
			return utils.ToolResults.NewError("page_update", err), PageUpdateOutput{}, nil
		}

		// Convert content to HTML - use explicit type if provided, otherwise auto-detect
		var convertedHTML string
		var detectedFormat utils.TextFormat
		var formatSource string

		if contentType != "" {
			// Use explicit content type
			var err error
			convertedHTML, detectedFormat, err = utils.ConvertToHTMLWithType(content, contentType)
			if err != nil {
				logger.LogError(err, "contentType", contentType)
				return utils.ToolResults.NewError("page_update", fmt.Errorf("invalid content type: %v", err)), PageUpdateOutput{}, nil
			}
			formatSource = "explicit"
			logger.LogDebug("Content converted with explicit type",
				"specified_type", contentType,
				"final_format", detectedFormat.String(),
				"original_length", len(content),
				"converted_length", len(convertedHTML))
		} else {
			// Auto-detect format
			convertedHTML, detectedFormat = utils.ConvertToHTML(content)
			formatSource = "detected"
			logger.LogDebug("Content format detection",
				"detected_format", detectedFormat.String(),
				"original_length", len(content),
				"converted_length", len(convertedHTML))
		}

		err := pageClient.UpdatePageContentSimple(pageID, convertedHTML)
		if err != nil {
			logger.LogError(err, "page_id", pageID)
			return utils.ToolResults.NewError("page_update", fmt.Errorf("failed to update page content: %v", err)), PageUpdateOutput{}, nil
		}

		output := PageUpdateOutput{
			Success:         true,
			Message:         "Page content updated successfully",
			Format:          detectedFormat.String(),
			FormatSource:   formatSource,
			ContentLength:  len(content),
			HTMLLength:     len(convertedHTML),
		}

		if contentType != "" {
			output.SpecifiedType = contentType
		}

		logger.LogSuccess()
		return utils.ToolResults.NewJSONResult("page_update", output), output, nil
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "page_update",
		Description: resources.MustGetToolDescription("page_update"),
	}, pageUpdateHandler)

	// page_update_advanced: Update page content with advanced commands
	pageUpdateAdvancedHandler := func(ctx context.Context, req *mcp.CallToolRequest, input PageUpdateAdvancedInput) (*mcp.CallToolResult, PageUpdateAdvancedOutput, error) {
		logger := utils.NewToolLogger("page_update_advanced")
		pageID := input.PageID
		commandsJSON := input.Commands
		contentType := input.ContentType

		logger.LogDebug("Advanced page update parameters", "pageID", pageID, "commands_length", len(commandsJSON), "contentType", contentType)

		// SECURITY: Verify the page belongs to the currently selected notebook
		if err := verifyPageNotebookOwnership(ctx, pageID, pageClient, notebookCache, "page_update_advanced"); err != nil {
			return utils.ToolResults.NewError("page_update_advanced", err), PageUpdateAdvancedOutput{}, nil
		}

		// Parse the commands JSON
		var commands []pages.UpdateCommand
		if errUnmarshal := json.Unmarshal([]byte(commandsJSON), &commands); errUnmarshal != nil {
			logger.LogError(errUnmarshal, "commands_json", commandsJSON)
			return utils.ToolResults.NewError("page_update_advanced", fmt.Errorf("failed to parse commands JSON: %v", errUnmarshal)), PageUpdateAdvancedOutput{}, nil
		}

		logger.LogDebug("Commands parsed", "command_count", len(commands))

		// Apply format detection and conversion to each command's content
		var formatDetectionResults []map[string]interface{}
		var formatSource string
		if contentType != "" {
			formatSource = "explicit"
		} else {
			formatSource = "detected"
		}

		for i, command := range commands {
			if command.Content != "" {
				originalContent := command.Content
				var convertedHTML string
				var detectedFormat utils.TextFormat

				if contentType != "" {
					// Use explicit content type for all commands
					var err error
					convertedHTML, detectedFormat, err = utils.ConvertToHTMLWithType(command.Content, contentType)
					if err != nil {
						logger.LogError(err, "command_index", i, "contentType", contentType)
						return utils.ToolResults.NewError("page_update_advanced", fmt.Errorf("invalid content type for command %d: %v", i, err)), PageUpdateAdvancedOutput{}, nil
					}
				} else {
					// Auto-detect format for each command
					convertedHTML, detectedFormat = utils.ConvertToHTML(command.Content)
				}

				commands[i].Content = convertedHTML

				// Track format detection results
				result := map[string]interface{}{
					"command_index":    i,
					"target":          command.Target,
					"action":          command.Action,
					"format":          detectedFormat.String(),
					"format_source":   formatSource,
					"original_length": len(originalContent),
					"html_length":     len(convertedHTML),
				}

				if contentType != "" {
					result["specified_type"] = contentType
				}

				formatDetectionResults = append(formatDetectionResults, result)

				logger.LogDebug("Command content format processing",
					"command_index", i,
					"target", command.Target,
					"action", command.Action,
					"format", detectedFormat.String(),
					"format_source", formatSource,
					"original_length", len(originalContent),
					"converted_length", len(convertedHTML))
			}
		}

		err := pageClient.UpdatePageContent(pageID, commands)
		if err != nil {
			logger.LogError(err, "page_id", pageID)
			return utils.ToolResults.NewError("page_update_advanced", fmt.Errorf("failed to update page content: %v", err)), PageUpdateAdvancedOutput{}, nil
		}

		output := PageUpdateAdvancedOutput{
			Success:                true,
			Message:               "Page content updated successfully with advanced commands",
			CommandsProcessed:    len(commands),
			FormatDetection:      formatDetectionResults,
		}

		logger.LogSuccess()
		return utils.ToolResults.NewJSONResult("page_update_advanced", output), output, nil
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "page_update_advanced",
		Description: resources.MustGetToolDescription("page_update_advanced"),
	}, pageUpdateAdvancedHandler)

	// page_delete: Delete a page by ID
	pageDeleteHandler := func(ctx context.Context, req *mcp.CallToolRequest, input PageDeleteInput) (*mcp.CallToolResult, PageDeleteOutput, error) {
		logger := utils.NewToolLogger("page_delete")
		pageID := input.PageID

		// SECURITY: Verify the page belongs to the currently selected notebook
		if err := verifyPageNotebookOwnership(ctx, pageID, pageClient, notebookCache, "page_delete"); err != nil {
			return utils.ToolResults.NewError("page_delete", err), PageDeleteOutput{}, nil
		}

		logger.LogDebug("Page delete parameter", "pageID", pageID)

		err := pageClient.DeletePage(pageID)
		if err != nil {
			logger.LogError(err, "page_id", pageID)
			return utils.ToolResults.NewError("page_delete", fmt.Errorf("failed to delete page: %v", err)), PageDeleteOutput{}, nil
		}

		// Clear all pages cache since we don't know which section the deleted page was in
		notebookCache.ClearAllPagesCache()
		logger.LogDebug("Cleared all pages cache", "page_id", pageID)

		output := PageDeleteOutput{
			Success: true,
			Message: "Page deleted successfully",
		}

		logger.LogSuccess()
		return utils.ToolResults.NewJSONResult("page_delete", output), output, nil
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "page_delete",
		Description: resources.MustGetToolDescription("page_delete"),
	}, pageDeleteHandler)

	// page_content: Get the HTML content of a page by ID
	pageContentHandler := func(ctx context.Context, req *mcp.CallToolRequest, input PageContentInput) (*mcp.CallToolResult, PageContentOutput, error) {
		logger := utils.NewToolLogger("page_content")
		pageID := input.PageID
		forUpdateStr := input.ForUpdate
		forUpdate := forUpdateStr == "true"
		formatStr := input.Format
		if formatStr == "" {
			formatStr = "HTML"
		}
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
			logger.LogError(fmt.Errorf("invalid format: %s", format), "valid_formats", validFormats)
			return utils.ToolResults.NewError("page_content", fmt.Errorf("invalid format '%s'. Valid formats are: HTML, Markdown, Text", formatStr)), PageContentOutput{}, nil
		}

		// Check for incompatible parameter combinations
		if forUpdate && format != "HTML" {
			logger.LogError(fmt.Errorf("forUpdate incompatible with non-HTML format"), "forUpdate", forUpdate, "format", format)
			return utils.ToolResults.NewError("page_content", fmt.Errorf("forUpdate parameter can only be used with HTML format (for page updates)")), PageContentOutput{}, nil
		}

		logger.LogDebug("Page content parameters", "pageID", pageID, "forUpdate", forUpdate, "format", format)

		// SECURITY: Verify the page belongs to the currently selected notebook
		if err := verifyPageNotebookOwnership(ctx, pageID, pageClient, notebookCache, "page_content"); err != nil {
			return utils.ToolResults.NewError("page_content", err), PageContentOutput{}, nil
		}

		// Always get HTML content first
		htmlContent, err := pageClient.GetPageContent(pageID, forUpdate)
		if err != nil {
			logger.LogError(err, "page_id", pageID)
			return utils.ToolResults.NewError("page_content", fmt.Errorf("failed to get page content: %v", err)), PageContentOutput{}, nil
		}

		var finalContent string
		var conversionError error

		// Convert content based on requested format
		switch format {
		case "HTML":
			finalContent = htmlContent
			logger.LogDebug("Returning HTML content as-is")
		case "MARKDOWN":
			logger.LogDebug("Converting HTML to Markdown")
			finalContent, conversionError = utils.ConvertHTMLToMarkdown(htmlContent)
			if conversionError != nil {
				logger.LogError(conversionError)
				return utils.ToolResults.NewError("page_content", fmt.Errorf("failed to convert content to Markdown: %v", conversionError)), PageContentOutput{}, nil
			}
		case "TEXT":
			logger.LogDebug("Converting HTML to plain text")
			finalContent, conversionError = utils.ConvertHTMLToText(htmlContent)
			if conversionError != nil {
				logger.LogError(conversionError)
				return utils.ToolResults.NewError("page_content", fmt.Errorf("failed to convert content to plain text: %v", conversionError)), PageContentOutput{}, nil
			}
		}

		logger.LogSuccess(
			"original_html_length", len(htmlContent),
			"final_content_length", len(finalContent),
			"output_format", format)

		// Create response with format information
		output := PageContentOutput{
			Content:            finalContent,
			Format:            format,
			OriginalHTMLLength: len(htmlContent),
			FinalContentLength: len(finalContent),
		}

		// Add forUpdate info if applicable
		if forUpdate {
			output.ForUpdate = true
		}

		return utils.ToolResults.NewJSONResult("page_content", output), output, nil
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "page_content",
		Description: resources.MustGetToolDescription("page_content"),
	}, pageContentHandler)

	// page_item_content: Get a OneNote page item (e.g., image) by page item ID, returns binary data with proper MIME type
	pageItemContentHandler := func(ctx context.Context, req *mcp.CallToolRequest, input PageItemContentInput) (*mcp.CallToolResult, map[string]interface{}, error) {
		logger := utils.NewToolLogger("page_item_content")
		pageID := input.PageID
		pageItemID := input.PageItemID
		customFilename := input.Filename
		fullSize := input.FullSize

		// SECURITY: Verify the page belongs to the currently selected notebook
		if err := verifyPageNotebookOwnership(ctx, pageID, pageClient, notebookCache, "page_item_content"); err != nil {
			return utils.ToolResults.NewError("page_item_content", err), nil, nil
		}

		logger.LogDebug("Page item content parameters", "pageID", pageID, "pageItemID", pageItemID, "filename", customFilename, "fullSize", fullSize)

		// Use the enhanced GetPageItem with fullSize parameter
		pageItemData, err := pageClient.GetPageItem(pageID, pageItemID, fullSize)
		if err != nil {
			logger.LogError(err, "page_id", pageID, "page_item_id", pageItemID)
			return utils.ToolResults.NewError("page_item_content", fmt.Errorf("failed to get page item: %v", err)), nil, nil
		}

		logger.LogDebug("Page item retrieved", "filename", pageItemData.Filename, "content_type", pageItemData.ContentType, "size_bytes", pageItemData.Size, "tag_name", pageItemData.TagName, "full_size", fullSize)

		// Use custom filename if provided, otherwise use the one from page item data
		filename := pageItemData.Filename
		if customFilename != "" {
			filename = customFilename
			logger.LogDebug("Using custom filename", "filename", filename)
		}

		// Return binary data with proper MIME type using mcp.NewToolResultImage
		// This should provide the raw binary data with the correct content type
		encoded := base64.StdEncoding.EncodeToString(pageItemData.Content)
		logger.LogDebug("Prepared content for return", "encoded_length", len(encoded), "content_type", pageItemData.ContentType, "filename", filename)

		// Create response with binary data as base64 encoded text
		response := map[string]interface{}{
			"content_type": pageItemData.ContentType,
			"filename":     filename,
			"size_bytes":   pageItemData.Size,
			"data":         encoded,
			"encoding":     "base64",
			"full_size":    fullSize,
		}

		logger.LogSuccess("scaled", !fullSize && strings.HasPrefix(pageItemData.ContentType, "image/"))
		return utils.ToolResults.NewJSONResult("page_item_content", response), response, nil
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "page_item_content",
		Description: resources.MustGetToolDescription("page_item_content"),
	}, pageItemContentHandler)

	// page_items: List all OneNote page items (images, files, etc.) for a specific page
	pageItemsHandler := func(ctx context.Context, req *mcp.CallToolRequest, input PageItemsInput) (*mcp.CallToolResult, PageItemsOutput, error) {
		logger := utils.NewToolLogger("page_items")
		pageID := input.PageID

		// SECURITY: Verify the page belongs to the currently selected notebook
		if err := verifyPageNotebookOwnership(ctx, pageID, pageClient, notebookCache, "page_items"); err != nil {
			return utils.ToolResults.NewError("page_items", err), PageItemsOutput{}, nil
		}

		logger.LogDebug("Page items parameter", "pageID", pageID)

		pageItems, err := pageClient.ListPageItems(pageID)
		if err != nil {
			logger.LogError(err, "page_id", pageID)
			return utils.ToolResults.NewError("page_items", fmt.Errorf("failed to list page items: %v", err)), PageItemsOutput{}, nil
		}

		output := PageItemsOutput{
			Items: pageItems,
		}

		logger.LogSuccess("items_count", len(pageItems))
		return utils.ToolResults.NewJSONResult("page_items", output), output, nil
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "page_items",
		Description: resources.MustGetToolDescription("page_items"),
	}, pageItemsHandler)

	// page_copy: Copy a page from one section to another
	pageCopyHandler := func(ctx context.Context, req *mcp.CallToolRequest, input PageCopyInput) (*mcp.CallToolResult, PageCopyOutput, error) {
		logger := utils.NewToolLogger("page_copy")
		pageID := input.PageID
		targetSectionID := input.TargetSectionID

		// SECURITY: Verify the page belongs to the currently selected notebook
		if err := verifyPageNotebookOwnership(ctx, pageID, pageClient, notebookCache, "page_copy"); err != nil {
			return utils.ToolResults.NewError("page_copy", err), PageCopyOutput{}, nil
		}

		logger.LogDebug("Page copy parameters", "pageID", pageID, "targetSectionID", targetSectionID)

		result, err := pageClient.CopyPage(pageID, targetSectionID)
		if err != nil {
			logger.LogError(err, "page_id", pageID)
			return utils.ToolResults.NewError("page_copy", fmt.Errorf("failed to copy page: %v", err)), PageCopyOutput{}, nil
		}

		// Clear pages cache for target section since we added a new page
		notebookCache.ClearPagesCache(targetSectionID)
		logger.LogDebug("Cleared pages cache for target section", "target_section_id", targetSectionID)

		output := PageCopyOutput{
			Success: true,
			Result:  result,
		}

		logger.LogSuccess()
		return utils.ToolResults.NewJSONResult("page_copy", output), output, nil
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "page_copy",
		Description: resources.MustGetToolDescription("page_copy"),
	}, pageCopyHandler)

	// page_move: Move a page from one section to another (copy then delete)
	pageMoveHandler := func(ctx context.Context, req *mcp.CallToolRequest, input PageMoveInput) (*mcp.CallToolResult, PageMoveOutput, error) {
		logger := utils.NewToolLogger("page_move")
		pageID := input.PageID
		targetSectionID := input.TargetSectionID

		// SECURITY: Verify the page belongs to the currently selected notebook
		if err := verifyPageNotebookOwnership(ctx, pageID, pageClient, notebookCache, "page_move"); err != nil {
			return utils.ToolResults.NewError("page_move", err), PageMoveOutput{}, nil
		}

		logger.LogDebug("Page move parameters", "pageID", pageID, "targetSectionID", targetSectionID)

		result, err := pageClient.MovePage(pageID, targetSectionID)
		if err != nil {
			logger.LogError(err, "page_id", pageID)
			return utils.ToolResults.NewError("page_move", fmt.Errorf("failed to move page: %v", err)), PageMoveOutput{}, nil
		}

		// Clear all pages cache since page_move affects both source and target sections
		// (we don't know the source section ID)
		notebookCache.ClearAllPagesCache()
		logger.LogDebug("Cleared all pages cache", "page_id", pageID, "target_section_id", targetSectionID)

		output := PageMoveOutput{
			Success: true,
			Result:  result,
		}

		logger.LogSuccess()
		return utils.ToolResults.NewJSONResult("page_move", output), output, nil
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "page_move",
		Description: resources.MustGetToolDescription("page_move"),
	}, pageMoveHandler)

	// quick_note: Add a timestamped note to a configured page
	quickNoteHandler := func(ctx context.Context, req *mcp.CallToolRequest, input QuickNoteInput) (*mcp.CallToolResult, QuickNoteOutput, error) {
		logger := utils.NewToolLogger("quick_note")
		content := input.Content

		// Extract progress token from request metadata for progress notifications
		progressToken := utils.ExtractProgressToken(req)

		// Check if quicknote is configured
		if cfg.QuickNote == nil {
			logger.LogError(fmt.Errorf("quickNote not configured"))
			return utils.ToolResults.NewError("quick_note", fmt.Errorf("QuickNote is not configured. Please set page_name and optionally notebook_name and date_format in your configuration")), QuickNoteOutput{}, nil
		}

		if cfg.QuickNote.PageName == "" {
			logger.LogError(fmt.Errorf("quickNote page name not configured"), "page", cfg.QuickNote.PageName)
			return utils.ToolResults.NewError("quick_note", fmt.Errorf("QuickNote page_name is required in quicknote configuration")), QuickNoteOutput{}, nil
		}

		// Determine target notebook name - use quicknote-specific notebook or fall back to default
		targetNotebookName := cfg.QuickNote.NotebookName
		if targetNotebookName == "" {
			targetNotebookName = cfg.NotebookName
			logger.LogDebug("Using default notebook name", "default_notebook", targetNotebookName)
		}

		if targetNotebookName == "" {
			logger.LogError(fmt.Errorf("no notebook name available"), "quicknote_notebook", cfg.QuickNote.NotebookName, "default_notebook", cfg.NotebookName)
			return utils.ToolResults.NewError("quick_note", fmt.Errorf("No notebook name configured. Please set either quicknote.notebook_name or notebook_name in your configuration")), QuickNoteOutput{}, nil
		}

		logger.LogDebug("QuickNote parameters",
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
			logger.LogError(err, "notebook_name", targetNotebookName)
			return utils.ToolResults.NewError("quick_note", fmt.Errorf("failed to find notebook '%s': %v", targetNotebookName, err)), QuickNoteOutput{}, nil
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
			logger.LogError(fmt.Errorf("notebook ID not found in response"))
			return utils.ToolResults.NewError("quick_note", fmt.Errorf("Notebook ID not found in response")), QuickNoteOutput{}, nil
		}

		// Progress notification will be sent based on cache status inside findPageInNotebookWithCache
		targetPage, targetSectionID, pageFromCache, err := findPageInNotebookWithCache(pageClient, sectionClient, notebookCache, s, notebookID, cfg.QuickNote.PageName, progressToken, ctx)
		if err != nil {
			logger.LogError(err, "page_name", cfg.QuickNote.PageName, "notebook_id", notebookID)
			return utils.ToolResults.NewError("quick_note", fmt.Errorf("failed to find page '%s' in notebook '%s': %v", cfg.QuickNote.PageName, targetNotebookName, err)), QuickNoteOutput{}, nil
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
		logger.LogDebug("Content format detection",
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
			logger.LogError(fmt.Errorf("page ID not found in response"))
			return utils.ToolResults.NewError("quick_note", fmt.Errorf("Page ID not found in response")), QuickNoteOutput{}, nil
		}

		err = pageClient.UpdatePageContent(pageID, commands)
		if err != nil {
			logger.LogError(err, "page_id", pageID)
			return utils.ToolResults.NewError("quick_note", fmt.Errorf("failed to add quicknote to page: %v", err)), QuickNoteOutput{}, nil
		}

		// Clear pages cache for this section and page search cache
		sendProgressNotification(s, ctx, progressToken, 90, 100, "Clearing cache and finalizing...")
		notebookCache.ClearPagesCache(targetSectionID)
		notebookCache.ClearAllPageSearchCache() // Clear page search cache since page content changed
		// Note: We don't clear notebook lookup cache as notebook metadata shouldn't change from page updates
		logger.LogDebug("Cleared pages cache and page search cache", "section_id", targetSectionID)

		logger.LogSuccess(
			"notebook", targetNotebookName,
			"page", cfg.QuickNote.PageName,
			"timestamp", formattedDate,
			"content_length", len(content))

		output := QuickNoteOutput{
			Success:         true,
			Timestamp:       formattedDate,
			Notebook:        targetNotebookName,
			Page:            cfg.QuickNote.PageName,
			Message:         "Quicknote added successfully",
			DetectedFormat: detectedFormat.String(),
			ContentLength:  len(content),
			HTMLLength:     len(convertedHTML),
		}

		// Send final progress notification
		sendProgressNotification(s, ctx, progressToken, 100, 100, "Quicknote added successfully!")

		return utils.ToolResults.NewJSONResult("quick_note", output), output, nil
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "quick_note",
		Description: resources.MustGetToolDescription("quick_note"),
	}, quickNoteHandler)

	logger := utils.NewToolLogger("pagetools")
	logger.LogDebug("Page tools registered successfully")
}


// findPageInNotebookWithCache searches for a page by name using cached listPages calls
// This leverages the existing page cache and progress notification system
// Returns the page data, section ID where the page was found, whether result was from cache, and any error
func findPageInNotebookWithCache(pageClient *pages.PageClient, sectionClient *sections.SectionClient, notebookCache *NotebookCache, s *mcp.Server, notebookID string, pageName string, progressToken string, ctx context.Context) (map[string]interface{}, string, bool, error) {
	logger := utils.NewToolLogger("findPageInNotebookWithCache")
	logger.LogDebug("Searching for page in notebook using cached listPages", "notebook_id", notebookID, "page_name", pageName)

	// Check if page search results are already cached
	if cachedResult, cached := notebookCache.GetPageSearchCache(notebookID, pageName); cached {
		logger.LogDebug("Found cached page search result",
			"notebook_id", notebookID,
			"page_name", pageName,
			"found", cachedResult.Found)

		if cachedResult.Found {
			sendProgressNotification(s, ctx, progressToken, 30, 100, "Page found in search cache")
			logger.LogSuccess("page_title", pageName, "section_id", cachedResult.SectionID)
			return cachedResult.Page, cachedResult.SectionID, true, nil
		} else {
			// Cached result indicates page was not found
			sendProgressNotification(s, ctx, progressToken, 30, 100, "Page not found (from cache)")
			return nil, "", true, fmt.Errorf("page '%s' not found in notebook (from cache)", pageName)
		}
	}

	logger.LogDebug("No cached search result found, performing fresh search")
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
	logger.LogDebug("Found sections in notebook for cached search", "notebook_id", notebookID, "sections_count", totalSections)

	sendProgressNotification(s, ctx, progressToken, 35, 100, fmt.Sprintf("Found %d sections, starting page search...", totalSections))

	// Search through each section using cached listPages calls with progress
	for i, section := range sections {
		sectionID, ok := section["id"].(string)
		if !ok {
			logger.LogDebug("Skipping section with invalid ID in cached search", "section_index", i+1)
			continue // Skip if ID is not available
		}

		sectionName, ok := section["displayName"].(string)
		if !ok {
			sectionName = "unknown"
		}

		// Calculate progress for this section (35-70% range)
		baseProgress := 35 + int(float64(i)/float64(totalSections)*35) // Distribute 35% across sections
		sendProgressNotification(s, ctx, progressToken, baseProgress, 100, fmt.Sprintf("Searching section %d of %d: %s", i+1, totalSections, sectionName))

		logger.LogDebug("Searching in section using cached listPages",
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
				logger.LogDebug("Using cached pages for section",
					"section_id", sectionID,
					"section_name", sectionName,
					"pages_count", len(pages))
			}
		}

		if !fromCache {
			// Cache miss - fetch from API with progress
			logger.LogDebug("Cache miss, fetching pages from API with progress",
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
				logger.LogError(fmt.Errorf("failed to list pages in section during cached search: %v", err),
					"section_id", sectionID,
					"section_name", sectionName)
				continue // Continue searching other sections
			}
			pages = pagesFromAPI
			// Cache the results
			notebookCache.SetPagesCache(sectionID, pages)
			logger.LogDebug("Cached fresh pages data",
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

		logger.LogDebug("Found pages in section (cached search)",
			"section_id", sectionID,
			"section_name", sectionName,
			"pages_count", len(pages),
			"page_names", pageNames,
			"from_cache", fromCache)

		// Look for the page by name
		for j, page := range pages {
			pageTitle, ok := page["title"].(string)
			if !ok {
				logger.LogDebug("Skipping page with invalid title in cached search",
					"section_name", sectionName,
					"page_index", j+1)
				continue // Skip if title is not available
			}

			logger.LogDebug("Examining page in cached search",
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
					logger.LogDebug("Found matching page but invalid ID in cached search",
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
				logger.LogDebug("Cached successful page search result",
					"notebook_id", notebookID,
					"page_name", pageName,
					"section_id", sectionID)

				// Send completion notification if available
				sendProgressNotification(s, ctx, progressToken, 75, 100, fmt.Sprintf("Page found in section: %s", sectionName))

				logger.LogSuccess("page_id", pageID,
					"page_title", pageTitle,
					"section_id", sectionID,
					"section_name", sectionName,
					"from_cache", fromCache)
				return page, sectionID, false, nil
			}
		}

		logger.LogDebug("Page not found in section (cached search)",
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
	logger.LogDebug("Cached failed page search result",
		"notebook_id", notebookID,
		"page_name", pageName)

	return nil, "", false, fmt.Errorf("page '%s' not found in notebook", pageName)
}

// getDetailedNotebookByNameCached retrieves comprehensive notebook information by display name with caching
// This is a cached wrapper around notebooks.GetDetailedNotebookByName
// Returns the notebook data and a boolean indicating if it was from cache
func getDetailedNotebookByNameCached(notebookClient *notebooks.NotebookClient, notebookCache *NotebookCache, notebookName string) (map[string]interface{}, bool, error) {
	logger := utils.NewToolLogger("getDetailedNotebookByNameCached")
	logger.LogDebug("Getting detailed notebook by name with caching", "notebook_name", notebookName)

	// Check if notebook lookup is already cached
	if cachedNotebook, cached := notebookCache.GetNotebookLookupCache(notebookName); cached {
		logger.LogDebug("Found cached notebook lookup result",
			"notebook_name", notebookName,
			"cached_notebook_id", cachedNotebook["id"],
			"cached_notebook_data", cachedNotebook)
		logger.LogSuccess("notebook_name", notebookName, "notebook_id", cachedNotebook["id"])
		return cachedNotebook, true, nil
	}

	logger.LogDebug("No cached notebook lookup found, performing fresh lookup")

	// Cache miss - fetch from API
	notebook, err := notebookClient.GetDetailedNotebookByName(notebookName)
	if err != nil {
		logger.LogError(err, "notebook_name", notebookName)
		return nil, false, err
	}

	// Log the fresh notebook data before caching
	logger.LogDebug("Fresh notebook lookup result",
		"notebook_name", notebookName,
		"fresh_notebook_id", notebook["id"],
		"fresh_notebook_data", notebook)

	// Cache the successful result
	notebookCache.SetNotebookLookupCache(notebookName, notebook)
	logger.LogDebug("Cached notebook lookup result", "notebook_name", notebookName, "notebook_id", notebook["id"])

	return notebook, false, nil
}

// Helper functions for progress notifications and section population

// extractProgressToken extracts progress token from MCP request for notifications

// populateSectionsForAuthorization fetches sections to populate cache for authorization context
func populateSectionsForAuthorization(s *mcp.Server, ctx context.Context, graphClient *graph.Client, notebookCache *NotebookCache, progressToken string) error {
	// Check if notebook is selected
	notebookID, isSet := notebookCache.GetNotebookID()
	if !isSet {
		return fmt.Errorf("no notebook selected")
	}

	logger := utils.NewToolLogger("populateSectionsForAuthorization")
	logger.LogDebug("Populating sections for authorization context",
		"notebook_id", notebookID,
		"has_progress_token", progressToken != "")

	// Create section client for fetching sections
	sectionClient := sections.NewSectionClient(graphClient)

	// Get sections with progress notifications
	if progressToken != "" {
		sendProgressNotification(s, ctx, progressToken, 15, 100, "Fetching notebook sections for authorization...")
	}

	// Create a progress context with typed keys (compatible with fetchAllNotebookContentWithProgress)
	progressCtx := context.WithValue(ctx, utils.MCPServerKey, s)
	progressCtx = context.WithValue(progressCtx, utils.ProgressTokenKey, progressToken)

	if progressToken != "" {
		sendProgressNotification(s, ctx, progressToken, 18, 100, "Calling sections API...")
	}

	// Fetch all sections and section groups recursively using the same function as getNotebookSections
	sectionItems, err := fetchAllNotebookContentWithProgress(sectionClient, notebookID, progressCtx)
	if err != nil {
		logger.LogError(fmt.Errorf("failed to fetch sections for authorization context: %v", err))
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

	logger.LogSuccess("sections_count", len(sectionItems))
	return nil
}

