// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

// OneNote MCP Server Resources
//
// This file implements MCP (Model Context Protocol) resources for accessing Microsoft OneNote data
// through a hierarchical REST-like URI structure. The resources provide AI models with structured
// access to OneNote notebooks, sections, and pages.
//
// Note: Notebook-related resources have been moved to NotebookResources.go for better organization.
// This file now contains only non-notebook resources and the main registration function.

package main

import (
	"github.com/mark3labs/mcp-go/server"

	"github.com/gebl/onenote-mcp-server/internal/config"
	"github.com/gebl/onenote-mcp-server/internal/graph"
	"github.com/gebl/onenote-mcp-server/internal/logging"
)

// registerResources registers all MCP resources for the OneNote server
func registerResources(s *server.MCPServer, graphClient *graph.Client, cfg *config.Config) {
	logging.MainLogger.Debug("Starting resource registration process")

	// Register notebook-related resources from the separate NotebookResources.go file
	registerNotebookResources(s, graphClient, cfg)

	// Register section-related resources from the separate SectionResources.go file
	registerSectionResources(s, graphClient, cfg)

	// Register page-related resources from the separate PageResources.go file
	registerPageResources(s, graphClient, cfg)

	// Future: Additional resources can be registered here
	// For example: direct section access, etc.

	logging.MainLogger.Debug("Resource registration completed successfully")
}
