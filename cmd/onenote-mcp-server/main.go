// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

// main.go - Entry point for the OneNote MCP Server.
//
// This file sets up the Model Context Protocol (MCP) server that provides seamless
// integration with Microsoft OneNote via the Microsoft Graph API. The server enables
// AI assistants and other MCP clients to read, create, update, and manage OneNote
// notebooks, sections, pages, and embedded content.
//
// Key Features:
// - OAuth 2.0 PKCE authentication with automatic token refresh
// - Complete OneNote CRUD operations (Create, Read, Update, Delete)
// - Rich content handling with HTML support and embedded media
// - Image optimization and metadata extraction
// - Search capabilities using OData filters
// - Comprehensive error handling and logging
//
// Available MCP Tools:
// - listNotebooks: List all OneNote notebooks for the user
// - listAllSections: Get all sections across all notebooks (useful for searching)
// - listSections: List sections within a specific notebook or section group (requires containerId)
// - listPages: List pages within a section
// - createSection: Create new sections in a notebook
// - createPage: Create new pages with HTML content
// - updatePageContent: Update existing page content
// - deletePage: Delete pages by ID
// - copyPage: Copy pages between sections using Microsoft Graph API
// - movePage: Move pages between sections (copy then delete)
// - getPageContent: Retrieve page HTML content
// - listPageItems: List embedded items (images, files) in a page
// - getPageItemContent: Get complete page item data with binary content
//
// Authentication Flow:
// 1. Server loads configuration from environment variables or config file
// 2. OAuth 2.0 PKCE flow handles user authentication
// 3. Access and refresh tokens are stored locally
// 4. Automatic token refresh prevents authentication failures
//
// Configuration:
// - Environment variables: ONENOTE_CLIENT_ID, ONENOTE_TENANT_ID, ONENOTE_REDIRECT_URI
// - Optional config file: Set ONENOTE_MCP_CONFIG environment variable
// - Logging: Set MCP_LOG_FILE for file-based logging
//
// Usage:
//   go build -o onenote-mcp-server ./cmd/onenote-mcp-server
//   ./onenote-mcp-server                    # stdio mode (default)
//   ./onenote-mcp-server -mode=streamable  # Streamable HTTP mode on port 8080
//   ./onenote-mcp-server -mode=streamable -port=8081 # Streamable HTTP mode on custom port
//
// Docker:
//   docker build -t onenote-mcp-server .
//   docker run -p 8080:8080 onenote-mcp-server
//
// For detailed setup instructions, see README.md and docs/setup.md

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/server"

	"github.com/gebl/onenote-mcp-server/internal/auth"
	"github.com/gebl/onenote-mcp-server/internal/config"
	"github.com/gebl/onenote-mcp-server/internal/graph"
	"github.com/gebl/onenote-mcp-server/internal/logging"
	"github.com/gebl/onenote-mcp-server/internal/notebooks"
	"github.com/gebl/onenote-mcp-server/internal/sections"
)

const (
	// Version is the current version of the OneNote MCP server
	Version        = "1.7.0"
	emptyJSONArray = "[]"
)

// PageSearchResult holds the result of a page search operation
type PageSearchResult struct {
	Page      map[string]interface{} // The found page data
	SectionID string                 // Section ID where the page was found
	Found     bool                   // Whether the page was found
}

// NotebookCache holds the currently selected notebook information and sections cache
type NotebookCache struct {
	mu                  sync.RWMutex
	notebook            map[string]interface{}
	notebookID          string
	displayName         string
	isSet               bool
	sectionsTree        map[string]interface{}              // Cached sections tree structure
	sectionsCached      bool                                // Whether sections have been cached
	pagesCache          map[string][]map[string]interface{} // Pages cache by section ID
	pagesCacheTime      map[string]time.Time                // Cache timestamps by section ID
	pageSearchCache     map[string]PageSearchResult         // Page search results cache by notebook:page key
	pageSearchTime      map[string]time.Time                // Page search cache timestamps
	notebookLookupCache map[string]map[string]interface{}   // Notebook lookup results cache by notebook name
	notebookLookupTime  map[string]time.Time                // Notebook lookup cache timestamps
	
	// Optional references for API fallback in authorization
	graphClient         interface{}                         // Graph client for API calls
	mcpServer           interface{}                         // MCP server for progress notifications
}

// NewNotebookCache creates a new notebook cache
func NewNotebookCache() *NotebookCache {
	return &NotebookCache{
		notebook:            make(map[string]interface{}),
		sectionsTree:        make(map[string]interface{}),
		pagesCache:          make(map[string][]map[string]interface{}),
		pagesCacheTime:      make(map[string]time.Time),
		pageSearchCache:     make(map[string]PageSearchResult),
		pageSearchTime:      make(map[string]time.Time),
		notebookLookupCache: make(map[string]map[string]interface{}),
		notebookLookupTime:  make(map[string]time.Time),
	}
}

