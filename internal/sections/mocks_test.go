// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package sections

import (
	"io"
	"net/http"

	"github.com/stretchr/testify/mock"

	"github.com/gebl/onenote-mcp-server/internal/graph"
)

// MockGraphClient is a mock implementation of the graph.Client for testing
type MockGraphClient struct {
	mock.Mock
	Client *graph.Client
}

// SanitizeOneNoteID mocks the SanitizeOneNoteID method
func (m *MockGraphClient) SanitizeOneNoteID(id string, paramName string) (string, error) {
	args := m.Called(id, paramName)
	return args.String(0), args.Error(1)
}

// MakeAuthenticatedRequest mocks the MakeAuthenticatedRequest method
func (m *MockGraphClient) MakeAuthenticatedRequest(method, url string, body io.Reader, headers map[string]string) (*http.Response, error) {
	args := m.Called(method, url, body, headers)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*http.Response), args.Error(1)
}

// HandleHTTPResponse mocks the HandleHTTPResponse method
func (m *MockGraphClient) HandleHTTPResponse(resp *http.Response, operation string) error {
	args := m.Called(resp, operation)
	return args.Error(0)
}

// ReadResponseBody mocks the ReadResponseBody method
func (m *MockGraphClient) ReadResponseBody(resp *http.Response, operation string) ([]byte, error) {
	args := m.Called(resp, operation)
	return args.Get(0).([]byte), args.Error(1)
}
