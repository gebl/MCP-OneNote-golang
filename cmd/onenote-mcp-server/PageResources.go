// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

// OneNote MCP Server Page Resources
//
// This file implements MCP (Model Context Protocol) page-related resources for accessing
// Microsoft OneNote pages through a hierarchical REST-like URI structure.
//
// ## Page Resource URIs
//
// ### Available Page Resource URIs
//
// #### 1. List Pages in a Section by ID or Name
// **URI:** `onenote://pages/{sectionIdOrName}`
// **Purpose:** Get all pages within a specific section, identified by either section ID or display name
// **Parameters:**
//   - `{sectionIdOrName}`: URL-encoded section ID or display name
// **API Equivalent:** `https://graph.microsoft.com/v1.0/me/onenote/sections/{sectionId}/pages`
// **Returns:** JSON object with pages containing title and pageId fields
//
// #### 2. Get Page Content for Update
// **URI:** `onenote://page/{pageId}`
// **Purpose:** Get HTML content for a specific page with data-id attributes included for update operations
// **Parameters:**
//   - `{pageId}`: URL-encoded page ID
// **API Equivalent:** `https://graph.microsoft.com/v1.0/me/onenote/pages/{pageId}/content?includeIDs=true`
// **Returns:** HTML content with data-id attributes for targeted updates
//

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/gebl/onenote-mcp-server/internal/config"
	"github.com/gebl/onenote-mcp-server/internal/graph"
	"github.com/gebl/onenote-mcp-server/internal/logging"
	"github.com/gebl/onenote-mcp-server/internal/pages"
	"github.com/gebl/onenote-mcp-server/internal/sections"
)

// registerPageResources registers all page-related MCP resources
func registerPageResources(s *server.MCPServer, graphClient *graph.Client, cfg *config.Config) {
	logging.MainLogger.Debug("Starting page resource registration process")

	// Register pages by section ID or name resource template
	logging.MainLogger.Debug("Creating pages by section resource template",
		"template_pattern", "onenote://pages/{sectionIdOrName}",
		"resource_type", "template_resource")
	pagesTemplate := mcp.NewResourceTemplate(
		"onenote://pages/{sectionIdOrName}",
		"OneNote Pages for Section",
		mcp.WithTemplateDescription("List all pages in a specific section by either section ID or display name, returning page titles and IDs"),
		mcp.WithTemplateMIMEType("application/json"),
	)

	s.AddResourceTemplate(pagesTemplate, func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		logging.MainLogger.Debug("Page resource template handler invoked",
			"template_pattern", "onenote://pages/{sectionIdOrName}",
			"request_uri", request.Params.URI,
			"handler_type", "section_pages")

		// Extract section ID or name from URI
		sectionIdOrName := extractSectionIdOrNameFromPagesURI(request.Params.URI)
		if sectionIdOrName == "" {
			logging.MainLogger.Error("Invalid section ID or name in pages URI",
				"request_uri", request.Params.URI,
				"extracted_value", sectionIdOrName)
			return nil, fmt.Errorf("invalid section ID or name in URI: %s", request.Params.URI)
		}

		logging.MainLogger.Debug("Extracted section ID or name from pages URI",
			"section_id_or_name", sectionIdOrName,
			"request_uri", request.Params.URI)

		// Call the pages resource handler with progress support
		jsonData, err := getPagesForSectionResource(ctx, s, graphClient, sectionIdOrName)
		if err != nil {
			logging.MainLogger.Error("Failed to get pages for section resource",
				"section_id_or_name", sectionIdOrName,
				"error", err)
			return nil, err
		}

		responseSize := len(jsonData)
		logging.MainLogger.Debug("Successfully prepared pages resource response",
			"section_id_or_name", sectionIdOrName,
			"request_uri", request.Params.URI,
			"response_size_bytes", responseSize)

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: "application/json",
				Text:     string(jsonData),
			},
		}, nil
	})
	logging.MainLogger.Debug("Registered pages by section resource template successfully",
		"template_pattern", "onenote://pages/{sectionIdOrName}")

	// Register page content by page ID resource template
	logging.MainLogger.Debug("Creating page content by page ID resource template",
		"template_pattern", "onenote://page/{pageId}",
		"resource_type", "template_resource")
	pageContentTemplate := mcp.NewResourceTemplate(
		"onenote://page/{pageId}",
		"OneNote Page Content for Update",
		mcp.WithTemplateDescription("Get HTML content for a specific page with data-id attributes included for update operations"),
		mcp.WithTemplateMIMEType("text/html"),
	)

	s.AddResourceTemplate(pageContentTemplate, func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		logging.MainLogger.Debug("Page content resource template handler invoked",
			"template_pattern", "onenote://page/{pageId}",
			"request_uri", request.Params.URI,
			"handler_type", "page_content")

		// Extract page ID from URI
		pageID := extractPageIdFromPageURI(request.Params.URI)
		if pageID == "" {
			logging.MainLogger.Error("Invalid page ID in page content URI",
				"request_uri", request.Params.URI,
				"extracted_value", pageID)
			return nil, fmt.Errorf("invalid page ID in URI: %s", request.Params.URI)
		}

		logging.MainLogger.Debug("Extracted page ID from page content URI",
			"page_id", pageID,
			"request_uri", request.Params.URI)

		// Call the page content resource handler with progress support
		htmlContent, err := getPageContentForResource(ctx, s, graphClient, pageID)
		if err != nil {
			logging.MainLogger.Error("Failed to get page content for resource",
				"page_id", pageID,
				"error", err)
			return nil, err
		}

		responseSize := len(htmlContent)
		logging.MainLogger.Debug("Successfully prepared page content resource response",
			"page_id", pageID,
			"request_uri", request.Params.URI,
			"response_size_bytes", responseSize)

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: "text/html",
				Text:     htmlContent,
			},
		}, nil
	})
	logging.MainLogger.Debug("Registered page content by page ID resource template successfully",
		"template_pattern", "onenote://page/{pageId}")

	logging.MainLogger.Debug("Page resource registration completed successfully",
		"total_resources", 2,
		"template_resources", 2,
		"static_resources", 0)
}

