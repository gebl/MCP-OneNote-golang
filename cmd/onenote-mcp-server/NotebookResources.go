// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

// OneNote MCP Server Notebook Resources
//
// This file implements MCP (Model Context Protocol) notebook-related resources for accessing
// Microsoft OneNote data through a hierarchical REST-like URI structure. The resources provide
// AI models with structured access to OneNote notebooks.
//
// ## Notebook Resource URIs
//
// ### Available Notebook Resource URIs
//
// #### 1. List All Notebooks
// **URI:** `onenote://notebooks`
// **Purpose:** Get a complete list of all accessible OneNote notebooks
// **Returns:** JSON array of notebook objects with metadata (IDs, names, creation dates, etc.)
//
// #### 2. Get Specific Notebook Details
// **URI:** `onenote://notebooks/{NotebookDisplayName}`
// **Purpose:** Get detailed information about a specific notebook by its display name
// **Parameters:**
//   - `{NotebookDisplayName}`: URL-encoded display name of the notebook (spaces become %20)
// **Returns:** JSON object with comprehensive notebook metadata
//
// Note: Section-related resources have been moved to SectionResources.go for better organization.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/gebl/onenote-mcp-server/internal/graph"
	"github.com/gebl/onenote-mcp-server/internal/logging"
	"github.com/gebl/onenote-mcp-server/internal/notebooks"
)

