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
)

const (
	// Version is the current version of the OneNote MCP server
	Version        = "1.7.0"
	emptyJSONArray = "[]"
)

// NotebookCache holds the currently selected notebook information and sections cache
type NotebookCache struct {
	mu             sync.RWMutex
	notebook       map[string]interface{}
	notebookID     string
	displayName    string
	isSet          bool
	sectionsTree   map[string]interface{}              // Cached sections tree structure
	sectionsCached bool                                // Whether sections have been cached
	pagesCache     map[string][]map[string]interface{} // Pages cache by section ID
	pagesCacheTime map[string]time.Time                // Cache timestamps by section ID
}

// NewNotebookCache creates a new notebook cache
func NewNotebookCache() *NotebookCache {
	return &NotebookCache{
		notebook:       make(map[string]interface{}),
		sectionsTree:   make(map[string]interface{}),
		pagesCache:     make(map[string][]map[string]interface{}),
		pagesCacheTime: make(map[string]time.Time),
	}
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

	// Register MCP Tools, Resources, and Completions
	registerTools(s, graphClient, authManager, globalNotebookCache)
	registerResources(s, graphClient)
	registerCompletions(s, graphClient)

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
