# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Development Commands

### Core Build Commands
```bash
# Build the main server executable
go build -o onenote-mcp-server.exe ./cmd/onenote-mcp-server

# Run with different modes
./onenote-mcp-server.exe                    # stdio mode (default)
./onenote-mcp-server.exe -mode=streamable  # Streamable HTTP mode on port 8080
./onenote-mcp-server.exe -mode=streamable -port=8081 # Custom port
```

### Development Commands
```bash
# Download and clean dependencies
go mod tidy

# Run all tests
go test ./...

# Run specific package tests
go test ./cmd/onenote-mcp-server     # Main server tests (configuration, tools, prompts)
go test ./internal/notebooks         # Notebook operations
go test ./internal/pages             # Page operations  
go test ./internal/sections          # Section operations

# Build for Docker
docker build -t onenote-mcp-server .
docker run -p 8080:8080 onenote-mcp-server
```

## Architecture Overview

### Core Components Architecture

The project follows a **modular MCP server architecture** with clear separation of concerns:

**Entry Point (`cmd/onenote-mcp-server/`)**:
- `main.go`: Server initialization, mode selection (stdio/streamable), token management, notebook cache
- `tools.go`: Main MCP registration orchestrator (tools, resources, completions)
- `AuthTools.go`: Authentication-related MCP tools
- `NotebookTools.go`: Notebook and section management MCP tools  
- `PageTools.go`: Page content and item management MCP tools
- `resources.go`: Main MCP resource registration orchestrator
- `NotebookResources.go`: Notebook-related MCP resources (notebooks list, notebook details, notebook sections)
- `SectionResources.go`: Section-related MCP resources (global sections, notebook sections)
- `PageResources.go`: Page-related MCP resources (pages by section, page content for updates)
- `completions.go`: MCP completion definitions for autocomplete support

**Domain-Specific Modules (`internal/`)**:
- `auth/`: OAuth 2.0 PKCE flow, token refresh, secure storage
- `config/`: Multi-source configuration (env vars, JSON files, defaults)
- `graph/`: Microsoft Graph SDK integration and HTTP client
- `notebooks/`, `pages/`, `sections/`: Domain-specific OneNote operations
- `utils/`: Validation, image processing, and text format detection utilities

### Key Architectural Patterns

**Client Composition Pattern**: The main `graph.Client` is composed into specialized domain clients (`NotebookClient`, `PageClient`, `SectionClient`) that add domain-specific operations while sharing the core HTTP/auth functionality.

**Token Refresh Integration**: Authentication failures automatically trigger token refresh across all HTTP operations through the `StaticTokenProvider` and `Client.MakeAuthenticatedRequest` pattern.

**MCP Tool Registration**: Tools are organized into specialized modules (`AuthTools`, `NotebookTools`, `PageTools`) with handlers that create domain clients, call operations, and return standardized MCP responses. Each tool includes comprehensive error handling and logging.

**Global Notebook Cache**: A thread-safe notebook cache system maintains the currently selected notebook and its sections tree for improved performance and user experience. Cache is automatically updated when notebooks change.

**Multi-Layer Caching System**: A comprehensive caching architecture with three levels - page metadata by section ID, page search results by notebook:page key, and notebook lookup results by name. All cache layers provide 5-minute expiration, automatic invalidation on operations, cache-aware progress notifications, and significant performance improvements.

**MCP Resources and Completions**: Server provides MCP resources for data discovery and completions for autocomplete support, enhancing the development experience.

**Container Hierarchy Validation**: OneNote's strict container hierarchy (Notebooks → Section Groups → Sections → Pages) is enforced through `determineContainerType` validation in section operations.

**Progress Notification System**: MCP tools support real-time progress notifications for long-running operations. Progress tokens are extracted from request metadata and used to send incremental progress updates to clients. HTTP request logging middleware is disabled in streamable mode to prevent interference with progress streaming.

**Intelligent Text Format Detection**: Advanced text processing utilities automatically detect content format (HTML, Markdown, ASCII) and convert to appropriate HTML for OneNote. Uses `gomarkdown` library for high-quality Markdown to HTML conversion with support for tables, code blocks, lists, and other common Markdown features.

**Granular Authorization System**: Comprehensive permission-based access control system providing fine-grained control over MCP tool access, OneNote operations, and resource permissions. Supports hierarchical permission inheritance with explicit overrides, secure default-deny policies, and flexible configuration through JSON files.

## Special Tools

### QuickNote Tool

