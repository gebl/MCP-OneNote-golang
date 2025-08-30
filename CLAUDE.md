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

# Run comprehensive test coverage analysis
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html

# Test authorization security model
./scripts/test-security-model.sh     # Linux/Mac
scripts/test-security-model.bat      # Windows

# Build for Docker
docker build -f docker/Dockerfile -t onenote-mcp-server .

# Run in stdio mode
docker run --rm \
  -e ONENOTE_CLIENT_ID=your-client-id \
  -e ONENOTE_TENANT_ID=common \
  -e ONENOTE_REDIRECT_URI=http://localhost:8080/callback \
  onenote-mcp-server

# Run in HTTP mode
docker run -d -p 8080:8080 \
  -e ONENOTE_CLIENT_ID=your-client-id \
  -e ONENOTE_TENANT_ID=common \
  -e ONENOTE_REDIRECT_URI=http://localhost:8080/callback \
  -e MCP_AUTH_ENABLED=true \
  -e MCP_BEARER_TOKEN=your-secret-token \
  onenote-mcp-server -mode=streamable

# Using Docker Compose
cd docker && docker-compose up onenote-mcp-server       # Stdio mode
cd docker && docker-compose --profile http up           # HTTP mode
```

## Architecture Overview

### Core Components Architecture

The project follows a **modular MCP server architecture** with clear separation of concerns:

**Entry Point (`cmd/onenote-mcp-server/`)**:
- `main.go`: Server initialization, mode selection (stdio/streamable), token management, notebook cache
- `tools.go`: Main MCP registration orchestrator (tools, resources)
- `AuthTools.go`: Authentication-related MCP tools
- `NotebookTools.go`: Notebook and section management MCP tools  
- `PageTools.go`: Page content and item management MCP tools
- `resources.go`: Main MCP resource registration orchestrator
- `NotebookResources.go`: Notebook-related MCP resources (notebooks list, notebook details, notebook sections)
- `SectionResources.go`: Section-related MCP resources (global sections, notebook sections)
- `PageResources.go`: Page-related MCP resources (pages by section, page content for updates)

**Domain-Specific Modules (`internal/`)**:
- `auth/`: OAuth 2.0 PKCE flow, token refresh, secure storage
- `config/`: Multi-source configuration (env vars, JSON files, defaults)
- `graph/`: Microsoft Graph SDK integration and HTTP client
- `http/`: Shared HTTP utilities for automatic resource cleanup and safe request handling
- `notebooks/`, `pages/`, `sections/`: Domain-specific OneNote operations
- `utils/`: Validation, image processing, and text format detection utilities

### Key Architectural Patterns

**Client Composition Pattern**: The main `graph.Client` is composed into specialized domain clients (`NotebookClient`, `PageClient`, `SectionClient`) that add domain-specific operations while sharing the core HTTP/auth functionality.

**Token Refresh Integration**: Authentication failures automatically trigger token refresh across all HTTP operations through the `StaticTokenProvider` and `Client.MakeAuthenticatedRequest` pattern.

**Shared HTTP Utilities (`internal/http/`)**: Centralized HTTP request handling with automatic resource cleanup eliminates manual `defer resp.Body.Close()` patterns throughout the codebase. Provides consistent error handling, request validation, and response processing with utilities like `SafeRequestWithBody`, `SafeRequestWithCustomHandler`, and `WithAutoCleanup`. This architectural pattern ensures proper resource management and reduces code duplication across all HTTP operations in the graph, auth, pages, and sections modules.

**MCP Tool Registration**: Tools are organized into specialized modules (`AuthTools`, `NotebookTools`, `PageTools`) with handlers that create domain clients, call operations, and return standardized MCP responses. Each tool includes comprehensive error handling and logging.

**Global Notebook Cache**: A thread-safe notebook cache system maintains the currently selected notebook and its sections tree for improved performance and user experience. Cache is automatically updated when notebooks change.

**Multi-Layer Caching System**: A comprehensive caching architecture with three levels - page metadata by section ID, page search results by notebook:page key, and notebook lookup results by name. All cache layers provide 5-minute expiration, automatic invalidation on operations, cache-aware progress notifications, and significant performance improvements.

**MCP Resources**: Server provides MCP resources for data discovery, enhancing the development experience.

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
The MCP OneNote Server includes a comprehensive authorization system that provides fine-grained access control over all operations. The system implements a **simplified hierarchical permission model** with secure defaults, current notebook scoping, and powerful pattern matching capabilities.

### Key Features
- **Simplified Model**: No complex tool categories - just notebook-level selection with resource-level permissions
- **Current Notebook Scoping**: All operations must be within a selected notebook context for security
- **Pattern Matching**: Support for wildcards (`*`) and recursive patterns (`**`) for flexible resource matching
- **Fail-Closed Security**: Unknown resources default to no access
- **Resource Filtering**: Automatically filters notebooks, sections, and pages based on permissions
- **Security Monitoring**: Comprehensive logging of all authorization decisions and security violations

### Permission Modes
- **`none`**: No access - operations are denied
- **`read`**: Read-only access - view operations only  
- **`write`**: Full read/write access - all operations allowed
- **`full`**: Same as write (reserved for future use)

### Core Architecture

#### Notebook-Centric Security Model
All operations are scoped to a currently selected notebook:

1. **Authentication Tools**: Always allowed (getAuthStatus, initiateAuth, etc.)
2. **Notebook Discovery**: `listNotebooks` is allowed but results are filtered by permissions
3. **Notebook Selection**: `selectNotebook` validates permission before setting current context
4. **Scoped Operations**: All other tools require a selected notebook and validate access within that scope

#### Security Enforcement
- **Cross-Notebook Protection**: Prevents access to resources outside the current notebook
- **Resource Resolution**: Validates notebook ownership of pages/sections using API calls when needed
- **Fail-Closed Model**: Denies access when resource ownership cannot be determined

### Configuration Structure

#### Core Authorization Settings
```json
{
  "authorization": {
    "enabled": true,                              // Enable/disable authorization system
    "default_notebook_permissions": "read",       // Default permission for any notebook
    "notebook_permissions": {                     // Specific notebook permissions
      "Work*": "write",                          // Pattern: any notebook starting with "Work"
      "Personal Notes": "read",                  // Exact match
      "Archive/**": "none"                       // Recursive pattern: Archive and all sub-paths
    },
    "section_permissions": {                      // Section permissions (with optional notebook prefix)
      "Meeting Notes": "write",                  // Section name only
      "Work*/Drafts": "read",                   // Notebook pattern + section name
      "Personal Notes/Private": "none"          // Full path specification
    },
    "page_permissions": {                        // Page permissions by title
      "Daily Journal": "write",                 // Exact page title match
      "Draft*": "read",                         // Pattern: pages starting with "Draft"
      "**confidential**": "none"                // Pattern: pages containing "confidential"
    }
  }
}
```

### Pattern Matching System

#### Supported Pattern Types

**Exact Matches** (Highest Precedence):
```json
{
  "notebook_permissions": {
    "Work Notebook": "write"         // Exact notebook name
  },
  "page_permissions": {
    "Daily Journal": "write"         // Exact page title
  }
}
```

**Prefix Patterns**:
```json
{
  "notebook_permissions": {
    "Work*": "write",               // Any notebook starting with "Work"
    "Project*": "read"              // Any notebook starting with "Project"
  }
}
```

**Suffix Patterns**:
```json
{
  "page_permissions": {
    "*Notes": "read",               // Any page ending with "Notes"
    "*Draft": "write"               // Any page ending with "Draft"
  }
}
```

**Recursive Patterns** (Lowest Precedence):
```json
{
  "section_permissions": {
    "Archive/**": "none",           // Archive and all subsections
    "**/Private/**": "none"         // Any section path containing "Private"
  }
}
```

#### Pattern Precedence Rules
Patterns are matched in order of specificity (higher precedence wins):

1. **Exact matches**: `Daily Journal`
2. **Simple prefix/suffix**: `Work*`, `*Notes`  
3. **Complex patterns**: `Work*/Draft*`
4. **Recursive patterns**: `**/Private/**`
5. **Default permissions**: Fallback when no pattern matches

### Security Features

#### Fail-Closed Security Model
- **Secure Defaults**: Default to `"none"` or `"read"` permissions
- **Unknown Resources**: Deny access to unrecognized notebooks/sections/pages
- **Cross-Notebook Prevention**: Block access to resources outside current notebook scope
- **Resource Validation**: Verify page/section ownership before allowing operations

#### Current Notebook Enforcement
```json
// Security workflow:
// 1. User calls: selectNotebook("Work Notebook") 
// 2. System validates: notebook permission >= read
// 3. System sets: current notebook context
// 4. All subsequent operations validated within this scope
// 5. Cross-notebook access attempts are blocked and logged
```

#### Comprehensive Security Logging
```bash
# Successful authorization
[auth] Authorization granted: notebook=Work Notebook, operation=write, source=exact_match

# Security violation detected  
[auth] SECURITY VIOLATION: Cross-notebook access attempt blocked
[auth] SECURITY: Denying page access - notebook ownership could not be determined

# Permission denied
[auth] Authorization denied: resource=Archive Notes, permission=none, reason=pattern_match
```

### Configuration Examples

#### QuickNote-Only Access (Restrictive)
Perfect for dedicated note-taking with minimal access:
```json
{
  "authorization": {
    "enabled": true,
    "default_notebook_permissions": "none",      // Block all notebooks by default
    "notebook_permissions": {
      "Personal Notes": "write"                  // Allow only Personal Notes notebook
    },
    "page_permissions": {
      "Daily Journal": "write",                  // Allow quickNote target page
      "Quick Notes": "write",                    // Allow additional note pages
      "Archive*": "none"                         // Block archive pages
    }
  },
  "quicknote": {
    "notebook_name": "Personal Notes",
    "page_name": "Daily Journal"
  }
}
```

#### Development Environment (Permissive with Protection)
Allow broad access while protecting sensitive data:
```json
{
  "authorization": {
    "enabled": true,
    "default_notebook_permissions": "read",      // Read access to most notebooks
    "notebook_permissions": {
      "Development*": "write",                   // Full access to dev notebooks
      "Test*": "write",                         // Full access to test notebooks
      "Production*": "none",                    // Block production data
      "Customer*": "none",                      // Block customer data
      "Personal*": "none"                       // Block personal notebooks
    },
    "section_permissions": {
      "**/Confidential": "none",                // Block confidential sections
      "**/Archive/**": "read"                   // Read-only access to archives
    }
  }
}
```

#### Work Environment (Pattern-Based)
Use patterns for flexible workspace organization:
```json
{
  "authorization": {
    "enabled": true,
    "default_notebook_permissions": "read",
    "notebook_permissions": {
      "Work*": "write",                         // All work notebooks writable
      "Project*": "write",                      // All project notebooks writable
      "Team*": "read",                          // Team notebooks read-only
      "Archive*": "read",                       // Archive notebooks read-only
      "Personal*": "none"                       // Block personal content
    },
    "section_permissions": {
      "Work*/Meeting Notes": "write",           // Meeting notes in work notebooks
      "Project*/Status": "write",               // Status sections in projects
      "**/Private": "none"                      // Block private sections everywhere
    },
    "page_permissions": {
      "*Weekly Report*": "write",               // Weekly reports anywhere
      "*Action Items*": "write",                // Action items anywhere  
      "**confidential**": "none"                // Block confidential pages
    }
  }
}
```

### Resource Filtering

#### Automatic Filtering Behavior
The authorization system automatically filters results from list operations:

**Filtered Operations**:
- `listNotebooks`: Only shows notebooks with `read`, `write`, or `full` permissions
- `getNotebookSections`: Only shows sections accessible within current notebook
- `listPages`: Only shows pages accessible based on section and page permissions

**Security Benefits**:
- Users only see resources they can access
- Prevents information disclosure about restricted resources
- Maintains clean user experience while enforcing security

#### Filtering Examples
```json
// User sees filtered results based on permissions:
{
  "available_notebooks": [
    "Work Notebook",      // Has 'write' permission
    "Project Alpha",      // Has 'read' permission  
    // "Archive" hidden - has 'none' permission
    // "Personal" hidden - blocked by pattern
  ],
  "filtered_info": {
    "total_notebooks": 4,
    "visible_notebooks": 2,
    "filtered_by_authorization": true
  }
}
```

### Environment Variables Support

Authorization can be configured via environment variables:
```bash
# Enable authorization
AUTHORIZATION_ENABLED=true