// SetAPIReferences sets the graph client and MCP server references for API fallback
func (nc *NotebookCache) SetAPIReferences(graphClient interface{}, mcpServer interface{}) {
	nc.mu.Lock()
	defer nc.mu.Unlock()
	
	nc.graphClient = graphClient
	nc.mcpServer = mcpServer
	
	logging.ToolsLogger.Debug("API references set in notebook cache for authorization fallback",
		"has_graph_client", graphClient != nil,
		"has_mcp_server", mcpServer != nil)
}

// GetAPIReferences returns the stored API references
func (nc *NotebookCache) GetAPIReferences() (interface{}, interface{}) {
	nc.mu.RLock()
	defer nc.mu.RUnlock()
	
	return nc.graphClient, nc.mcpServer
}

// SetNotebook sets the selected notebook in cache and clears sections cache
func (nc *NotebookCache) SetNotebook(notebook map[string]interface{}) {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	nc.notebook = notebook
	nc.isSet = true

	// Clear sections cache when notebook changes
	nc.sectionsTree = make(map[string]interface{})
	nc.sectionsCached = false

	// Clear pages cache when notebook changes
	nc.pagesCache = make(map[string][]map[string]interface{})
	nc.pagesCacheTime = make(map[string]time.Time)

	// Clear page search cache when notebook changes
	nc.pageSearchCache = make(map[string]PageSearchResult)
	nc.pageSearchTime = make(map[string]time.Time)

	// Clear notebook lookup cache when notebook changes
	nc.notebookLookupCache = make(map[string]map[string]interface{})
	nc.notebookLookupTime = make(map[string]time.Time)

	// Extract ID and display name for easy access
	if id, ok := notebook["id"].(string); ok {
		nc.notebookID = id
	}
	if name, ok := notebook["displayName"].(string); ok {
		nc.displayName = name
	}
}

// GetNotebook returns the currently selected notebook
func (nc *NotebookCache) GetNotebook() (map[string]interface{}, bool) {
	nc.mu.RLock()
	defer nc.mu.RUnlock()

	if !nc.isSet {
		return nil, false
	}

	// Return a copy to prevent race conditions
	notebookCopy := make(map[string]interface{})
	for k, v := range nc.notebook {
		notebookCopy[k] = v
	}

	return notebookCopy, true
}

// GetNotebookID returns the currently selected notebook ID
func (nc *NotebookCache) GetNotebookID() (string, bool) {
	nc.mu.RLock()
	defer nc.mu.RUnlock()

	return nc.notebookID, nc.isSet
}

// GetDisplayName returns the currently selected notebook display name
func (nc *NotebookCache) GetDisplayName() (string, bool) {
	nc.mu.RLock()
	defer nc.mu.RUnlock()

	return nc.displayName, nc.isSet
}

// GetSectionName returns the section name for a given section ID by searching through the cached sections tree
func (nc *NotebookCache) GetSectionName(sectionID string) (string, bool) {
	nc.mu.RLock()
	defer nc.mu.RUnlock()

	if !nc.sectionsCached {
		return "", false
	}

	// Search through the sections tree to find the section with the given ID
	return nc.findSectionNameInTree(nc.sectionsTree, sectionID)
}

// GetSectionNameWithAutoFetch returns the section name for a given section ID, 
// optionally fetching sections if they're not cached (for authorization context)
func (nc *NotebookCache) GetSectionNameWithAutoFetch(sectionID string, graphClient interface{}, enableAutoFetch bool) (string, bool) {
	// First try the fast cache lookup
	if name, found := nc.GetSectionName(sectionID); found {
		return name, true
	}

	// If not found and auto-fetch is disabled, return false
	if !enableAutoFetch {
		return "", false
	}

	// If sections aren't cached and we have a graph client, try to fetch them
	// This is primarily for authorization context when sections haven't been loaded yet
	if !nc.IsSectionsCached() && graphClient != nil {
		// This is a simplified auto-fetch - in a real implementation we would:
		// 1. Create a notebook client from the graph client
		// 2. Fetch sections with progress notifications
		// 3. Cache the results
		// 4. Retry the lookup
		// For now, we'll just return false to avoid complexity
		return "", false
	}

	return "", false
}