The `quickNote` tool provides rapid note-taking functionality that appends timestamped entries to a pre-configured OneNote page. This tool is ideal for quick journaling, meeting notes, or capturing thoughts on the fly.

**Features:**
- **Timestamped entries**: Each note is prefixed with an H3 heading containing the current date/time
- **Intelligent format detection**: Automatically detects if content is HTML, Markdown, or plain ASCII text
- **Multi-format support**: Converts Markdown to HTML, preserves HTML as-is, and wraps ASCII text in proper HTML paragraphs
- **Configurable formatting**: Date format is customizable via Go time layout strings
- **Smart page lookup**: Finds the target page by searching through all sections using cached listPages calls with multi-layer result caching
- **Performance optimized**: Leverages three-layer caching system (page metadata, search results, notebook lookups) for fast repeated operations
- **Cache-aware progress notifications**: Provides real-time progress updates that differentiate between cached and API operations
- **Append-only operation**: New content is always appended to the page body, preserving existing content

**Configuration Required:**
```json
{
  "quicknote": {
    "notebook_name": "Personal Notes",    // Target notebook name (optional - falls back to notebook_name)
    "page_name": "Daily Journal",         // Target page name (required)
    "date_format": "January 2, 2006 - 3:04 PM"  // Go time format layout (optional)
  }
}
```

**Fallback Behavior:**
- If `quicknote.notebook_name` is not specified, the tool will use the default `notebook_name` from the main configuration
- If neither is specified, the tool will return an error
- Only `page_name` is strictly required in the quicknote configuration

**Usage Examples:**

*Plain Text:*
```json
{
  "method": "tools/call",
  "params": {
    "name": "quickNote",
    "arguments": {
      "content": "Had a great idea for the project - implement real-time collaboration features"
    }
  }
}
```

*Markdown Content:*
```json
{
  "method": "tools/call",
  "params": {
    "name": "quickNote",
    "arguments": {
      "content": "# Meeting Notes\n\n- Discussed new features\n- **Action item**: Review API design\n- *Deadline*: End of week"
    }
  }
}
```

*HTML Content:*
```json
{
  "method": "tools/call",
  "params": {
    "name": "quickNote",
    "arguments": {
      "content": "<p>This is <strong>HTML</strong> content with <em>formatting</em></p>"
    }
  }
}
```

**Output Format:**
The tool detects the content format and converts it appropriately:

*Plain Text Output:*
```html
<h3>January 15, 2025 - 2:30 PM</h3>
<p>Had a great idea for the project - implement real-time collaboration features</p>
```

*Markdown Content Output:*
```html
<h3>January 15, 2025 - 2:30 PM</h3>
<h1 id="meeting-notes">Meeting Notes</h1>
<ul>
<li>Discussed new features</li>
<li><strong>Action item</strong>: Review API design</li>
<li><em>Deadline</em>: End of week</li>
</ul>
```

*HTML Content Output:*
```html
<h3>January 15, 2025 - 2:30 PM</h3>
<p>This is <strong>HTML</strong> content with <em>formatting</em></p>
```

**Error Handling:**
- Validates that page_name is configured in quicknote settings
- Falls back to default notebook_name if quicknote.notebook_name is not set
- Returns clear error messages if no notebook name is available (neither quicknote.notebook_name nor default notebook_name)
- Returns clear error messages if target notebook or page cannot be found
- Handles authentication and API errors gracefully with detailed logging

## Configuration Requirements

### Required Environment Variables
```bash
ONENOTE_CLIENT_ID=your-azure-app-client-id
ONENOTE_TENANT_ID=your-azure-tenant-id
ONENOTE_REDIRECT_URI=http://localhost:8080/callback
```

### Optional Configuration
```bash
ONENOTE_DEFAULT_NOTEBOOK_NAME="My Notebook"  # Default notebook for operations
ONENOTE_TOOLSETS="notebooks,sections,pages,content"  # Enabled toolsets
ONENOTE_MCP_CONFIG="configs/config.json"  # JSON config file path

# QuickNote Configuration
QUICKNOTE_NOTEBOOK_NAME="Personal Notes"  # Target notebook for quicknote entries
QUICKNOTE_PAGE_NAME="Daily Journal"       # Target page for quicknote entries  
QUICKNOTE_DATE_FORMAT="January 2, 2006 - 3:04 PM"  # Go time format for timestamps

# MCP Authentication Configuration (for HTTP mode only)
MCP_AUTH_ENABLED="true"    # Enable bearer token authentication
MCP_BEARER_TOKEN="your-secret-bearer-token"  # Bearer token for HTTP authentication

# Logging Configuration (can also be set in JSON config file)
LOG_LEVEL="DEBUG"          # General logging level: DEBUG, INFO, WARN, ERROR
LOG_FORMAT="text"          # Log format: "text" or "json"
MCP_LOG_FILE="mcp-server.log"  # File-based logging
CONTENT_LOG_LEVEL="DEBUG"  # Content logging verbosity: DEBUG, INFO, WARN, ERROR, OFF
```

