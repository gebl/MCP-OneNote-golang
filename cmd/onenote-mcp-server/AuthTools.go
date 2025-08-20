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
	// getAuthStatus: Get current authentication status
	getAuthStatusTool := mcp.NewTool(
		"getAuthStatus",
		mcp.WithDescription(resources.MustGetToolDescription("getAuthStatus")),
	)
	getAuthStatusHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logger := utils.NewToolLogger("getAuthStatus")

		if authManager == nil {
			return mcp.NewToolResultError("Authentication manager not available"), nil
		}

		status := authManager.GetAuthStatus()

		logger.LogSuccess("authenticated", status.Authenticated)
		return utils.ToolResults.NewJSONResult("getAuthStatus", status), nil
	}
	// getAuthStatus doesn't require authorization since it's needed to check auth state
	s.AddTool(getAuthStatusTool, server.ToolHandlerFunc(getAuthStatusHandler))

	// refreshToken: Manually refresh authentication token
	refreshTokenTool := mcp.NewTool(
		"refreshToken",
		mcp.WithDescription(resources.MustGetToolDescription("refreshToken")),
	)
	refreshTokenHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logger := utils.NewToolLogger("refreshToken")

		if authManager == nil {
			return mcp.NewToolResultError("Authentication manager not available"), nil
		}

		status, err := authManager.RefreshToken()
		if err != nil {
			logger.LogError(err)
			return utils.ToolResults.NewError("refresh token", err), nil
		}

		logger.LogSuccess()
		return utils.ToolResults.NewJSONResult("refreshToken", status), nil
	}
	s.AddTool(refreshTokenTool, server.ToolHandlerFunc(authorization.AuthorizedToolHandler("refreshToken", refreshTokenHandler, authConfig, cache, quickNoteConfig)))

	// initiateAuth: Start new authentication flow
	initiateAuthTool := mcp.NewTool(
		"initiateAuth",
		mcp.WithDescription(resources.MustGetToolDescription("initiateAuth")),
	)
	initiateAuthHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logger := utils.NewToolLogger("initiateAuth")

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
		return utils.ToolResults.NewJSONResult("initiateAuth", response), nil
	}
	// initiateAuth doesn't require authorization since it's needed to establish auth
	s.AddTool(initiateAuthTool, server.ToolHandlerFunc(initiateAuthHandler))

	// clearAuth: Clear stored authentication tokens
	clearAuthTool := mcp.NewTool(
		"clearAuth",
		mcp.WithDescription(resources.MustGetToolDescription("clearAuth")),
	)
	clearAuthHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logger := utils.NewToolLogger("clearAuth")

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
			"message": "Authentication tokens cleared successfully. Use initiateAuth to re-authenticate.",
		}

		logger.LogSuccess()
		return utils.ToolResults.NewJSONResult("clearAuth", response), nil
	}
	// clearAuth doesn't require authorization since it clears auth state
	s.AddTool(clearAuthTool, server.ToolHandlerFunc(clearAuthHandler))
}
