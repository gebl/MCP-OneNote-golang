// pages.go - Page operations for the Microsoft Graph API client.
//
// This file contains all page-related operations including listing pages,
// creating, updating, deleting, copying, and moving pages, as well as
// handling page items (images, files) and content management.
//
// Key Features:
// - Complete page CRUD operations (Create, Read, Update, Delete)
// - Page content management with HTML support
// - Page item handling (images, files, objects)
// - Advanced content updates with commands
// - Asynchronous page operations (copy, move)
// - Image optimization and metadata extraction
// - HTML parsing and content extraction
//
// Operations Supported:
// - ListPages: List all pages in a section with pagination
// - GetPageContent: Retrieve page HTML content
// - CreatePage: Create new pages with HTML content
// - UpdatePageContent: Update page content with advanced commands
// - UpdatePageContentSimple: Simple content replacement
// - DeletePage: Delete pages by ID
// - CopyPage: Copy pages between sections with async operations
// - MovePage: Move pages between sections
// - GetPageItem: Get complete page item data with binary content
// - ListPageItems: List embedded items in a page
//
// Usage Example:
//   pageClient := pages.NewPageClient(graphClient)
//   pages, err := pageClient.ListPages(sectionID)
//   if err != nil {
//       logging.PageLogger.Error("Failed to list pages", "error", err)
//   }
//
//   result, err := pageClient.CreatePage(sectionID, "My Page", "<html><body>Content</body></html>")
//   if err != nil {
//       logging.PageLogger.Error("Failed to create page", "error", err)
//   }

package pages

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"

	abstractions "github.com/microsoft/kiota-abstractions-go"
	msgraphmodels "github.com/microsoftgraph/msgraph-sdk-go/models"

	"github.com/gebl/onenote-mcp-server/internal/graph"
	"github.com/gebl/onenote-mcp-server/internal/logging"
	"github.com/gebl/onenote-mcp-server/internal/utils"
)

const (
	imgTag    = "img"
	objectTag = "object"
)

// PageClient provides page-specific operations
type PageClient struct {
	*graph.Client
}

// NewPageClient creates a new PageClient
func NewPageClient(client *graph.Client) *PageClient {
	return &PageClient{Client: client}
}

// PageItemData represents the complete data for a OneNote page item (e.g., image) including metadata and content.
type PageItemData struct {
	ContentType string            `json:"contentType"` // MIME type of the page item
	Filename    string            `json:"filename"`    // Suggested filename for the page item
	Size        int64             `json:"size"`        // Size of the page item in bytes
	Content     []byte            `json:"content"`     // Raw binary content of the page item
	TagName     string            `json:"tagName"`     // HTML tag name (img, object, etc.)
	Attributes  map[string]string `json:"attributes"`  // All HTML attributes from the tag
	OriginalURL string            `json:"originalUrl"` // Original URL from the HTML tag
}

// UpdateCommand represents a single update operation for OneNote page content.
// Based on Microsoft Graph OneNote API documentation.
type UpdateCommand struct {
	Target   string `json:"target"`             // Element to update (data-id, generated id, body, title)
	Action   string `json:"action"`             // Action to perform (append, insert, prepend, replace)
	Position string `json:"position,omitempty"` // Position for insert/append (before, after) - omitted for append action
	Content  string `json:"content"`            // HTML content to add/replace
}

// MarshalJSON implements custom JSON marshaling to exclude position field for append actions
func (uc UpdateCommand) MarshalJSON() ([]byte, error) {
	type UpdateCommandAlias UpdateCommand // Create alias to avoid infinite recursion

	// For append action, create a copy without the position field
	if uc.Action == "append" {
		alias := UpdateCommandAlias{
			Target:  uc.Target,
			Action:  uc.Action,
			Content: uc.Content,
			// Position field is omitted
		}
		return json.Marshal(alias)
	}

	// For other actions, marshal normally
	return json.Marshal(UpdateCommandAlias(uc))
}

// PageItemInfo represents a parsed page item with attributes and extracted ID
type PageItemInfo struct {
	TagName     string            `json:"tagName"`     // "img" or "object"
	PageItemID  string            `json:"pageItemId"`  // Extracted from src/data URL
	Attributes  map[string]string `json:"attributes"`  // All attributes from the tag
	OriginalURL string            `json:"originalUrl"` // Original src/data URL
}

// ListPages fetches all pages in a section by sectionID using the Microsoft Graph SDK.
// Returns a slice of page metadata maps and an error, if any.
// This function handles pagination using the SDK's nextLink mechanism.
func (c *PageClient) ListPages(sectionID string) ([]map[string]interface{}, error) {
	logging.PageLogger.Info("Starting ListPages operation", "section_id", sectionID)

	if c.TokenManager != nil && c.TokenManager.IsExpired() {
		logging.PageLogger.Debug("Token expired, refreshing before ListPages", "section_id", sectionID)
		if err := c.RefreshTokenIfNeeded(); err != nil {
			logging.PageLogger.Error("Token refresh failed during ListPages", "section_id", sectionID, "error", err)
			return nil, fmt.Errorf("token expired and refresh failed: %v", err)
		}
		logging.PageLogger.Debug("Token refreshed successfully for ListPages", "section_id", sectionID)
	}

	ctx := context.Background()
	logging.PageLogger.Debug("Making Graph API call to list pages", "section_id", sectionID)
	result, err := c.GraphClient.Me().Onenote().Sections().ByOnenoteSectionId(sectionID).Pages().Get(ctx, nil)
	if err != nil {
		logging.PageLogger.Debug("Initial Graph API call failed", "section_id", sectionID, "error", err)
		if strings.Contains(err.Error(), "JWT") || strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "403") {
			logging.PageLogger.Debug("Authentication error detected, attempting token refresh", "section_id", sectionID)
			if c.TokenManager != nil && c.OAuthConfig != nil {
				if refreshErr := c.RefreshTokenIfNeeded(); refreshErr != nil {
					logging.PageLogger.Error("Token refresh failed after auth error", "section_id", sectionID, "refresh_error", refreshErr, "original_error", err)
					return nil, fmt.Errorf("authentication failed and token refresh failed: %v", refreshErr)
				}
				logging.PageLogger.Debug("Token refreshed, retrying Graph API call", "section_id", sectionID)
				result, err = c.GraphClient.Me().Onenote().Sections().ByOnenoteSectionId(sectionID).Pages().Get(ctx, nil)
				if err != nil {
					logging.PageLogger.Error("Graph API call failed after token refresh", "section_id", sectionID, "error", err)
					return nil, err
				}
				logging.PageLogger.Debug("Graph API call succeeded after token refresh", "section_id", sectionID)
			} else {
				logging.PageLogger.Error("Authentication error but no token manager available", "section_id", sectionID, "error", err)
				return nil, err
			}
		} else {
			logging.PageLogger.Error("Graph API call failed with non-auth error", "section_id", sectionID, "error", err)
			return nil, err
		}
	}

	var pages []map[string]interface{}
	if result != nil {
		pages = append(pages, extractOnenotePageList(result)...)
		logging.PageLogger.Debug("Initial page batch retrieved", "section_id", sectionID, "page_count", len(pages))
	}
	nextLink := result.GetOdataNextLink()
	paginationCount := 0
	for nextLink != nil {
		paginationCount++
		logging.PageLogger.Debug("Processing pagination", "section_id", sectionID, "pagination_count", paginationCount, "next_link", *nextLink)
		requestInfo := abstractions.NewRequestInformation()
		requestInfo.UrlTemplate = *nextLink
		requestInfo.Method = abstractions.GET
		resp, err := c.GraphClient.GetAdapter().Send(ctx, requestInfo, msgraphmodels.CreateOnenotePageCollectionResponseFromDiscriminatorValue, nil)
		if err != nil {
			logging.PageLogger.Error("Pagination request failed", "section_id", sectionID, "pagination_count", paginationCount, "error", err)
			return nil, fmt.Errorf("failed to fetch next page of pages: %v", err)
		}
		pageResp := resp.(msgraphmodels.OnenotePageCollectionResponseable)
		batchPages := extractOnenotePageList(pageResp)
		pages = append(pages, batchPages...)
		logging.PageLogger.Debug("Pagination batch retrieved", "section_id", sectionID, "pagination_count", paginationCount, "batch_size", len(batchPages), "total_pages", len(pages))
		nextLink = pageResp.GetOdataNextLink()
	}
	logging.PageLogger.Info("ListPages completed successfully", "section_id", sectionID, "total_pages", len(pages), "pagination_requests", paginationCount)
	return pages, nil
}