### JSON Configuration File Example
```json
{
  "client_id": "your-azure-app-client-id",
  "tenant_id": "your-azure-tenant-id",
  "redirect_uri": "http://localhost:8080/callback",
  "notebook_name": "My Default Notebook",
  "toolsets": ["notebooks", "sections", "pages", "content"],
  "authorization": {
    "enabled": true,
    "default_mode": "read",
    "default_tool_mode": "read",
    "default_notebook_mode": "none",
    "default_section_mode": "read",
    "default_page_mode": "none",
    "tool_permissions": {
      "auth_tools": "full",
      "page_write": "write",
      "notebook_management": "read"
    },
    "notebook_permissions": {
      "My Notebook": "write",
      "Work Notes": "read"
    },
    "section_permissions": {
      "Meeting Notes": "write",
      "Projects": "read"
    },
    "page_permissions": {
      "Daily Journal": "write",
      "Quick Notes": "write"
    }
  },
  "quicknote": {
    "notebook_name": "Personal Notes",
    "page_name": "Daily Journal",
    "date_format": "January 2, 2006 - 3:04 PM"
  },
  "mcp_auth": {
    "enabled": true,
    "bearer_token": "your-secret-bearer-token-here"
  },
  "log_level": "DEBUG",
  "log_format": "text",
  "log_file": "mcp-server.log",
  "content_log_level": "DEBUG"
}
```

### Azure App Registration Requirements
- **API Permissions**: `Notes.ReadWrite` (delegated)
- **Redirect URI**: `http://localhost:8080/callback`
- **Authentication**: OAuth 2.0 PKCE flow (no client secret needed)

### MCP Server Authentication (HTTP mode only)
The server supports bearer token authentication for securing HTTP transport mode:
- **Stdio mode**: No authentication needed (runs locally)
- **HTTP mode**: Optional bearer token authentication via `Authorization: Bearer <token>` header
- **Configuration**: Set `MCP_AUTH_ENABLED=true` and `MCP_BEARER_TOKEN=your-secret-token`
- **Security**: Use HTTPS in production when bearer token authentication is enabled
- **Logging**: All authentication attempts are logged for security monitoring

## Authorization System

### Overview
The MCP OneNote Server includes a comprehensive authorization system that provides fine-grained access control over all operations. The system implements a hierarchical permission model with secure defaults and explicit overrides.

### Permission Modes
- **`none`**: No access - operations are denied
- **`read`**: Read-only access - view operations only
- **`write`**: Full read/write access - all operations allowed
- **`full`**: Administrative access (for tool permissions only)

### Configuration Structure

#### Core Authorization Settings
```json
{
  "authorization": {
    "enabled": true,                    // Enable/disable authorization system
    "default_mode": "read",             // Global default permission level
    "default_tool_mode": "read",        // Default for tool access
    "default_notebook_mode": "none",    // Default for notebook access
    "default_section_mode": "read",     // Default for section access  
    "default_page_mode": "none"         // Default for page access
  }
}
```

#### Permission Hierarchies

**Tool-Level Permissions** (`tool_permissions`):
Controls access to specific MCP tool categories:
- `auth_tools`: Authentication tools (getAuthStatus, initiateAuth, etc.)
- `notebook_management`: Notebook creation, deletion, and management
- `section_management`: Section creation, deletion, and management  
- `page_write`: Page creation, content updates, and deletion
- `page_read`: Page content reading and listing
- `content_management`: Advanced content operations

**Resource-Level Permissions**:
Controls access to specific OneNote resources by name:
- `notebook_permissions`: Permissions by notebook display name
- `section_permissions`: Permissions by section display name
- `page_permissions`: Permissions by page title

#### Permission Resolution Order
The system evaluates permissions in this hierarchical order:
1. **Tool Permission Check**: Does the user have access to the tool category?
2. **Resource-Specific Permission**: Explicit permissions for the target resource
3. **Resource-Type Default**: Default permission for the resource type (notebook/section/page)
4. **Global Default**: Fallback to the global default mode

