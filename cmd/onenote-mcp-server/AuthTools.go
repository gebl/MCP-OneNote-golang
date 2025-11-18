// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package main

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/gebl/onenote-mcp-server/internal/auth"
	"github.com/gebl/onenote-mcp-server/internal/authorization"
	"github.com/gebl/onenote-mcp-server/internal/resources"
	"github.com/gebl/onenote-mcp-server/internal/utils"
)

// Input/Output structs for auth tools
type AuthStatusInput struct{}
type AuthStatusOutput struct {
	Status interface{} `json:"status"`
}

type AuthRefreshInput struct{}
type AuthRefreshOutput struct {
	Status interface{} `json:"status"`
}

type AuthInitiateInput struct{}
type AuthInitiateOutput struct {
	AuthURL         string `json:"authUrl"`
	Instructions    string `json:"instructions"`
	LocalServerPort int    `json:"localServerPort"`
	TimeoutMinutes  int    `json:"timeoutMinutes"`
	State           string `json:"state"`
}

type AuthClearInput struct{}
type AuthClearOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// registerAuthTools registers authentication-related MCP tools
func registerAuthTools(s *mcp.Server, authManager *auth.AuthManager, authConfig *authorization.AuthorizationConfig, cache authorization.NotebookCache, quickNoteConfig authorization.QuickNoteConfig) {
	// auth_status: Get current authentication status
	auth_statusHandler := func(ctx context.Context, req *mcp.CallToolRequest, input AuthStatusInput) (*mcp.CallToolResult, AuthStatusOutput, error) {
		logger := utils.NewToolLogger("auth_status")

		if authManager == nil {
			return utils.ToolResults.NewError("auth_status", fmt.Errorf("authentication manager not available")), AuthStatusOutput{}, nil
		}

		status := authManager.GetAuthStatus()

		logger.LogSuccess("authenticated", status.Authenticated)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Authentication status: %+v", status)},
			},
		}, AuthStatusOutput{Status: status}, nil
	}

	// Register auth_status tool
	mcp.AddTool(s, &mcp.Tool{
		Name:        "auth_status",
		Description: resources.MustGetToolDescription("auth_status"),
	}, auth_statusHandler)

	// auth_refresh: Manually refresh authentication token
	auth_refreshHandler := func(ctx context.Context, req *mcp.CallToolRequest, input AuthRefreshInput) (*mcp.CallToolResult, AuthRefreshOutput, error) {
		logger := utils.NewToolLogger("auth_refresh")

		if authManager == nil {
			return utils.ToolResults.NewError("auth_refresh", fmt.Errorf("authentication manager not available")), AuthRefreshOutput{}, nil
		}

		status, err := authManager.RefreshToken()
		if err != nil {
			logger.LogError(err)
			return utils.ToolResults.NewError("refresh token", err), AuthRefreshOutput{}, nil
		}

		logger.LogSuccess()
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Token refresh result: %+v", status)},
			},
		}, AuthRefreshOutput{Status: status}, nil
	}

	// Register auth_refresh tool (will need authorization wrapper)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "auth_refresh",
		Description: resources.MustGetToolDescription("auth_refresh"),
	}, auth_refreshHandler)

	// auth_initiate: Start new authentication flow
	auth_initiateHandler := func(ctx context.Context, req *mcp.CallToolRequest, input AuthInitiateInput) (*mcp.CallToolResult, AuthInitiateOutput, error) {
		logger := utils.NewToolLogger("auth_initiate")

		if authManager == nil {
			return utils.ToolResults.NewError("auth_initiate", fmt.Errorf("authentication manager not available")), AuthInitiateOutput{}, nil
		}

		session, err := authManager.InitiateAuth()
		if err != nil {
			logger.LogError(err)
			return utils.ToolResults.NewError("initiate authentication", err), AuthInitiateOutput{}, nil
		}

		output := AuthInitiateOutput{
			AuthURL:         session.AuthURL,
			Instructions:    "Visit this URL in your browser to authenticate with Microsoft. The authentication will complete automatically.",
			LocalServerPort: session.LocalServerPort,
			TimeoutMinutes:  session.TimeoutMinutes,
			State:           session.State,
		}

		logger.LogSuccess("auth_url_generated", true)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Visit this URL to authenticate: %s", session.AuthURL)},
			},
		}, output, nil
	}

	// Register auth_initiate tool
	mcp.AddTool(s, &mcp.Tool{
		Name:        "auth_initiate",
		Description: resources.MustGetToolDescription("auth_initiate"),
	}, auth_initiateHandler)

	// auth_clear: Clear stored authentication tokens
	auth_clearHandler := func(ctx context.Context, req *mcp.CallToolRequest, input AuthClearInput) (*mcp.CallToolResult, AuthClearOutput, error) {
		logger := utils.NewToolLogger("auth_clear")

		if authManager == nil {
			return utils.ToolResults.NewError("auth_clear", fmt.Errorf("authentication manager not available")), AuthClearOutput{}, nil
		}

		err := authManager.ClearAuth()
		if err != nil {
			logger.LogError(err)
			return utils.ToolResults.NewError("clear tokens", err), AuthClearOutput{Success: false, Message: err.Error()}, nil
		}

		logger.LogSuccess()
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "Authentication tokens cleared successfully"},
			},
		}, AuthClearOutput{Success: true, Message: "Tokens cleared successfully"}, nil
	}

	// Register auth_clear tool
	mcp.AddTool(s, &mcp.Tool{
		Name:        "auth_clear",
		Description: resources.MustGetToolDescription("auth_clear"),
	}, auth_clearHandler)
}