// Page-specific URI extraction functions

// extractSectionIdOrNameFromPagesURI extracts the section ID or name from a URI like "onenote://pages/{sectionIdOrName}"
func extractSectionIdOrNameFromPagesURI(uri string) string {
	// URI format: onenote://pages/{sectionIdOrName}
	parts := strings.Split(uri, "/")
	if len(parts) >= 4 && parts[0] == "onenote:" && parts[2] == "pages" {
		// URL decode the section ID or name since it's part of a URI
		decodedValue, err := url.QueryUnescape(parts[3])
		if err != nil {
			logging.MainLogger.Warn("Failed to URL decode section ID or name from pages URI, using raw value",
				"raw_value", parts[3],
				"error", err,
				"uri", uri)
			return parts[3] // fallback to raw value
		}
		logging.MainLogger.Debug("URL decoded section ID or name from pages URI",
			"raw_value", parts[3],
			"decoded_value", decodedValue,
			"uri", uri)
		return decodedValue
	}
	return ""
}

// extractPageIdFromPageURI extracts the page ID from a URI like "onenote://page/{pageId}"
func extractPageIdFromPageURI(uri string) string {
	// URI format: onenote://page/{pageId}
	parts := strings.Split(uri, "/")
	if len(parts) >= 4 && parts[0] == "onenote:" && parts[2] == "page" {
		// URL decode the page ID since it's part of a URI
		decodedValue, err := url.QueryUnescape(parts[3])
		if err != nil {
			logging.MainLogger.Warn("Failed to URL decode page ID from page URI, using raw value",
				"raw_value", parts[3],
				"error", err,
				"uri", uri)
			return parts[3] // fallback to raw value
		}
		logging.MainLogger.Debug("URL decoded page ID from page URI",
			"raw_value", parts[3],
			"decoded_value", decodedValue,
			"uri", uri)
		return decodedValue
	}
	return ""
}