### Security Features

#### Default-Deny Security Model
- **Secure Defaults**: Most defaults are set to `"none"` or `"read"` 
- **Explicit Grants**: Write access must be explicitly configured
- **Principle of Least Privilege**: Users get minimum necessary permissions

#### Authentication Integration
- **OAuth Integration**: Authorization checks occur after OAuth authentication
- **Token Validation**: Expired or invalid tokens result in authentication errors before authorization
- **Session Security**: Authorization state is evaluated per-request

#### Audit and Logging
- **Permission Decisions**: All authorization decisions are logged with context
- **Access Attempts**: Failed authorization attempts are logged for security monitoring
- **Structured Logging**: Authorization logs include user context, resource details, and decision rationale

### Configuration Examples

#### Restrictive Configuration (Default-Deny)
```json
{
  "authorization": {
    "enabled": true,
    "default_mode": "none",
    "default_tool_mode": "none", 
    "default_notebook_mode": "none",
    "default_section_mode": "none",
    "default_page_mode": "none",
    "tool_permissions": {
      "auth_tools": "full",
      "page_read": "read"
    },
    "notebook_permissions": {
      "Public Notes": "read"
    },
    "page_permissions": {
      "Daily Journal": "write"
    }
  }
}
```

#### Permissive Configuration (Read-Heavy)
```json
{
  "authorization": {
    "enabled": true,
    "default_mode": "read",
    "default_tool_mode": "read",
    "default_notebook_mode": "read", 
    "default_section_mode": "read",
    "default_page_mode": "read",
    "tool_permissions": {
      "auth_tools": "full",
      "page_write": "write"
    },
    "notebook_permissions": {
      "Work Notes": "write",
      "Archive": "none"
    }
  }
}
```

#### QuickNote-Focused Configuration
```json
{
  "authorization": {
    "enabled": true,
    "default_mode": "read",
    "default_tool_mode": "read",
    "default_notebook_mode": "none",
    "default_page_mode": "none",
    "tool_permissions": {
      "auth_tools": "full",
      "page_write": "write"
    },
    "notebook_permissions": {
      "Personal Notes": "write"
    },
    "page_permissions": {
      "Daily Journal": "write",
      "Quick Notes": "write"
    }
  }
}
```

### Usage Scenarios

#### Read-Only Access for Browsing
Enable users to browse and read OneNote content without modification capabilities:
```json
{
  "authorization": {
    "enabled": true,
    "default_mode": "read",
    "default_tool_mode": "read",
    "tool_permissions": {
      "auth_tools": "full"
    }
  }
}
```

#### Dedicated Note-Taking Setup
Configure access for a specific note-taking workflow:
```json
{
  "authorization": {
    "enabled": true,
    "default_mode": "read",
    "default_notebook_mode": "none",
    "default_page_mode": "none",
    "tool_permissions": {
      "auth_tools": "full",
      "page_write": "write"
    },
    "notebook_permissions": {
      "Meeting Notes": "write"
    },
    "page_permissions": {
      "Action Items": "write",
      "Daily Standup": "write"
    }
  }
}
```

#### Development and Testing Environment
Allow broader access for development while protecting sensitive notebooks:
```json
{
  "authorization": {
    "enabled": true,
    "default_mode": "read",
    "default_tool_mode": "write",
    "tool_permissions": {
      "auth_tools": "full"
    },
    "notebook_permissions": {
      "Production Data": "none",
      "Test Notebook": "write"
    }
  }
}
```

### Best Practices

#### Security Recommendations
1. **Enable Authorization**: Always enable the authorization system in production
2. **Use Secure Defaults**: Start with restrictive defaults and explicitly grant permissions
3. **Regular Audits**: Review permission configurations periodically
4. **Monitor Logs**: Watch authorization logs for unexpected access patterns
5. **Principle of Least Privilege**: Grant only the minimum permissions needed

#### Configuration Management
1. **Version Control**: Store authorization configurations in version control
2. **Environment Separation**: Use different configurations for dev/staging/production
3. **Documentation**: Document permission grants and their business justification
4. **Testing**: Test permission configurations with representative use cases

#### Performance Considerations
1. **Caching**: Permission decisions are cached per request for performance
2. **Early Termination**: Authorization failures fail fast without expensive operations
3. **Logging Efficiency**: Use appropriate log levels to balance security and performance

## OneNote Operations Architecture

