// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

// helpers.go - Shared HTTP utilities for automatic resource cleanup and safe request handling.
//
// This module provides a centralized location for HTTP request utilities that eliminate
// manual defer resp.Body.Close() patterns throughout the codebase. By consolidating
// HTTP handling logic into reusable utilities, we achieve consistent error handling,
// automatic resource cleanup, and reduced code duplication.
//
// Architectural Benefits:
// - Clean separation of concerns with shared infrastructure
// - Both internal/graph and business logic layers can use the same utilities
// - Eliminates dependency cycles (graph doesn't depend on higher-level utils)
// - Ensures consistent HTTP handling patterns across all modules
//
// Key Features:
// - Automatic response body cleanup with guaranteed resource management
// - Error-safe request handling with proper cleanup on failures
// - Consistent HTTP error handling and response validation
// - Simplified API for common HTTP patterns (body reading, custom handlers)
// - Support for multiple request patterns and conditional logic
//
// Usage across modules:
// - internal/graph: Core HTTP client operations and utilities
// - internal/auth: OAuth token exchange and refresh operations
// - internal/pages: Page content operations and multipart requests
// - internal/sections: Section and section group management
//
// This eliminates 26+ instances of manual defer resp.Body.Close() patterns
// and ensures proper resource cleanup even in complex error scenarios.

package http

import (
	"fmt"
	"io"
	"net/http"
)

// HTTPRequestFunc represents a function that makes HTTP requests
type HTTPRequestFunc func(method, url string, body io.Reader, headers map[string]string) (*http.Response, error)

// HTTPResponseHandler represents a function that handles HTTP responses
type HTTPResponseHandler func(resp *http.Response, operation string) error

// HTTPBodyReader represents a function that reads response bodies
type HTTPBodyReader func(resp *http.Response, operation string) ([]byte, error)

// SafeHTTPClient wraps HTTP operations with automatic resource cleanup
type SafeHTTPClient struct {
	makeRequest   HTTPRequestFunc
	handleResponse HTTPResponseHandler
	readBody      HTTPBodyReader
}

// NewSafeHTTPClient creates a new SafeHTTPClient with the provided functions
func NewSafeHTTPClient(
	makeRequest HTTPRequestFunc,
	handleResponse HTTPResponseHandler,
	readBody HTTPBodyReader,
) *SafeHTTPClient {
	return &SafeHTTPClient{
		makeRequest:   makeRequest,
		handleResponse: handleResponse,
		readBody:      readBody,
	}
}

// SafeHTTPResponse wraps a response with automatic cleanup
type SafeHTTPResponse struct {
	resp   *http.Response
	closed bool
}

// StatusCode returns the HTTP status code
func (sr *SafeHTTPResponse) StatusCode() int {
	if sr.resp == nil {
		return 0
	}
	return sr.resp.StatusCode
}

// Header returns the response headers
func (sr *SafeHTTPResponse) Header() http.Header {
	if sr.resp == nil {
		return nil
	}
	return sr.resp.Header
}

// Body returns the response body (caller is responsible for closing)
func (sr *SafeHTTPResponse) Body() io.ReadCloser {
	if sr.resp == nil {
		return nil
	}
	return sr.resp.Body
}

// Close ensures the response body is closed
func (sr *SafeHTTPResponse) Close() error {
	if !sr.closed && sr.resp != nil && sr.resp.Body != nil {
		sr.closed = true
		return sr.resp.Body.Close()
	}
	return nil
}

// ExecuteRequest makes an HTTP request with automatic cleanup
func (c *SafeHTTPClient) ExecuteRequest(
	method, url string,
	body io.Reader,
	headers map[string]string,
	operation string,
) (*SafeHTTPResponse, error) {
	resp, err := c.makeRequest(method, url, body, headers)
	if err != nil {
		return nil, err
	}

	safeResp := &SafeHTTPResponse{resp: resp}

	// Handle HTTP response validation
	if c.handleResponse != nil {
		if err := c.handleResponse(resp, operation); err != nil {
			safeResp.Close() // Ensure cleanup on error
			return nil, err
		}
	}

	return safeResp, nil
}

