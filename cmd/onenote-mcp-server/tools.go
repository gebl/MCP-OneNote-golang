// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package main

import (
	"github.com/mark3labs/mcp-go/server"

	"github.com/gebl/onenote-mcp-server/internal/auth"
	"github.com/gebl/onenote-mcp-server/internal/authorization"
	"github.com/gebl/onenote-mcp-server/internal/config"
	"github.com/gebl/onenote-mcp-server/internal/graph"
	"github.com/gebl/onenote-mcp-server/internal/logging"
	"github.com/gebl/onenote-mcp-server/internal/pages"
)

// registerTools registers all MCP tools for the OneNote server
func registerTools(s *server.MCPServer, graphClient *graph.Client, authManager *auth.AuthManager, notebookCache *NotebookCache, cfg *config.Config) {
	// Create specialized clients for each domain
	pageClient := pages.NewPageClient(graphClient)
	logging.ToolsLogger.Debug("Starting tool registration")

	// Create authorization adapters if authorization is enabled
	var cacheAdapter authorization.NotebookCache
	var quickNoteAdapter authorization.QuickNoteConfig
	var authConfig *authorization.AuthorizationConfig
	
	if cfg != nil && cfg.Authorization != nil && cfg.Authorization.Enabled {
		authConfig = cfg.Authorization
		cacheAdapter = authorization.NewNotebookCacheAdapter(notebookCache)
		quickNoteAdapter = authorization.NewQuickNoteConfigAdapter(cfg.QuickNote, cfg.NotebookName)
		
		authInfo := authorization.GetAuthorizationInfo(cfg.Authorization)
		logging.ToolsLogger.Info("Authorization is enabled and integrated into tool registration",
			"enabled", authInfo.Enabled,
			"default_mode", authInfo.DefaultMode,
			"tool_categories", authInfo.ToolCategories,
			"notebook_rules", authInfo.NotebookRules,
			"section_rules", authInfo.SectionRules,
			"compiled_matchers", authInfo.CompiledMatchers)
	} else {
		logging.ToolsLogger.Debug("Authorization is disabled or not configured")
	}

	// Register authentication tools
	registerAuthTools(s, authManager, authConfig, cacheAdapter, quickNoteAdapter)

	// Register notebook and section tools
	registerNotebookTools(s, graphClient, notebookCache, authConfig, cacheAdapter, quickNoteAdapter)

	// Register page tools
	registerPageTools(s, pageClient, graphClient, notebookCache, cfg, authConfig, cacheAdapter, quickNoteAdapter)

	// Register test tools
	registerTestTools(s, authConfig, cacheAdapter, quickNoteAdapter)

	logging.ToolsLogger.Debug("All tools registered successfully")
}