### Container Hierarchy Rules
OneNote enforces strict container relationships that the code validates:
- **Notebooks**: Can contain sections and section groups
- **Section Groups**: Can contain sections and other section groups  
- **Sections**: Can contain pages only (NOT other sections/section groups)

### Advanced Page Content Updates
The `updatePageContentAdvanced` tool uses command-based updates targeting specific HTML elements:
- **Get data-id values**: Use `getPageContent` with `forUpdate=true`
- **Target elements**: Use `data-id:element-123`, `title`, or element selectors
- **Commands**: `append`, `insert`, `replace`, `delete` with position options
- **Table restrictions**: Tables must be updated as complete units, never individual cells

### Image Processing Pipeline
Images are automatically optimized through `utils/image.go`:
- **Size validation**: Large images are scaled down for performance (1024x768 max by default)
- **Format detection**: Content-Type extraction from HTML data-src-type attributes and HTTP headers
- **Base64 encoding**: For JSON transport in MCP responses
- **Full-size option**: `getPageItem` supports `fullSize` parameter to bypass automatic scaling
- **Smart filename generation**: Generates appropriate filenames with extensions based on content type

## Authentication Management

### **Server Startup Behavior**
The server now starts immediately without blocking for authentication:

- **No tokens**: Server starts with empty authentication state
- **Expired tokens**: Server starts and allows re-authentication via MCP tools  
- **Valid tokens**: Server starts normally with authentication active

### **MCP Authentication Tools**

**`getAuthStatus`**: Check current authentication state
```json
{
  "authenticated": true,
  "tokenExpiry": "2025-01-15T10:30:00Z", 
  "tokenExpiresIn": "45 minutes",
  "refreshTokenAvailable": true,
  "authMethod": "OAuth2_PKCE"
}
```

**`initiateAuth`**: Start new OAuth authentication flow
```json
{
  "authUrl": "https://login.microsoftonline.com/...",
  "instructions": "Visit this URL in your browser to authenticate",
  "localServerPort": 8080,
  "timeoutMinutes": 10
}
```

**`refreshToken`**: Manually refresh authentication tokens
```json
{
  "authenticated": true,
  "message": "Token refreshed successfully"
}
```

**`clearAuth`**: Clear stored tokens (logout)
```json
{
  "success": true,
  "message": "Authentication tokens cleared. Use initiateAuth to re-authenticate."
}
```

### **Authentication Workflow**

**Initial Setup**:
1. Start server: `./onenote-mcp-server.exe` (starts immediately)
2. Check status: Use `getAuthStatus` MCP tool
3. Authenticate: Use `initiateAuth` MCP tool
4. Visit provided URL in browser to complete OAuth flow
5. Server automatically receives callback and updates tokens

**Token Management**:
1. `TokenManager.IsExpired()` checks expiration before requests
2. `OAuth2Config.RefreshToken()` obtains new tokens  
3. `Client` automatically retries failed requests after refresh
4. New tokens saved to disk for persistence
5. MCP tools allow manual token refresh and re-authentication

## MCP Features

### MCP Resources
The server provides resources for data discovery through a hierarchical URI structure:

#### Notebook Resources
- **`onenote://notebooks`**: Lists all available notebooks with metadata
- **`onenote://notebooks/{name}`**: Get specific notebook details by display name
- **`onenote://notebooks/{name}/sections`**: Hierarchical view of sections and section groups within a notebook

#### Section Resources  
- **`onenote://sections`**: Get all sections across all notebooks using global sections endpoint
- **`onenote://notebooks/{name}/sections`**: Get sections within a specific notebook

#### Page Resources
- **`onenote://pages/{sectionIdOrName}`**: List all pages in a specific section by either section ID or display name, returning page titles and IDs
- **`onenote://page/{pageId}`**: Get HTML content for a specific page with data-id attributes included for update operations

#### Resource Features
- **Dynamic Content**: Resources are generated from live OneNote data
- **Progress Notifications**: Long-running resource requests support progress streaming
- **URI Encoding**: All names and IDs are properly URL-encoded in resource URIs
- **Integration**: MCP clients can discover and access OneNote information through standardized resource URIs

### MCP Completions  
The server provides intelligent autocomplete support:
- **Notebook Name Completion**: Autocomplete notebook names in tool parameters
- **Context-Aware**: Completions based on current user's OneNote data
- **Performance**: Cached results for fast autocomplete responses

