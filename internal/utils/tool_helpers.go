// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

// tool_helpers.go - Common utilities for MCP tool handlers to reduce code duplication.
//
// This module provides centralized functions for common MCP tool handler patterns:
// - Error result creation with consistent formatting
// - JSON marshaling with standardized error handling
// - Tool operation logging with timing and context
// - HTTP request/response handling with automatic cleanup
//
// These utilities eliminate repeated boilerplate code across tool handlers and ensure
// consistent error handling, logging, and response formatting throughout the codebase.

package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/gebl/onenote-mcp-server/internal/logging"
)

// ToolResult provides helper functions for creating consistent MCP tool results
type ToolResult struct{}

// NewError creates a standardized error result with consistent formatting
func (tr ToolResult) NewError(operation string, err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Failed to %s: %v", operation, err)},
		},
	}
}

// NewErrorf creates a standardized error result with formatted message
func (tr ToolResult) NewErrorf(operation string, format string, args ...interface{}) *mcp.CallToolResult {
	message := fmt.Sprintf(format, args...)
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Failed to %s: %s", operation, message)},
		},
	}
}

// NewJSONResult marshals data to JSON and returns a text result, or error result if marshaling fails
func (tr ToolResult) NewJSONResult(operation string, data interface{}) *mcp.CallToolResult {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		logging.ToolsLogger.Error("Failed to marshal JSON response", "operation", operation, "error", err)
		return tr.NewError(fmt.Sprintf("marshal %s response", operation), err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(jsonBytes)},
		},
	}
}

// NewJSONResultWithFallback marshals data to JSON, with fallback text if marshaling fails
func (tr ToolResult) NewJSONResultWithFallback(operation string, data interface{}, fallbackText string) *mcp.CallToolResult {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		logging.ToolsLogger.Warn("Failed to marshal JSON response, using fallback",
			"operation", operation, "error", err, "fallback", fallbackText)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fallbackText},
			},
		}
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(jsonBytes)},
		},
	}
}

// Global instance for easy access
var ToolResults = ToolResult{}

// ToolLogger provides standardized logging for tool operations
type ToolLogger struct {
	operation string
	startTime time.Time
}

// NewToolLogger creates a new tool logger for an operation
func NewToolLogger(operation string) *ToolLogger {
	startTime := time.Now()
	logging.ToolsLogger.Info("Starting tool operation", "operation", operation, "type", "tool_invocation")
	return &ToolLogger{
		operation: operation,
		startTime: startTime,
	}
}

// LogError logs an error with operation context
func (tl *ToolLogger) LogError(err error, extraFields ...interface{}) {
	fields := []interface{}{"operation", tl.operation, "error", err}
	fields = append(fields, extraFields...)
	logging.ToolsLogger.Error("Tool operation failed", fields...)
}

// LogDebug logs debug information with operation context
func (tl *ToolLogger) LogDebug(message string, extraFields ...interface{}) {
	fields := []interface{}{"operation", tl.operation}
	fields = append(fields, extraFields...)
	logging.ToolsLogger.Debug(message, fields...)
}

// LogSuccess logs successful completion with duration
func (tl *ToolLogger) LogSuccess(extraFields ...interface{}) {
	elapsed := time.Since(tl.startTime)
	fields := []interface{}{"operation", tl.operation, "duration", elapsed}
	fields = append(fields, extraFields...)
	logging.ToolsLogger.Debug("Tool operation completed successfully", fields...)
}

// HTTPResponse wraps an HTTP response with automatic cleanup and error handling
type HTTPResponse struct {
	*http.Response
	cleaned bool
}

// NewHTTPResponse creates a new HTTPResponse wrapper that ensures proper cleanup
func NewHTTPResponse(resp *http.Response) *HTTPResponse {
	return &HTTPResponse{Response: resp, cleaned: false}
}

// Close closes the response body if not already closed
func (hr *HTTPResponse) Close() error {
	if !hr.cleaned && hr.Response != nil && hr.Response.Body != nil {
		hr.cleaned = true
		return hr.Response.Body.Close()
	}
	return nil
}

// ReadAll reads the entire response body and automatically closes it
func (hr *HTTPResponse) ReadAll() ([]byte, error) {
	if hr.Response == nil || hr.Response.Body == nil {
		return nil, fmt.Errorf("no response body available")
	}
	
	defer hr.Close()
	return io.ReadAll(hr.Response.Body)
}

// HTTPClient provides utilities for making HTTP requests with automatic cleanup
type HTTPClient struct {
	client interface {
		MakeAuthenticatedRequest(method, url string, body io.Reader, headers map[string]string) (*http.Response, error)
		HandleHTTPResponse(resp *http.Response, operation string) error
		ReadResponseBody(resp *http.Response, operation string) ([]byte, error)
	}
}

