package main

import (
	"github.com/mark3labs/mcp-go/server"

	"github.com/gebl/onenote-mcp-server/internal/auth"
	"github.com/gebl/onenote-mcp-server/internal/graph"
	"github.com/gebl/onenote-mcp-server/internal/logging"
	"github.com/gebl/onenote-mcp-server/internal/pages"
)

// registerTools registers all MCP tools for the OneNote server
func registerTools(s *server.MCPServer, graphClient *graph.Client, authManager *auth.AuthManager, notebookCache *NotebookCache) {
	// Create specialized clients for each domain
	pageClient := pages.NewPageClient(graphClient)
	logging.ToolsLogger.Debug("Starting tool registration")

	// Register authentication tools
	registerAuthTools(s, authManager)

	// Register notebook and section tools
	registerNotebookTools(s, graphClient, notebookCache)

	// Register page tools
	registerPageTools(s, pageClient, graphClient, notebookCache)

	// Register test tools
	registerTestTools(s)

	logging.ToolsLogger.Debug("All tools registered successfully")
}