# Set default permissions
AUTHORIZATION_DEFAULT_NOTEBOOK_PERMISSIONS=read

# Configure permissions (JSON format)
AUTHORIZATION_NOTEBOOK_PERMISSIONS='{"Work*":"write","Archive*":"read"}'
AUTHORIZATION_SECTION_PERMISSIONS='{"**/Private":"none"}'
AUTHORIZATION_PAGE_PERMISSIONS='{"Daily Journal":"write"}'
```

**Note**: JSON configuration takes precedence over environment variables.

### Migration and Troubleshooting

#### Migrating to Authorization
1. **Start Permissive**: Begin with `default_notebook_permissions: "read"`
2. **Test Operations**: Verify all required functionality works
3. **Add Restrictions**: Gradually add `"none"` permissions for sensitive resources
4. **Monitor Logs**: Watch for unexpected authorization denials

#### Common Issues

**Issue**: `"No notebook selected"` error
**Cause**: Attempting operations without calling `selectNotebook` first
**Solution**: Always call `selectNotebook` before other operations

**Issue**: `"Cross-notebook access denied"` 
**Cause**: Trying to access resources outside current notebook
**Solution**: Select the correct notebook before accessing its resources

**Issue**: QuickNote authorization failures
**Cause**: Missing permissions for target notebook or page
**Solution**: Ensure quickNote target has appropriate permissions:
```json
{
  "notebook_permissions": {
    "Personal Notes": "write"    // QuickNote notebook needs write access
  },
  "page_permissions": {
    "Daily Journal": "write"     // QuickNote page needs write access
  }
}
```

#### Debug Authorization Issues
Enable detailed authorization logging:
```json
{
  "log_level": "DEBUG",
  "content_log_level": "DEBUG"
}
```

Look for log messages like:
```bash
[auth] Authorization granted: notebook=Work, operation=write, matched_pattern=Work*
[auth] Authorization denied: resource=Archive, permission=none, matched_pattern=Archive*
[auth] SECURITY VIOLATION: Cross-notebook access attempt detected
```

### Best Practices

#### Security Recommendations
1. **Always Enable in Production**: Set `"enabled": true` for production deployments
2. **Use Secure Defaults**: Start with `"none"` or `"read"` for default permissions
3. **Explicit Write Grants**: Only grant `"write"` permissions where needed
4. **Pattern Strategy**: Use patterns for scalable permission management
5. **Monitor Security Logs**: Watch for security violations and unauthorized access attempts

#### Performance Optimization
1. **Pattern Efficiency**: Exact matches are fastest, complex patterns are slower
2. **Cache Utilization**: Authorization leverages existing cache infrastructure
3. **Early Termination**: Failed authorization stops expensive operations immediately
4. **Structured Logging**: Use appropriate log levels to balance security and performance

#### Configuration Management
1. **Version Control**: Store authorization configs in source control
2. **Environment Separation**: Use different permissions for dev/test/prod
3. **Regular Audits**: Review permissions periodically for security and compliance
4. **Documentation**: Document permission grants and their business justification

## Docker Deployment

### Container Features

The OneNote MCP Server includes production-ready Docker configuration with:

**Security-First Design:**
- Non-root execution with dedicated `mcpuser` account
- Minimal Alpine Linux base image for reduced attack surface
- Read-only configuration mounting
- Secure token persistence via Docker volumes

**Multi-Mode Support:**
- **Stdio Mode**: For local development and Claude Desktop integration
- **HTTP Mode**: For remote access and web-based MCP clients
- **Docker Compose**: Simplified orchestration with environment file support

**Configuration Management:**
- Complete environment variable support for all settings
- JSON configuration file mounting for authorization settings
- Automatic token persistence across container restarts
- Comprehensive logging with configurable verbosity

### Docker Build and Deployment

#### Basic Container Usage

**Stdio Mode (Local Development):**
```bash
docker run --rm \
  -e ONENOTE_CLIENT_ID=your-client-id \
  -e ONENOTE_TENANT_ID=common \
  -e ONENOTE_REDIRECT_URI=http://localhost:8080/callback \
  -v tokens_volume:/app \
  onenote-mcp-server