// GetSectionNameWithProgress returns the section name for a given section ID with progress notification support.
// If not found in cache, it will perform a live API lookup with progress notifications.
func (nc *NotebookCache) GetSectionNameWithProgress(ctx context.Context, sectionID string, mcpServer interface{}, progressToken string, graphClient interface{}) (string, bool) {
	
	logging.ToolsLogger.Debug("Looking up section name with progress support",
		"section_id", sectionID,
		"has_progress_token", progressToken != "",
		"progress_token", progressToken)

	// First try the fast cache lookup
	if name, found := nc.GetSectionName(sectionID); found {
		logging.ToolsLogger.Debug("Section name found in cache",
			"section_id", sectionID,
			"section_name", name)
		return name, true
	}

	logging.ToolsLogger.Debug("Section name not found in cache, attempting live API lookup",
		"section_id", sectionID,
		"sections_cached", nc.IsSectionsCached())

	// Cast the graph client and MCP server
	graphClientTyped, ok := graphClient.(*graph.Client)
	if !ok {
		logging.ToolsLogger.Warn("Invalid graph client type for section lookup",
			"section_id", sectionID,
			"client_type", fmt.Sprintf("%T", graphClient))
		return "", false
	}

	var mcpServerTyped *server.MCPServer
	if mcpServer != nil {
		mcpServerTyped, _ = mcpServer.(*server.MCPServer)
	}

	// Send progress notification for API lookup
	if mcpServerTyped != nil && progressToken != "" {
		err := mcpServerTyped.SendNotificationToClient(ctx, "notifications/progress", map[string]any{
			"progressToken": progressToken,
			"progress":      5,
			"total":         100,
			"message":       fmt.Sprintf("Looking up section name for %s via API...", sectionID),
		})
		if err != nil {
			logging.ToolsLogger.Warn("Failed to send progress notification for section lookup",
				"error", err,
				"section_id", sectionID)
		}
	}

	// Create section client for direct API call
	sectionClient := sections.NewSectionClient(graphClientTyped)

	// Try to fetch section details directly by ID
	logging.ToolsLogger.Debug("Attempting direct section lookup by ID",
		"section_id", sectionID)

	// Send progress notification for API call
	if mcpServerTyped != nil && progressToken != "" {
		err := mcpServerTyped.SendNotificationToClient(ctx, "notifications/progress", map[string]any{
			"progressToken": progressToken,
			"progress":      10,
			"total":         100,
			"message":       "Fetching section details from OneNote API...",
		})
		if err != nil {
			logging.ToolsLogger.Warn("Failed to send API progress notification",
				"error", err,
				"section_id", sectionID)
		}
	}

	// Try to get section details using the section ID directly
	// This uses the Microsoft Graph API endpoint: /me/onenote/sections/{section-id}
	sectionDetails, err := sectionClient.GetSectionByID(sectionID)
	if err != nil {
		logging.ToolsLogger.Warn("Failed to fetch section details by ID",
			"section_id", sectionID,
			"error", err)

		// Send failure progress notification
		if mcpServerTyped != nil && progressToken != "" {
			err := mcpServerTyped.SendNotificationToClient(ctx, "notifications/progress", map[string]any{
				"progressToken": progressToken,
				"progress":      15,
				"total":         100,
				"message":       "Section lookup failed, section name unavailable",
			})
			if err != nil {
				logging.ToolsLogger.Warn("Failed to send failure progress notification", "error", err)
			}
		}

		return "", false
	}

	// Extract section name from the response
	sectionName := ""
	if displayName, ok := sectionDetails["displayName"].(string); ok {
		sectionName = displayName
	}

	if sectionName == "" {
		logging.ToolsLogger.Warn("Section details retrieved but displayName is missing",
			"section_id", sectionID,
			"section_details", sectionDetails)

		// Send failure progress notification
		if mcpServerTyped != nil && progressToken != "" {
			err := mcpServerTyped.SendNotificationToClient(ctx, "notifications/progress", map[string]any{
				"progressToken": progressToken,
				"progress":      20,
				"total":         100,
				"message":       "Section found but name is missing",
			})
			if err != nil {
				logging.ToolsLogger.Warn("Failed to send missing name progress notification", "error", err)
			}
		}

		return "", false
	}

	// Send success progress notification
	if mcpServerTyped != nil && progressToken != "" {
		err := mcpServerTyped.SendNotificationToClient(ctx, "notifications/progress", map[string]any{
			"progressToken": progressToken,
			"progress":      25,
			"total":         100,
			"message":       fmt.Sprintf("Section name resolved: %s", sectionName),
		})
		if err != nil {
			logging.ToolsLogger.Warn("Failed to send success progress notification", "error", err)
		}
	}

	logging.ToolsLogger.Debug("Successfully resolved section name via API",
		"section_id", sectionID,
		"section_name", sectionName)

	return sectionName, true
}

// findSectionNameInTree recursively searches through a sections tree structure to find a section by ID
func (nc *NotebookCache) findSectionNameInTree(tree map[string]interface{}, targetSectionID string) (string, bool) {
	// Check if this tree node has sections
	if sectionsInterface, exists := tree["sections"]; exists {
		if sections, ok := sectionsInterface.([]interface{}); ok {
			for _, sectionInterface := range sections {
				if section, ok := sectionInterface.(map[string]interface{}); ok {
					// Check if this section matches the target ID
					if sectionID, idExists := section["id"].(string); idExists && sectionID == targetSectionID {
						if displayName, nameExists := section["displayName"].(string); nameExists {
							return displayName, true
						}
					}
				}
			}
		}
	}

	// Check if this tree node has section groups
	if sectionGroupsInterface, exists := tree["sectionGroups"]; exists {
		if sectionGroups, ok := sectionGroupsInterface.([]interface{}); ok {
			for _, sectionGroupInterface := range sectionGroups {
				if sectionGroup, ok := sectionGroupInterface.(map[string]interface{}); ok {
					// Recursively search in section groups
					if name, found := nc.findSectionNameInTree(sectionGroup, targetSectionID); found {
						return name, true
					}
				}
			}
		}
	}

	return "", false
}