// extractOnenotePageList extracts page info from a OnenotePageCollectionResponseable
func extractOnenotePageList(result msgraphmodels.OnenotePageCollectionResponseable) []map[string]interface{} {
	var pages []map[string]interface{}
	for _, page := range result.GetValue() {
		m := map[string]interface{}{}
		if page.GetId() != nil {
			m["pageId"] = *page.GetId()
		}
		if page.GetTitle() != nil {
			m["title"] = *page.GetTitle()
		}
		pages = append(pages, m)
	}
	return pages
}

// GetPageContent fetches the HTML content of a page by pageID using the Microsoft Graph SDK.
// pageID: ID of the page to fetch content for.
// forUpdate: Optional boolean that when true includes includeIDs=true parameter for update operations.
// Returns the HTML content as a string and an error, if any.
func (c *PageClient) GetPageContent(pageID string, forUpdate ...bool) (string, error) {
	logging.PageLogger.Info("Starting GetPageContent operation", "page_id", pageID, "for_update", len(forUpdate) > 0 && forUpdate[0])

	// Check if token is expired and refresh if needed
	if c.TokenManager != nil && c.TokenManager.IsExpired() {
		logging.PageLogger.Debug("Token expired, refreshing before GetPageContent", "page_id", pageID)
		if err := c.RefreshTokenIfNeeded(); err != nil {
			logging.PageLogger.Error("Token refresh failed during GetPageContent", "page_id", pageID, "error", err)
			return "", fmt.Errorf("token expired and refresh failed: %v", err)
		}
		logging.PageLogger.Debug("Token refreshed successfully for GetPageContent", "page_id", pageID)
	}

	// Check if forUpdate parameter is provided and true
	includeIDs := false
	if len(forUpdate) > 0 && forUpdate[0] {
		includeIDs = true
	}
	logging.PageLogger.Debug("GetPageContent parameters determined", "page_id", pageID, "include_ids", includeIDs)

	ctx := context.Background()

	// Use direct HTTP call to support includeIDs parameter
	if includeIDs {
		// Use direct HTTP call with includeIDs parameter
		url := fmt.Sprintf("https://graph.microsoft.com/v1.0/me/onenote/pages/%s/content?includeIDs=true", pageID)
		logging.PageLogger.Debug("Using direct HTTP call with includeIDs", "page_id", pageID, "url", url)

		resp, err := c.MakeAuthenticatedRequest("GET", url, nil, nil)
		if err != nil {
			logging.PageLogger.Error("Direct HTTP request failed for GetPageContent", "page_id", pageID, "url", url, "error", err)
			return "", err
		}
		defer resp.Body.Close()

		if errHandle := c.HandleHTTPResponse(resp, "GetPageContent"); errHandle != nil {
			logging.PageLogger.Error("HTTP response handling failed for GetPageContent", "page_id", pageID, "status", resp.StatusCode, "error", errHandle)
			return "", errHandle
		}

		content, err := c.ReadResponseBody(resp, "GetPageContent")
		if err != nil {
			logging.PageLogger.Error("Failed to read response body for GetPageContent", "page_id", pageID, "error", err)
			return "", err
		}

		logging.PageLogger.Info("GetPageContent completed successfully via HTTP", "page_id", pageID, "content_length", len(content), "include_ids", true)
		return string(content), nil
	}
	// Use SDK for normal content retrieval
	logging.PageLogger.Debug("Using Graph SDK for normal content retrieval", "page_id", pageID)
	content, err := c.GraphClient.Me().Onenote().Pages().ByOnenotePageId(pageID).Content().Get(ctx, nil)
	if err != nil {
		logging.PageLogger.Debug("SDK content retrieval failed", "page_id", pageID, "error", err)
		// Check if this is an auth error and try token refresh
		if strings.Contains(err.Error(), "JWT") || strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "403") {
			logging.PageLogger.Debug("Authentication error detected in SDK call, attempting token refresh", "page_id", pageID)
			if c.TokenManager != nil && c.OAuthConfig != nil {
				if refreshErr := c.RefreshTokenIfNeeded(); refreshErr != nil {
					logging.PageLogger.Error("Token refresh failed after SDK auth error", "page_id", pageID, "refresh_error", refreshErr, "original_error", err)
					return "", fmt.Errorf("authentication failed and token refresh failed: %v", refreshErr)
				}
				logging.PageLogger.Debug("Token refreshed, retrying SDK call", "page_id", pageID)
				// Retry the operation with fresh token
				content, err = c.GraphClient.Me().Onenote().Pages().ByOnenotePageId(pageID).Content().Get(ctx, nil)
				if err != nil {
					logging.PageLogger.Error("SDK call failed after token refresh", "page_id", pageID, "error", err)
					return "", err
				}
				logging.PageLogger.Debug("SDK call succeeded after token refresh", "page_id", pageID)
			} else {
				logging.PageLogger.Error("Authentication error but no token manager available", "page_id", pageID, "error", err)
				return "", err
			}
		} else {
			logging.PageLogger.Error("SDK call failed with non-auth error", "page_id", pageID, "error", err)
			return "", err
		}
	}
	logging.PageLogger.Info("GetPageContent completed successfully via SDK", "page_id", pageID, "content_length", len(content), "include_ids", false)
	return string(content), nil
}

// CreatePage creates a new page in a section.
// sectionID: ID of the section to create the page in.
// title: Title of the new page.
// content: HTML content for the page.
// Returns the created page metadata and an error, if any.
func (c *PageClient) CreatePage(sectionID, title, content string) (map[string]interface{}, error) {
	logging.PageLogger.Info("Starting CreatePage operation", "section_id", sectionID, "title", title, "content_length", len(content))
	// Log the original content at configurable verbosity level
	logging.LogContent(logging.PageLogger, slog.LevelDebug, "CreatePage original content", "section_id", sectionID, "title", title, "original_content", content)
	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/me/onenote/sections/%s/pages", sectionID)

	// Ensure the HTML content includes a <title> tag
	finalContent := content
	if !strings.Contains(strings.ToLower(content), "<title>") {
		finalContent = "<html><head><title>" + htmlEscape(title) + "</title></head><body>" + content + "</body></html>"
		logging.PageLogger.Debug("Added HTML wrapper with title tag", "section_id", sectionID, "title", title, "final_content_length", len(finalContent))
	} else {
		logging.PageLogger.Debug("Content already contains title tag", "section_id", sectionID, "title", title, "content_length", len(finalContent))
	}

	// Log the actual content at configurable verbosity level
	logging.LogContent(logging.PageLogger, slog.LevelDebug, "CreatePage final content", "section_id", sectionID, "title", title, "final_content", finalContent)

	// Make authenticated request
	headers := map[string]string{"Content-Type": "application/xhtml+xml"}
	logging.PageLogger.Debug("Making authenticated request to create page", "section_id", sectionID, "url", url, "content_type", headers["Content-Type"])
	resp, err := c.MakeAuthenticatedRequest("POST", url, strings.NewReader(finalContent), headers)
	if err != nil {
		logging.PageLogger.Error("Authenticated request failed for CreatePage", "section_id", sectionID, "title", title, "error", err)
		return nil, err
	}
	defer resp.Body.Close()

	// Handle HTTP response
	if err := c.HandleHTTPResponse(resp, "CreatePage"); err != nil {
		logging.PageLogger.Error("HTTP response handling failed for CreatePage", "section_id", sectionID, "title", title, "status", resp.StatusCode, "error", err)
		return nil, err
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		logging.PageLogger.Error("Failed to decode response for CreatePage", "section_id", sectionID, "title", title, "error", err)
		return nil, err
	}

	// Log successful creation with page ID if available
	pageID, hasID := result["id"].(string)
	if hasID {
		logging.PageLogger.Info("CreatePage completed successfully", "section_id", sectionID, "title", title, "page_id", pageID, "content_length", len(content))
	} else {
		logging.PageLogger.Info("CreatePage completed successfully but no page ID in response", "section_id", sectionID, "title", title, "content_length", len(content))
	}
	return result, nil
}

