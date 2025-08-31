# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Quick Start Commands

### Build and Run
```bash
# Build server
go build -o onenote-mcp-server.exe ./cmd/onenote-mcp-server

# Run modes
./onenote-mcp-server.exe                    # stdio mode (default)
./onenote-mcp-server.exe -mode=streamable  # HTTP mode on port 8080
./onenote-mcp-server.exe -mode=streamable -port=8081 # Custom port
```

### Development
```bash
go mod tidy                # Dependencies
go test ./...              # All tests
go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out -o coverage.html  # Coverage

# Test specific packages
go test ./cmd/onenote-mcp-server -v    # Server tests
go test ./internal/notebooks -v        # Notebook operations
go test ./internal/pages -v           # Page operations
go test ./internal/sections -v        # Section operations

# Security model testing
./scripts/test-security-model.sh       # Linux/Mac
scripts/test-security-model.bat        # Windows
```

### Docker
```bash
# Build image
docker build -f docker/Dockerfile -t onenote-mcp-server .

# Run stdio mode
docker run --rm -e ONENOTE_CLIENT_ID=your-id -e ONENOTE_TENANT_ID=common -e ONENOTE_REDIRECT_URI=http://localhost:8080/callback onenote-mcp-server

# Run HTTP mode
docker run -d -p 8080:8080 -e ONENOTE_CLIENT_ID=your-id -e MCP_AUTH_ENABLED=true -e MCP_BEARER_TOKEN=token onenote-mcp-server -mode=streamable

# Docker Compose
cd docker && docker-compose up onenote-mcp-server       # Stdio
cd docker && docker-compose --profile http up           # HTTP
```

## Architecture

### Core Structure
**Entry Point (`cmd/onenote-mcp-server/`)**:
- `main.go`: Server initialization, modes, token management, notebook cache
- `*Tools.go`: MCP tool handlers (Auth, Notebook, Page)
- `*Resources.go`: MCP resource providers
- `tools.go`, `resources.go`: Registration orchestrators

**Internal Modules (`internal/`)**:
- `auth/`: OAuth 2.0 PKCE, token refresh, storage
- `config/`: Multi-source configuration (env, JSON, defaults)
- `graph/`: Microsoft Graph SDK integration
- `http/`: Shared HTTP utilities with auto-cleanup
- `notebooks/`, `pages/`, `sections/`: Domain operations
- `utils/`: Validation, image processing, text format detection

### Key Patterns
- **Client Composition**: Specialized domain clients (NotebookClient, PageClient, SectionClient) compose the main graph.Client
- **Auto Token Refresh**: Authentication failures trigger automatic token refresh
- **Shared HTTP Utilities**: Centralized request handling with auto-cleanup via `SafeRequestWithBody`, `WithAutoCleanup`
- **Multi-Layer Caching**: Page metadata (5min), search results, notebook lookups with smart invalidation
- **Progress Notifications**: Real-time progress for long operations via MCP streaming
- **Authorization System**: Granular permissions with pattern matching and fail-closed security

## Configuration

### Required Environment Variables
```bash
ONENOTE_CLIENT_ID=your-azure-app-client-id
ONENOTE_TENANT_ID=your-azure-tenant-id
ONENOTE_REDIRECT_URI=http://localhost:8080/callback
```

### Optional Configuration
```bash
ONENOTE_DEFAULT_NOTEBOOK_NAME="My Notebook"
ONENOTE_TOOLSETS="notebooks,sections,pages,content"
ONENOTE_MCP_CONFIG="configs/config.json"

# QuickNote
QUICKNOTE_NOTEBOOK_NAME="Personal Notes"
QUICKNOTE_PAGE_NAME="Daily Journal"
QUICKNOTE_DATE_FORMAT="January 2, 2006 - 3:04 PM"

# HTTP Mode Authentication
MCP_AUTH_ENABLED="true"
MCP_BEARER_TOKEN="your-secret-token"

# Logging
LOG_LEVEL="INFO"                     # DEBUG, INFO, WARN, ERROR
LOG_FORMAT="text"                    # text, json
CONTENT_LOG_LEVEL="INFO"             # Content logging: DEBUG, INFO, WARN, ERROR, OFF
```

### JSON Configuration Example
```json
{
  "client_id": "your-azure-app-client-id",
  "tenant_id": "your-azure-tenant-id", 
  "redirect_uri": "http://localhost:8080/callback",
  "notebook_name": "My Notebook",
  "toolsets": ["notebooks", "sections", "pages", "content"],
  "quicknote": {
    "notebook_name": "Personal Notes",
    "page_name": "Daily Journal",
    "date_format": "January 2, 2006 - 3:04 PM"
  },
  "authorization": {
    "enabled": true,
    "default_notebook_permissions": "read",
    "notebook_permissions": {
      "Work*": "write",
      "Personal Notes": "write",
      "Archive*": "read"
    },
    "page_permissions": {
      "Daily Journal": "write",
      "**confidential**": "none"
    }
  },
  "mcp_auth": {
    "enabled": true,
    "bearer_token": "your-token"
  },
  "log_level": "INFO",
  "content_log_level": "INFO"
}
```

### Azure App Requirements
- **API Permissions**: `Notes.ReadWrite` (delegated)
- **Redirect URI**: `http://localhost:8080/callback`
- **Authentication**: OAuth 2.0 PKCE (no client secret needed)

## QuickNote Tool

Rapid note-taking that appends timestamped entries to a configured OneNote page.

