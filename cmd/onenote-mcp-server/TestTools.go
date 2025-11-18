// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package main

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/gebl/onenote-mcp-server/internal/authorization"
	"github.com/gebl/onenote-mcp-server/internal/logging"
)

// registerTestTools registers test and utility MCP tools
func registerTestTools(s *mcp.Server, authConfig *authorization.AuthorizationConfig, cache authorization.NotebookCache, quickNoteConfig authorization.QuickNoteConfig) {
	// Currently no test tools are registered
	// This function is kept for future test tools if needed

	logging.ToolsLogger.Debug("Test tools registered successfully")
}