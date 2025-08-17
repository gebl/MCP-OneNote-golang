package main

import (
	"context"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/gebl/onenote-mcp-server/internal/authorization"
	"github.com/gebl/onenote-mcp-server/internal/logging"
)

// registerTestTools registers test and utility MCP tools
func registerTestTools(s *server.MCPServer, authConfig *authorization.AuthorizationConfig, cache authorization.NotebookCache, quickNoteConfig authorization.QuickNoteConfig) {
	// testProgress: A tool that emits progress messages for testing progress notifications
	testProgressTool := mcp.NewTool(
		"testProgress",
		mcp.WithDescription("A tool that emits progress messages to test progress notification functionality. Sends progress updates from 0 to 10 over 10 seconds."),
	)
	testProgressHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logging.ToolsLogger.Info("Starting test progress tool", "operation", "testProgress", "type", "tool_invocation")

		mcpServer := server.ServerFromContext(ctx)
		if mcpServer == nil {
			logging.ToolsLogger.Error("testProgress: No server context available")
			return mcp.NewToolResultError("Server context not available"), nil
		}

		// Get progress token from meta if available
		var progressToken mcp.ProgressToken
		if req.Params.Meta != nil && req.Params.Meta.ProgressToken != nil {
			progressToken = req.Params.Meta.ProgressToken
		} else {
			// Use a default token if none provided
			progressToken = "test-progress"
		}

		logging.ToolsLogger.Debug("testProgress: Using progress token", "token", progressToken)

		// Send initial progress notification
		total := float64(10)
		message := "Starting progress test"
		err := mcpServer.SendNotificationToClient(ctx, "notifications/progress", map[string]any{
			"progressToken": progressToken,
			"progress":      0.0,
			"total":         total,
			"message":       message,
		})
		if err != nil {
			logging.ToolsLogger.Error("testProgress: Failed to send initial progress", "error", err)
		}

		logging.ToolsLogger.Debug("testProgress: Sent initial progress notification", "message", message)

		// Send progress updates from 1 to 10
		for i := 1; i <= 10; i++ {
			time.Sleep(1 * time.Second)
			progress := float64(i)
			progressMsg := fmt.Sprintf("Progress: %d/10", i)

			err := mcpServer.SendNotificationToClient(ctx, "notifications/progress", map[string]any{
				"progressToken": progressToken,
				"progress":      progress,
				"total":         total,
				"message":       progressMsg,
			})
			if err != nil {
				logging.ToolsLogger.Error("testProgress: Failed to send progress update", "error", err, "step", i)
			}

			logging.ToolsLogger.Debug("testProgress: Sent progress update", "step", i, "message", progressMsg)
		}

		logging.ToolsLogger.Info("testProgress operation completed successfully")
		// Return final result
		return mcp.NewToolResultText("Test progress completed successfully"), nil
	}
	s.AddTool(testProgressTool, server.ToolHandlerFunc(authorization.AuthorizedToolHandler("testProgress", testProgressHandler, authConfig, cache, quickNoteConfig)))

	logging.ToolsLogger.Debug("Test tools registered successfully")
}
