// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package utils

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/gebl/onenote-mcp-server/internal/logging"
)

// ProgressNotifier provides centralized progress notification functionality
// for MCP tools and internal operations
type ProgressNotifier struct {
	server *server.MCPServer
	ctx    context.Context
	token  string
}

// NewProgressNotifier creates a new progress notifier instance
func NewProgressNotifier(s *server.MCPServer, ctx context.Context, token string) *ProgressNotifier {
	return &ProgressNotifier{
		server: s,
		ctx:    ctx,
		token:  token,
	}
}

// ExtractProgressToken extracts the progress token from MCP request metadata
func ExtractProgressToken(req mcp.CallToolRequest) string {
	// Extract progress token from request metadata if available
	if req.Params.Meta != nil && req.Params.Meta.ProgressToken != nil {
		if token, ok := req.Params.Meta.ProgressToken.(string); ok {
			return token
		}
		// Handle case where ProgressToken is not a string (e.g., mcp.ProgressToken type)
		return fmt.Sprintf("%v", req.Params.Meta.ProgressToken)
	}
	return ""
}

// SendNotification sends a progress notification with percentage calculation
func (pn *ProgressNotifier) SendNotification(progress, total int, message string) {
	SendProgressNotification(pn.server, pn.ctx, pn.token, progress, total, message)
}

// SendMessage sends a simple progress message without percentage
func (pn *ProgressNotifier) SendMessage(message string) {
	SendProgressMessage(pn.server, pn.ctx, pn.token, message)
}

// IsValid returns whether this notifier has the required components to send notifications
func (pn *ProgressNotifier) IsValid() bool {
	return pn.server != nil && pn.token != ""
}

// SendProgressNotification sends a progress notification with percentage-based progress
func SendProgressNotification(s *server.MCPServer, ctx context.Context, progressToken string, progress int, total int, message string) {
	if progressToken == "" {
		logging.UtilsLogger.Debug("Skipping progress notification - no progress token provided",
			"progress", progress,
			"total", total,
			"message", message)
		return
	}

	// Calculate percentage for enhanced logging
	percentage := float64(progress) / float64(total) * 100

	logging.UtilsLogger.Debug("Preparing to send progress notification to client",
		"progressToken", progressToken,
		"progress", progress,
		"total", total,
		"percentage", fmt.Sprintf("%.1f%%", percentage),
		"message", message,
		"has_server", s != nil)

	if s == nil {
		logging.UtilsLogger.Warn("Cannot send progress notification - MCP server is nil",
			"progressToken", progressToken,
			"progress", progress,
			"message", message)
		return
	}

	err := s.SendNotificationToClient(ctx, "notifications/progress", map[string]any{
		"progressToken": progressToken,
		"progress":      progress,
		"total":         total,
		"message":       message,
	})

	if err != nil {
		logging.UtilsLogger.Warn("Failed to send progress notification to client",
			"error", err,
			"progressToken", progressToken,
			"progress", progress,
			"total", total,
			"percentage", fmt.Sprintf("%.1f%%", percentage),
			"message", message)
	} else {
		logging.UtilsLogger.Debug("Successfully sent progress notification to client",
			"progressToken", progressToken,
			"progress", progress,
			"total", total,
			"percentage", fmt.Sprintf("%.1f%%", percentage),
			"message", message)
	}
}

// SendProgressMessage sends a simple progress message without percentage
func SendProgressMessage(s *server.MCPServer, ctx context.Context, progressToken string, message string) {
	if progressToken == "" {
		logging.UtilsLogger.Debug("Skipping progress message - no progress token provided",
			"message", message)
		return
	}

	logging.UtilsLogger.Debug("Preparing to send progress message to client",
		"progressToken", progressToken,
		"message", message,
		"has_server", s != nil)

	if s == nil {
		logging.UtilsLogger.Warn("Cannot send progress message - MCP server is nil",
			"progressToken", progressToken,
			"message", message)
		return
	}

	err := s.SendNotificationToClient(ctx, "notifications/progress", map[string]any{
		"progressToken": progressToken,
		"message":       message,
	})

	if err != nil {
		logging.UtilsLogger.Warn("Failed to send progress message to client",
			"error", err,
			"progressToken", progressToken,
			"message", message)
	} else {
		logging.UtilsLogger.Debug("Successfully sent progress message to client",
			"progressToken", progressToken,
			"message", message)
	}
}

// Context keys for progress notification system
type contextKey string

const (
	MCPServerKey     contextKey = "mcpServer"
	ProgressTokenKey contextKey = "progressToken"
)

// ExtractFromContext extracts MCP server and progress token from context
// This supports the pattern used by internal clients like SectionClient
func ExtractFromContext(ctx context.Context) (*server.MCPServer, string) {
	var mcpServer *server.MCPServer
	var progressToken string

	if serverVal := ctx.Value(MCPServerKey); serverVal != nil {
		if s, ok := serverVal.(*server.MCPServer); ok {
			mcpServer = s
		}
	}

	if tokenVal := ctx.Value(ProgressTokenKey); tokenVal != nil {
		if token, ok := tokenVal.(string); ok {
			progressToken = token
		}
	}

	return mcpServer, progressToken
}

// SendContextualMessage sends a progress message using context values
// This is used by internal clients that store progress info in context
func SendContextualMessage(ctx context.Context, message string, logger interface{}) {
	mcpServer, progressToken := ExtractFromContext(ctx)
	
	if mcpServer != nil && progressToken != "" {
		SendProgressMessage(mcpServer, ctx, progressToken, message)
	} else {
		// Log the attempt for debugging
		if logger != nil {
			logging.UtilsLogger.Debug("Progress notification context incomplete",
				"message", message,
				"has_server", mcpServer != nil,
				"has_token", progressToken != "",
				"progressToken", progressToken)
		}
	}
}