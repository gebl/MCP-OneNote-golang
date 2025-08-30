// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package http

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock implementations for testing
type mockHTTPRequestFunc struct {
	responses []mockHTTPResponse
	callCount int
	requests  []mockRequestCall
}

type mockRequestCall struct {
	method  string
	url     string
	body    string
	headers map[string]string
}

type mockHTTPResponse struct {
	resp *http.Response
	err  error
}

func (m *mockHTTPRequestFunc) call(method, url string, body io.Reader, headers map[string]string) (*http.Response, error) {
	// Record the call
	bodyStr := ""
	if body != nil {
		bodyBytes, _ := io.ReadAll(body)
		bodyStr = string(bodyBytes)
	}
	
	m.requests = append(m.requests, mockRequestCall{
		method:  method,
		url:     url,
		body:    bodyStr,
		headers: headers,
	})

	if m.callCount >= len(m.responses) {
		return nil, fmt.Errorf("unexpected request call %d", m.callCount)
	}

	response := m.responses[m.callCount]
	m.callCount++
	return response.resp, response.err
}

func newMockResponse(statusCode int, body string, contentType string) *http.Response {
	resp := &http.Response{
		StatusCode: statusCode,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	if contentType != "" {
		resp.Header.Set("Content-Type", contentType)
	}
	return resp
}

// Mock response handler
type mockHTTPResponseHandler struct {
	shouldError bool
	errorMsg    string
	calls       []mockHandlerCall
}

type mockHandlerCall struct {
	statusCode int
	operation  string
}

func (m *mockHTTPResponseHandler) handle(resp *http.Response, operation string) error {
	m.calls = append(m.calls, mockHandlerCall{
		statusCode: resp.StatusCode,
		operation:  operation,
	})
	
	if m.shouldError {
		return fmt.Errorf(m.errorMsg)
	}
	return nil
}

// Mock body reader
type mockHTTPBodyReader struct {
	shouldError bool
	errorMsg    string
	responses   []string
	callCount   int
}

func (m *mockHTTPBodyReader) read(resp *http.Response, operation string) ([]byte, error) {
	if m.shouldError {
		return nil, fmt.Errorf(m.errorMsg)
	}
	
	if m.callCount >= len(m.responses) {
		// Fallback to actual body reading
		return io.ReadAll(resp.Body)
	}
	
	response := m.responses[m.callCount]
	m.callCount++
	return []byte(response), nil
}

// Test SafeHTTPClient creation
func TestNewSafeHTTPClient(t *testing.T) {
	mockRequest := &mockHTTPRequestFunc{}
	mockHandler := &mockHTTPResponseHandler{}
	mockReader := &mockHTTPBodyReader{}

	client := NewSafeHTTPClient(
		mockRequest.call,
		mockHandler.handle,
		mockReader.read,
	)

	assert.NotNil(t, client)
	assert.NotNil(t, client.makeRequest)
	assert.NotNil(t, client.handleResponse)
	assert.NotNil(t, client.readBody)
}

// Test SafeHTTPResponse wrapper
func TestSafeHTTPResponse(t *testing.T) {
	t.Run("normal response", func(t *testing.T) {
		resp := newMockResponse(200, "test body", "application/json")
		safeResp := &SafeHTTPResponse{resp: resp}

		assert.Equal(t, 200, safeResp.StatusCode())
		assert.Equal(t, "application/json", safeResp.Header().Get("Content-Type"))
		assert.NotNil(t, safeResp.Body())
		
		// Test close functionality
		err := safeResp.Close()
		assert.NoError(t, err)
		assert.True(t, safeResp.closed)
		
		// Second close should be safe
		err = safeResp.Close()
		assert.NoError(t, err)
	})

	t.Run("nil response", func(t *testing.T) {
		safeResp := &SafeHTTPResponse{resp: nil}

		assert.Equal(t, 0, safeResp.StatusCode())
		assert.Nil(t, safeResp.Header())
		assert.Nil(t, safeResp.Body())
		
		err := safeResp.Close()
		assert.NoError(t, err)
	})
}

// Test ExecuteRequest method
func TestSafeHTTPClient_ExecuteRequest(t *testing.T) {
	t.Run("successful request", func(t *testing.T) {
		mockRequest := &mockHTTPRequestFunc{
			responses: []mockHTTPResponse{
				{resp: newMockResponse(200, "success", "application/json"), err: nil},
			},
		}
		mockHandler := &mockHTTPResponseHandler{}
		
		client := NewSafeHTTPClient(mockRequest.call, mockHandler.handle, nil)
		
		safeResp, err := client.ExecuteRequest("GET", "https://example.com", nil, nil, "test_op")
		
		assert.NoError(t, err)
		assert.NotNil(t, safeResp)
		assert.Equal(t, 200, safeResp.StatusCode())
		assert.Equal(t, 1, len(mockRequest.requests))
		assert.Equal(t, "GET", mockRequest.requests[0].method)
		assert.Equal(t, "https://example.com", mockRequest.requests[0].url)
		assert.Equal(t, 1, len(mockHandler.calls))
		assert.Equal(t, "test_op", mockHandler.calls[0].operation)
		
		safeResp.Close()
	})

	t.Run("request failure", func(t *testing.T) {
		mockRequest := &mockHTTPRequestFunc{
			responses: []mockHTTPResponse{
				{resp: nil, err: fmt.Errorf("network error")},
			},
		}
		
		client := NewSafeHTTPClient(mockRequest.call, nil, nil)
		
		safeResp, err := client.ExecuteRequest("GET", "https://example.com", nil, nil, "test_op")
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "network error")
		assert.Nil(t, safeResp)
	})

	t.Run("response handler failure", func(t *testing.T) {
		mockRequest := &mockHTTPRequestFunc{
			responses: []mockHTTPResponse{
				{resp: newMockResponse(400, "error", "application/json"), err: nil},
			},
		}
		mockHandler := &mockHTTPResponseHandler{
			shouldError: true,
			errorMsg:    "HTTP 400 error",
		}
		
		client := NewSafeHTTPClient(mockRequest.call, mockHandler.handle, nil)
		
		safeResp, err := client.ExecuteRequest("GET", "https://example.com", nil, nil, "test_op")
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "HTTP 400 error")
		assert.Nil(t, safeResp)
	})
}

