// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

// utils.go - Utility functions for Microsoft Graph API client.
//
// This file contains utility functions for ID sanitization, filename generation,
// and other common operations used across the Graph API client.
//
// Key Features:
// - OneNote ID validation and sanitization
// - Filename generation based on content type
// - Security-focused input validation
//
// Usage Example:
//   sanitizedID, err := sanitizeOneNoteID(id, "sectionID")
//   if err != nil {
//       return nil, err
//   }
//
//   filename := generateFilename(pageItemID, contentType)

package graph

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gebl/onenote-mcp-server/internal/logging"
)

// SanitizeOneNoteID validates and sanitizes OneNote IDs to prevent injection attacks.
// OneNote IDs typically contain alphanumeric characters, hyphens, and exclamation marks.
// Example format: "0-4D24C77F19546939!40109"
func (c *Client) SanitizeOneNoteID(id, idType string) (string, error) {
	if id == "" {
		logging.GraphLogger.Debug("Empty ID provided", "id_type", idType)
		return "", fmt.Errorf("%s cannot be empty", idType)
	}

	// Remove whitespace
	sanitizedID := strings.TrimSpace(id)

	// Validate that the ID contains only allowed characters
	allowedChars := "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-!"
	for _, char := range sanitizedID {
		if !strings.ContainsRune(allowedChars, char) {
			logging.GraphLogger.Debug("Invalid character in ID", "id_type", idType, "char", string(char))
			return "", fmt.Errorf("%s contains invalid characters", idType)
		}
	}

	// Additional validation: ensure it's not too long
	if len(sanitizedID) > 100 {
		logging.GraphLogger.Debug("ID too long", "id_type", idType, "length", len(sanitizedID))
		return "", fmt.Errorf("%s is too long", idType)
	}

	logging.GraphLogger.Debug("Sanitized ID", "id_type", idType, "sanitized_id", sanitizedID)
	return sanitizedID, nil
}

// GenerateFilename creates a filename based on page item ID and content type.
func (c *Client) GenerateFilename(pageItemID, contentType string) string {
	var filename string
	if strings.HasPrefix(contentType, "image/") {
		// Map content types to file extensions
		switch contentType {
		case "image/jpeg":
			filename = pageItemID + ".jpg"
		case "image/png":
			filename = pageItemID + ".png"
		case "image/gif":
			filename = pageItemID + ".gif"
		case "image/webp":
			filename = pageItemID + ".webp"
		default:
			filename = pageItemID + ".bin"
		}
	} else {
		filename = pageItemID + ".bin"
	}

	logging.GraphLogger.Debug("Generated filename", "filename", filename)
	return filename
}

// GetOnenoteOperation retrieves the status of an asynchronous OneNote operation.
// operationID: ID of the operation to check.
// Returns the operation status and metadata, and an error, if any.
func (c *Client) GetOnenoteOperation(operationID string) (map[string]interface{}, error) {
	logging.GraphLogger.Info("Getting OneNote operation status", "operation_id", operationID)

	// Validate and sanitize the operation ID
	sanitizedOperationID, err := c.SanitizeOneNoteID(operationID, "operationID")
	if err != nil {
		return nil, err
	}

	// Construct the URL for getting operation status
	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/me/onenote/operations/%s", sanitizedOperationID)
	logging.GraphLogger.Debug("Operation status URL", "url", url)

	// Make authenticated request
	resp, err := c.MakeAuthenticatedRequest("GET", url, nil, nil)
	if err != nil {
		logging.GraphLogger.Debug("Authenticated request failed", "error", err)
		return nil, err
	}
	defer resp.Body.Close()

	logging.GraphLogger.Debug("Received response", "status", resp.StatusCode, "headers", resp.Header)

	// Handle HTTP response
	if errHandle := c.HandleHTTPResponse(resp, "GetOnenoteOperation"); errHandle != nil {
		logging.GraphLogger.Debug("HTTP response handling failed", "error", errHandle)
		return nil, errHandle
	}

	// Read response body
	content, err := c.ReadResponseBody(resp, "GetOnenoteOperation")
	if err != nil {
		logging.GraphLogger.Debug("Failed to read response body", "error", err)
		return nil, err
	}

	logging.GraphLogger.Debug("Response body", "body", string(content))

	// Parse the response JSON
	var result map[string]interface{}
	if err := json.Unmarshal(content, &result); err != nil {
		logging.GraphLogger.Debug("Failed to unmarshal response", "error", err)
		return nil, fmt.Errorf("failed to parse operation status response: %v", err)
	}

	logging.GraphLogger.Info("Successfully retrieved operation status", "operation_id", operationID)
	return result, nil
}
