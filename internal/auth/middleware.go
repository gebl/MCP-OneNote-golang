// middleware.go - HTTP authentication middleware for MCP server.
//
// This file provides HTTP authentication middleware for securing the OneNote MCP server
// when running in HTTP/SSE transport modes. The middleware validates Bearer tokens
// in the Authorization header against a configured token value.
//
// Key Features:
// - Simple bearer token validation using string comparison
// - Configurable authentication bypass for health check endpoints
// - Comprehensive logging of authentication attempts
// - Standards-compliant HTTP responses for authentication failures
// - No external dependencies or complex crypto operations
//
// Security Considerations:
// - Bearer tokens should be transmitted over HTTPS only in production
// - Tokens should be sufficiently long and random for security
// - Failed authentication attempts are logged for security monitoring
// - Authentication is only applied to HTTP/SSE modes, not stdio mode
//
// Usage:
//   middleware := auth.BearerTokenMiddleware("your-secret-token")
//   handler := middleware(httpHandler)
//
// HTTP Client Usage:
//   Authorization: Bearer your-secret-token
//
// For detailed configuration options, see internal/config/config.go

package auth

import (
	"net/http"
	"strings"

	"github.com/gebl/onenote-mcp-server/internal/logging"
)

// BearerTokenMiddleware creates HTTP middleware that validates Bearer tokens
// against the provided expectedToken. Returns 401 Unauthorized for invalid
// or missing tokens.
//
// The middleware:
// - Extracts the Authorization header from incoming requests
// - Validates the Bearer token format and value
// - Allows requests with valid tokens to proceed
// - Returns 401 for missing, malformed, or invalid tokens
// - Logs authentication attempts for security monitoring
func BearerTokenMiddleware(expectedToken string) func(http.Handler) http.Handler {
	logger := logging.AuthLogger

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip authentication for health check endpoints if needed
			if r.URL.Path == "/health" || r.URL.Path == "/ping" {
				logger.Debug("Skipping authentication for health check endpoint", "path", r.URL.Path)
				next.ServeHTTP(w, r)
				return
			}

			// Extract Authorization header
			auth := r.Header.Get("Authorization")
			if auth == "" {
				logger.Warn("Authentication failed: missing Authorization header",
					"remote_addr", r.RemoteAddr,
					"user_agent", r.Header.Get("User-Agent"),
					"path", r.URL.Path,
					"method", r.Method)
				w.Header().Set("WWW-Authenticate", "Bearer")
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			// Extract token - handle both "Bearer token" and raw "token" formats
			var token string
			if strings.HasPrefix(auth, "Bearer ") {
				// Standard Bearer token format
				token = strings.TrimPrefix(auth, "Bearer ")
				logger.Debug("Received standard Bearer token format")
			} else {
				// Raw token format (for compatibility with some MCP clients)
				token = auth
				logger.Debug("Received raw token format (no Bearer prefix)")
			}

			if token == "" {
				logger.Warn("Authentication failed: empty token",
					"remote_addr", r.RemoteAddr,
					"user_agent", r.Header.Get("User-Agent"),
					"path", r.URL.Path,
					"method", r.Method)
				w.Header().Set("WWW-Authenticate", "Bearer")
				http.Error(w, "Token cannot be empty", http.StatusUnauthorized)
				return
			}

			// Validate token against expected value
			if token != expectedToken {
				logger.Warn("Authentication failed: invalid Bearer token",
					"remote_addr", r.RemoteAddr,
					"user_agent", r.Header.Get("User-Agent"),
					"path", r.URL.Path,
					"method", r.Method,
					"token_length", len(token),
					"expected_length", len(expectedToken))
				w.Header().Set("WWW-Authenticate", "Bearer")
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			// Authentication successful
			logger.Debug("Authentication successful",
				"remote_addr", r.RemoteAddr,
				"user_agent", r.Header.Get("User-Agent"),
				"path", r.URL.Path,
				"method", r.Method,
				"token_length", len(token))

			// Proceed to the next handler
			next.ServeHTTP(w, r)
		})
	}
}

// RequestLoggingMiddleware creates HTTP middleware that logs request details
// including method, path, remote address, and user agent. This is useful
// for debugging HTTP/SSE transport issues and monitoring server usage.
//
// The middleware logs:
// - HTTP method (GET, POST, etc.)
// - Request path and query parameters
// - Remote client address
// - User-Agent header
// - Response status code (after request completion)
func RequestLoggingMiddleware() func(http.Handler) http.Handler {
	logger := logging.MainLogger

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Log incoming request
			logger.Info("HTTP request received",
				"method", r.Method,
				"path", r.URL.Path,
				"query", r.URL.RawQuery,
				"remote_addr", r.RemoteAddr,
				"user_agent", r.Header.Get("User-Agent"),
				"content_length", r.ContentLength,
				"host", r.Host)

			// Create a response writer wrapper to capture status code
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Call the next handler
			next.ServeHTTP(wrapped, r)

			// Log response with additional context for errors
			if wrapped.statusCode >= 400 {
				logger.Warn("HTTP request completed with error",
					"method", r.Method,
					"path", r.URL.Path,
					"query", r.URL.RawQuery,
					"status_code", wrapped.statusCode,
					"remote_addr", r.RemoteAddr,
					"user_agent", r.Header.Get("User-Agent"))
			} else {
				logger.Info("HTTP request completed",
					"method", r.Method,
					"path", r.URL.Path,
					"status_code", wrapped.statusCode,
					"remote_addr", r.RemoteAddr)
			}
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code and calls the underlying WriteHeader
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Write calls the underlying Write method
func (rw *responseWriter) Write(b []byte) (int, error) {
	return rw.ResponseWriter.Write(b)
}
