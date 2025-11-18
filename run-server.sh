#!/bin/bash
# OneNote MCP Server - Build and Run Script (Linux/macOS)
# Usage: ./run-server.sh [mode] [port]
#   mode: stdio (default) or http
#   port: port number for HTTP mode (default: 8080)

echo "OneNote MCP Server - Build and Run"
echo "====================================="

# Set default values
MODE=${1:-stdio}
PORT=${2:-8080}

echo "Building server..."
go build -o onenote-mcp-server ./cmd/onenote-mcp-server
if [ $? -ne 0 ]; then
    echo "Build failed!"
    exit 1
fi

echo "Build successful!"
echo

if [ "$MODE" = "http" ]; then
    echo "Starting OneNote MCP Server in HTTP mode on port $PORT..."
    echo "Press Ctrl+C to stop the server"
    echo
    ./onenote-mcp-server -mode=http -port=$PORT
else
    echo "Starting OneNote MCP Server in STDIO mode..."
    echo "Use Ctrl+C to stop, or send MCP messages via stdin"
    echo
    ./onenote-mcp-server
fi