// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package main

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/gebl/onenote-mcp-server/internal/auth"
	"github.com/gebl/onenote-mcp-server/internal/authorization"
	"github.com/gebl/onenote-mcp-server/internal/resources"
	"github.com/gebl/onenote-mcp-server/internal/utils"
)

// registerAuthTools registers authentication-related MCP tools
func registerAuthTools(s *server.MCPServer, authManager *auth.AuthManager, authConfig *authorization.AuthorizationConfig, cache authorization.NotebookCache, quickNoteConfig authorization.QuickNoteConfig) {
	// auth_status: Get current authentication status
	auth_statusTool := mcp.NewTool(
		"auth_status",
		mcp.WithDescription(resources.MustGetToolDescription("auth_status")),
	)
	auth_statusHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logger := utils.NewToolLogger("auth_status")

		if authManager == nil {
			return mcp.NewToolResultError("Authentication manager not available"), nil
		}

		status := authManager.GetAuthStatus()

		logger.LogSuccess("authenticated", status.Authenticated)
		return utils.ToolResults.NewJSONResult("auth_status", status), nil
	}
	// auth_status doesn't require authorization since it's needed to check auth state
	s.AddTool(auth_statusTool, server.ToolHandlerFunc(auth_statusHandler))

	// auth_refresh: Manually refresh authentication token
	auth_refreshTool := mcp.NewTool(
		"auth_refresh",
		mcp.WithDescription(resources.MustGetToolDescription("auth_refresh")),
	)
	auth_refreshHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logger := utils.NewToolLogger("auth_refresh")

		if authManager == nil {
			return mcp.NewToolResultError("Authentication manager not available"), nil
		}

		status, err := authManager.RefreshToken()
		if err != nil {
			logger.LogError(err)
			return utils.ToolResults.NewError("refresh token", err), nil
		}

		logger.LogSuccess()
		return utils.ToolResults.NewJSONResult("auth_refresh", status), nil
	}
	s.AddTool(auth_refreshTool, server.ToolHandlerFunc(authorization.AuthorizedToolHandler("auth_refresh", auth_refreshHandler, authConfig, cache, quickNoteConfig)))

	// auth_initiate: Start new authentication flow
	auth_initiateTool := mcp.NewTool(
		"auth_initiate",
		mcp.WithDescription(resources.MustGetToolDescription("auth_initiate")),
	)
	auth_initiateHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logger := utils.NewToolLogger("auth_initiate")

		if authManager == nil {
			return mcp.NewToolResultError("Authentication manager not available"), nil
		}

		session, err := authManager.InitiateAuth()
		if err != nil {
			logger.LogError(err)
			return utils.ToolResults.NewError("initiate authentication", err), nil
		}

		// Create response with user instructions
		response := map[string]interface{}{
			"authUrl":         session.AuthURL,
			"instructions":    "Visit this URL in your browser to authenticate with Microsoft. The authentication will complete automatically.",
			"localServerPort": session.LocalServerPort,
			"timeoutMinutes":  session.TimeoutMinutes,
			"state":           session.State,
		}

		logger.LogSuccess("auth_url_generated", true)
		return utils.ToolResults.NewJSONResult("auth_initiate", response), nil
	}
	// auth_initiate doesn't require authorization since it's needed to establish auth
	s.AddTool(auth_initiateTool, server.ToolHandlerFunc(auth_initiateHandler))

	// auth_clear: Clear stored authentication tokens
	auth_clearTool := mcp.NewTool(
		"auth_clear",
		mcp.WithDescription(resources.MustGetToolDescription("auth_clear")),
	)
	auth_clearHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logger := utils.NewToolLogger("auth_clear")

		if authManager == nil {
			return mcp.NewToolResultError("Authentication manager not available"), nil
		}

		err := authManager.ClearAuth()
		if err != nil {
			logger.LogError(err)
			return utils.ToolResults.NewError("clear authentication", err), nil
		}

		response := map[string]interface{}{
			"success": true,
			"message": "Authentication tokens cleared successfully. Use auth_initiate to re-authenticate.",
		}

		logger.LogSuccess()
		return utils.ToolResults.NewJSONResult("auth_clear", response), nil
	}
	// auth_clear doesn't require authorization since it clears auth state
	s.AddTool(auth_clearTool, server.ToolHandlerFunc(auth_clearHandler))
}