```

**HTTP Mode (Production):**
```bash
docker run -d -p 8080:8080 \
  -e ONENOTE_CLIENT_ID=your-client-id \
  -e ONENOTE_TENANT_ID=common \
  -e ONENOTE_REDIRECT_URI=http://localhost:8080/callback \
  -e MCP_AUTH_ENABLED=true \
  -e MCP_BEARER_TOKEN=your-secret-token \
  -e AUTHORIZATION_ENABLED=true \
  -v /path/to/config.json:/app/config.json \
  -v tokens_volume:/app \
  onenote-mcp-server -mode=streamable
```

#### Docker Compose Deployment

**Environment Configuration (`docker/.env`):**
```bash
# Azure App Registration
ONENOTE_CLIENT_ID=your-azure-app-client-id
ONENOTE_TENANT_ID=common
ONENOTE_REDIRECT_URI=http://localhost:8080/callback

# Authorization (when using JSON config)
AUTHORIZATION_ENABLED=true
AUTHORIZATION_DEFAULT_MODE=read

# MCP Authentication
MCP_AUTH_ENABLED=true
MCP_BEARER_TOKEN=your-secret-bearer-token

# Logging
LOG_LEVEL=INFO
CONTENT_LOG_LEVEL=INFO
```

**Service Deployment:**
```bash
cd docker

