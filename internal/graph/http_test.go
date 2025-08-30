// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package graph

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gebl/onenote-mcp-server/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_MakeAuthenticatedRequest(t *testing.T) {
	tests := []struct {
		name          string
		accessToken   string
		tokenManager  *auth.TokenManager
		method        string
		url           string
		body          io.Reader
		headers       map[string]string
		expectedError string
	}{
		{
			name:          "empty access token",
			accessToken:   "",
			tokenManager:  nil,
			method:        "GET",
			url:           "https://graph.microsoft.com/v1.0/me",
			body:          nil,
			headers:       nil,
			expectedError: "authentication required: tokens have been cleared",
		},
		{
			name:        "cleared token manager",
			accessToken: "valid-token",
			tokenManager: &auth.TokenManager{
				AccessToken:  "",
				RefreshToken: "",
				Expiry:       0,
			},
			method:        "GET",
			url:           "https://graph.microsoft.com/v1.0/me",
			body:          nil,
			headers:       nil,
			expectedError: "authentication required: tokens have been cleared",
		},
		{
			name:        "invalid method",
			accessToken: "valid-token",
			tokenManager: &auth.TokenManager{
				AccessToken: "valid-token",
				Expiry:      time.Now().Add(time.Hour).Unix(),
			},
			method:        "INVALID\nMETHOD",
			url:           "https://graph.microsoft.com/v1.0/me",
			body:          nil,
			headers:       nil,
			expectedError: "invalid method",
		},
		{
			name:        "invalid url",
			accessToken: "valid-token",
			tokenManager: &auth.TokenManager{
				AccessToken: "valid-token",
				Expiry:      time.Now().Add(time.Hour).Unix(),
			},
			method:        "GET",
			url:           "://invalid-url",
			body:          nil,
			headers:       nil,
			expectedError: "missing protocol scheme",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				AccessToken:  tt.accessToken,
				TokenManager: tt.tokenManager,
			}

			resp, err := client.MakeAuthenticatedRequest(tt.method, tt.url, tt.body, tt.headers)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, resp)
			} else {
				// Note: This will fail in unit tests without proper HTTP mocking
				// In a real scenario, we'd mock the auth.MakeAuthenticatedRequestWithCallback
				assert.Error(t, err) // Expected to fail without proper HTTP setup
			}
		})
	}
}

func TestClient_HandleHTTPResponse(t *testing.T) {
	client := NewClient("test-token")

	tests := []struct {
		name         string
		statusCode   int
		body         string
		operation    string
		expectError  bool
		errorContains string
	}{
		{
			name:        "success 200",
			statusCode:  200,
			body:        `{"success": true}`,
			operation:   "test_operation",
			expectError: false,
		},
		{
			name:        "success 201",
			statusCode:  201,
			body:        `{"created": true}`,
			operation:   "create_operation",
			expectError: false,
		},
		{
			name:        "success 204",
			statusCode:  204,
			body:        "",
			operation:   "delete_operation",
			expectError: false,
		},
		{
			name:          "error 400",
			statusCode:    400,
			body:          `{"error": "Bad Request"}`,
			operation:     "invalid_operation",
			expectError:   true,
			errorContains: "HTTP 400",
		},
		{
			name:          "error 401",
			statusCode:    401,
			body:          `{"error": "Unauthorized"}`,
			operation:     "unauthorized_operation",
			expectError:   true,
			errorContains: "HTTP 401",
		},
		{
			name:          "error 404",
			statusCode:    404,
			body:          `{"error": "Not Found"}`,
			operation:     "missing_resource",
			expectError:   true,
			errorContains: "HTTP 404",
		},
		{
			name:          "error 500",
			statusCode:    500,
			body:          `{"error": "Internal Server Error"}`,
			operation:     "server_error",
			expectError:   true,
			errorContains: "HTTP 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock response
			resp := &http.Response{
				StatusCode: tt.statusCode,
				Body:       io.NopCloser(strings.NewReader(tt.body)),
			}

			err := client.HandleHTTPResponse(resp, tt.operation)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
				assert.Contains(t, err.Error(), tt.operation)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClient_ReadResponseBody(t *testing.T) {
	client := NewClient("test-token")

	tests := []struct {
		name        string
		body        string
		operation   string
		expectError bool
	}{
		{
			name:        "valid json body",
			body:        `{"message": "success", "data": [1, 2, 3]}`,
			operation:   "json_operation",
			expectError: false,
		},
		{
			name:        "empty body",
			body:        "",
			operation:   "empty_operation",
			expectError: false,
		},
		{
			name:        "large body",
			body:        strings.Repeat("test data ", 1000),
			operation:   "large_operation",
			expectError: false,
		},
		{
			name:        "binary data",
			body:        "\x00\x01\x02\x03\xFF",
			operation:   "binary_operation",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				Body: io.NopCloser(strings.NewReader(tt.body)),
			}

			content, err := client.ReadResponseBody(resp, tt.operation)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.body, string(content))
			}
		})
	}
}