// registerNotebookResources registers all notebook-related MCP resources
func registerNotebookResources(s *server.MCPServer, graphClient *graph.Client) {
	logging.MainLogger.Debug("Starting notebook resource registration process")

	// Register notebooks resource - list all notebooks with full metadata
	logging.MainLogger.Debug("Creating notebooks resource",
		"resource_uri", "onenote://notebooks",
		"resource_type", "static_resource")
	notebooksResource := mcp.NewResource(
		"onenote://notebooks",
		"OneNote Notebooks",
		mcp.WithResourceDescription("List of all OneNote notebooks accessible to the authenticated user with comprehensive metadata including timestamps, ownership, links, and properties"),
		mcp.WithMIMEType("application/json"),
	)

	s.AddResource(notebooksResource, func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		logging.MainLogger.Debug("Resource handler invoked",
			"resource_uri", "onenote://notebooks",
			"request_uri", request.Params.URI,
			"handler_type", "notebooks_list")

		// Create specialized client for notebooks
		notebookClient := notebooks.NewNotebookClient(graphClient)
		logging.MainLogger.Debug("Created notebook client for resource request")

		// Get notebooks with detailed information
		logging.MainLogger.Debug("Calling ListNotebooksDetailed from resource handler")
		notebooks, err := notebookClient.ListNotebooksDetailed()
		if err != nil {
			logging.MainLogger.Error("Failed to list detailed notebooks for resource",
				"error", err,
				"resource_uri", "onenote://notebooks")
			return nil, fmt.Errorf("failed to list detailed notebooks: %v", err)
		}

		logging.MainLogger.Debug("Retrieved detailed notebooks for resource",
			"count", len(notebooks),
			"resource_uri", "onenote://notebooks")

		// Convert to JSON
		logging.MainLogger.Debug("Marshaling notebooks data to JSON")
		jsonData, err := json.Marshal(notebooks)
		if err != nil {
			logging.MainLogger.Error("Failed to marshal detailed notebooks to JSON",
				"error", err,
				"notebooks_count", len(notebooks))
			return nil, fmt.Errorf("failed to marshal detailed notebooks: %v", err)
		}

		responseSize := len(jsonData)
		logging.MainLogger.Debug("Successfully prepared resource response",
			"resource_uri", "onenote://notebooks",
			"response_size_bytes", responseSize,
			"notebooks_count", len(notebooks))

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "onenote://notebooks",
				MIMEType: "application/json",
				Text:     string(jsonData),
			},
		}, nil
	})
	logging.MainLogger.Debug("Registered notebooks resource successfully",
		"resource_uri", "onenote://notebooks")

	// Register notebook by name resource template
	logging.MainLogger.Debug("Creating notebook by name resource template",
		"template_pattern", "onenote://notebooks/{name}",
		"resource_type", "template_resource")
	notebookByNameTemplate := mcp.NewResourceTemplate(
		"onenote://notebooks/{name}",
		"OneNote Notebook by Name",
		mcp.WithTemplateDescription("Get a specific OneNote notebook by its display name with comprehensive metadata including timestamps, ownership, links, and properties"),
		mcp.WithTemplateMIMEType("application/json"),
	)

	s.AddResourceTemplate(notebookByNameTemplate, func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		logging.MainLogger.Debug("Resource template handler invoked",
			"template_pattern", "onenote://notebooks/{name}",
			"request_uri", request.Params.URI,
			"handler_type", "notebook_by_name")

		// Extract notebook name from URI
		notebookName := extractNotebookNameFromURI(request.Params.URI)
		if notebookName == "" {
			logging.MainLogger.Error("Invalid notebook name in URI",
				"request_uri", request.Params.URI,
				"extracted_name", notebookName)
			return nil, fmt.Errorf("invalid notebook name in URI: %s", request.Params.URI)
		}

		logging.MainLogger.Debug("Extracted notebook name from URI",
			"notebook_name", notebookName,
			"request_uri", request.Params.URI)

		// Create specialized client for notebooks
		notebookClient := notebooks.NewNotebookClient(graphClient)
		logging.MainLogger.Debug("Created notebook client for notebook by name resource request")

		// Get detailed notebook information by name using the existing function
		logging.MainLogger.Debug("Calling GetDetailedNotebookByName from resource handler",
			"notebook_name", notebookName)
		detailedNotebook, err := notebookClient.GetDetailedNotebookByName(notebookName)
		if err != nil {
			logging.MainLogger.Error("Failed to get detailed notebook for resource",
				"notebook_name", notebookName,
				"error", err,
				"request_uri", request.Params.URI)
			return nil, fmt.Errorf("failed to get detailed notebook '%s': %v", notebookName, err)
		}

		logging.MainLogger.Debug("Retrieved detailed notebook by name",
			"notebook_name", notebookName,
			"attributes_count", len(detailedNotebook),
			"request_uri", request.Params.URI)

		logging.MainLogger.Debug("Marshaling detailed notebook data to JSON",
			"notebook_name", notebookName)
		jsonData, err := json.Marshal(detailedNotebook)
		if err != nil {
			logging.MainLogger.Error("Failed to marshal detailed notebook to JSON",
				"error", err,
				"notebook_name", notebookName,
				"attributes_count", len(detailedNotebook))
			return nil, fmt.Errorf("failed to marshal detailed notebook: %v", err)
		}

		responseSize := len(jsonData)
		logging.MainLogger.Debug("Successfully prepared notebook by name resource response",
			"notebook_name", notebookName,
			"request_uri", request.Params.URI,
			"response_size_bytes", responseSize,
			"attributes_count", len(detailedNotebook))

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: "application/json",
				Text:     string(jsonData),
			},
		}, nil
	})
	logging.MainLogger.Debug("Registered notebook by name resource template successfully",
		"template_pattern", "onenote://notebooks/{name}")

	logging.MainLogger.Debug("Notebook resource registration completed successfully",
		"resources_registered", 2,
		"static_resources", 1,
		"template_resources", 1)
}

// Notebook-specific URI extraction functions

// extractNotebookIDFromURI extracts the notebook ID from a URI like "onenote://notebooks/{id}"
func extractNotebookIDFromURI(uri string) string {
	// URI format: onenote://notebooks/{id}
	parts := strings.Split(uri, "/")
	if len(parts) >= 3 && parts[0] == "onenote:" && parts[2] == "notebooks" && len(parts) >= 4 {
		return parts[3]
	}
	return ""
}

// extractNotebookNameFromURI extracts the notebook name from a URI like "onenote://notebooks/{name}"
func extractNotebookNameFromURI(uri string) string {
	// URI format: onenote://notebooks/{name}
	parts := strings.Split(uri, "/")
	if len(parts) >= 4 && parts[0] == "onenote:" && parts[2] == "notebooks" {
		// URL decode the notebook name since it's part of a URI
		decodedName, err := url.QueryUnescape(parts[3])
		if err != nil {
			logging.MainLogger.Warn("Failed to URL decode notebook name, using raw value",
				"raw_name", parts[3],
				"error", err)
			return parts[3] // fallback to raw value
		}
		logging.MainLogger.Debug("URL decoded notebook name",
			"raw_name", parts[3],
			"decoded_name", decodedName)
		return decodedName
	}
	return ""
}