// IsSet returns whether a notebook is currently selected
func (nc *NotebookCache) IsSet() bool {
	nc.mu.RLock()
	defer nc.mu.RUnlock()

	return nc.isSet
}

// SetSectionsTree sets the cached sections tree structure
func (nc *NotebookCache) SetSectionsTree(sectionsTree map[string]interface{}) {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	nc.sectionsTree = sectionsTree
	nc.sectionsCached = true
}

// GetSectionsTree returns the cached sections tree structure
func (nc *NotebookCache) GetSectionsTree() (map[string]interface{}, bool) {
	nc.mu.RLock()
	defer nc.mu.RUnlock()

	if !nc.sectionsCached {
		return nil, false
	}

	// Return a copy to prevent race conditions
	treeCopy := make(map[string]interface{})
	for k, v := range nc.sectionsTree {
		treeCopy[k] = v
	}

	return treeCopy, true
}

// IsSectionsCached returns whether sections have been cached
func (nc *NotebookCache) IsSectionsCached() bool {
	nc.mu.RLock()
	defer nc.mu.RUnlock()

	return nc.sectionsCached
}

// ClearSectionsCache clears the sections cache
func (nc *NotebookCache) ClearSectionsCache() {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	nc.sectionsTree = make(map[string]interface{})
	nc.sectionsCached = false
}

// SetPagesCache sets the cached pages for a specific section
func (nc *NotebookCache) SetPagesCache(sectionID string, pages []map[string]interface{}) {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	nc.pagesCache[sectionID] = pages
	nc.pagesCacheTime[sectionID] = time.Now()
}

// GetPagesCache returns the cached pages for a specific section
func (nc *NotebookCache) GetPagesCache(sectionID string) ([]map[string]interface{}, bool) {
	nc.mu.RLock()
	defer nc.mu.RUnlock()

	pages, exists := nc.pagesCache[sectionID]
	if !exists {
		return nil, false
	}

	// Check if cache is still fresh (within 5 minutes)
	cacheTime, timeExists := nc.pagesCacheTime[sectionID]
	if !timeExists || time.Since(cacheTime) > 5*time.Minute {
		return nil, false
	}

	// Return a copy to prevent race conditions
	result := make([]map[string]interface{}, len(pages))
	for i, page := range pages {
		pageCopy := make(map[string]interface{})
		for k, v := range page {
			pageCopy[k] = v
		}
		result[i] = pageCopy
	}

	return result, true
}

// IsPagesCached returns whether pages are cached for a specific section
func (nc *NotebookCache) IsPagesCached(sectionID string) bool {
	nc.mu.RLock()
	defer nc.mu.RUnlock()

	_, exists := nc.pagesCache[sectionID]
	if !exists {
		return false
	}

	// Check if cache is still fresh
	cacheTime, timeExists := nc.pagesCacheTime[sectionID]
	if !timeExists || time.Since(cacheTime) > 5*time.Minute {
		return false
	}

	return true
}

// ClearPagesCache clears the pages cache for a specific section
func (nc *NotebookCache) ClearPagesCache(sectionID string) {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	delete(nc.pagesCache, sectionID)
	delete(nc.pagesCacheTime, sectionID)
}

// ClearAllPagesCache clears all pages cache
func (nc *NotebookCache) ClearAllPagesCache() {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	nc.pagesCache = make(map[string][]map[string]interface{})
	nc.pagesCacheTime = make(map[string]time.Time)
}

// ClearAllCache clears all cached data (notebook, sections, and pages)
func (nc *NotebookCache) ClearAllCache() {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	// Clear notebook cache (but don't unset the selection)
	// Only clear cached data, keep the notebook selection intact

	// Clear sections cache
	nc.sectionsTree = make(map[string]interface{})
	nc.sectionsCached = false

	// Clear all pages cache
	nc.pagesCache = make(map[string][]map[string]interface{})
	nc.pagesCacheTime = make(map[string]time.Time)

	// Clear page search cache
	nc.pageSearchCache = make(map[string]PageSearchResult)
	nc.pageSearchTime = make(map[string]time.Time)

	// Clear notebook lookup cache
	nc.notebookLookupCache = make(map[string]map[string]interface{})
	nc.notebookLookupTime = make(map[string]time.Time)
}

// GetPageSearchCacheKey creates a cache key for page search results
func (nc *NotebookCache) GetPageSearchCacheKey(notebookID, pageName string) string {
	return fmt.Sprintf("%s:%s", notebookID, pageName)
}