// htmlEscape escapes special HTML characters in a string for safe insertion in <title>.
func htmlEscape(s string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
		"'", "&#39;",
	)
	return replacer.Replace(s)
}

// validateTableUpdates checks if any commands are attempting to update individual table elements
// and returns an error with guidance if table elements are targeted individually.
func validateTableUpdates(commands []UpdateCommand) error {
	var tableElementTargets []string

	for _, cmd := range commands {
		// Check if target is a table element (td, th, tr)
		if strings.HasPrefix(cmd.Target, "td:") ||
			strings.HasPrefix(cmd.Target, "th:") ||
			strings.HasPrefix(cmd.Target, "tr:") {
			tableElementTargets = append(tableElementTargets, cmd.Target)
		}
	}

	if len(tableElementTargets) > 0 {
		return fmt.Errorf(`TABLE UPDATE RESTRICTION: You are attempting to update individual table elements (%v). 

OneNote requires that tables be updated as complete units, not individual cells or rows. 

SOLUTION: Instead of updating individual table elements, you must:
1. Target the entire table element (table:data-id) 
2. Replace the complete table HTML with your updated content
3. Include all table structure (table, tr, td/th elements) in your replacement

Example of CORRECT approach:
- Target: "table:{table-data-id}" 
- Action: "replace"
- Content: "<table>...complete table HTML...</table>"

Example of INCORRECT approach (what you're doing):
- Target: "td:{cell-data-id}"
- Action: "replace" 
- Content: "<td>new content</td>"

This restriction ensures table integrity and prevents layout corruption in OneNote`, tableElementTargets)
	}

	return nil
}

// UpdatePageContent updates the HTML content of a page using Microsoft Graph OneNote API.
// commands: Array of update commands defining the changes to make.
// Returns an error if the update fails.
func (c *PageClient) UpdatePageContent(pageID string, commands []UpdateCommand) error {

	// Sanitize and validate the pageID to prevent injection attacks
	sanitizedPageID, err := c.SanitizeOneNoteID(pageID, "pageID")
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/me/onenote/pages/%s/content", sanitizedPageID)

	if len(commands) == 0 {
		logging.PageLogger.Error("No update commands provided", "page_id", pageID)
		return fmt.Errorf("no update commands provided")
	}

	// Log command details for debugging including actual content at verbose level
	for i, cmd := range commands {
		logging.PageLogger.Debug("Update command details", "page_id", pageID, "command_index", i, "target", cmd.Target, "action", cmd.Action, "position", cmd.Position, "content_length", len(cmd.Content))
		// Log the actual command content at configurable verbosity level
		logging.LogContent(logging.PageLogger, slog.LevelDebug, "Update command content", "page_id", pageID, "command_index", i, "target", cmd.Target, "action", cmd.Action, "content", cmd.Content)
	}

	// Validate table updates to prevent individual table element modifications
	logging.PageLogger.Debug("Validating table update restrictions", "page_id", pageID)
	if errValidate := validateTableUpdates(commands); errValidate != nil {
		logging.PageLogger.Error("Table update validation failed", "page_id", pageID, "error", errValidate)
		return errValidate
	}
	logging.PageLogger.Debug("Table update validation passed", "page_id", pageID)

	// --- NEW LOGIC: Scan for <img> or <object> tags with Graph API URLs ---
	type resourcePart struct {
		ContentID   string
		Content     []byte
		ContentType string
		Filename    string
	}
	resourceParts := []resourcePart{}
	resourceCounter := 1

	rewriteHTML := func(htmlIn string) (string, error) {
		doc, errParse := html.Parse(strings.NewReader(htmlIn))
		if errParse != nil {
			return htmlIn, nil // fallback: return original if parse fails
		}
		var changed bool
		var traverse func(*html.Node)
		traverse = func(n *html.Node) {
			if n.Type == html.ElementNode && (n.Data == imgTag || n.Data == objectTag) {
				var urlAttr string
				if n.Data == imgTag {
					urlAttr = "src"
				} else if n.Data == objectTag {
					urlAttr = "data"
				}
				for _, attr := range n.Attr {
					if attr.Key == urlAttr && strings.HasPrefix(attr.Val, "https://graph.microsoft.com/") {
						// Extract resource ID
						pageItemID := extractPageItemID(attr.Val)
						if pageItemID != "" {
							// Download the resource
							item, itemErr := c.GetPageItem(pageID, pageItemID)
							if itemErr != nil {
								continue
							}
							contentID := fmt.Sprintf("part%d", resourceCounter)
							resourceCounter++
							resourceParts = append(resourceParts, resourcePart{
								ContentID:   contentID,
								Content:     item.Content,
								ContentType: item.ContentType,
								Filename:    item.Filename,
							})
							// Remove all attributes and only include the src/data attribute
							n.Attr = []html.Attribute{{Key: urlAttr, Val: "name:" + contentID}}
							changed = true
							break
						}
					}
				}
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				traverse(c)
			}
		}
		traverse(doc)
		if !changed {
			return htmlIn, nil
		}
		var buf bytes.Buffer
		html.Render(&buf, doc)
		return buf.String(), nil
	}

	// Rewrite HTML in all commands and collect resource parts
	logging.PageLogger.Debug("Rewriting HTML in commands to handle embedded resources", "page_id", pageID)
	for i := range commands {
		if commands[i].Content != "" {
			originalLength := len(commands[i].Content)
			rewritten, _ := rewriteHTML(commands[i].Content)
			commands[i].Content = rewritten
			logging.PageLogger.Debug("Rewrote command HTML content", "page_id", pageID, "command_index", i, "original_length", originalLength, "rewritten_length", len(rewritten))
		}
	}
	logging.PageLogger.Debug("HTML rewriting completed", "page_id", pageID, "resource_parts_found", len(resourceParts))

	commandsJSON, err := json.Marshal(commands)
	if err != nil {
		logging.PageLogger.Error("Failed to marshal commands to JSON", "page_id", pageID, "error", err)
		return fmt.Errorf("failed to marshal commands: %v", err)
	}
	logging.PageLogger.Debug("Commands marshaled to JSON", "page_id", pageID, "json_length", len(commandsJSON))
	// Log the actual JSON commands at configurable verbosity level
	logging.LogContent(logging.PageLogger, slog.LevelDebug, "Commands JSON content", "page_id", pageID, "commands_json", string(commandsJSON))

	// Create multipart form data
	logging.PageLogger.Debug("Creating multipart form data", "page_id", pageID, "resource_parts", len(resourceParts))
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add the commands part
	commandsPart, err := writer.CreateFormFile("commands", "commands.json")
	if err != nil {
		logging.PageLogger.Error("Failed to create commands part", "page_id", pageID, "error", err)
		return fmt.Errorf("failed to create commands part: %v", err)
	}

	_, err = commandsPart.Write(commandsJSON)
	if err != nil {
		logging.PageLogger.Error("Failed to write commands to form file", "page_id", pageID, "error", err)
		return fmt.Errorf("failed to write commands to form file: %v", err)
	}
	logging.PageLogger.Debug("Commands part added to multipart form", "page_id", pageID)

	// Add resource parts
	for i, part := range resourceParts {
		logging.PageLogger.Debug("Adding resource part to multipart form", "page_id", pageID, "part_index", i, "content_id", part.ContentID, "filename", part.Filename, "content_type", part.ContentType, "size", len(part.Content))
		partHeader := make(textproto.MIMEHeader)
		partHeader.Set("Content-Disposition", fmt.Sprintf("form-data; name=\"%s\"; filename=\"%s\"", part.ContentID, part.Filename))
		partHeader.Set("Content-Type", part.ContentType)
		resourceWriter, errPart := writer.CreatePart(partHeader)
		if errPart != nil {
			logging.PageLogger.Debug("Failed to create resource part, skipping", "page_id", pageID, "content_id", part.ContentID, "error", errPart)
			continue
		}
		_, err = resourceWriter.Write(part.Content)
		if err != nil {
			logging.PageLogger.Debug("Failed to write resource content, skipping", "page_id", pageID, "content_id", part.ContentID, "error", err)
			continue
		}
	}

	// Close the writer
	writer.Close()

	// Get the final multipart content type
	contentType := writer.FormDataContentType()

	// Make authenticated request with multipart content type
	headers := map[string]string{"Content-Type": contentType}
	logging.PageLogger.Debug("Making authenticated PATCH request", "page_id", pageID, "url", url, "content_type", contentType, "body_size", buf.Len())

	resp, err := c.MakeAuthenticatedRequest("PATCH", url, &buf, headers)
	if err != nil {
		logging.PageLogger.Error("Authenticated PATCH request failed for UpdatePageContent", "page_id", pageID, "error", err)
		return err
	}
	defer resp.Body.Close()

	// Handle HTTP response
	if err := c.HandleHTTPResponse(resp, "UpdatePageContent"); err != nil {
		logging.PageLogger.Error("HTTP response handling failed for UpdatePageContent", "page_id", pageID, "status", resp.StatusCode, "error", err)
		return err
	}

	logging.PageLogger.Info("UpdatePageContent completed successfully", "page_id", pageID, "command_count", len(commands), "resource_parts", len(resourceParts))
	return nil
}