// Test ExecuteAndReadBody method
func TestSafeHTTPClient_ExecuteAndReadBody(t *testing.T) {
	t.Run("successful request with body reading", func(t *testing.T) {
		expectedBody := `{"message": "success"}`
		mockRequest := &mockHTTPRequestFunc{
			responses: []mockHTTPResponse{
				{resp: newMockResponse(200, expectedBody, "application/json"), err: nil},
			},
		}
		mockHandler := &mockHTTPResponseHandler{}
		mockReader := &mockHTTPBodyReader{
			responses: []string{expectedBody},
		}
		
		client := NewSafeHTTPClient(mockRequest.call, mockHandler.handle, mockReader.read)
		
		body, err := client.ExecuteAndReadBody("POST", "https://api.example.com", 
			strings.NewReader(`{"test": true}`), 
			map[string]string{"Content-Type": "application/json"}, 
			"create_resource")
		
		assert.NoError(t, err)
		assert.Equal(t, expectedBody, string(body))
		assert.Equal(t, 1, len(mockRequest.requests))
		assert.Equal(t, "POST", mockRequest.requests[0].method)
		assert.Equal(t, `{"test": true}`, mockRequest.requests[0].body)
		assert.Equal(t, "application/json", mockRequest.requests[0].headers["Content-Type"])
	})

	t.Run("fallback body reading", func(t *testing.T) {
		expectedBody := "fallback response"
		mockRequest := &mockHTTPRequestFunc{
			responses: []mockHTTPResponse{
				{resp: newMockResponse(200, expectedBody, "text/plain"), err: nil},
			},
		}
		
		client := NewSafeHTTPClient(mockRequest.call, nil, nil)
		
		body, err := client.ExecuteAndReadBody("GET", "https://example.com", nil, nil, "get_resource")
		
		assert.NoError(t, err)
		assert.Equal(t, expectedBody, string(body))
	})

	t.Run("body reader failure", func(t *testing.T) {
		mockRequest := &mockHTTPRequestFunc{
			responses: []mockHTTPResponse{
				{resp: newMockResponse(200, "success", "application/json"), err: nil},
			},
		}
		mockReader := &mockHTTPBodyReader{
			shouldError: true,
			errorMsg:    "failed to read body",
		}
		
		client := NewSafeHTTPClient(mockRequest.call, nil, mockReader.read)
		
		body, err := client.ExecuteAndReadBody("GET", "https://example.com", nil, nil, "test_op")
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read body")
		assert.Nil(t, body)
	})
}

