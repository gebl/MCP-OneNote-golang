# OneNote MCP Server - Product Requirements Document

## Overview

Build a Go-based Model Context Protocol (MCP) server that provides seamless integration with Microsoft OneNote via the Microsoft Graph API. This server will enable AI assistants to interact with OneNote notebooks, sections, and pages, following the same architectural patterns as the official GitHub MCP server.

## Project Goals

- **Primary Goal**: Create a robust, production-ready MCP server for OneNote integration
- **Secondary Goals**: 
  - Enable AI assistants to read, create, and manage OneNote content
  - Provide secure authentication and authorization
  - Support Docker deployment with easy configuration
  - Follow MCP specification best practices

## Technical Stack

- **Language**: Go (Golang)
- **Protocol**: Model Context Protocol (MCP) 2024-11-05 specification
- **Transport**: stdio (primary), with optional HTTP support
- **API**: Microsoft Graph API v1.0
- **Authentication**: OAuth 2.0 with Microsoft Identity Platform
- **Deployment**: Docker + local binary
- **Dependencies**: Minimal external dependencies, prefer standard library

## Architecture Reference

Follow the architecture patterns from the GitHub MCP Server:
- Similar project structure with cmd/ directory for main executable
- Docker-first deployment with environment variable configuration
- Toolset-based capability organization
- Configuration override system via JSON files and environment variables

## Core Features

### 1. Authentication & Authorization

**Requirements:**
- OAuth 2.0 flow with Microsoft Identity Platform
- Support delegated authentication (app-only authentication deprecated March 31, 2025)
- Secure token storage and refresh handling
- Support for both personal and organizational accounts

**Environment Variables:**
- `ONENOTE_CLIENT_ID` - Azure AD application client ID
- `ONENOTE_CLIENT_SECRET` - Azure AD application client secret
- `ONENOTE_TENANT_ID` - Azure AD tenant ID (optional, defaults to common)
- `ONENOTE_REDIRECT_URI` - OAuth redirect URI

### 2. MCP Server Capabilities

**Server Information:**
- Name: "OneNote MCP Server"
- Version: "1.0.0"
- Protocol Version: "2024-11-05"

**Supported Capabilities:**
- `tools`: Execute OneNote operations
- `resources`: Access OneNote content as resources
- `logging`: Debug and operational logging

### 3. Tool Categories (Toolsets)

Following the GitHub MCP server pattern, organize tools into logical toolsets:

#### 3.1 Notebooks Toolset (`notebooks`)
- `list_notebooks` - List all notebooks accessible to the user
- `get_notebook` - Get details of a specific notebook
- `create_notebook` - Create a new notebook
- `search_notebooks` - Search notebooks by name/content

#### 3.2 Sections Toolset (`sections`)
- `list_sections` - List sections in a notebook
- `get_section` - Get details of a specific section  
- `create_section` - Create a new section in a notebook
- `update_section` - Update section properties
- `delete_section` - Delete a section

#### 3.3 Pages Toolset (`pages`)
- `list_pages` - List pages in a section with filtering options
- `get_page` - Get page content and metadata
- `create_page` - Create a new page with HTML content
- `update_page` - Update existing page content
- `delete_page` - Delete a page
- `search_pages` - Search pages with full-text search capabilities

#### 3.4 Content Toolset (`content`)
- `get_page_content` - Get page content in various formats (HTML, preview)
- `append_page_content` - Append content to existing page
- `extract_text` - Extract text content using OCR capabilities

#### 3.5 Authentication Toolset (`auth`)
- `auth_status` - Get current authentication status and token information
- `auth_initiate` - Start new OAuth authentication flow with browser redirect
- `auth_refresh` - Manually refresh authentication tokens
- `auth_clear` - Clear stored tokens (logout functionality)

### 4. Resource Templates

Provide MCP resources for direct content access:

```
onenote://notebooks/{notebook-id}
onenote://notebooks/{notebook-id}/sections/{section-id}
onenote://notebooks/{notebook-id}/sections/{section-id}/pages/{page-id}
onenote://search/pages?q={query}
onenote://recent/pages?days={days}
```

### 5. Tool Specifications

#### Example: `create_page`
```json
{
  "name": "create_page",
  "description": "Create a new page in a OneNote section",
  "inputSchema": {
    "type": "object",
    "properties": {
      "section_id": {
        "type": "string",
        "description": "The ID of the section to create the page in"
      },
      "title": {
        "type": "string", 
        "description": "The title of the new page"
      },
      "content": {
        "type": "string",
        "description": "HTML content for the page body"
      }
    },
    "required": ["section_id", "title"]
  }
}
```

#### Example: `search_pages`
```json
{
  "name": "search_pages",
  "description": "Search for pages across all accessible notebooks",
  "inputSchema": {
    "type": "object",
    "properties": {
      "query": {
        "type": "string",
        "description": "Search query string"
      },
      "top": {
        "type": "integer",
        "description": "Maximum number of results to return",
        "default": 20,
        "maximum": 100
      },
      "filter": {
        "type": "string", 
        "description": "OData filter expression"
      }
    },
    "required": ["query"]
  }
}
```

