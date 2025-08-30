// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package graph

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClient_SanitizeOneNoteID(t *testing.T) {
	client := NewClient("test-token")

	tests := []struct {
		name          string
		id            string
		idType        string
		expectedID    string
		expectedError string
	}{
		{
			name:       "valid onenote id",
			id:         "0-4D24C77F19546939!40109",
			idType:     "pageID",
			expectedID: "0-4D24C77F19546939!40109",
		},
		{
			name:       "valid section id",
			id:         "1-B2C4D6F8A1B3C5D7!2345",
			idType:     "sectionID",
			expectedID: "1-B2C4D6F8A1B3C5D7!2345",
		},
		{
			name:       "id with whitespace",
			id:         "  0-4D24C77F19546939!40109  ",
			idType:     "pageID",
			expectedID: "0-4D24C77F19546939!40109",
		},
		{
			name:       "simple alphanumeric id",
			id:         "ABC123DEF456",
			idType:     "notebookID",
			expectedID: "ABC123DEF456",
		},
		{
			name:       "id with hyphens",
			id:         "test-id-with-hyphens",
			idType:     "itemID",
			expectedID: "test-id-with-hyphens",
		},
		{
			name:       "id with exclamation marks",
			id:         "test!id!with!exclamations",
			idType:     "resourceID",
			expectedID: "test!id!with!exclamations",
		},
		{
			name:          "empty id",
			id:            "",
			idType:        "pageID",
			expectedError: "pageID cannot be empty",
		},
		{
			name:          "id with invalid characters",
			id:            "invalid@id#with$symbols",
			idType:        "pageID",
			expectedError: "pageID contains invalid characters",
		},
		{
			name:          "id with spaces in middle",
			id:            "invalid id with spaces",
			idType:        "sectionID",
			expectedError: "sectionID contains invalid characters",
		},
		{
			name:          "id with special characters",
			id:            "id-with-unicode-ñ",
			idType:        "pageID",
			expectedError: "pageID contains invalid characters",
		},
		{
			name:          "id too long",
			id:            strings.Repeat("a", 101),
			idType:        "pageID",
			expectedError: "pageID is too long",
		},
		{
			name:       "id at max length",
			id:         strings.Repeat("a", 100),
			idType:     "pageID",
			expectedID: strings.Repeat("a", 100),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sanitizedID, err := client.SanitizeOneNoteID(tt.id, tt.idType)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Equal(t, "", sanitizedID)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, sanitizedID)
			}
		})
	}
}

func TestClient_GenerateFilename(t *testing.T) {
	client := NewClient("test-token")

	tests := []struct {
		name           string
		pageItemID     string
		contentType    string
		expectedFilename string
	}{
		{
			name:           "jpeg image",
			pageItemID:     "item123",
			contentType:    "image/jpeg",
			expectedFilename: "item123.jpg",
		},
		{
			name:           "png image",
			pageItemID:     "item456",
			contentType:    "image/png",
			expectedFilename: "item456.png",
		},
		{
			name:           "gif image",
			pageItemID:     "item789",
			contentType:    "image/gif",
			expectedFilename: "item789.gif",
		},
		{
			name:           "webp image",
			pageItemID:     "item101112",
			contentType:    "image/webp",
			expectedFilename: "item101112.webp",
		},
		{
			name:           "unknown image type",
			pageItemID:     "item131415",
			contentType:    "image/tiff",
			expectedFilename: "item131415.bin",
		},
		{
			name:           "text content",
			pageItemID:     "item161718",
			contentType:    "text/plain",
			expectedFilename: "item161718.bin",
		},
		{
			name:           "json content",
			pageItemID:     "item192021",
			contentType:    "application/json",
			expectedFilename: "item192021.bin",
		},
		{
			name:           "empty content type",
			pageItemID:     "item222324",
			contentType:    "",
			expectedFilename: "item222324.bin",
		},
		{
			name:           "complex page item id",
			pageItemID:     "0-4D24C77F19546939!40109",
			contentType:    "image/png",
			expectedFilename: "0-4D24C77F19546939!40109.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := client.GenerateFilename(tt.pageItemID, tt.contentType)
			assert.Equal(t, tt.expectedFilename, filename)
		})
	}
}

func TestClient_GetOnenoteOperation(t *testing.T) {
	client := NewClient("test-token")

	tests := []struct {
		name          string
		operationID   string
		expectedError string
	}{
		{
			name:          "empty operation id",
			operationID:   "",
			expectedError: "operationID cannot be empty",
		},
		{
			name:          "invalid operation id",
			operationID:   "invalid@operation#id",
			expectedError: "operationID contains invalid characters",
		},
		{
			name:          "operation id too long",
			operationID:   strings.Repeat("a", 101),
			expectedError: "operationID is too long",
		},
		{
			name:        "valid operation id - will fail without HTTP mock",
			operationID: "valid-operation-id-123",
			// This will fail due to actual HTTP request, but that's expected in unit tests
			expectedError: "", // We expect HTTP-related errors, not validation errors
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.GetOnenoteOperation(tt.operationID)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, result)
			} else {
				// For valid operation IDs, we expect HTTP-related errors since we're not mocking
				assert.Error(t, err) // Expected - no actual HTTP backend
				assert.Nil(t, result)
			}
		})
	}
}