// UpdatePageContentSimple provides backward compatibility for simple content replacement.
// This function replaces the entire body content of a page.
// For more complex updates, use UpdatePageContent with UpdateCommand array.
func (c *PageClient) UpdatePageContentSimple(pageID, content string) error {
	commands := []UpdateCommand{
		{
			Target:  "body",
			Action:  "replace",
			Content: content,
		},
	}

	return c.UpdatePageContent(pageID, commands)
}

// DeletePage deletes a page by pageID using the Microsoft Graph SDK.
// Returns an error if the deletion fails.
func (c *PageClient) DeletePage(pageID string) error {
	logging.PageLogger.Info("Starting DeletePage operation", "page_id", pageID)
	ctx := context.Background()
	err := c.GraphClient.Me().Onenote().Pages().ByOnenotePageId(pageID).Delete(ctx, nil)
	if err != nil {
		logging.PageLogger.Error("DeletePage failed", "page_id", pageID, "error", err)
		return fmt.Errorf("[pages] DeletePage failed: %v", err)
	}
	logging.PageLogger.Info("DeletePage completed successfully", "page_id", pageID)
	return nil
}

// CopyPage copies a page from one section to another using direct HTTP API calls.
// pageID: ID of the page to copy.
// targetSectionID: ID of the target section to copy the page to.
// Returns the copied page metadata and an error, if any.
func (c *PageClient) CopyPage(pageID string, targetSectionID string) (map[string]interface{}, error) {
	logging.PageLogger.Info("Starting CopyPage operation", "page_id", pageID, "target_section_id", targetSectionID)

	// Validate and sanitize inputs
	sanitizedPageID, err := c.SanitizeOneNoteID(pageID, "pageID")
	if err != nil {
		logging.PageLogger.Error("Page ID sanitization failed", "page_id", pageID, "error", err)
		return nil, err
	}
	sanitizedTargetSectionID, err := c.SanitizeOneNoteID(targetSectionID, "targetSectionID")
	if err != nil {
		logging.PageLogger.Error("Target section ID sanitization failed", "target_section_id", targetSectionID, "error", err)
		return nil, err
	}
	logging.PageLogger.Debug("Input sanitization completed", "sanitized_page_id", sanitizedPageID, "sanitized_target_section_id", sanitizedTargetSectionID)

	// Construct the URL for copying a page to a section
	url := fmt.Sprintf("https://graph.microsoft.com/beta/me/onenote/pages/%s/copyToSection", sanitizedPageID)
	logging.PageLogger.Debug("Copy URL constructed", "page_id", pageID, "url", url)

	// Create the request body for copying to section
	requestBody := map[string]interface{}{
		"id": sanitizedTargetSectionID,
	}

	// Marshal the request body to JSON
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		logging.PageLogger.Error("Failed to marshal request body", "page_id", pageID, "error", err)
		return nil, fmt.Errorf("failed to marshal request body: %v", err)
	}
	logging.PageLogger.Debug("Request body marshaled", "page_id", pageID, "body_length", len(jsonBody))

	// Make authenticated request
	headers := map[string]string{"Content-Type": "application/json"}
	logging.PageLogger.Debug("Making authenticated request to copy page", "page_id", pageID, "target_section_id", targetSectionID)
	resp, err := c.MakeAuthenticatedRequest("POST", url, bytes.NewBuffer(jsonBody), headers)
	if err != nil {
		logging.PageLogger.Error("Authenticated request failed for CopyPage", "page_id", pageID, "target_section_id", targetSectionID, "error", err)
		return nil, err
	}
	defer resp.Body.Close()

	// Check for 202 status code (Accepted - asynchronous operation)
	logging.PageLogger.Debug("Copy request response received", "page_id", pageID, "status_code", resp.StatusCode)
	if resp.StatusCode != 202 {
		logging.PageLogger.Error("Copy operation failed with unexpected status", "page_id", pageID, "expected_status", 202, "actual_status", resp.StatusCode)
		return nil, fmt.Errorf("copy operation failed: expected status 202, got %d", resp.StatusCode)
	}

	// Read response body
	content, err := c.ReadResponseBody(resp, "CopyPage")
	if err != nil {
		logging.PageLogger.Error("Failed to read response body for CopyPage", "page_id", pageID, "error", err)
		return nil, err
	}
	logging.PageLogger.Debug("Copy response body read", "page_id", pageID, "content_length", len(content))

	// Parse the response JSON
	var result map[string]interface{}
	if err := json.Unmarshal(content, &result); err != nil {
		logging.PageLogger.Error("Failed to parse copy response JSON", "page_id", pageID, "error", err)
		return nil, fmt.Errorf("failed to parse copy response: %v", err)
	}

	// Extract status and id from the response
	status, statusExists := result["status"].(string)
	if !statusExists {
		logging.PageLogger.Error("No status field found in copy response", "page_id", pageID)
		return nil, fmt.Errorf("no status field found in copy response")
	}

	operationID, idExists := result["id"].(string)
	if !idExists {
		logging.PageLogger.Error("No operation ID found in copy response", "page_id", pageID)
		return nil, fmt.Errorf("no id field found in copy response")
	}
	logging.PageLogger.Debug("Async copy operation initiated", "page_id", pageID, "operation_id", operationID, "initial_status", status)

	// Poll for operation completion
	maxAttempts := 30
	logging.PageLogger.Debug("Starting operation polling", "page_id", pageID, "operation_id", operationID, "max_attempts", maxAttempts)
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		logging.PageLogger.Debug("Polling operation status", "page_id", pageID, "operation_id", operationID, "attempt", attempt)

		// Get operation status
		operationResult, err := c.GetOnenoteOperation(operationID)
		if err != nil {
			logging.PageLogger.Error("Failed to get operation status", "page_id", pageID, "operation_id", operationID, "attempt", attempt, "error", err)
			return nil, fmt.Errorf("failed to get operation status: %v", err)
		}

		// Check operation status
		operationStatus, statusExists := operationResult["status"].(string)
		if !statusExists {
			logging.PageLogger.Error("No status field in operation result", "page_id", pageID, "operation_id", operationID, "attempt", attempt)
			return nil, fmt.Errorf("no status field in operation result")
		}
		logging.PageLogger.Debug("Operation status retrieved", "page_id", pageID, "operation_id", operationID, "attempt", attempt, "status", operationStatus)

		// Check if operation is completed
		if operationStatus == "Completed" {
			logging.PageLogger.Debug("Copy operation completed", "page_id", pageID, "operation_id", operationID, "attempt", attempt)

			// Extract page ID from resourceLocation
			newPageID, err := c.extractPageIDFromResourceLocation(operationResult)
			if err != nil {
				logging.PageLogger.Error("Failed to extract new page ID from operation result", "original_page_id", pageID, "operation_id", operationID, "error", err)
				return nil, fmt.Errorf("failed to extract page ID from operation result: %v", err)
			}

			logging.PageLogger.Info("CopyPage completed successfully", "original_page_id", pageID, "new_page_id", newPageID, "target_section_id", targetSectionID, "operation_id", operationID, "attempts", attempt)
			return map[string]interface{}{
				"id":          newPageID,
				"operationId": operationID,
				"status":      "Completed",
			}, nil
		}

		// Check if operation failed
		if operationStatus == "Failed" {
			logging.PageLogger.Error("Copy operation failed", "page_id", pageID, "operation_id", operationID, "attempt", attempt)
			return nil, fmt.Errorf("copy operation failed")
		}

		// Handle special "Running" status (returned when 503 error is encountered)
		if operationStatus == "Running" {
			// Check if we got additional information about the 503 response
			if note, hasNote := operationResult["note"].(string); hasNote {
				logging.PageLogger.Info("Copy operation still in progress (503 response)",
					"page_id", pageID,
					"operation_id", operationID,
					"attempt", attempt,
					"note", note)
			} else {
				logging.PageLogger.Debug("Copy operation still running", "page_id", pageID, "operation_id", operationID, "attempt", attempt)
			}
		}

		// If not completed, wait before next attempt
		if attempt < maxAttempts {
			// Random delay between 1-3 seconds
			delay := time.Duration(1+rand.Intn(2+attempt)) * time.Second

			// Log differently based on whether we got a 503 or just normal progress
			if operationStatus == "Running" {
				logging.PageLogger.Info("Continuing to wait for copy operation completion after 503 response",
					"page_id", pageID,
					"operation_id", operationID,
					"attempt", attempt,
					"delay", delay,
					"status", operationStatus)
			} else {
				logging.PageLogger.Debug("Waiting before next polling attempt",
					"page_id", pageID,
					"operation_id", operationID,
					"attempt", attempt,
					"delay", delay,
					"status", operationStatus)
			}

			time.Sleep(delay)
		}
	}

	logging.PageLogger.Error("Copy operation timed out", "page_id", pageID, "operation_id", operationID, "max_attempts", maxAttempts)
	return nil, fmt.Errorf("copy operation did not complete within %d attempts", maxAttempts)
}