// Test ExecuteMultipleRequests method
func TestSafeHTTPClient_ExecuteMultipleRequests(t *testing.T) {
	t.Run("multiple successful requests", func(t *testing.T) {
		mockRequest := &mockHTTPRequestFunc{
			responses: []mockHTTPResponse{
				{resp: newMockResponse(200, "response1", "application/json"), err: nil},
				{resp: newMockResponse(201, "response2", "application/json"), err: nil},
				{resp: newMockResponse(200, "response3", "text/plain"), err: nil},
			},
		}
		
		client := NewSafeHTTPClient(mockRequest.call, nil, nil)
		
		requests := []HTTPRequestSpec{
			{Method: "GET", URL: "https://api1.example.com", Body: nil, Headers: nil},
			{Method: "POST", URL: "https://api2.example.com", Body: strings.NewReader("data"), Headers: map[string]string{"Content-Type": "text/plain"}},
			{Method: "PUT", URL: "https://api3.example.com", Body: nil, Headers: nil},
		}
		
		responses, err := client.ExecuteMultipleRequests(requests, "batch_operation")
		
		assert.NoError(t, err)
		assert.Equal(t, 3, len(responses))
		assert.Equal(t, 200, responses[0].StatusCode())
		assert.Equal(t, 201, responses[1].StatusCode())
		assert.Equal(t, 200, responses[2].StatusCode())
		
		// Clean up responses
		for _, resp := range responses {
			resp.Close()
		}
	})

	t.Run("failure in middle request", func(t *testing.T) {
		mockRequest := &mockHTTPRequestFunc{
			responses: []mockHTTPResponse{
				{resp: newMockResponse(200, "response1", "application/json"), err: nil},
				{resp: nil, err: fmt.Errorf("network failure")},
			},
		}
		
		client := NewSafeHTTPClient(mockRequest.call, nil, nil)
		
		requests := []HTTPRequestSpec{
			{Method: "GET", URL: "https://api1.example.com", Body: nil, Headers: nil},
			{Method: "POST", URL: "https://api2.example.com", Body: nil, Headers: nil},
		}
		
		responses, err := client.ExecuteMultipleRequests(requests, "batch_operation")
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "network failure")
		assert.Nil(t, responses)
	})
}

// Test WithAutoCleanup function
func TestWithAutoCleanup(t *testing.T) {
	t.Run("successful operation", func(t *testing.T) {
		resp := newMockResponse(200, "test body", "application/json")
		var capturedStatusCode int
		
		err := WithAutoCleanup(resp, func(r *http.Response) error {
			capturedStatusCode = r.StatusCode
			return nil
		})
		
		assert.NoError(t, err)
		assert.Equal(t, 200, capturedStatusCode)
	})

	t.Run("nil response", func(t *testing.T) {
		err := WithAutoCleanup(nil, func(r *http.Response) error {
			return nil
		})
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nil response provided")
	})

	t.Run("function returns error", func(t *testing.T) {
		resp := newMockResponse(500, "error", "application/json")
		
		err := WithAutoCleanup(resp, func(r *http.Response) error {
			return fmt.Errorf("processing error")
		})
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "processing error")
	})
}