// SetPageSearchCache sets the cached page search result
func (nc *NotebookCache) SetPageSearchCache(notebookID, pageName string, result PageSearchResult) {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	key := nc.GetPageSearchCacheKey(notebookID, pageName)
	nc.pageSearchCache[key] = result
	nc.pageSearchTime[key] = time.Now()
}

// GetPageSearchCache returns the cached page search result
func (nc *NotebookCache) GetPageSearchCache(notebookID, pageName string) (PageSearchResult, bool) {
	nc.mu.RLock()
	defer nc.mu.RUnlock()

	key := nc.GetPageSearchCacheKey(notebookID, pageName)
	result, exists := nc.pageSearchCache[key]
	if !exists {
		return PageSearchResult{}, false
	}

	// Check if cache is still fresh (within 5 minutes)
	cacheTime, timeExists := nc.pageSearchTime[key]
	if !timeExists || time.Since(cacheTime) > 5*time.Minute {
		return PageSearchResult{}, false
	}

	return result, true
}

// IsPageSearchCached returns whether page search results are cached
func (nc *NotebookCache) IsPageSearchCached(notebookID, pageName string) bool {
	nc.mu.RLock()
	defer nc.mu.RUnlock()

	key := nc.GetPageSearchCacheKey(notebookID, pageName)
	_, exists := nc.pageSearchCache[key]
	if !exists {
		return false
	}

	// Check if cache is still fresh
	cacheTime, timeExists := nc.pageSearchTime[key]
	if !timeExists || time.Since(cacheTime) > 5*time.Minute {
		return false
	}

	return true
}

// ClearPageSearchCache clears the page search cache for a specific notebook/page combination
func (nc *NotebookCache) ClearPageSearchCache(notebookID, pageName string) {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	key := nc.GetPageSearchCacheKey(notebookID, pageName)
	delete(nc.pageSearchCache, key)
	delete(nc.pageSearchTime, key)
}

// ClearAllPageSearchCache clears all page search cache
func (nc *NotebookCache) ClearAllPageSearchCache() {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	nc.pageSearchCache = make(map[string]PageSearchResult)
	nc.pageSearchTime = make(map[string]time.Time)
}

// SetNotebookLookupCache sets the cached notebook lookup result
func (nc *NotebookCache) SetNotebookLookupCache(notebookName string, notebook map[string]interface{}) {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	logging.NotebookLogger.Debug("Storing notebook in cache", 
		"notebook_name", notebookName, 
		"notebook_id", notebook["id"], 
		"notebook_data", notebook)

	nc.notebookLookupCache[notebookName] = notebook
	nc.notebookLookupTime[notebookName] = time.Now()
}

// GetNotebookLookupCache returns the cached notebook lookup result
func (nc *NotebookCache) GetNotebookLookupCache(notebookName string) (map[string]interface{}, bool) {
	nc.mu.RLock()
	defer nc.mu.RUnlock()

	notebook, exists := nc.notebookLookupCache[notebookName]
	if !exists {
		logging.NotebookLogger.Debug("Notebook lookup cache miss - not found in cache", "notebook_name", notebookName)
		return nil, false
	}

	// Check if cache is still fresh (within 5 minutes)
	cacheTime, timeExists := nc.notebookLookupTime[notebookName]
	if !timeExists {
		logging.NotebookLogger.Debug("Notebook lookup cache miss - no timestamp", "notebook_name", notebookName)
		return nil, false
	}
	
	age := time.Since(cacheTime)
	if age > 5*time.Minute {
		logging.NotebookLogger.Debug("Notebook lookup cache expired", "notebook_name", notebookName, "age", age)
		return nil, false
	}

	logging.NotebookLogger.Debug("Notebook lookup cache hit", "notebook_name", notebookName, "age", age, "notebook_id", notebook["id"])

	// Return a copy to prevent race conditions
	notebookCopy := make(map[string]interface{})
	for k, v := range notebook {
		notebookCopy[k] = v
	}

	return notebookCopy, true
}

// IsNotebookLookupCached returns whether notebook lookup results are cached
func (nc *NotebookCache) IsNotebookLookupCached(notebookName string) bool {
	nc.mu.RLock()
	defer nc.mu.RUnlock()

	_, exists := nc.notebookLookupCache[notebookName]
	if !exists {
		return false
	}

	// Check if cache is still fresh
	cacheTime, timeExists := nc.notebookLookupTime[notebookName]
	if !timeExists || time.Since(cacheTime) > 5*time.Minute {
		return false
	}

	return true
}

// ClearNotebookLookupCache clears the notebook lookup cache for a specific notebook name
func (nc *NotebookCache) ClearNotebookLookupCache(notebookName string) {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	delete(nc.notebookLookupCache, notebookName)
	delete(nc.notebookLookupTime, notebookName)
}

// ClearAllNotebookLookupCache clears all notebook lookup cache
func (nc *NotebookCache) ClearAllNotebookLookupCache() {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	nc.notebookLookupCache = make(map[string]map[string]interface{})
	nc.notebookLookupTime = make(map[string]time.Time)
}