// GetOnenoteOperation retrieves the status of an asynchronous OneNote operation.
// operationID: ID of the operation to check.
// Returns the operation status and metadata, and an error, if any.
func (c *PageClient) GetOnenoteOperation(operationID string) (map[string]interface{}, error) {
	logging.PageLogger.Debug("Starting GetOnenoteOperation", "operation_id", operationID)

	// Validate and sanitize the operation ID
	sanitizedOperationID, err := c.SanitizeOneNoteID(operationID, "operationID")
	if err != nil {
		logging.PageLogger.Error("Operation ID sanitization failed", "operation_id", operationID, "error", err)
		return nil, err
	}

	// Construct the URL for getting operation status
	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/me/onenote/operations/%s", sanitizedOperationID)
	logging.PageLogger.Debug("Operation status URL constructed", "operation_id", operationID, "url", url)

	// Make authenticated request
	resp, err := c.MakeAuthenticatedRequest("GET", url, nil, nil)
	if err != nil {
		logging.PageLogger.Error("Authenticated request failed for GetOnenoteOperation", "operation_id", operationID, "error", err)
		return nil, err
	}
	defer resp.Body.Close()

	// Handle HTTP response - special handling for 503 Service Unavailable
	if resp.StatusCode == 503 {
		// 503 indicates the operation is still in progress, not an error condition
		logging.PageLogger.Warn("Operation status check returned 503 Service Unavailable - operation still in progress",
			"operation_id", operationID,
			"status_code", resp.StatusCode,
			"note", "This is expected during long-running copy operations")

		// Return a result indicating the operation is still running
		return map[string]interface{}{
			"status": "Running",
			"id":     operationID,
			"note":   "Operation is still in progress (503 response received)",
		}, nil
	}

	// Handle other HTTP responses normally
	if errHandle := c.HandleHTTPResponse(resp, "GetOnenoteOperation"); errHandle != nil {
		logging.PageLogger.Error("HTTP response handling failed for GetOnenoteOperation", "operation_id", operationID, "status", resp.StatusCode, "error", errHandle)
		return nil, errHandle
	}

	// Read response body
	content, err := c.ReadResponseBody(resp, "GetOnenoteOperation")
	if err != nil {
		logging.PageLogger.Error("Failed to read response body for GetOnenoteOperation", "operation_id", operationID, "error", err)
		return nil, err
	}

	// Parse the response JSON
	var result map[string]interface{}
	if err := json.Unmarshal(content, &result); err != nil {
		logging.PageLogger.Error("Failed to parse operation status response", "operation_id", operationID, "error", err)
		return nil, fmt.Errorf("failed to parse operation status response: %v", err)
	}

	// Log the operation status if available
	if status, hasStatus := result["status"].(string); hasStatus {
		logging.PageLogger.Debug("GetOnenoteOperation completed", "operation_id", operationID, "status", status)
	} else {
		logging.PageLogger.Debug("GetOnenoteOperation completed but no status in response", "operation_id", operationID)
	}
	return result, nil
}