// Test SafeRequest function
func TestSafeRequest(t *testing.T) {
	t.Run("successful request", func(t *testing.T) {
		mockRequest := &mockHTTPRequestFunc{
			responses: []mockHTTPResponse{
				{resp: newMockResponse(200, "success", "application/json"), err: nil},
			},
		}
		mockHandler := &mockHTTPResponseHandler{}
		
		err := SafeRequest(mockRequest.call, mockHandler.handle, "GET", "https://example.com", nil, nil, "safe_operation")
		
		assert.NoError(t, err)
		assert.Equal(t, 1, len(mockRequest.requests))
		assert.Equal(t, 1, len(mockHandler.calls))
		assert.Equal(t, "safe_operation", mockHandler.calls[0].operation)
	})

	t.Run("request failure", func(t *testing.T) {
		mockRequest := &mockHTTPRequestFunc{
			responses: []mockHTTPResponse{
				{resp: nil, err: fmt.Errorf("connection refused")},
			},
		}
		
		err := SafeRequest(mockRequest.call, nil, "GET", "https://invalid.example.com", nil, nil, "failed_operation")
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connection refused")
	})

	t.Run("handler failure", func(t *testing.T) {
		mockRequest := &mockHTTPRequestFunc{
			responses: []mockHTTPResponse{
				{resp: newMockResponse(404, "not found", "application/json"), err: nil},
			},
		}
		mockHandler := &mockHTTPResponseHandler{
			shouldError: true,
			errorMsg:    "resource not found",
		}
		
		err := SafeRequest(mockRequest.call, mockHandler.handle, "GET", "https://example.com/missing", nil, nil, "not_found_operation")
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "resource not found")
	})

	t.Run("no handler provided", func(t *testing.T) {
		mockRequest := &mockHTTPRequestFunc{
			responses: []mockHTTPResponse{
				{resp: newMockResponse(200, "success", "application/json"), err: nil},
			},
		}
		
		err := SafeRequest(mockRequest.call, nil, "GET", "https://example.com", nil, nil, "no_handler_operation")
		
		assert.NoError(t, err)
	})
}

// Test SafeRequestWithBody function - this is the core functionality
func TestSafeRequestWithBody(t *testing.T) {
	t.Run("complete successful flow", func(t *testing.T) {
		expectedResponse := `{"id": "123", "status": "created"}`
		mockRequest := &mockHTTPRequestFunc{
			responses: []mockHTTPResponse{
				{resp: newMockResponse(201, expectedResponse, "application/json"), err: nil},
			},
		}
		mockHandler := &mockHTTPResponseHandler{}
		mockReader := &mockHTTPBodyReader{
			responses: []string{expectedResponse},
		}
		
		requestBody := `{"name": "test resource"}`
		headers := map[string]string{
			"Content-Type": "application/json",
			"Authorization": "Bearer token123",
		}
		
		body, err := SafeRequestWithBody(
			mockRequest.call,
			mockHandler.handle,
			mockReader.read,
			"POST",
			"https://api.example.com/resources",
			strings.NewReader(requestBody),
			headers,
			"create_resource",
		)
		
		assert.NoError(t, err)
		assert.Equal(t, expectedResponse, string(body))
		
		// Verify request details
		require.Equal(t, 1, len(mockRequest.requests))
		req := mockRequest.requests[0]
		assert.Equal(t, "POST", req.method)
		assert.Equal(t, "https://api.example.com/resources", req.url)
		assert.Equal(t, requestBody, req.body)
		assert.Equal(t, "application/json", req.headers["Content-Type"])
		assert.Equal(t, "Bearer token123", req.headers["Authorization"])
		
		// Verify handler was called
		require.Equal(t, 1, len(mockHandler.calls))
		assert.Equal(t, 201, mockHandler.calls[0].statusCode)
		assert.Equal(t, "create_resource", mockHandler.calls[0].operation)
	})

	t.Run("fallback body reading", func(t *testing.T) {
		expectedResponse := "plain text response"
		mockRequest := &mockHTTPRequestFunc{
			responses: []mockHTTPResponse{
				{resp: newMockResponse(200, expectedResponse, "text/plain"), err: nil},
			},
		}
		
		body, err := SafeRequestWithBody(
			mockRequest.call,
			nil,    // No response handler
			nil,    // No body reader - should fallback to io.ReadAll
			"GET",
			"https://example.com/text",
			nil,
			nil,
			"get_text",
		)
		
		assert.NoError(t, err)
		assert.Equal(t, expectedResponse, string(body))
	})

	t.Run("request error", func(t *testing.T) {
		mockRequest := &mockHTTPRequestFunc{
			responses: []mockHTTPResponse{
				{resp: nil, err: fmt.Errorf("DNS resolution failed")},
			},
		}
		
		body, err := SafeRequestWithBody(
			mockRequest.call,
			nil,
			nil,
			"GET",
			"https://nonexistent.domain.com",
			nil,
			nil,
			"failed_request",
		)
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DNS resolution failed")
		assert.Nil(t, body)
	})

	t.Run("response handler error", func(t *testing.T) {
		mockRequest := &mockHTTPRequestFunc{
			responses: []mockHTTPResponse{
				{resp: newMockResponse(403, "forbidden", "application/json"), err: nil},
			},
		}
		mockHandler := &mockHTTPResponseHandler{
			shouldError: true,
			errorMsg:    "HTTP 403 Forbidden",
		}
		
		body, err := SafeRequestWithBody(
			mockRequest.call,
			mockHandler.handle,
			nil,
			"GET",
			"https://api.example.com/restricted",
			nil,
			nil,
			"forbidden_access",
		)
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "HTTP 403 Forbidden")
		assert.Nil(t, body)
	})

	t.Run("body reader error", func(t *testing.T) {
		mockRequest := &mockHTTPRequestFunc{
			responses: []mockHTTPResponse{
				{resp: newMockResponse(200, "success", "application/json"), err: nil},
			},
		}
		mockReader := &mockHTTPBodyReader{
			shouldError: true,
			errorMsg:    "failed to parse JSON response",
		}
		
		body, err := SafeRequestWithBody(
			mockRequest.call,
			nil,
			mockReader.read,
			"GET",
			"https://api.example.com/data",
			nil,
			nil,
			"parse_error",
		)
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse JSON response")
		assert.Nil(t, body)
	})
}