#### Example: `auth_status`
```json
{
  "name": "auth_status",
  "description": "Get current authentication status and token information without exposing sensitive data",
  "inputSchema": {
    "type": "object",
    "properties": {},
    "required": []
  }
}
```

#### Example: `auth_initiate`
```json
{
  "name": "auth_initiate", 
  "description": "Start new OAuth authentication flow with browser redirect",
  "inputSchema": {
    "type": "object",
    "properties": {},
    "required": []
  }
}
```

## Microsoft Graph API Integration

### Base URL
https://graph.microsoft.com/v1.0/me/onenote/

### Key Endpoints to Implement

**Notebooks:**
- `GET /notebooks` - List notebooks
- `POST /notebooks` - Create notebook
- `GET /notebooks/{id}` - Get notebook details

**Sections:**
- `GET /notebooks/{id}/sections` - List sections
- `POST /notebooks/{id}/sections` - Create section
- `GET /sections/{id}` - Get section details

**Pages:**
- GET /pages - List all pages with optional filtering
- GET /sections/{id}/pages - List pages in section
- `POST /sections/{id}/pages` - Create page
- `GET /pages/{id}` - Get page details
- `GET /pages/{id}/content` - Get page content

### API Features to Leverage
- OData query parameters: $filter, $select, $expand, $orderby, $top
- OCR capabilities for image processing
- Full-text search across notebooks
- HTML content creation and updates

## Configuration System

### 1. Toolset Management
Support enabling/disabling toolsets via:

**Command Line:**
```bash
./onenote-mcp-server --toolsets notebooks,sections,pages
```

**Environment Variable:**
```bash
ONENOTE_TOOLSETS="notebooks,sections,pages,content"
```

### 2. Tool Description Overrides

**Config File** (`onenote-mcp-server-config.json`):
```json
{
  "TOOL_CREATE_PAGE_DESCRIPTION": "Create a new OneNote page with custom content",
  "TOOL_SEARCH_PAGES_DESCRIPTION": "Search across all OneNote pages using full-text search"
}
```

**Environment Variables:**
```bash
ONENOTE_MCP_TOOL_CREATE_PAGE_DESCRIPTION="Custom description"
```

### 3. Export Translations
```bash
./onenote-mcp-server --export-translations
```

## Project Structure

```
onenote-mcp-server/
├── cmd/
│   └── onenote-mcp-server/
│       └── main.go              # Entry point
├── internal/
│   ├── auth/                    # OAuth 2.0 authentication
│   ├── graph/                   # Microsoft Graph API client
│   ├── mcp/                     # MCP protocol implementation
│   ├── tools/                   # Tool implementations
│   └── config/                  # Configuration management
├── pkg/                         # Public packages (if any)
├── docker/
│   ├── Dockerfile
│   └── docker-compose.yml
├── configs/
│   └── example-config.json
├── docs/
│   ├── setup.md
│   └── api.md
├── go.mod
├── go.sum
├── README.md
└── LICENSE
```

## Docker Configuration

### Dockerfile
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o onenote-mcp-server ./cmd/onenote-mcp-server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/onenote-mcp-server .
ENTRYPOINT ["./onenote-mcp-server"]
```

### Docker Compose
```yaml
version: '3.8'
services:
  onenote-mcp-server:
    build: .
    environment:
      - ONENOTE_CLIENT_ID=${ONENOTE_CLIENT_ID}
      - ONENOTE_CLIENT_SECRET=${ONENOTE_CLIENT_SECRET}
      - ONENOTE_TENANT_ID=${ONENOTE_TENANT_ID}
    stdin_open: true
    tty: true