// GetPageName returns the page name for a given page ID by searching through all cached pages
func (nc *NotebookCache) GetPageName(pageID string) (string, bool) {
	nc.mu.RLock()
	defer nc.mu.RUnlock()

	// Search through all cached pages in all sections
	for sectionID, pages := range nc.pagesCache {
		// Check if this section's cache is still fresh
		if cacheTime, exists := nc.pagesCacheTime[sectionID]; !exists || time.Since(cacheTime) > 5*time.Minute {
			continue // Skip expired cache entries
		}

		// Search through pages in this section
		for _, page := range pages {
			// Check both possible field names for page ID
			var currentPageID string
			if id, ok := page["id"].(string); ok {
				currentPageID = id
			} else if id, ok := page["pageId"].(string); ok {
				currentPageID = id
			}

			if currentPageID == pageID {
				if title, ok := page["title"].(string); ok {
					logging.AuthorizationLogger.Debug("Page name found in cache",
						"page_id", pageID,
						"page_name", title,
						"section_id", sectionID)
					return title, true
				}
			}
		}
	}

	logging.AuthorizationLogger.Debug("Page name not found in cache", "page_id", pageID)
	return "", false
}

// GetPageNameWithProgress returns the page name for a given page ID with progress notification support.
// If not found in cache, it will perform a live API lookup with progress notifications.
func (nc *NotebookCache) GetPageNameWithProgress(ctx context.Context, pageID string, mcpServer interface{}, progressToken string, graphClient interface{}) (string, bool) {
	logging.AuthorizationLogger.Debug("Looking up page name with progress support",
		"page_id", pageID,
		"has_progress_token", progressToken != "",
		"progress_token", progressToken)

	// First try the fast cache lookup
	if name, found := nc.GetPageName(pageID); found {
		logging.AuthorizationLogger.Debug("Page name found in cache",
			"page_id", pageID,
			"page_name", name)
		return name, true
	}

	logging.AuthorizationLogger.Debug("Page name not found in cache, attempting live API lookup",
		"page_id", pageID)

	// Cast the graph client and MCP server
	graphClientTyped, ok := graphClient.(*graph.Client)
	if !ok {
		logging.AuthorizationLogger.Warn("Invalid graph client type for page lookup",
			"page_id", pageID,
			"client_type", fmt.Sprintf("%T", graphClient))
		return "", false
	}

	var mcpServerTyped *server.MCPServer
	if mcpServer != nil {
		mcpServerTyped, _ = mcpServer.(*server.MCPServer)
	}

	// Send progress notification for API lookup
	if mcpServerTyped != nil && progressToken != "" {
		err := mcpServerTyped.SendNotificationToClient(ctx, "notifications/progress", map[string]any{
			"progressToken": progressToken,
			"progress":      5,
			"total":         100,
			"message":       fmt.Sprintf("Looking up page name for %s via API...", pageID),
		})
		if err != nil {
			logging.AuthorizationLogger.Warn("Failed to send progress notification for page lookup",
				"error", err,
				"page_id", pageID)
		}
	}

	// Create a basic page client to fetch page details
	// We use the GetPageContent method to get page details including title
	pageApiURL := fmt.Sprintf("https://graph.microsoft.com/v1.0/me/onenote/pages/%s?$select=id,title", pageID)
	
	if mcpServerTyped != nil && progressToken != "" {
		err := mcpServerTyped.SendNotificationToClient(ctx, "notifications/progress", map[string]any{
			"progressToken": progressToken,
			"progress":      10,
			"total":         100,
			"message":       "Fetching page details from OneNote API...",
		})
		if err != nil {
			logging.AuthorizationLogger.Warn("Failed to send API progress notification",
				"error", err,
				"page_id", pageID)
		}
	}

	// Make API call to get page details
	httpResponse, err := graphClientTyped.MakeAuthenticatedRequest("GET", pageApiURL, nil, nil)
	if err != nil {
		logging.AuthorizationLogger.Warn("Failed to fetch page details by ID",
			"page_id", pageID,
			"error", err)

		// Send failure progress notification
		if mcpServerTyped != nil && progressToken != "" {
			err := mcpServerTyped.SendNotificationToClient(ctx, "notifications/progress", map[string]any{
				"progressToken": progressToken,
				"progress":      15,
				"total":         100,
				"message":       "Page lookup failed, page name unavailable",
			})
			if err != nil {
				logging.AuthorizationLogger.Warn("Failed to send failure progress notification", "error", err)
			}
		}

		return "", false
	}
	defer httpResponse.Body.Close()

	// Read and parse the response body
	var pageDetails map[string]interface{}
	err = json.NewDecoder(httpResponse.Body).Decode(&pageDetails)
	if err != nil {
		logging.AuthorizationLogger.Warn("Failed to parse page details response",
			"page_id", pageID,
			"error", err)
		return "", false
	}

	// Extract page name from the response
	pageName := ""
	if title, ok := pageDetails["title"].(string); ok {
		pageName = title
	}

	if pageName == "" {
		logging.AuthorizationLogger.Warn("Page details retrieved but title is missing",
			"page_id", pageID,
			"page_details", pageDetails)

		// Send failure progress notification
		if mcpServerTyped != nil && progressToken != "" {
			err := mcpServerTyped.SendNotificationToClient(ctx, "notifications/progress", map[string]any{
				"progressToken": progressToken,
				"progress":      20,
				"total":         100,
				"message":       "Page found but title is missing",
			})
			if err != nil {
				logging.AuthorizationLogger.Warn("Failed to send missing title progress notification", "error", err)
			}
		}

		return "", false
	}

	// Send success progress notification
	if mcpServerTyped != nil && progressToken != "" {
		err := mcpServerTyped.SendNotificationToClient(ctx, "notifications/progress", map[string]any{
			"progressToken": progressToken,
			"progress":      25,
			"total":         100,
			"message":       fmt.Sprintf("Page name resolved: %s", pageName),
		})
		if err != nil {
			logging.AuthorizationLogger.Warn("Failed to send success progress notification", "error", err)
		}
	}

	logging.AuthorizationLogger.Info("Page name resolved via API lookup",
		"page_id", pageID,
		"page_name", pageName)

	return pageName, true
}