// Test SafeRequestWithCustomHandler function
func TestSafeRequestWithCustomHandler(t *testing.T) {
	t.Run("successful custom handling", func(t *testing.T) {
		mockRequest := &mockHTTPRequestFunc{
			responses: []mockHTTPResponse{
				{resp: newMockResponse(200, "custom response", "text/plain"), err: nil},
			},
		}
		
		var capturedBody string
		customHandler := func(resp *http.Response) error {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			capturedBody = string(body)
			return nil
		}
		
		err := SafeRequestWithCustomHandler(
			mockRequest.call,
			customHandler,
			"GET",
			"https://example.com/custom",
			nil,
			map[string]string{"Accept": "text/plain"},
		)
		
		assert.NoError(t, err)
		assert.Equal(t, "custom response", capturedBody)
	})

	t.Run("custom handler error", func(t *testing.T) {
		mockRequest := &mockHTTPRequestFunc{
			responses: []mockHTTPResponse{
				{resp: newMockResponse(200, "response", "text/plain"), err: nil},
			},
		}
		
		customHandler := func(resp *http.Response) error {
			return fmt.Errorf("custom processing failed")
		}
		
		err := SafeRequestWithCustomHandler(
			mockRequest.call,
			customHandler,
			"POST",
			"https://example.com/process",
			strings.NewReader("data"),
			nil,
		)
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "custom processing failed")
	})
}

// Test ConditionalRequest function
func TestConditionalRequest(t *testing.T) {
	t.Run("condition true - success handler called", func(t *testing.T) {
		mockRequest := &mockHTTPRequestFunc{
			responses: []mockHTTPResponse{
				{resp: newMockResponse(200, "success", "application/json"), err: nil},
			},
		}
		
		var successCalled bool
		var failureCalled bool
		
		condition := func(resp *http.Response) bool {
			return resp.StatusCode == 200
		}
		
		onSuccess := func(resp *http.Response) error {
			successCalled = true
			return nil
		}
		
		onFailure := func(resp *http.Response) error {
			failureCalled = true
			return nil
		}
		
		err := ConditionalRequest(
			mockRequest.call,
			condition,
			onSuccess,
			onFailure,
			"GET",
			"https://example.com",
			nil,
			nil,
		)
		
		assert.NoError(t, err)
		assert.True(t, successCalled)
		assert.False(t, failureCalled)
	})

	t.Run("condition false - failure handler called", func(t *testing.T) {
		mockRequest := &mockHTTPRequestFunc{
			responses: []mockHTTPResponse{
				{resp: newMockResponse(404, "not found", "application/json"), err: nil},
			},
		}
		
		var successCalled bool
		var failureCalled bool
		
		condition := func(resp *http.Response) bool {
			return resp.StatusCode < 400
		}
		
		onSuccess := func(resp *http.Response) error {
			successCalled = true
			return nil
		}
		
		onFailure := func(resp *http.Response) error {
			failureCalled = true
			return fmt.Errorf("resource not found")
		}
		
		err := ConditionalRequest(
			mockRequest.call,
			condition,
			onSuccess,
			onFailure,
			"GET",
			"https://example.com/missing",
			nil,
			nil,
		)
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "resource not found")
		assert.False(t, successCalled)
		assert.True(t, failureCalled)
	})

	t.Run("no handlers provided", func(t *testing.T) {
		mockRequest := &mockHTTPRequestFunc{
			responses: []mockHTTPResponse{
				{resp: newMockResponse(200, "success", "application/json"), err: nil},
			},
		}
		
		condition := func(resp *http.Response) bool {
			return true
		}
		
		err := ConditionalRequest(
			mockRequest.call,
			condition,
			nil,
			nil,
			"GET",
			"https://example.com",
			nil,
			nil,
		)
		
		assert.NoError(t, err)
	})
}