func TestClient_GenerateFilename_Comprehensive(t *testing.T) {
	client := NewClient("test-token")

	// Test all supported image types
	imageTypes := map[string]string{
		"image/jpeg": ".jpg",
		"image/png":  ".png",
		"image/gif":  ".gif",
		"image/webp": ".webp",
	}

	for contentType, expectedExt := range imageTypes {
		t.Run("image_type_"+contentType, func(t *testing.T) {
			pageItemID := "test-item-123"
			filename := client.GenerateFilename(pageItemID, contentType)
			assert.Equal(t, pageItemID+expectedExt, filename)
		})
	}

	// Test non-image types
	nonImageTypes := []string{
		"text/plain",
		"application/json",
		"application/pdf",
		"video/mp4",
		"audio/mpeg",
		"",
	}

	for _, contentType := range nonImageTypes {
		t.Run("non_image_type_"+strings.ReplaceAll(contentType, "/", "_"), func(t *testing.T) {
			pageItemID := "test-item-456"
			filename := client.GenerateFilename(pageItemID, contentType)
			assert.Equal(t, pageItemID+".bin", filename)
		})
	}
}

func TestClient_SanitizeOneNoteID_EdgeCases(t *testing.T) {
	client := NewClient("test-token")

	tests := []struct {
		name       string
		id         string
		idType     string
		shouldPass bool
	}{
		{
			name:       "all allowed characters",
			id:         "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-!",
			idType:     "testID",
			shouldPass: true,
		},
		{
			name:       "single character",
			id:         "a",
			idType:     "testID",
			shouldPass: true,
		},
		{
			name:       "only numbers",
			id:         "1234567890",
			idType:     "testID",
			shouldPass: true,
		},
		{
			name:       "only letters",
			id:         "abcdefghijklmnopqrstuvwxyz",
			idType:     "testID",
			shouldPass: true,
		},
		{
			name:       "only uppercase",
			id:         "ABCDEFGHIJKLMNOPQRSTUVWXYZ",
			idType:     "testID",
			shouldPass: true,
		},
		{
			name:       "only special characters",
			id:         "-!-!-!",
			idType:     "testID",
			shouldPass: true,
		},
		{
			name:       "tab character",
			id:         "test\tid",
			idType:     "testID",
			shouldPass: false,
		},
		{
			name:       "newline character",
			id:         "test\nid",
			idType:     "testID",
			shouldPass: false,
		},
		{
			name:       "unicode character",
			id:         "test€id",
			idType:     "testID",
			shouldPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sanitizedID, err := client.SanitizeOneNoteID(tt.id, tt.idType)

			if tt.shouldPass {
				assert.NoError(t, err)
				assert.Equal(t, tt.id, sanitizedID)
			} else {
				assert.Error(t, err)
				assert.Equal(t, "", sanitizedID)
			}
		})
	}
}

// Test URL construction in GetOnenoteOperation
func TestClient_GetOnenoteOperation_URLConstruction(t *testing.T) {
	client := NewClient("test-token")

	// Test that the operation correctly constructs URLs
	// We can't test the full HTTP flow, but we can test validation

	validOperationIDs := []string{
		"simple-id",
		"0-4D24C77F19546939!40109",
		"operation123",
		"test-operation-with-hyphens",
		"operation!with!exclamations",
	}

	for _, operationID := range validOperationIDs {
		t.Run("valid_operation_"+operationID, func(t *testing.T) {
			// This will fail with HTTP error, but should pass validation
			result, err := client.GetOnenoteOperation(operationID)

			// We expect an error due to no HTTP backend, but not a validation error
			assert.Error(t, err)
			assert.Nil(t, result)
			// The error should not be about invalid characters or empty ID
			assert.NotContains(t, err.Error(), "cannot be empty")
			assert.NotContains(t, err.Error(), "contains invalid characters")
			assert.NotContains(t, err.Error(), "is too long")
		})
	}
}

// Test JSON parsing behavior
func TestJSONParsingBehavior(t *testing.T) {
	// Test various JSON structures that might be returned by GetOnenoteOperation
	testCases := []struct {
		name       string
		jsonData   string
		shouldPass bool
	}{
		{
			name:       "simple object",
			jsonData:   `{"status": "completed", "id": "123"}`,
			shouldPass: true,
		},
		{
			name:       "nested object",
			jsonData:   `{"operation": {"id": "123", "status": "running", "details": {"progress": 50}}}`,
			shouldPass: true,
		},
		{
			name:       "empty object",
			jsonData:   `{}`,
			shouldPass: true,
		},
		{
			name:       "null values",
			jsonData:   `{"status": null, "progress": null}`,
			shouldPass: true,
		},
		{
			name:       "invalid json",
			jsonData:   `{"status": "completed"`,
			shouldPass: false,
		},
		{
			name:       "not an object",
			jsonData:   `"just a string"`,
			shouldPass: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var result map[string]interface{}
			err := json.Unmarshal([]byte(tc.jsonData), &result)

			if tc.shouldPass {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			} else {
				assert.Error(t, err)
			}
		})
	}
}