func TestClient_ReadResponseBody_ErrorReading(t *testing.T) {
	client := NewClient("test-token")

	// Create a mock reader that always returns an error
	errorReader := &errorReader{err: io.ErrUnexpectedEOF}
	resp := &http.Response{
		Body: io.NopCloser(errorReader),
	}

	content, err := client.ReadResponseBody(resp, "error_operation")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read error_operation response body")
	assert.Contains(t, err.Error(), "unexpected EOF")
	assert.Nil(t, content)
}

func TestClient_GetContentTypeFromResponse(t *testing.T) {
	client := NewClient("test-token")

	tests := []struct {
		name               string
		contentTypeHeader  string
		expectedContentType string
	}{
		{
			name:               "application/json",
			contentTypeHeader:  "application/json",
			expectedContentType: "application/json",
		},
		{
			name:               "application/json with charset",
			contentTypeHeader:  "application/json; charset=utf-8",
			expectedContentType: "application/json; charset=utf-8",
		},
		{
			name:               "image/png",
			contentTypeHeader:  "image/png",
			expectedContentType: "image/png",
		},
		{
			name:               "text/html",
			contentTypeHeader:  "text/html; charset=utf-8",
			expectedContentType: "text/html; charset=utf-8",
		},
		{
			name:               "no content type header",
			contentTypeHeader:  "",
			expectedContentType: "application/octet-stream",
		},
		{
			name:               "empty content type header",
			contentTypeHeader:  "",
			expectedContentType: "application/octet-stream",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create response with headers
			resp := &http.Response{
				Header: make(http.Header),
			}

			if tt.contentTypeHeader != "" {
				resp.Header.Set("Content-Type", tt.contentTypeHeader)
			}

			contentType := client.GetContentTypeFromResponse(resp)

			assert.Equal(t, tt.expectedContentType, contentType)
		})
	}
}

func TestClient_HTTPIntegration(t *testing.T) {
	// Integration test combining multiple HTTP methods
	client := NewClient("test-token")

	tests := []struct {
		name        string
		statusCode  int
		body        string
		contentType string
	}{
		{
			name:        "json response",
			statusCode:  200,
			body:        `{"test": "data"}`,
			contentType: "application/json",
		},
		{
			name:        "created response",
			statusCode:  201,
			body:        `{"id": "123", "created": true}`,
			contentType: "application/json; charset=utf-8",
		},
		{
			name:        "no content response",
			statusCode:  204,
			body:        "",
			contentType: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create response with headers and body
			resp := &http.Response{
				StatusCode: tt.statusCode,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(tt.body)),
			}

			if tt.contentType != "" {
				resp.Header.Set("Content-Type", tt.contentType)
			}

			// Test HandleHTTPResponse
			err := client.HandleHTTPResponse(resp, "integration_test")
			assert.NoError(t, err)

			// Reset body for reading
			resp.Body = io.NopCloser(strings.NewReader(tt.body))

			// Test ReadResponseBody
			content, err := client.ReadResponseBody(resp, "integration_test")
			require.NoError(t, err)
			assert.Equal(t, tt.body, string(content))

			// Test GetContentTypeFromResponse
			actualContentType := client.GetContentTypeFromResponse(resp)
			if tt.contentType != "" {
				assert.Equal(t, tt.contentType, actualContentType)
			} else {
				assert.Equal(t, "application/octet-stream", actualContentType)
			}
		})
	}
}

// Helper struct for testing error scenarios
type errorReader struct {
	err error
}

func (r *errorReader) Read(p []byte) (n int, err error) {
	return 0, r.err
}