// Test TryMultipleEndpoints function
func TestTryMultipleEndpoints(t *testing.T) {
	t.Run("first endpoint succeeds", func(t *testing.T) {
		expectedResponse := "first endpoint response"
		mockRequest := &mockHTTPRequestFunc{
			responses: []mockHTTPResponse{
				{resp: newMockResponse(200, expectedResponse, "application/json"), err: nil},
			},
		}
		
		endpoints := []HTTPRequestSpec{
			{Method: "GET", URL: "https://primary.example.com", Body: nil, Headers: nil},
			{Method: "GET", URL: "https://secondary.example.com", Body: nil, Headers: nil},
			{Method: "GET", URL: "https://tertiary.example.com", Body: nil, Headers: nil},
		}
		
		body, err := TryMultipleEndpoints(
			mockRequest.call,
			nil,
			nil,
			endpoints,
			"fallback_operation",
		)
		
		assert.NoError(t, err)
		assert.Equal(t, expectedResponse, string(body))
		assert.Equal(t, 1, mockRequest.callCount) // Only first endpoint should be called
	})

	t.Run("first endpoint fails, second succeeds", func(t *testing.T) {
		expectedResponse := "second endpoint response"
		mockRequest := &mockHTTPRequestFunc{
			responses: []mockHTTPResponse{
				{resp: nil, err: fmt.Errorf("connection timeout")},
				{resp: newMockResponse(200, expectedResponse, "application/json"), err: nil},
			},
		}
		
		endpoints := []HTTPRequestSpec{
			{Method: "GET", URL: "https://primary.example.com", Body: nil, Headers: nil},
			{Method: "GET", URL: "https://secondary.example.com", Body: nil, Headers: nil},
		}
		
		body, err := TryMultipleEndpoints(
			mockRequest.call,
			nil,
			nil,
			endpoints,
			"fallback_operation",
		)
		
		assert.NoError(t, err)
		assert.Equal(t, expectedResponse, string(body))
		assert.Equal(t, 2, mockRequest.callCount)
	})

	t.Run("all endpoints fail", func(t *testing.T) {
		mockRequest := &mockHTTPRequestFunc{
			responses: []mockHTTPResponse{
				{resp: nil, err: fmt.Errorf("connection timeout")},
				{resp: nil, err: fmt.Errorf("DNS resolution failed")},
				{resp: nil, err: fmt.Errorf("server unavailable")},
			},
		}
		
		endpoints := []HTTPRequestSpec{
			{Method: "GET", URL: "https://primary.example.com", Body: nil, Headers: nil},
			{Method: "GET", URL: "https://secondary.example.com", Body: nil, Headers: nil},
			{Method: "GET", URL: "https://tertiary.example.com", Body: nil, Headers: nil},
		}
		
		body, err := TryMultipleEndpoints(
			mockRequest.call,
			nil,
			nil,
			endpoints,
			"fallback_operation",
		)
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "all endpoints failed")
		assert.Contains(t, err.Error(), "server unavailable") // Last error
		assert.Nil(t, body)
		assert.Equal(t, 3, mockRequest.callCount)
	})

	t.Run("no endpoints provided", func(t *testing.T) {
		mockRequest := &mockHTTPRequestFunc{}
		
		body, err := TryMultipleEndpoints(
			mockRequest.call,
			nil,
			nil,
			[]HTTPRequestSpec{},
			"empty_operation",
		)
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no endpoints provided")
		assert.Nil(t, body)
	})

	t.Run("endpoint succeeds but handler fails", func(t *testing.T) {
		mockRequest := &mockHTTPRequestFunc{
			responses: []mockHTTPResponse{
				{resp: newMockResponse(500, "server error", "text/plain"), err: nil},
				{resp: newMockResponse(200, "success", "application/json"), err: nil},
			},
		}
		
		endpoints := []HTTPRequestSpec{
			{Method: "GET", URL: "https://broken.example.com", Body: nil, Headers: nil},
			{Method: "GET", URL: "https://working.example.com", Body: nil, Headers: nil},
		}
		
		// Reset handler error for second call
		callCount := 0
		handlerFunc := func(resp *http.Response, operation string) error {
			callCount++
			if callCount == 1 {
				return fmt.Errorf("HTTP 500 error")
			}
			return nil
		}
		
		body, err := TryMultipleEndpoints(
			mockRequest.call,
			handlerFunc,
			nil,
			endpoints,
			"retry_operation",
		)
		
		assert.NoError(t, err)
		assert.Equal(t, "success", string(body))
		assert.Equal(t, 2, mockRequest.callCount)
	})
}