// ExecuteAndReadBody makes a request and reads the body, with automatic cleanup
func (c *SafeHTTPClient) ExecuteAndReadBody(
	method, url string,
	body io.Reader,
	headers map[string]string,
	operation string,
) ([]byte, error) {
	resp, err := c.makeRequest(method, url, body, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() // Safe to use defer here since it's encapsulated

	// Handle HTTP response validation
	if c.handleResponse != nil {
		if err := c.handleResponse(resp, operation); err != nil {
			return nil, err
		}
	}

	// Read response body
	if c.readBody != nil {
		return c.readBody(resp, operation)
	}

	// Fallback to direct body reading
	return io.ReadAll(resp.Body)
}

// ExecuteMultipleRequests executes multiple HTTP requests and ensures all are cleaned up
func (c *SafeHTTPClient) ExecuteMultipleRequests(
	requests []HTTPRequestSpec,
	operation string,
) ([]*SafeHTTPResponse, error) {
	var responses []*SafeHTTPResponse
	var cleanupResponses = func() {
		for _, resp := range responses {
			if resp != nil {
				resp.Close()
			}
		}
	}

	for i, req := range requests {
		resp, err := c.ExecuteRequest(req.Method, req.URL, req.Body, req.Headers, 
			fmt.Sprintf("%s_request_%d", operation, i))
		if err != nil {
			cleanupResponses() // Clean up all previous responses
			return nil, err
		}
		responses = append(responses, resp)
	}

	return responses, nil
}

// HTTPRequestSpec defines a single HTTP request specification
type HTTPRequestSpec struct {
	Method  string
	URL     string
	Body    io.Reader
	Headers map[string]string
}

// WithAutoCleanup executes a function with a response and ensures cleanup
func WithAutoCleanup(resp *http.Response, fn func(*http.Response) error) error {
	if resp == nil {
		return fmt.Errorf("nil response provided")
	}
	defer resp.Body.Close()
	return fn(resp)
}

// SafeRequest executes an HTTP request with automatic cleanup and error handling
func SafeRequest(
	makeRequest HTTPRequestFunc,
	handleResponse HTTPResponseHandler,
	method, url string,
	body io.Reader,
	headers map[string]string,
	operation string,
) error {
	resp, err := makeRequest(method, url, body, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if handleResponse != nil {
		return handleResponse(resp, operation)
	}

	return nil
}

// SafeRequestWithBody executes a request and returns the response body with automatic cleanup
func SafeRequestWithBody(
	makeRequest HTTPRequestFunc,
	handleResponse HTTPResponseHandler,
	readBody HTTPBodyReader,
	method, url string,
	body io.Reader,
	headers map[string]string,
	operation string,
) ([]byte, error) {
	resp, err := makeRequest(method, url, body, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if handleResponse != nil {
		if err := handleResponse(resp, operation); err != nil {
			return nil, err
		}
	}

	if readBody != nil {
		return readBody(resp, operation)
	}

	return io.ReadAll(resp.Body)
}

// SafeRequestWithCustomHandler executes a request with a custom response handler and automatic cleanup
func SafeRequestWithCustomHandler(
	makeRequest HTTPRequestFunc,
	handler func(*http.Response) error,
	method, url string,
	body io.Reader,
	headers map[string]string,
) error {
	resp, err := makeRequest(method, url, body, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return handler(resp)
}

// ConditionalRequest makes a request with conditional logic and automatic cleanup
func ConditionalRequest(
	makeRequest HTTPRequestFunc,
	condition func(*http.Response) bool,
	onSuccess func(*http.Response) error,
	onFailure func(*http.Response) error,
	method, url string,
	body io.Reader,
	headers map[string]string,
) error {
	resp, err := makeRequest(method, url, body, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if condition(resp) {
		if onSuccess != nil {
			return onSuccess(resp)
		}
	} else {
		if onFailure != nil {
			return onFailure(resp)
		}
	}

	return nil
}

// TryMultipleEndpoints tries multiple HTTP endpoints until one succeeds, with automatic cleanup
func TryMultipleEndpoints(
	makeRequest HTTPRequestFunc,
	handleResponse HTTPResponseHandler,
	readBody HTTPBodyReader,
	endpoints []HTTPRequestSpec,
	operation string,
) ([]byte, error) {
	var lastErr error

	for i, endpoint := range endpoints {
		resp, err := makeRequest(endpoint.Method, endpoint.URL, endpoint.Body, endpoint.Headers)
		if err != nil {
			lastErr = err
			continue
		}

		// Use the WithAutoCleanup helper to ensure proper cleanup
		var bodyData []byte
		err = WithAutoCleanup(resp, func(r *http.Response) error {
			if handleResponse != nil {
				if err := handleResponse(r, fmt.Sprintf("%s_endpoint_%d", operation, i)); err != nil {
					return err
				}
			}

			if readBody != nil {
				data, err := readBody(r, operation)
				bodyData = data
				return err
			}

			data, err := io.ReadAll(r.Body)
			bodyData = data
			return err
		})

		if err != nil {
			lastErr = err
			continue
		}

		return bodyData, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("all endpoints failed, last error: %v", lastErr)
	}

	return nil, fmt.Errorf("no endpoints provided")
}