// extractPageIDFromResourceLocation extracts the page ID from a resourceLocation URL in JSON response.
// jsonData: JSON data that contains a resourceLocation field with a page URL.
// Returns the page ID extracted from the URL and an error, if any.
func (c *PageClient) extractPageIDFromResourceLocation(jsonData map[string]interface{}) (string, error) {
	logging.PageLogger.Debug("Starting extractPageIDFromResourceLocation")

	// Check if resourceLocation exists in the JSON
	resourceLocation, exists := jsonData["resourceLocation"]
	if !exists {
		logging.PageLogger.Debug("No resourceLocation field found in JSON data")
		return "", fmt.Errorf("resourceLocation field not found in JSON data")
	}

	// Convert to string
	url, ok := resourceLocation.(string)
	if !ok {
		logging.PageLogger.Debug("ResourceLocation is not a string", "type", fmt.Sprintf("%T", resourceLocation))
		return "", fmt.Errorf("resourceLocation is not a string")
	}
	logging.PageLogger.Debug("Found resourceLocation URL", "url", url)

	// Extract page ID from the URL using regex
	// Pattern matches URLs like: https://graph.microsoft.com/beta/users/.../onenote/pages/{pageId}
	pageIDRegex := regexp.MustCompile(`/onenote/pages/([A-Za-z0-9\-!]+)`)
	matches := pageIDRegex.FindStringSubmatch(url)

	if len(matches) < 2 {
		logging.PageLogger.Debug("Could not extract page ID from URL", "url", url)
		return "", fmt.Errorf("could not extract page ID from URL: %s", url)
	}

	pageID := matches[1]
	logging.PageLogger.Debug("Extracted page ID from URL", "url", url, "page_id", pageID)

	// Validate the extracted page ID
	if _, err := c.SanitizeOneNoteID(pageID, "extracted page ID"); err != nil {
		logging.PageLogger.Debug("Extracted page ID failed validation", "page_id", pageID, "error", err)
		return "", fmt.Errorf("extracted page ID failed validation: %v", err)
	}

	logging.PageLogger.Debug("Successfully extracted and validated page ID", "page_id", pageID)
	return pageID, nil
}

// MovePage moves a page from one section to another by copying it to the target section and deleting the original.
// pageID: ID of the page to move.
// targetSectionID: ID of the target section to move the page to.
// Returns the moved page metadata and an error, if any.
func (c *PageClient) MovePage(pageID string, targetSectionID string) (map[string]interface{}, error) {
	logging.PageLogger.Info("Starting MovePage operation", "page_id", pageID, "target_section_id", targetSectionID)

	// First, copy the page to the target section
	logging.PageLogger.Debug("Copying page for move operation", "page_id", pageID, "target_section_id", targetSectionID)
	copiedPage, err := c.CopyPage(pageID, targetSectionID)
	if err != nil {
		logging.PageLogger.Error("Copy operation failed during move", "page_id", pageID, "target_section_id", targetSectionID, "error", err)
		return nil, fmt.Errorf("failed to copy page for move operation: %v", err)
	}
	logging.PageLogger.Info("Page copied successfully, proceeding with deletion", "original_page_id", pageID, "new_page_id", copiedPage["id"], "target_section_id", targetSectionID)

	// Then, delete the original page
	logging.PageLogger.Debug("Deleting original page after successful copy", "page_id", pageID)
	err = c.DeletePage(pageID)
	if err != nil {
		logging.PageLogger.Warn("Failed to delete original page after copy, but move operation succeeded", "original_page_id", pageID, "new_page_id", copiedPage["id"], "delete_error", err)
		// Note: We don't return an error here because the copy was successful
		// The user now has the page in the target section, even though the original wasn't deleted
	} else {
		logging.PageLogger.Debug("Original page deleted successfully", "page_id", pageID)
	}

	logging.PageLogger.Info("MovePage completed successfully", "original_page_id", pageID, "new_page_id", copiedPage["id"], "target_section_id", targetSectionID)
	return copiedPage, nil
}