// Test error handling and resource cleanup
func TestResourceCleanup(t *testing.T) {
	t.Run("response body is closed on error", func(t *testing.T) {
		// Create a response that we can track if it's closed
		body := &trackingReadCloser{
			Reader: strings.NewReader("test data"),
			closed: false,
		}
		
		resp := &http.Response{
			StatusCode: 200,
			Header:     make(http.Header),
			Body:       body,
		}
		
		mockRequest := &mockHTTPRequestFunc{
			responses: []mockHTTPResponse{
				{resp: resp, err: nil},
			},
		}
		
		mockHandler := &mockHTTPResponseHandler{
			shouldError: true,
			errorMsg:    "processing failed",
		}
		
		client := NewSafeHTTPClient(mockRequest.call, mockHandler.handle, nil)
		
		// This should fail and trigger cleanup
		safeResp, err := client.ExecuteRequest("GET", "https://example.com", nil, nil, "test_cleanup")
		
		assert.Error(t, err)
		assert.Nil(t, safeResp)
		assert.True(t, body.closed, "Response body should be closed on error")
	})
}

// Helper struct to track if response body is closed
type trackingReadCloser struct {
	io.Reader
	closed bool
}

func (t *trackingReadCloser) Close() error {
	t.closed = true
	return nil
}

// Test integration scenarios
func TestHTTPHelpers_Integration(t *testing.T) {
	t.Run("real world API call simulation", func(t *testing.T) {
		// Simulate a complete API interaction: auth, request, response handling
		responses := []mockHTTPResponse{
			{resp: newMockResponse(200, `{"access_token":"token123","expires_in":3600}`, "application/json"), err: nil},
			{resp: newMockResponse(201, `{"id":"resource123","status":"created"}`, "application/json"), err: nil},
		}
		
		mockRequest := &mockHTTPRequestFunc{responses: responses}
		
		mockHandler := &mockHTTPResponseHandler{}
		
		mockReader := &mockHTTPBodyReader{
			responses: []string{
				`{"access_token":"token123","expires_in":3600}`,
				`{"id":"resource123","status":"created"}`,
			},
		}
		
		client := NewSafeHTTPClient(mockRequest.call, mockHandler.handle, mockReader.read)
		
		// Step 1: Get access token
		authBody, err := client.ExecuteAndReadBody(
			"POST",
			"https://auth.example.com/token",
			strings.NewReader("grant_type=client_credentials"),
			map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			"get_token",
		)
		
		assert.NoError(t, err)
		assert.Contains(t, string(authBody), "access_token")
		
		// Step 2: Create resource using token
		resourceBody, err := client.ExecuteAndReadBody(
			"POST",
			"https://api.example.com/resources",
			strings.NewReader(`{"name":"test resource"}`),
			map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer token123",
			},
			"create_resource",
		)
		
		assert.NoError(t, err)
		assert.Contains(t, string(resourceBody), "resource123")
		
		// Verify both requests were made with correct parameters
		assert.Equal(t, 2, len(mockRequest.requests))
		assert.Equal(t, "POST", mockRequest.requests[0].method)
		assert.Equal(t, "https://auth.example.com/token", mockRequest.requests[0].url)
		assert.Equal(t, "POST", mockRequest.requests[1].method)
		assert.Equal(t, "https://api.example.com/resources", mockRequest.requests[1].url)
		assert.Equal(t, "Bearer token123", mockRequest.requests[1].headers["Authorization"])
	})
}