**Features:**
- Timestamped entries with H3 date/time headers
- Auto-format detection: HTML, Markdown, ASCII text
- Smart page lookup with multi-layer caching
- Append-only operation preserving existing content

**Configuration:**
```json
{
  "quicknote": {
    "notebook_name": "Personal Notes",    // Optional - falls back to default
    "page_name": "Daily Journal",         // Required
    "date_format": "January 2, 2006 - 3:04 PM"  // Optional Go time format
  }
}
```

**Usage:**
```json
{
  "method": "tools/call",
  "params": {
    "name": "quickNote", 
    "arguments": {
      "content": "Meeting notes:\n- Discussed API design\n- **Action**: Review by Friday"
    }
  }
}
```

## Authorization System

### Overview
Comprehensive permission-based access control with notebook-centric security model.

**Key Features:**
- Current notebook scoping - all operations within selected notebook context
- Pattern matching with wildcards (`*`) and recursive patterns (`**`)
- Fail-closed security - unknown resources default to no access
- Resource filtering - automatically filters lists based on permissions

**Permission Modes:** `none`, `read`, `write`, `full`

### Pattern Examples
```json
{
  "authorization": {
    "enabled": true,
    "default_notebook_permissions": "read",
    "notebook_permissions": {
      "Work*": "write",                    // Prefix pattern
      "Personal Notes": "write",           // Exact match
      "Archive/**": "none"                 // Recursive pattern
    },
    "section_permissions": {
      "Meeting Notes": "write",            // Section name
      "Work*/Drafts": "read",             // Notebook + section
      "**/Private": "none"                // Any private section
    },
    "page_permissions": {
      "Daily Journal": "write",           // Exact page
      "Draft*": "read",                   // Prefix pattern
      "**confidential**": "none"          // Contains pattern
    }
  }
}
```

### Security Workflow
1. `selectNotebook("Work Notebook")` - validates notebook permission
2. Sets current notebook context
3. All operations validated within scope
4. Cross-notebook access blocked and logged

## OneNote Operations

### Container Hierarchy
OneNote enforces strict relationships:
- **Notebooks** → sections, section groups
- **Section Groups** → sections, other section groups
- **Sections** → pages only (NOT sections/groups)

### Advanced Page Updates
`updatePageContentAdvanced` uses command-based targeting:
- Get data-id values with `getPageContent(forUpdate=true)`
- Target: `data-id:element-123`, `title`, or element selectors
- Commands: `append`, `insert`, `replace`, `delete`
- Table restriction: update tables as complete units

### Image Processing
Auto-optimization via `utils/image.go`:
- Size validation (max 1024x768 by default)
- Format detection from Content-Type/HTML attributes
- Base64 encoding for JSON transport
- `fullSize` parameter bypasses scaling

## Authentication

### Server Startup
Server starts immediately without blocking for auth:
- No tokens: empty auth state
- Expired tokens: allows re-auth via MCP tools
- Valid tokens: normal startup with auth active

### MCP Auth Tools
- `getAuthStatus`: Check current state
- `initiateAuth`: Start OAuth flow
- `refreshToken`: Manual token refresh
- `clearAuth`: Clear tokens (logout)

### Workflow
1. Start server: `./onenote-mcp-server.exe`
2. Check: `getAuthStatus` tool
3. Auth: `initiateAuth` tool → visit URL
4. Auto token refresh on API failures

## MCP Resources

Hierarchical URI structure for data discovery:

**Notebooks:**
- `onenote://notebooks` - All notebooks with metadata
- `onenote://notebooks/{name}` - Specific notebook details
- `onenote://notebooks/{name}/sections` - Notebook sections tree

**Sections:**
- `onenote://sections` - Global sections across notebooks
- `onenote://notebooks/{name}/sections` - Notebook-specific sections

**Pages:**
- `onenote://pages/{sectionId}` - Pages in section
- `onenote://page/{pageId}` - Page HTML with data-id attributes

## Development Notes

- **Tokens**: Default `tokens.json`, configurable via `auth.GetTokenPath()`
- **Logging**: Structured with component prefixes `[tools]`, `[graph]`, `[auth]`
- **Two-Phase Logging**: DEBUG during config load → configured level after
- **Content Logging**: `CONTENT_LOG_LEVEL` controls HTML/JSON payload verbosity
- **Validation**: OneNote illegal chars: `?*\\/:<>|&#''%%~`
- **Caching**: 5-minute page metadata cache with auto-invalidation
- **Progress**: Real-time notifications for long operations
- **Error Handling**: `mcp.NewToolResultError()` for failures, `mcp.NewToolResultText()` for success

## Recent Improvements

### Microsoft Graph API Fixes
- **Page Resolution Bug**: Fixed empty parent objects with `$select` by switching to `$expand`
- **Configuration**: Environment variables now properly override JSON config

### Test Coverage
- 10,000+ lines of unit tests across all modules
- Mock-based testing with `testify/mock`
- Dedicated security model test scripts
- HTML coverage reports with `go tool cover`

### Enhanced Logging
- Component-based structured logging
- Raw JSON response debugging for Graph API
- Performance metrics and cache hit/miss tracking
- Security event logging for authorization decisions

## Testing

```bash
# All tests with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html

# Component-specific
go test ./cmd/onenote-mcp-server -v    # Server, config, tools
go test ./internal/notebooks -v        # Notebook operations  
go test ./internal/pages -v           # Page CRUD, content, images
go test ./internal/sections -v        # Section ops, hierarchy
```

**Philosophy:** Unit tests for business logic, mocking for Graph API, no E2E (handled by mcp-go library).