# Stdio mode (default)
docker-compose up onenote-mcp-server

# HTTP mode with authentication
docker-compose --profile http up onenote-mcp-server-http

# Background deployment
docker-compose up -d onenote-mcp-server-http
```

#### Authorization with Docker

**Configuration File Mounting:**
```bash
# Create authorization config
cat > docker/configs/auth-config.json << 'EOF'
{
  "client_id": "your-client-id",
  "tenant_id": "common",
  "redirect_uri": "http://localhost:8080/callback",
  "authorization": {
    "enabled": true,
    "default_mode": "read",
    "tool_permissions": {
      "auth_tools": "full",
      "page_write": "write"
    },
    "notebook_permissions": {
      "Work Notes": "write",
      "Personal": "read"
    }
  }
}
EOF

# Run with authorization
docker run -d -p 8080:8080 \
  -v $(pwd)/docker/configs:/app/configs:ro \
  -v tokens_volume:/app \
  -e ONENOTE_MCP_CONFIG=/app/configs/auth-config.json \
  onenote-mcp-server -mode=streamable
```

### Production Deployment Recommendations

#### Security Hardening
1. **Use HTTPS**: Always use HTTPS in production with proper TLS certificates
2. **Network Isolation**: Deploy in private networks with restricted access
3. **Secrets Management**: Use Docker secrets or external secret management
4. **Resource Limits**: Set appropriate CPU and memory limits
5. **Health Monitoring**: Implement health checks and monitoring

#### Example Production Setup
```yaml
version: '3.8'
services:
  onenote-mcp-server:
    image: onenote-mcp-server:latest
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      - ONENOTE_CLIENT_ID=${ONENOTE_CLIENT_ID}
      - ONENOTE_TENANT_ID=${ONENOTE_TENANT_ID}
      - ONENOTE_REDIRECT_URI=https://your-domain.com/callback
      - MCP_AUTH_ENABLED=true
      - AUTHORIZATION_ENABLED=true
      - LOG_LEVEL=INFO
      - LOG_FORMAT=json
    volumes:
      - ./configs:/app/configs:ro
      - tokens_data:/app
      - ./logs:/app/logs
    deploy:
      resources:
        limits:
          memory: 256M
          cpus: '0.5'
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
    command: ["-mode=streamable"]