```

## Installation & Usage

### MCP Host Configuration

**Claude Desktop:**
```json
{
  "mcpServers": {
    "onenote": {
      "command": "docker",
      "args": [
        "run", "-i", "--rm",
        "-e", "ONENOTE_CLIENT_ID",
        "-e", "ONENOTE_CLIENT_SECRET", 
        "-e", "ONENOTE_TENANT_ID",
        "onenote-mcp-server"
      ],
      "env": {
        "ONENOTE_CLIENT_ID": "<YOUR_CLIENT_ID>",
        "ONENOTE_CLIENT_SECRET": "<YOUR_CLIENT_SECRET>",
        "ONENOTE_TENANT_ID": "<YOUR_TENANT_ID>"
      }
    }
  }
}
```

**VS Code MCP:**
```json
{
  "mcp": {
    "inputs": [
      {
        "type": "promptString",
        "id": "onenote_client_id",
        "description": "OneNote Client ID",
        "password": false
      },
      {
        "type": "promptString", 
        "id": "onenote_client_secret",
        "description": "OneNote Client Secret",
        "password": true
      }
    ],
    "servers": {
      "onenote": {
        "command": "docker",
        "args": [
          "run", "-i", "--rm",
          "-e", "ONENOTE_CLIENT_ID",
          "-e", "ONENOTE_CLIENT_SECRET",
          "onenote-mcp-server"
        ],
        "env": {
          "ONENOTE_CLIENT_ID": "${input:onenote_client_id}",
          "ONENOTE_CLIENT_SECRET": "${input:onenote_client_secret}"
        }
      }
    }
  }
}
```

## Security Considerations

1. **Token Security**: Implement secure token storage and automatic refresh
2. **Scope Limitation**: Request minimal required permissions (Notes.ReadWrite)
3. **Input Validation**: Validate all tool inputs and sanitize content
4. **Error Handling**: Don't expose sensitive information in error messages
5. **Rate Limiting**: Implement client-side rate limiting for Graph API calls

## Error Handling

1. **Authentication Errors**: Clear guidance on token refresh and re-authentication
2. **API Errors**: Map Graph API errors to meaningful MCP error responses
3. **Network Errors**: Implement retry logic with exponential backoff
4. **Validation Errors**: Provide detailed input validation feedback

## Testing Strategy

1. **Unit Tests**: Test individual tool implementations
2. **Integration Tests**: Test against Microsoft Graph API sandbox
3. **MCP Protocol Tests**: Validate MCP message format compliance
4. **Docker Tests**: Test containerized deployment scenarios

## Performance Requirements

1. **Response Times**: < 2 seconds for most operations
2. **Memory Usage**: < 100MB baseline memory footprint
3. **Concurrent Operations**: Support multiple simultaneous API calls
4. **Caching**: Cache notebooks/sections metadata to reduce API calls

## Documentation Requirements

1. **Setup Guide**: Azure AD app registration and configuration
2. **API Reference**: Complete tool and resource documentation  
3. **Examples**: Common usage patterns and code samples
4. **Troubleshooting**: Common issues and solutions

## Future Enhancements

1. **Webhook Support**: If/when Microsoft Graph adds OneNote webhook support
2. **Attachment Handling**: Support for file attachments and images
3. **Real-time Sync**: Polling mechanism for change detection
4. **Advanced Search**: Support for more complex search queries
5. **Collaboration Features**: Share and permission management

## Success Criteria

1. **Functionality**: All core OneNote operations work reliably
2. **Compatibility**: Works with major MCP hosts (Claude, VS Code)
3. **Performance**: Meets response time and resource usage requirements
4. **Security**: Passes security review for enterprise usage
5. **Documentation**: Complete setup and usage documentation
6. **Community**: Positive feedback from early adopters

## Risks & Mitigation

1. **API Changes**: Monitor Microsoft Graph API versioning and deprecations
2. **Authentication Flow**: Implement robust OAuth error handling
3. **Rate Limits**: Implement intelligent rate limiting and retry logic
4. **Content Formatting**: Handle HTML content sanitization properly

This PRD provides a comprehensive foundation for implementing a OneNote MCP server that follows industry best practices and integrates seamlessly with the MCP ecosystem.

## Appendix: Missing Features from Original PRD

The following features were planned in the original PRD but have not yet been implemented:

### A.1 MCP Resources
Originally planned URI-based resource access:
```
onenote://notebooks/{notebook-id}
onenote://notebooks/{notebook-id}/sections/{section-id}
onenote://notebooks/{notebook-id}/sections/{section-id}/pages/{page-id}
onenote://search/pages?q={query}
onenote://recent/pages?days={days}
```

### A.2 Missing Tools
- `get_notebook` - Get details of a specific notebook
- `create_notebook` - Create a new notebook
- `search_notebooks` - Search notebooks by name/content
- `get_section` - Get details of a specific section
- `update_section` - Update section properties
- `delete_section` - Delete a section
- `get_page` - Get page content and metadata
- `update_page` - Update existing page content (replaced by updatePageContent)
- `append_page_content` - Append content to existing page
- `extract_text` - Extract text content using OCR capabilities

### A.3 Configuration Features
- `--export-translations` command line option
- Tool description overrides via config file
- Full toolset management system

### A.4 Advanced Features Not Implemented
- Webhook support for real-time sync
- OCR capabilities for text extraction
- Attachment handling beyond basic image/file retrieval
- Advanced search with full-text search capabilities
- Collaboration features (sharing and permissions)
- Real-time polling mechanism for change detection

### A.5 Authentication Differences
The original PRD specified OAuth 2.0 with client secret, but the implementation uses the more secure OAuth 2.0 PKCE flow which doesn't require storing client secrets.

## Implemented Authentication Features

### MCP Authentication Tools
The server includes 4 authentication management tools:

1. **`auth_status`** - Check authentication state without exposing tokens
2. **`auth_initiate`** - Start OAuth flow with automatic HTTP server handling
3. **`auth_refresh`** - Manual token refresh capability
4. **`auth_clear`** - Logout and clear stored tokens

### Non-Blocking Startup
Unlike the original design, the server starts immediately without waiting for authentication:
- **No tokens**: Server starts and user can authenticate via `auth_initiate` 
- **Expired tokens**: Server starts and allows re-authentication
- **Valid tokens**: Server starts normally

This provides better user experience and follows MCP patterns where authentication is handled through tools rather than blocking server startup.