// getPageContentForResource fetches HTML content for a specific page with data-id attributes for updates
func getPageContentForResource(ctx context.Context, s *server.MCPServer, graphClient *graph.Client, pageID string) (string, error) {
	logging.MainLogger.Debug("getPageContentForResource called", "page_id", pageID)

	// Extract progress token from request metadata (MCP spec for resources)
	var progressToken string
	logging.MainLogger.Debug("Resource progress support - checking for progress token",
		"page_id", pageID)

	// Send initial progress notification
	if s != nil {
		sendProgressNotification(s, ctx, progressToken, 0, 100, "Starting to fetch page content...")
	}

	// Create specialized page client
	pageClient := pages.NewPageClient(graphClient)

	// Create progress context
	progressCtx := context.WithValue(ctx, mcpServerKey, s)
	progressCtx = context.WithValue(progressCtx, progressTokenKey, progressToken)

	// Send progress for API call
	if s != nil {
		sendProgressNotification(s, progressCtx, progressToken, 30, 100, "Fetching page content with update IDs...")
	}

	// Get page content with forUpdate=true to include data-id attributes
	htmlContent, err := pageClient.GetPageContent(pageID, true)
	if err != nil {
		logging.MainLogger.Error("Failed to fetch page content for resource",
			"page_id", pageID,
			"error", err)
		return "", fmt.Errorf("failed to fetch page content: %v", err)
	}

	// Send progress for completion
	if s != nil {
		sendProgressNotification(s, progressCtx, progressToken, 100, 100, "Completed fetching page content")
	}

	logging.MainLogger.Debug("Successfully prepared page content for resource",
		"page_id", pageID,
		"content_length", len(htmlContent))

	return htmlContent, nil
}

// getPagesForSectionResource fetches all pages for a section identified by ID or name
func getPagesForSectionResource(ctx context.Context, s *server.MCPServer, graphClient *graph.Client, sectionIdOrName string) ([]byte, error) {
	logging.MainLogger.Debug("getPagesForSectionResource called", "section_id_or_name", sectionIdOrName)

	// Extract progress token from request metadata (MCP spec for resources)
	var progressToken string
	logging.MainLogger.Debug("Resource progress support - checking for progress token",
		"section_id_or_name", sectionIdOrName)

	// Send initial progress notification
	if s != nil {
		sendProgressNotification(s, ctx, progressToken, 0, 100, "Starting to fetch pages for section...")
	}

	// Create specialized clients
	pageClient := pages.NewPageClient(graphClient)
	sectionClient := sections.NewSectionClient(graphClient)

	// Create progress context
	progressCtx := context.WithValue(ctx, mcpServerKey, s)
	progressCtx = context.WithValue(progressCtx, progressTokenKey, progressToken)

	// First, try to determine if this is an ID or name and resolve to section ID
	sectionID, err := resolveSectionIdOrName(progressCtx, sectionClient, sectionIdOrName)
	if err != nil {
		logging.MainLogger.Error("Failed to resolve section ID or name",
			"section_id_or_name", sectionIdOrName,
			"error", err)
		return nil, fmt.Errorf("failed to resolve section '%s': %v", sectionIdOrName, err)
	}

	logging.MainLogger.Debug("Resolved section ID",
		"section_id_or_name", sectionIdOrName,
		"resolved_section_id", sectionID)

	// Send progress for API call
	if s != nil {
		sendProgressNotification(s, ctx, progressToken, 30, 100, "Fetching pages from section...")
	}

	// Get pages for the resolved section ID
	pagesData, err := pageClient.ListPages(sectionID)
	if err != nil {
		logging.MainLogger.Error("Failed to fetch pages for section resource",
			"section_id", sectionID,
			"section_id_or_name", sectionIdOrName,
			"error", err)
		return nil, fmt.Errorf("failed to fetch pages for section: %v", err)
	}

	// Send progress for processing
	if s != nil {
		sendProgressNotification(s, ctx, progressToken, 70, 100, "Processing pages data...")
	}

	// Filter the pages data to include only title and pageId
	var filteredPages []map[string]interface{}
	for _, page := range pagesData {
		filteredPage := make(map[string]interface{})

		// Extract title from the page data
		if title, exists := page["title"]; exists {
			filteredPage["title"] = title
		}

		// Extract pageId from the page data (could be "pageId" or "id")
		if pageId, exists := page["pageId"]; exists {
			filteredPage["pageId"] = pageId
		} else if id, exists := page["id"]; exists {
			filteredPage["pageId"] = id
		}

		filteredPages = append(filteredPages, filteredPage)
	}

	// Build response in the requested format
	response := map[string]interface{}{
		"pages":        filteredPages,
		"pages_count":  len(filteredPages),
		"section_id":   sectionID,
		"section_name": sectionIdOrName, // Original input (could be name or ID)
		"source":       "pages_resource_api",
	}

	// Send progress for marshaling
	if s != nil {
		sendProgressNotification(s, ctx, progressToken, 90, 100, "Preparing response...")
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		logging.MainLogger.Error("Failed to marshal pages for resource",
			"error", err,
			"section_id_or_name", sectionIdOrName)
		return nil, fmt.Errorf("failed to marshal pages: %v", err)
	}

	// Send final progress notification
	if s != nil {
		sendProgressNotification(s, ctx, progressToken, 100, 100, "Completed fetching pages for section")
	}

	logging.MainLogger.Debug("Successfully prepared pages for section resource",
		"section_id_or_name", sectionIdOrName,
		"section_id", sectionID,
		"pages_count", len(filteredPages))

	return jsonData, nil
}