### Default Notebook Management
- **Automatic Initialization**: Server attempts to set a default notebook on startup
- **Configuration Priority**: Uses `ONENOTE_DEFAULT_NOTEBOOK_NAME` if configured
- **Fallback**: Selects first available notebook if no default configured
- **Thread-Safe Cache**: Global notebook cache maintains current selection
- **Authentication Aware**: Only initializes when authentication is available

### Multi-Layer Caching System
- **Page Metadata Caching**: Pages are cached by section ID for efficient retrieval with 5-minute expiration
- **Page Search Result Caching**: Caches page search results by notebook:page key for the `quickNote` tool and other page lookup operations
- **Notebook Lookup Caching**: Caches detailed notebook information by name to avoid repeated API calls for notebook resolution
- **Fresh Cache Policy**: All cache layers expire after 5 minutes to ensure data freshness
- **Automatic Population**: Caches are populated during normal operations (`listPages`, page searches, notebook lookups)
- **Smart Invalidation**: Caches are cleared when pages are created, deleted, or moved
- **Cache-Aware Progress Notifications**: Progress notifications differentiate between cache hits and API operations for better user experience
- **Performance Optimization**: Subsequent calls for the same data return cached results instantly
- **Memory Efficient**: Only caches metadata (not full content) with proper memory management
- **Thread-Safe Operations**: All cache operations are protected by read-write mutexes
- **Cache Status Reporting**: Responses include cache hit/miss status and performance metrics
- **Structured Debug Logging**: Enhanced logging with key-value pairs for troubleshooting cache behavior

## Testing Strategy

### Test Coverage
The project has comprehensive unit tests covering core functionality:
- **`cmd/onenote-mcp-server/`**: Configuration loading, tool registration, resources, completions
- **`internal/notebooks/`**: Notebook operations and Microsoft Graph integration
- **`internal/pages/`**: Page CRUD operations, content handling, image processing
- **`internal/sections/`**: Section operations, container hierarchy validation

### Testing Philosophy
- **Unit Tests**: Focus on business logic and OneNote operations
- **Integration**: Protocol-level functionality (JSON-RPC, MCP) is handled by the `mcp-go` library
- **Mocking**: Uses testify/mock for Microsoft Graph API interactions
- **No E2E**: Stdio/HTTP protocol testing removed as it's handled by the MCP library

### Running Tests
```bash
# All tests
go test ./...

# Specific components
go test ./cmd/onenote-mcp-server -v    # Server initialization, tools, resources, completions
go test ./internal/notebooks -v        # Notebook operations
go test ./internal/pages -v           # Page operations
go test ./internal/sections -v        # Section operations
```

## Development Notes

- **Token Path**: Default is `tokens.json` in working directory, configurable via `auth.GetTokenPath()`
- **Logging**: All modules use structured logging with component prefixes like `[tools]`, `[graph]`, `[auth]`
- **Content Logging**: Configurable verbosity for logging actual HTML content and JSON payloads
  - Can be configured via environment variables or JSON config file (env vars take precedence)
  - Set `CONTENT_LOG_LEVEL=DEBUG` (default during development) to see all content
  - Set `CONTENT_LOG_LEVEL=OFF` to disable content logging for performance  
  - Content logging includes: CreatePage HTML content, UpdatePageContent commands and JSON
- **Two-Phase Logging Initialization**:
  - **Phase 1**: Maximum verbosity (DEBUG) during config loading to capture all setup details
  - **Phase 2**: Configured verbosity level after config is loaded and processed
  - This ensures config loading problems are always visible regardless of final log level
- **Error Handling**: MCP tools return `mcp.NewToolResultError()` for failures, `mcp.NewToolResultText()` for success
- **Validation**: Display names validated against OneNote illegal characters: `?*\\/:<>|&#''%%~`
- **Pagination**: All list operations automatically handle Microsoft Graph pagination
- **Caching Strategy**: Page metadata is cached for 5 minutes per section; cache automatically invalidated on create/delete/move operations
- **Cache Logging**: Cache hits and misses are logged at DEBUG level with performance metrics
- **Enhanced Response Format**: `listPages` returns structured responses with cache status:
  ```json
  {
    "pages": [...],
    "cached": true,
    "cache_hit": true,
    "pages_count": 5,
    "duration": "1.234ms"
  }
  ```
- **Progress Notifications**: MCP tools support real-time progress streaming for long-running operations like `getNotebookSections`. Progress tokens are extracted from request metadata and incremental progress notifications are sent to clients.
- **HTTP Transport Optimization**: Request logging middleware is disabled in streamable HTTP mode to prevent interference with progress notification streaming and improve performance.