// GetPageItem retrieves complete data for a OneNote page item (image, file, etc.) by its ID.
// This enhanced version matches the original sophisticated implementation with:
// - HTML metadata extraction using ListPageItems (avoids duplicate parsing)
// - Content type detection from HTML data-src-type attribute (primary) and HTTP headers (fallback)
// - Automatic image scaling for large images (can be disabled with fullSize parameter)
// - Rich metadata extraction from HTML attributes
// pageID: ID of the page containing the item.
// pageItemID: Resource ID of the page item to retrieve.
// fullSize: Optional parameter to skip image scaling (defaults to false = scale images)
// Returns PageItemData with metadata and binary content, and an error if retrieval fails.
func (c *PageClient) GetPageItem(pageID, pageItemID string, fullSize ...bool) (*PageItemData, error) {
	logging.PageLogger.Info("Starting GetPageItem operation", "page_id", pageID, "page_item_id", pageItemID, "full_size", len(fullSize) > 0 && fullSize[0])

	// Check if token is expired and refresh if needed
	if c.TokenManager != nil && c.TokenManager.IsExpired() {
		logging.PageLogger.Debug("Token expired, refreshing before GetPageItem", "page_id", pageID, "page_item_id", pageItemID)
		if err := c.RefreshTokenIfNeeded(); err != nil {
			logging.PageLogger.Error("Token refresh failed during GetPageItem", "page_id", pageID, "page_item_id", pageItemID, "error", err)
			return nil, fmt.Errorf("token expired and refresh failed: %v", err)
		}
		logging.PageLogger.Debug("Token refreshed successfully for GetPageItem", "page_id", pageID, "page_item_id", pageItemID)
	}

	// Validate and sanitize inputs
	sanitizedPageID, err := c.SanitizeOneNoteID(pageID, "pageID")
	if err != nil {
		logging.PageLogger.Error("Page ID sanitization failed", "page_id", pageID, "error", err)
		return nil, err
	}
	sanitizedPageItemID, err := c.SanitizeOneNoteID(pageItemID, "pageItemID")
	if err != nil {
		logging.PageLogger.Error("Page item ID sanitization failed", "page_item_id", pageItemID, "error", err)
		return nil, err
	}

	logging.PageLogger.Debug("Getting page item using sophisticated approach", "page_id", sanitizedPageID, "page_item_id", sanitizedPageItemID)

	// First, get the page items list to extract HTML metadata (original approach)
	logging.PageLogger.Debug("Getting page items list to extract HTML metadata")
	pageItemsList, err := c.ListPageItems(sanitizedPageID)
	if err != nil {
		logging.PageLogger.Error("Failed to get page items list", "page_id", pageID, "page_item_id", pageItemID, "error", err)
		return nil, fmt.Errorf("failed to get page items list: %v", err)
	}

	// Find the specific page item in the list to get its HTML metadata
	var htmlMetadata *PageItemInfo
	for _, itemMap := range pageItemsList {
		logging.PageLogger.Debug("Checking item in list", "item_map", itemMap)
		if itemID, exists := itemMap["pageItemId"].(string); exists && itemID == sanitizedPageItemID {
			logging.PageLogger.Debug("Found matching page item", "page_item_id", itemID)

			// Safely extract tagName
			tagName := ""
			if tagNameVal, exists := itemMap["tagName"]; exists {
				if tagNameStr, ok := tagNameVal.(string); ok {
					tagName = tagNameStr
				}
			}

			// Initialize attributes map and add data-attachment if present
			attributes := make(map[string]string)
			if dataAttachmentVal, exists := itemMap["data-attachment"]; exists {
				if dataAttachmentStr, ok := dataAttachmentVal.(string); ok {
					attributes["data-attachment"] = dataAttachmentStr
					logging.PageLogger.Debug("Found data-attachment attribute", "value", dataAttachmentStr)
				}
			}

			// Extract mimeType if available - check both data-src-type and type attributes
			var mimeTypeFromHTML string
			if mimeTypeVal, exists := itemMap["mimeType"]; exists {
				if mimeTypeStr, ok := mimeTypeVal.(string); ok {
					mimeTypeFromHTML = mimeTypeStr
					// Store the MIME type in appropriate attribute based on tag type
					if tagName == "object" {
						attributes["type"] = mimeTypeStr
					} else {
						attributes["data-src-type"] = mimeTypeStr
					}
					logging.PageLogger.Debug("Found mimeType from HTML", "value", mimeTypeStr, "tag_name", tagName)
				}
			}

			htmlMetadata = &PageItemInfo{
				TagName:     tagName,
				PageItemID:  itemID,
				Attributes:  attributes,
				OriginalURL: "", // No longer available in simplified structure
			}

			logging.PageLogger.Debug("Found HTML metadata for page item", "tag_name", htmlMetadata.TagName, "attributes", htmlMetadata.Attributes, "page_item_id", htmlMetadata.PageItemID, "mime_type_from_html", mimeTypeFromHTML)
			break
		}
	}

	// Construct the resource URL for downloading content
	resourceURL := fmt.Sprintf("https://graph.microsoft.com/v1.0/me/onenote/resources/%s/$value", sanitizedPageItemID)
	logging.PageLogger.Debug("Downloading item content", "page_id", pageID, "page_item_id", pageItemID, "url", resourceURL)

	// Make authenticated request to download the resource
	resp, err := c.MakeAuthenticatedRequest("GET", resourceURL, nil, nil)
	if err != nil {
		logging.PageLogger.Error("Failed to download page item content", "page_id", pageID, "page_item_id", pageItemID, "error", err)
		return nil, fmt.Errorf("failed to download page item: %v", err)
	}
	defer resp.Body.Close()

	// Handle HTTP response
	if errHandle := c.HandleHTTPResponse(resp, "GetPageItem"); errHandle != nil {
		logging.PageLogger.Error("HTTP response handling failed for GetPageItem", "page_id", pageID, "page_item_id", pageItemID, "status", resp.StatusCode, "error", errHandle)
		return nil, errHandle
	}

	// Read the binary content
	content, err := c.ReadResponseBody(resp, "GetPageItem")
	if err != nil {
		logging.PageLogger.Error("Failed to read page item content", "page_id", pageID, "page_item_id", pageItemID, "error", err)
		return nil, fmt.Errorf("failed to read page item content: %v", err)
	}

	// Create the page item data with content
	pageItemData := &PageItemData{
		Size:    int64(len(content)),
		Content: content,
	}

	// Add HTML metadata if found (original sophisticated approach)
	if htmlMetadata != nil {
		pageItemData.TagName = htmlMetadata.TagName
		pageItemData.Attributes = htmlMetadata.Attributes
		pageItemData.OriginalURL = htmlMetadata.OriginalURL

		logging.PageLogger.Debug("HTML metadata found", "tag_name", htmlMetadata.TagName, "attributes", htmlMetadata.Attributes, "data_src_type", htmlMetadata.Attributes["data-src-type"], "type", htmlMetadata.Attributes["type"])

		// Extract content type from HTML attributes - PRIMARY method
		var contentTypeFromHTML string
		// For object tags, check the type attribute first (HTML standard)
		if htmlMetadata.TagName == "object" {
			if contentType, exists := htmlMetadata.Attributes["type"]; exists && contentType != "" {
				contentTypeFromHTML = contentType
				logging.PageLogger.Debug("Content type from HTML type attribute (object tag)", "content_type", contentType)
			}
		}
		// If not found, check data-src-type (existing behavior)
		if contentTypeFromHTML == "" {
			if contentType, exists := htmlMetadata.Attributes["data-src-type"]; exists && contentType != "" {
				contentTypeFromHTML = contentType
				logging.PageLogger.Debug("Content type from HTML data-src-type attribute", "content_type", contentType)
			}
		}

		if contentTypeFromHTML != "" {
			pageItemData.ContentType = contentTypeFromHTML
		} else {
			// Fallback to HTTP content type if HTML attribute not found
			contentType := c.getContentTypeFromResponse(resp)
			pageItemData.ContentType = contentType
			logging.PageLogger.Debug("Content type from HTTP headers (fallback)", "content_type", contentType)
		}

		// Generate filename based on content type (enhanced version)
		pageItemData.Filename = c.generateFilenameFromContentType(pageItemID, pageItemData.ContentType)

		// Scale image if it's large and an image type (unless fullSize is requested)
		shouldScale := len(fullSize) == 0 || !fullSize[0] // Scale by default, skip if fullSize=true
		if shouldScale && strings.HasPrefix(pageItemData.ContentType, "image/") {
			logging.PageLogger.Debug("Attempting to scale image", "content_type", pageItemData.ContentType, "original_size", len(content))
			scaledContent, scaled, err := utils.ScaleImageIfNeeded(pageItemData.Content, pageItemData.ContentType, 1024, 768)
			if err != nil {
				logging.PageLogger.Debug("Failed to scale image", "error", err)
			} else if scaled {
				pageItemData.Content = scaledContent
				pageItemData.Size = int64(len(scaledContent))
				logging.PageLogger.Debug("Image was scaled down", "original_size", len(content), "scaled_size", len(scaledContent))
			} else {
				logging.PageLogger.Debug("Image size within limits, no scaling needed")
			}
		} else if len(fullSize) > 0 && fullSize[0] {
			logging.PageLogger.Debug("Skipping image scaling due to fullSize parameter")
		}

		logging.PageLogger.Debug("Added HTML metadata", "tag_name", pageItemData.TagName, "original_url", pageItemData.OriginalURL, "content_type", pageItemData.ContentType, "filename", pageItemData.Filename, "size_bytes", pageItemData.Size, "attributes", pageItemData.Attributes)
	} else {
		logging.PageLogger.Debug("No HTML metadata found for page item", "page_item_id", sanitizedPageItemID)
		// Fallback to HTTP content type and generated filename
		contentType := c.getContentTypeFromResponse(resp)
		pageItemData.ContentType = contentType
		pageItemData.Filename = c.generateFilenameFromContentType(pageItemID, contentType)
		logging.PageLogger.Debug("Using fallback content type and filename", "content_type", contentType, "filename", pageItemData.Filename)
	}

	// Determine if scaling was applied for logging
	scalingApplied := (len(fullSize) == 0 || !fullSize[0]) && strings.HasPrefix(pageItemData.ContentType, "image/")
	logging.PageLogger.Info("GetPageItem completed successfully", "page_id", pageID, "page_item_id", pageItemID, "filename", pageItemData.Filename, "content_type", pageItemData.ContentType, "size_bytes", pageItemData.Size, "tag_name", pageItemData.TagName, "scaling_applied", scalingApplied)
	return pageItemData, nil
}

