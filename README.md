# OneNote MCP Server

A Go-based Model Context Protocol (MCP) server for seamless integration with Microsoft OneNote via the Microsoft Graph API.

## Features
- Read, create, and manage OneNote content
- Secure OAuth 2.0 authentication
- Docker deployment
- Modular toolset architecture

## Project Structure
```
cmd/onenote-mcp-server/      # Main entrypoint
internal/auth/              # OAuth 2.0 authentication
internal/graph/             # Microsoft Graph API client
internal/mcp/               # MCP protocol implementation
internal/tools/             # Tool implementations
internal/config/            # Configuration management
pkg/                        # Public packages (if any)
docker/                     # Dockerfile and compose
configs/                    # Example configs
docs/                       # Documentation
```

## Setup
1. Set environment variables for Microsoft OAuth (see PRD)
2. Build and run: `go build -o onenote-mcp-server ./cmd/onenote-mcp-server`
3. Or use Docker: `docker build -t onenote-mcp-server .`

## See `onenote_mcp_prd.md` for full requirements and roadmap. 