volumes:
  tokens_data:
    driver: local
```

#### Container Image Management
```bash
# Build versioned image
docker build -f docker/Dockerfile -t onenote-mcp-server:1.7.0 .
docker tag onenote-mcp-server:1.7.0 onenote-mcp-server:latest

# Multi-architecture build
docker buildx build --platform linux/amd64,linux/arm64 \
  -f docker/Dockerfile -t onenote-mcp-server:1.7.0 .

# Registry deployment
docker tag onenote-mcp-server:1.7.0 your-registry.com/onenote-mcp-server:1.7.0
docker push your-registry.com/onenote-mcp-server:1.7.0
```

### Container Monitoring

**Health Checks:**
```bash
# Check container status
docker ps
docker logs onenote-mcp-server

# Monitor resource usage
docker stats onenote-mcp-server

# Execute health checks
curl -H "Authorization: Bearer your-token" \
  http://localhost:8080/mcp/v1/health
```

**Log Analysis:**
```bash
# Follow logs with JSON formatting
docker logs -f onenote-mcp-server | jq '.'

# Filter authorization events
docker logs onenote-mcp-server 2>&1 | grep authorization

# Export logs for analysis
docker logs onenote-mcp-server > onenote-mcp.log
```

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
- **`cmd/onenote-mcp-server/`**: Configuration loading, tool registration, resources
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
go test ./cmd/onenote-mcp-server -v    # Server initialization, tools, resources
go test ./internal/notebooks -v        # Notebook operations
go test ./internal/pages -v           # Page operations
go test ./internal/sections -v        # Section operations
```