// NewHTTPClient creates a new HTTPClient wrapper
func NewHTTPClient(client interface {
	MakeAuthenticatedRequest(method, url string, body io.Reader, headers map[string]string) (*http.Response, error)
	HandleHTTPResponse(resp *http.Response, operation string) error
	ReadResponseBody(resp *http.Response, operation string) ([]byte, error)
}) *HTTPClient {
	return &HTTPClient{client: client}
}

// MakeRequest makes an authenticated HTTP request with automatic cleanup and error handling
func (hc *HTTPClient) MakeRequest(method, url string, body io.Reader, headers map[string]string, operation string) (*HTTPResponse, error) {
	resp, err := hc.client.MakeAuthenticatedRequest(method, url, body, headers)
	if err != nil {
		return nil, err
	}
	
	httpResp := NewHTTPResponse(resp)
	
	// Handle HTTP response errors
	if err := hc.client.HandleHTTPResponse(resp, operation); err != nil {
		httpResp.Close() // Ensure cleanup on error
		return nil, err
	}
	
	return httpResp, nil
}

// MakeRequestAndReadBody makes a request and reads the response body in one operation
func (hc *HTTPClient) MakeRequestAndReadBody(method, url string, body io.Reader, headers map[string]string, operation string) ([]byte, error) {
	httpResp, err := hc.MakeRequest(method, url, body, headers, operation)
	if err != nil {
		return nil, err
	}
	
	return httpResp.ReadAll()
}

// ToolParameterExtractor provides utilities for extracting and validating tool parameters
type ToolParameterExtractor struct {
	req    *mcp.CallToolRequest
	logger *ToolLogger
}

// NewParameterExtractor creates a new parameter extractor for a tool request
func NewParameterExtractor(req *mcp.CallToolRequest, logger *ToolLogger) *ToolParameterExtractor {
	return &ToolParameterExtractor{req: req, logger: logger}
}

// RequireString extracts a required string parameter with validation
func (tpe *ToolParameterExtractor) RequireString(paramName string) (string, error) {
	// In the new SDK, arguments are in req.Params.Arguments as json.RawMessage
	if len(tpe.req.Params.Arguments) == 0 {
		tpe.logger.LogError(fmt.Errorf("no arguments provided"))
		return "", fmt.Errorf("no arguments provided")
	}

	// Parse the raw JSON arguments
	var args map[string]interface{}
	if err := json.Unmarshal(tpe.req.Params.Arguments, &args); err != nil {
		tpe.logger.LogError(fmt.Errorf("failed to parse arguments: %v", err))
		return "", fmt.Errorf("invalid arguments format")
	}

	value, exists := args[paramName]
	if !exists {
		tpe.logger.LogError(fmt.Errorf("missing required parameter: %s", paramName))
		return "", fmt.Errorf("%s is required", paramName)
	}

	strValue, ok := value.(string)
	if !ok {
		tpe.logger.LogError(fmt.Errorf("parameter %s is not a string", paramName))
		return "", fmt.Errorf("%s must be a string", paramName)
	}

	if strValue == "" {
		tpe.logger.LogError(fmt.Errorf("empty required parameter: %s", paramName))
		return "", fmt.Errorf("%s cannot be empty", paramName)
	}

	tpe.logger.LogDebug("Extracted parameter", paramName, strValue)
	return strValue, nil
}

// ProgressHandler provides utilities for handling progress notifications in tools
type ProgressHandler struct {
	progressToken string
	logger        *ToolLogger
}

// NewProgressHandler creates a new progress handler
func NewProgressHandler(req *mcp.CallToolRequest, logger *ToolLogger) *ProgressHandler {
	progressToken := ExtractProgressToken(req)
	return &ProgressHandler{progressToken: progressToken, logger: logger}
}

// HasProgressToken returns true if a progress token is available
func (ph *ProgressHandler) HasProgressToken() bool {
	return ph.progressToken != ""
}

// SendProgress sends a progress notification if a token is available
func (ph *ProgressHandler) SendProgress(ctx context.Context, s interface{}, progress int, total int, message string) {
	if ph.progressToken != "" && s != nil {
		if mcpServer, ok := s.(*mcp.Server); ok {
			SendProgressNotification(mcpServer, ctx, ph.progressToken, progress, total, message)
			ph.logger.LogDebug("Progress notification sent", "progress", progress, "total", total, "message", message)
		}
	}
}

// SendProgressMessage sends a simple progress message if a token is available
func (ph *ProgressHandler) SendProgressMessage(ctx context.Context, s interface{}, message string) {
	if ph.progressToken != "" && s != nil {
		if mcpServer, ok := s.(*mcp.Server); ok {
			SendProgressMessage(mcpServer, ctx, ph.progressToken, message)
			ph.logger.LogDebug("Progress message sent", "message", message)
		}
	}
}