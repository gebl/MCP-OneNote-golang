package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/gebl/onenote-mcp-server/internal/auth"
	"github.com/gebl/onenote-mcp-server/internal/authorization"
	"github.com/gebl/onenote-mcp-server/internal/logging"
	"github.com/gebl/onenote-mcp-server/internal/resources"
)

// registerAuthTools registers authentication-related MCP tools
func registerAuthTools(s *server.MCPServer, authManager *auth.AuthManager, authConfig *authorization.AuthorizationConfig, cache authorization.NotebookCache, quickNoteConfig authorization.QuickNoteConfig) {
	// getAuthStatus: Get current authentication status
	getAuthStatusTool := mcp.NewTool(
		"getAuthStatus",
		mcp.WithDescription(resources.MustGetToolDescription("getAuthStatus")),
	)
	getAuthStatusHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logging.ToolsLogger.Info("Checking authentication status", "operation", "getAuthStatus", "type", "tool_invocation")

		if authManager == nil {
			return mcp.NewToolResultError("Authentication manager not available"), nil
		}

		status := authManager.GetAuthStatus()

		jsonBytes, err := json.Marshal(status)
		if err != nil {
			logging.ToolsLogger.Error("getAuthStatus failed to marshal status", "error", err)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal auth status: %v", err)), nil
		}

		logging.ToolsLogger.Debug("getAuthStatus operation completed", "authenticated", status.Authenticated)
		return mcp.NewToolResultText(string(jsonBytes)), nil
	}
	// getAuthStatus doesn't require authorization since it's needed to check auth state
	s.AddTool(getAuthStatusTool, server.ToolHandlerFunc(getAuthStatusHandler))

	// refreshToken: Manually refresh authentication token
	refreshTokenTool := mcp.NewTool(
		"refreshToken",
		mcp.WithDescription(resources.MustGetToolDescription("refreshToken")),
	)
	refreshTokenHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logging.ToolsLogger.Info("Refreshing authentication token", "operation", "refreshToken", "type", "tool_invocation")

		if authManager == nil {
			return mcp.NewToolResultError("Authentication manager not available"), nil
		}

		status, err := authManager.RefreshToken()
		if err != nil {
			logging.ToolsLogger.Error("refreshToken operation failed", "error", err, "operation", "refreshToken")
			return mcp.NewToolResultError(fmt.Sprintf("Failed to refresh token: %v", err)), nil
		}

		jsonBytes, err := json.Marshal(status)
		if err != nil {
			logging.ToolsLogger.Error("refreshToken failed to marshal status", "error", err)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal refresh status: %v", err)), nil
		}

		logging.ToolsLogger.Debug("refreshToken completed successfully")
		return mcp.NewToolResultText(string(jsonBytes)), nil
	}
	s.AddTool(refreshTokenTool, server.ToolHandlerFunc(authorization.AuthorizedToolHandler("refreshToken", refreshTokenHandler, authConfig, cache, quickNoteConfig)))

	// initiateAuth: Start new authentication flow
	initiateAuthTool := mcp.NewTool(
		"initiateAuth",
		mcp.WithDescription(resources.MustGetToolDescription("initiateAuth")),
	)
	initiateAuthHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logging.ToolsLogger.Info("Starting authentication flow", "operation", "initiateAuth", "type", "tool_invocation")

		if authManager == nil {
			return mcp.NewToolResultError("Authentication manager not available"), nil
		}

		session, err := authManager.InitiateAuth()
		if err != nil {
			logging.ToolsLogger.Error("initiateAuth operation failed", "error", err, "operation", "initiateAuth")
			return mcp.NewToolResultError(fmt.Sprintf("Failed to initiate authentication: %v", err)), nil
		}

		// Create response with user instructions
		response := map[string]interface{}{
			"authUrl":         session.AuthURL,
			"instructions":    "Visit this URL in your browser to authenticate with Microsoft. The authentication will complete automatically.",
			"localServerPort": session.LocalServerPort,
			"timeoutMinutes":  session.TimeoutMinutes,
			"state":           session.State,
		}

		jsonBytes, err := json.Marshal(response)
		if err != nil {
			logging.ToolsLogger.Error("initiateAuth failed to marshal response", "error", err)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal auth response: %v", err)), nil
		}

		logging.ToolsLogger.Debug("initiateAuth completed, auth URL generated")
		return mcp.NewToolResultText(string(jsonBytes)), nil
	}
	// initiateAuth doesn't require authorization since it's needed to establish auth
	s.AddTool(initiateAuthTool, server.ToolHandlerFunc(initiateAuthHandler))

	// clearAuth: Clear stored authentication tokens
	clearAuthTool := mcp.NewTool(
		"clearAuth",
		mcp.WithDescription(resources.MustGetToolDescription("clearAuth")),
	)
	clearAuthHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logging.ToolsLogger.Info("Clearing authentication", "operation", "clearAuth", "type", "tool_invocation")

		if authManager == nil {
			return mcp.NewToolResultError("Authentication manager not available"), nil
		}

		err := authManager.ClearAuth()
		if err != nil {
			logging.ToolsLogger.Error("clearAuth operation failed", "error", err, "operation", "clearAuth")
			return mcp.NewToolResultError(fmt.Sprintf("Failed to clear authentication: %v", err)), nil
		}

		response := map[string]interface{}{
			"success": true,
			"message": "Authentication tokens cleared successfully. Use initiateAuth to re-authenticate.",
		}

		jsonBytes, err := json.Marshal(response)
		if err != nil {
			logging.ToolsLogger.Error("clearAuth failed to marshal response", "error", err)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal clear response: %v", err)), nil
		}

		logging.ToolsLogger.Debug("clearAuth completed successfully")
		return mcp.NewToolResultText(string(jsonBytes)), nil
	}
	// clearAuth doesn't require authorization since it clears auth state
	s.AddTool(clearAuthTool, server.ToolHandlerFunc(clearAuthHandler))

	logging.ToolsLogger.Debug("Authentication tools registered successfully")
}