// resolveSectionIdOrName resolves a section identifier (ID or name) to a section ID
// If the input is already a valid section ID, returns it as-is
// If the input is a section name, searches for the section and returns its ID
func resolveSectionIdOrName(ctx context.Context, sectionClient *sections.SectionClient, sectionIdOrName string) (string, error) {
	logging.MainLogger.Debug("Resolving section ID or name", "input", sectionIdOrName)

	// First, try to use it as an ID directly - if it's a valid OneNote ID format, try it
	if isValidOneNoteIDFormat(sectionIdOrName) {
		logging.MainLogger.Debug("Input appears to be a valid OneNote ID format, testing directly",
			"section_id", sectionIdOrName)

		// We can't easily test if a section ID is valid without making an API call
		// So we'll first try the global sections API to see if we can find it
		allSections, err := sectionClient.ListAllSections()
		if err == nil {
			for _, section := range allSections {
				if id, exists := section["id"].(string); exists && id == sectionIdOrName {
					logging.MainLogger.Debug("Found section by ID in global sections list",
						"section_id", sectionIdOrName)
					return sectionIdOrName, nil
				}
			}
		}

		logging.MainLogger.Debug("Section ID not found in global sections, treating as name",
			"input", sectionIdOrName)
	}

	// Try to find section by display name
	logging.MainLogger.Debug("Searching for section by name", "section_name", sectionIdOrName)

	// Get all sections and search by name
	allSections, err := sectionClient.ListAllSections()
	if err != nil {
		logging.MainLogger.Error("Failed to list all sections for name resolution",
			"section_name", sectionIdOrName,
			"error", err)
		return "", fmt.Errorf("failed to list sections for name resolution: %v", err)
	}

	// Search for section with matching display name
	for _, section := range allSections {
		if displayName, exists := section["displayName"].(string); exists && displayName == sectionIdOrName {
			if id, exists := section["id"].(string); exists {
				logging.MainLogger.Debug("Found section by name",
					"section_name", sectionIdOrName,
					"section_id", id)
				return id, nil
			}
		}
	}

	logging.MainLogger.Error("Section not found by ID or name",
		"section_id_or_name", sectionIdOrName)
	return "", fmt.Errorf("section not found: '%s' (searched by both ID and name)", sectionIdOrName)
}

// isValidOneNoteIDFormat checks if a string looks like a valid OneNote ID
// OneNote IDs typically contain alphanumeric characters, hyphens, and exclamation marks
func isValidOneNoteIDFormat(id string) bool {
	if len(id) < 10 { // OneNote IDs are typically longer
		return false
	}

	// OneNote IDs contain these characters: A-Za-z0-9-!
	for _, char := range id {
		if !((char >= 'A' && char <= 'Z') ||
			(char >= 'a' && char <= 'z') ||
			(char >= '0' && char <= '9') ||
			char == '-' || char == '!') {
			return false
		}
	}

	return true
}
