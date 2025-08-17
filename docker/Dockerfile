FROM golang:1.23-alpine AS builder

# Install git for potential Go module dependencies
RUN apk --no-cache add git

WORKDIR /app

# Copy go module files first for better Docker layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application with proper flags
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o onenote-mcp-server ./cmd/onenote-mcp-server

# Runtime stage
FROM alpine:latest

# Install CA certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user for security
RUN adduser -D -s /bin/sh mcpuser

WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/onenote-mcp-server .

# Create directory for configs and ensure proper permissions
RUN mkdir -p /app/configs && \
    chown -R mcpuser:mcpuser /app

# Switch to non-root user
USER mcpuser

# Expose port for HTTP mode
EXPOSE 8080

# Default command runs in stdio mode, can be overridden for HTTP mode
ENTRYPOINT ["./onenote-mcp-server"] 