// Global notebook cache instance
var globalNotebookCache *NotebookCache

// initializeDefaultNotebook initializes the default notebook on server startup
func initializeDefaultNotebook(graphClient *graph.Client, cfg *config.Config, cache *NotebookCache, logger *slog.Logger) {
	// Only initialize if we have valid authentication
	if graphClient.AccessToken == "" {
		logger.Info("No authentication available, skipping default notebook initialization")
		return
	}

	logger.Debug("Initializing default notebook", "configured_name", cfg.NotebookName)

	// Import the notebooks package for client creation
	notebookClient := notebooks.NewNotebookClient(graphClient)

	// Try to get the configured default notebook
	if cfg.NotebookName != "" {
		notebook, err := notebookClient.GetDetailedNotebookByName(cfg.NotebookName)
		if err != nil {
			logger.Debug("Failed to get configured default notebook, will try first available",
				"notebook_name", cfg.NotebookName, "error", err)
		} else {
			cache.SetNotebook(notebook)
			logger.Info("Initialized default notebook from configuration",
				"notebook_name", cfg.NotebookName,
				"notebook_id", notebook["id"])
			return
		}
	}

	// Fallback: Get the first available notebook
	notebooks, err := notebookClient.ListNotebooks()
	if err != nil {
		logger.Debug("Failed to list notebooks for default initialization", "error", err)
		return
	}

	if len(notebooks) == 0 {
		logger.Info("No notebooks found, default notebook not set")
		return
	}

	// Use the first notebook as default
	firstNotebook := notebooks[0]

	// Get detailed notebook info
	if notebookID, ok := firstNotebook["id"].(string); ok {
		// Get detailed notebooks and find the matching one
		detailedNotebooks, err := notebookClient.ListNotebooksDetailed()
		if err != nil {
			logger.Debug("Failed to get detailed notebooks list", "error", err)
			// Use basic info as fallback
			cache.SetNotebook(firstNotebook)
		} else {
			// Find the matching detailed notebook
			var detailedNotebook map[string]interface{}
			found := false
			for _, nb := range detailedNotebooks {
				if nbID, ok := nb["id"].(string); ok && nbID == notebookID {
					detailedNotebook = nb
					found = true
					break
				}
			}

			if found {
				cache.SetNotebook(detailedNotebook)
			} else {
				// Use basic info as fallback
				cache.SetNotebook(firstNotebook)
			}
		}

		if displayName, ok := firstNotebook["displayName"].(string); ok {
			logger.Info("Initialized default notebook (first available)",
				"notebook_name", displayName,
				"notebook_id", notebookID)
		}
	}
}