## Recent Improvements and Bug Fixes

### Microsoft Graph API Integration Fixes

**Page Notebook Resolution Enhancement**: Fixed a critical bug in `ResolvePageNotebook()` where Microsoft Graph API v1.0 would return empty parent objects when using `$select` parameter. The solution involved:

- **Root Cause**: Microsoft Graph API v1.0 has a limitation where `$select=parentSection,parentNotebook` on OneNote pages completely omits these fields from the response
- **Solution**: Changed from `$select=id,title,parentSection,parentNotebook` to `$expand=parentSection,parentNotebook` to force inclusion of parent objects
- **Impact**: This fixes authorization failures where pages couldn't be properly associated with their notebooks
- **Enhanced Debugging**: Added comprehensive logging of API responses to troubleshoot similar issues

**Configuration System Enhancements**:
- **Environment Variable Precedence**: Environment variables now properly override JSON configuration values
- **Default Toolsets**: Added sensible defaults for toolsets when not specified
- **Enhanced Error Handling**: Improved error messages and validation during configuration loading

### Comprehensive Test Coverage

**Test Suite Expansion**: Added over 10,000 lines of unit tests providing comprehensive coverage:

- **Domain Testing**: Full test coverage for notebooks, pages, sections, and authorization modules  
- **HTTP Layer Testing**: Comprehensive testing of Graph API integration and HTTP utilities
- **Configuration Testing**: Full validation of configuration loading from multiple sources
- **Authorization Testing**: Complete test coverage of the security model with various scenarios
- **Functional Testing**: End-to-end testing of core workflows and operations

**Security Model Testing**: Added dedicated test scripts for validation:
- `scripts/test-security-model.sh` (Linux/Mac)
- `scripts/test-security-model.bat` (Windows)

**Test Infrastructure**:
- **Mock-Based Testing**: Uses `testify/mock` for controlled Microsoft Graph API simulation
- **Coverage Analysis**: Support for HTML coverage reports with `go tool cover`
- **Isolated Testing**: Each test package can run independently without external dependencies

### Enhanced Debugging and Diagnostics

**Structured Logging Improvements**:
- **Component-Based Logging**: All modules use consistent structured logging with component prefixes
- **Enhanced API Debugging**: Raw JSON responses logged for Microsoft Graph API calls
- **Performance Metrics**: Cache hit/miss ratios and operation durations tracked
- **Security Event Logging**: Comprehensive logging of authorization decisions and security violations

**Development Tooling**:
- **Coverage Scripts**: Automated test coverage analysis and HTML report generation  
- **Security Testing**: Dedicated scripts for testing authorization scenarios
- **Enhanced Error Messages**: More descriptive error messages with context for troubleshooting

### Configuration and Environment Handling

**Environment Variable Processing**: Enhanced handling with proper precedence rules:
- Environment variables take precedence over JSON configuration
- Improved validation and error reporting for configuration issues
- Support for complex nested configurations via environment variables

**JSON Configuration Enhancements**:
- Better error handling for malformed JSON files
- Support for partial configuration overrides
- Enhanced validation of configuration values

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