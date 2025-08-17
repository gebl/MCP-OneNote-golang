// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

// http.go - HTTP utilities for Microsoft Graph API client.
//
// This file contains HTTP request and response handling utilities used across
// the Graph API client for making authenticated requests and processing responses.
//
// Key Features:
// - Authenticated HTTP request creation and execution
// - Response body reading and error handling
// - Content type extraction from responses
// - Token refresh integration for failed requests
//
// Usage Example:
//   resp, err := c.makeAuthenticatedRequest("GET", url, nil, nil)
//   if err != nil {
//       return nil, err
//   }
//   defer resp.Body.Close()
//
//   content, err := readResponseBody(resp, "operation")
//   if err != nil {
//       return nil, err
//   }

package graph

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gebl/onenote-mcp-server/internal/auth"
	"github.com/gebl/onenote-mcp-server/internal/logging"
)

// MakeAuthenticatedRequest creates and executes an authenticated HTTP request with token refresh support.
func (c *Client) MakeAuthenticatedRequest(method, url string, body io.Reader, headers map[string]string) (*http.Response, error) {
	logging.GraphLogger.Debug("Creating HTTP request", "method", method, "url", url)

	// Check if authentication has been cleared
	if c.AccessToken == "" || (c.TokenManager != nil && c.TokenManager.AccessToken == "" && c.TokenManager.RefreshToken == "") {
		logging.GraphLogger.Debug("No valid authentication tokens available")
		return nil, fmt.Errorf("authentication required: tokens have been cleared, use initiateAuth to re-authenticate")
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		logging.GraphLogger.Debug("Failed to create HTTP request", "error", err)
		return nil, err
	}

	// Set custom headers if provided
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	logging.GraphLogger.Debug("HTTP request created successfully")

	// Use the authenticated request method that handles token refresh
	logging.GraphLogger.Debug("Making authenticated request to Graph API")
	resp, err := auth.MakeAuthenticatedRequestWithCallback(req, c.AccessToken, c.OAuthConfig, c.TokenManager, c.TokenPath, c.UpdateToken)
	if err != nil {
		logging.GraphLogger.Debug("Authenticated request failed", "error", err)
		return nil, err
	}

	logging.GraphLogger.Debug("Response received", "status", resp.StatusCode, "headers", resp.Header)
	return resp, nil
}

// HandleHTTPResponse handles common HTTP response processing and error checking.
func (c *Client) HandleHTTPResponse(resp *http.Response, operation string) error {
	if resp.StatusCode != 200 && resp.StatusCode != 201 && resp.StatusCode != 204 {
		body, _ := io.ReadAll(resp.Body)
		logging.GraphLogger.Debug("Error response body", "body", string(body))
		return fmt.Errorf("%s failed: HTTP %d - %s", operation, resp.StatusCode, string(body))
	}
	return nil
}

// ReadResponseBody reads the entire response body and returns it as bytes.
func (c *Client) ReadResponseBody(resp *http.Response, operation string) ([]byte, error) {
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		logging.GraphLogger.Debug("Failed to read response body", "error", err)
		return nil, fmt.Errorf("failed to read %s response body: %v", operation, err)
	}
	logging.GraphLogger.Debug("Successfully read response body", "bytes", len(content))
	return content, nil
}

// GetContentTypeFromResponse extracts content type from HTTP response headers with fallback.
func (c *Client) GetContentTypeFromResponse(resp *http.Response) string {
	contentType := "application/octet-stream" // Default content type

	if resp.Header.Get("Content-Type") != "" {
		contentType = resp.Header.Get("Content-Type")
		logging.GraphLogger.Debug("Content type from HTTP headers", "content_type", contentType)
	} else {
		logging.GraphLogger.Debug("No Content-Type header found, using default", "content_type", contentType)
	}

	return contentType
}