func main() {
	// Initialize structured logging first
	logging.Initialize()
	logger := logging.MainLogger

	// Initialize notebook cache
	globalNotebookCache = NewNotebookCache()

	// Parse command line flags
	mode := flag.String("mode", "stdio", "Server mode: stdio or streamable")
	port := flag.String("port", "8080", "Port for HTTP server (used with streamable mode)")
	flag.Parse()

	// Log version on startup
	logger.Info("OneNote MCP Server starting", "version", Version, "mode", *mode, "port", *port)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Reinitialize logging with configuration values
	logging.InitializeFromConfig(cfg)
	logger = logging.MainLogger // Get fresh logger instance after reinitialization
	logger.Debug("Logging reconfigured based on loaded configuration")

	// Initialize authentication
	oauthConfig := auth.NewOAuth2Config(cfg.ClientID, cfg.TenantID, cfg.RedirectURI)

	// Get token file path from environment variable or use default
	tokenPath := auth.GetTokenPath("tokens.json")

	// Get absolute path for token file
	absTokenPath, err := filepath.Abs(tokenPath)
	if err != nil {
		logger.Debug("Could not get absolute path for token file", "path", tokenPath, "error", err)
		absTokenPath = "unknown"
	}
	logger.Debug("Loading tokens (non-blocking)", "path", absTokenPath)

	// Try to load existing tokens, but don't block if they don't exist or are invalid
	tokenManager, err := auth.LoadTokens(tokenPath)
	if err != nil {
		logger.Info("No valid tokens found, server will start without authentication", "error", err)
		logger.Info("Use the 'initiateAuth' MCP tool to authenticate")
		// Create empty token manager to allow server startup
		tokenManager = &auth.TokenManager{
			AccessToken:  "",
			RefreshToken: "",
			Expiry:       0,
		}
	} else if tokenManager.IsExpired() {
		logger.Info("Existing tokens are expired, server will start without authentication")
		logger.Info("Use the 'initiateAuth' MCP tool to re-authenticate")
		// Keep the existing token manager but it will be handled as expired
	} else {
		logger.Info("Valid authentication tokens loaded successfully")
	}

	// Create Graph client with token refresh capability
	graphConfig := &graph.Config{
		NotebookName: cfg.NotebookName,
	}
	logger.Debug("Creating Graph client", "token_path", absTokenPath)
	graphClient := graph.NewClientWithTokenRefresh(tokenManager.AccessToken, oauthConfig, tokenManager, tokenPath, graphConfig)

	// Create authentication manager for MCP tools
	authManager := auth.NewAuthManager(oauthConfig, tokenManager, tokenPath)

	// Set up token refresh callback to update the graph client
	authManager.SetTokenRefreshCallback(func(newAccessToken string) {
		logger.Debug("Updating graph client with new access token")
		graphClient.UpdateToken(newAccessToken)
	})

	logger.Debug("Authentication manager created")

	// Create MCP server with progress streaming support
	s := server.NewMCPServer("OneNote MCP Server", "1.6.0",
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true),
		server.WithPromptCapabilities(false))

	// Register MCP Tools and Resources
	registerTools(s, graphClient, authManager, globalNotebookCache, cfg)
	registerResources(s, graphClient, cfg)
	
	// Set API references in cache for authorization fallback
	globalNotebookCache.SetAPIReferences(graphClient, s)

	// Initialize default notebook if authentication is available
	initializeDefaultNotebook(graphClient, cfg, globalNotebookCache, logger)

	switch *mode {
	case "streamable":
		logger.Info("Starting MCP server", "transport", "Streamable HTTP", "port", *port, "request_logging", "enabled")
		streamableServer := server.NewStreamableHTTPServer(s,
			server.WithStateLess(*cfg.Stateless))
		handler := applyAuthIfEnabled(streamableServer, cfg)
		logger.Info("Streamable HTTP server listening", "address", fmt.Sprintf("http://localhost:%s", *port))
		if err := http.ListenAndServe(":"+*port, handler); err != nil {
			logger.Error("Streamable HTTP server error", "error", err)
			os.Exit(1)
		}
	case "stdio":
		logger.Info("Starting MCP server", "transport", "stdio")
		if err := server.ServeStdio(s); err != nil {
			logger.Error("Stdio server error", "error", err)
			os.Exit(1)
		}
	default:
		logger.Error("Invalid mode specified", "mode", *mode, "valid_modes", []string{"stdio", "streamable"})
		os.Exit(1)
	}
}

// applyAuthIfEnabled applies bearer token authentication middleware if enabled in configuration.
// Also applies request logging middleware for all HTTP/SSE requests.
// Returns the handler with middleware applied.
func applyAuthIfEnabled(handler http.Handler, cfg *config.Config) http.Handler {
	logger := logging.MainLogger

	// Always apply request logging middleware first (outermost)
	//handler = auth.RequestLoggingMiddleware()(handler)
	//logger.Debug("Request logging middleware enabled for HTTP transport")

	// Check if MCP authentication is enabled and properly configured
	if cfg.MCPAuth != nil && cfg.MCPAuth.Enabled {
		if cfg.MCPAuth.BearerToken == "" {
			logger.Warn("MCP authentication is enabled but no bearer token is configured",
				"recommendation", "set MCP_BEARER_TOKEN environment variable or add bearer_token to config file")
			return handler
		}

		logger.Info("MCP authentication enabled for HTTP transport",
			"token_length", len(cfg.MCPAuth.BearerToken))
		return auth.BearerTokenMiddleware(cfg.MCPAuth.BearerToken)(handler)
	}

	logger.Debug("MCP authentication disabled - HTTP endpoints are not protected")
	return handler
}

// stringifySections formats sections for output.
func stringifySections(sections interface{}) string {
	if sections == nil {
		return emptyJSONArray
	}

	jsonBytes, err := json.Marshal(sections)
	if err != nil {
		slog.Error("Failed to marshal sections", "error", err)
		return fmt.Sprintf("%v", sections)
	}
	return string(jsonBytes)
}