// ListPageItems lists all embedded items (images, files) in a OneNote page.
// This enhanced version uses the original sophisticated HTML parsing approach.
// pageID: ID of the page to list items for.
// Returns a slice of page item metadata and an error if listing fails.
func (c *PageClient) ListPageItems(pageID string) ([]map[string]interface{}, error) {
	logging.PageLogger.Info("Starting ListPageItems operation", "page_id", pageID)

	// Check if token is expired and refresh if needed
	if c.TokenManager != nil && c.TokenManager.IsExpired() {
		logging.PageLogger.Debug("Token expired, refreshing before ListPageItems", "page_id", pageID)
		if err := c.RefreshTokenIfNeeded(); err != nil {
			logging.PageLogger.Error("Token refresh failed during ListPageItems", "page_id", pageID, "error", err)
			return nil, fmt.Errorf("token expired and refresh failed: %v", err)
		}
		logging.PageLogger.Debug("Token refreshed successfully for ListPageItems", "page_id", pageID)
	}

	// Validate and sanitize page ID
	sanitizedPageID, err := c.SanitizeOneNoteID(pageID, "pageID")
	if err != nil {
		logging.PageLogger.Error("Page ID sanitization failed", "page_id", pageID, "error", err)
		return nil, err
	}

	logging.PageLogger.Debug("Getting page content to extract page items", "page_id", sanitizedPageID)

	// Get the page content first (original approach - don't need IDs for listing)
	pageContent, err := c.GetPageContent(sanitizedPageID, false)
	if err != nil {
		logging.PageLogger.Error("Failed to get page content", "page_id", pageID, "error", err)
		return nil, fmt.Errorf("failed to get page content: %v", err)
	}

	logging.PageLogger.Debug("Retrieved page content", "content_length", len(pageContent))

	// Parse the HTML content using the original sophisticated approach
	pageItems, err := c.parseHTMLForPageItemsOriginal(pageContent)
	if err != nil {
		logging.PageLogger.Error("Failed to parse page items from HTML", "page_id", pageID, "error", err)
		return nil, fmt.Errorf("failed to parse page items: %v", err)
	}

	logging.PageLogger.Debug("Found page items in HTML content", "items_count", len(pageItems))

	// Convert PageItemInfo structs to simplified JSON array with requested fields
	var result []map[string]interface{}
	for _, item := range pageItems {
		itemMap := map[string]interface{}{
			"pageItemId": item.PageItemID,
			"tagName":    item.TagName,
		}

		// Determine type based on tagName and attributes
		var itemType string
		switch item.TagName {
		case "img":
			itemType = "image"
		case "object":
			if dataType, exists := item.Attributes["data-attachment"]; exists {
				if dataType == "true" {
					itemType = "attachment"
				} else {
					itemType = "object"
				}
			} else {
				itemType = "object"
			}
		default:
			itemType = item.TagName
		}
		itemMap["type"] = itemType

		// Add data-attachment if it exists in attributes
		if dataAttachment, exists := item.Attributes["data-attachment"]; exists {
			itemMap["data-attachment"] = dataAttachment
		}

		// Add MIME type from attributes (check multiple sources)
		var mimeType string
		// First check for data-src-type (existing behavior)
		if dataSrcType, exists := item.Attributes["data-src-type"]; exists && dataSrcType != "" {
			mimeType = dataSrcType
		}
		// For object tags, also check the type attribute (HTML standard)
		if mimeType == "" && item.TagName == "object" {
			if typeAttr, exists := item.Attributes["type"]; exists && typeAttr != "" {
				mimeType = typeAttr
			}
		}
		// Set mimeType if we found one
		if mimeType != "" {
			itemMap["mimeType"] = mimeType
		}

		result = append(result, itemMap)
		logging.PageLogger.Debug("Page item found", "tag_name", item.TagName, "page_item_id", item.PageItemID, "type", itemType, "data_attachment", item.Attributes["data-attachment"], "mime_type", mimeType)
	}

	logging.PageLogger.Info("ListPageItems completed successfully", "page_id", pageID, "items_count", len(result))
	return result, nil
}

// getExtensionFromContentType returns a file extension based on MIME type.
// contentType: The MIME type.
// Returns the appropriate file extension.
func getExtensionFromContentType(contentType string) string {
	switch {
	case strings.HasPrefix(contentType, "image/jpeg"):
		return ".jpg"
	case strings.HasPrefix(contentType, "image/png"):
		return ".png"
	case strings.HasPrefix(contentType, "image/gif"):
		return ".gif"
	case strings.HasPrefix(contentType, "image/bmp"):
		return ".bmp"
	case strings.HasPrefix(contentType, "image/webp"):
		return ".webp"
	case strings.HasPrefix(contentType, "image/svg"):
		return ".svg"
	case strings.HasPrefix(contentType, "application/pdf"):
		return ".pdf"
	case strings.HasPrefix(contentType, "text/plain"):
		return ".txt"
	case strings.HasPrefix(contentType, "text/html"):
		return ".html"
	case strings.HasPrefix(contentType, "application/json"):
		return ".json"
	case strings.HasPrefix(contentType, "application/xml"):
		return ".xml"
	case strings.HasPrefix(contentType, "application/zip"):
		return ".zip"
	case strings.HasPrefix(contentType, "application/vnd.openxmlformats-officedocument"):
		if strings.Contains(contentType, "wordprocessingml") {
			return ".docx"
		} else if strings.Contains(contentType, "spreadsheetml") {
			return ".xlsx"
		} else if strings.Contains(contentType, "presentationml") {
			return ".pptx"
		}
		return ".office"
	default:
		return ""
	}
}

// getContentTypeFromResponse extracts content type from HTTP response headers with fallback.
func (c *PageClient) getContentTypeFromResponse(resp *http.Response) string {
	contentType := "application/octet-stream" // Default content type

	if resp.Header.Get("Content-Type") != "" {
		contentType = resp.Header.Get("Content-Type")
		logging.PageLogger.Debug("Content type from HTTP headers", "content_type", contentType)
	} else {
		logging.PageLogger.Debug("No Content-Type header found, using default", "content_type", contentType)
	}

	return contentType
}

// generateFilenameFromContentType creates a filename based on page item ID and content type.
// This version provides more comprehensive MIME type mapping than the original.
func (c *PageClient) generateFilenameFromContentType(pageItemID, contentType string) string {
	var filename string

	// Use the existing utility function for better extension mapping
	ext := getExtensionFromContentType(contentType)
	if ext != "" {
		filename = pageItemID + ext
	} else {
		filename = pageItemID + ".bin"
	}

	logging.PageLogger.Debug("Generated filename from content type", "page_item_id", pageItemID, "content_type", contentType, "filename", filename)
	return filename
}

// parseHTMLForPageItemsOriginal parses HTML content using the original sophisticated approach.
// This replaces the current regex-based parsing with proper HTML parser traversal.
func (c *PageClient) parseHTMLForPageItemsOriginal(htmlContent string) ([]*PageItemInfo, error) {
	var pageItems []*PageItemInfo

	logging.PageLogger.Debug("Starting HTML parsing with proper HTML parser", "content_length", len(htmlContent))

	// Parse HTML using the standard library HTML parser
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		logging.PageLogger.Error("Failed to parse HTML", "error", err)
		return pageItems, fmt.Errorf("failed to parse HTML: %v", err)
	}

	// Recursively traverse the HTML tree to find img and object elements
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if n.Data == "img" || n.Data == "object" {
				// Extract all attributes
				attributes := make(map[string]string)
				for _, attr := range n.Attr {
					attributes[attr.Key] = attr.Val
					logging.PageLogger.Debug("Found HTML attribute", "element", n.Data, "key", attr.Key, "value", attr.Val)
				}

				// Determine the URL attribute based on element type
				var urlAttr string
				var originalURL string

				if n.Data == "img" {
					urlAttr = "src"
				} else if n.Data == "object" {
					urlAttr = "data"
				}

				if url, exists := attributes[urlAttr]; exists && url != "" {
					originalURL = url
					pageItemID := extractPageItemID(url)
					if pageItemID != "" {
						logging.PageLogger.Debug("Extracted page item from HTML", "element", n.Data, "page_item_id", pageItemID, "url", url)

						pageItems = append(pageItems, &PageItemInfo{
							TagName:     n.Data,
							PageItemID:  pageItemID,
							Attributes:  attributes,
							OriginalURL: originalURL,
						})
					}
				}
			}
		}

		// Recursively process child nodes
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	// Start traversal from the root
	traverse(doc)

	logging.PageLogger.Debug("HTML parsing completed", "items_found", len(pageItems))
	return pageItems, nil
}

// extractPageItemID extracts the page item ID from OneNote resource URLs
func extractPageItemID(url string) string {
	// Pattern for OneNote resource URLs supports both:
	// - Legacy: https://www.onenote.com/api/v1.0/me/notes/resources/{id}/$value
	// - Microsoft Graph: https://graph.microsoft.com/v1.0/users(...)/onenote/resources/{id}/$value
	// OneNote IDs can contain alphanumeric chars, hyphens, and exclamation marks like: 0-896fbac8f72d01b02c5950345e65f588!1-4D24C77F19546939!39705
	re := regexp.MustCompile(`/resources/([A-Za-z0-9\-!]+)/\$value`)
	matches := re.FindStringSubmatch(url)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}
