# OneNote MCP Server

**Created by Gabriel Lawrence**

A Go-based Model Context Protocol (MCP) server that provides seamless integration with Microsoft OneNote via the Microsoft Graph API. This project was created to experiment with AI-assisted development using Claude, GitHub Copilot, and Cursor, while learning about MCP server architecture and capabilities.

This server enables AI assistants and other MCP clients to read, create, update, and manage OneNote notebooks, sections, pages, and embedded content with enterprise-grade features including comprehensive authorization, caching, and intelligent content processing.

## 🚀 Features

### 🎯 Rapid Note-Taking
- **QuickNote Tool**: Intelligent rapid note-taking with automatic timestamping, format detection (HTML/Markdown/ASCII), and smart page lookup
- **Multi-Format Support**: Automatic conversion between Markdown, HTML, and plain text with proper formatting preservation
- **Configurable Templates**: Customizable date formats and target notebook/page configurations

### 🔐 Enterprise Security & Authorization
- **Granular Authorization System**: Comprehensive permission-based access control with hierarchical inheritance
- **OAuth 2.0 PKCE Flow**: Secure authentication using Proof Key for Code Exchange with non-blocking startup
- **Default-Deny Security Model**: Secure defaults with explicit permission grants following principle of least privilege
- **Audit Logging**: Complete authorization decision logging with security monitoring capabilities
- **MCP Authentication Tools**: Built-in tools for authentication management (`getAuthStatus`, `initiateAuth`, `refreshToken`, `clearAuth`)

### 🚀 High-Performance Architecture
- **Multi-Layer Caching System**: Three-tier caching (page metadata, search results, notebook lookups) with 5-minute expiration
- **Progress Notification System**: Real-time progress updates for long-running operations with cache-aware notifications
- **Thread-Safe Operations**: Global notebook cache with concurrent-safe access patterns
- **Smart Cache Invalidation**: Automatic cache updates when content changes occur

### 📚 Core OneNote Operations
- **Notebook Management**: Complete notebook operations with automatic pagination and caching
- **Section & Section Group Operations**: Full hierarchical container management with strict validation
- **Advanced Page Operations**: Create, read, update, delete with intelligent content targeting and table handling
- **Content Management**: Rich HTML content support with format detection and absolute positioning
- **Page Item Handling**: Extract and manage embedded images, files, and objects with optimization
- **Search Capabilities**: Recursive search through all sections and section groups within notebooks

### 🌐 MCP Protocol Integration
- **MCP Resources**: Hierarchical URI-based resource discovery (`onenote://notebooks`, `onenote://sections`, etc.)
- **MCP Completions**: Intelligent autocomplete support for notebook names and context-aware suggestions
- **Streamlined MCP Protocol**: Focused on tools and resources following current MCP specification
- **Multi-Mode Support**: stdio and streamable HTTP protocols with built-in SSE streaming

### 🧠 Intelligent Content Processing
- **Format Detection**: Automatic detection and conversion between HTML, Markdown, and ASCII text
- **Image Optimization**: Smart image scaling and processing with configurable size limits
- **Table Handling**: Advanced table update restrictions to prevent layout corruption
- **Note Tags Support**: Built-in OneNote note tags with data-tag attribute support
- **Absolute Positioning**: Support for absolutely positioned elements on OneNote pages

### 🐳 Production-Ready Deployment
- **Docker Support**: Multi-stage builds with security-first design and non-root execution
- **Docker Compose**: Simplified orchestration with environment file support and multiple profiles
- **Configuration Management**: Multi-source configuration (environment variables, JSON files, defaults)
- **Health Monitoring**: Built-in health checks and comprehensive logging with configurable verbosity

## 📋 Prerequisites

1. **Microsoft Azure App Registration**:
   - Create an app registration in Azure Portal
   - Configure OAuth 2.0 redirect URI (e.g., `http://localhost:8080/callback`)
   - Grant `Notes.ReadWrite` API permissions

2. **Go Environment** (for development):
   - Go 1.21 or later
   - Git for version control

3. **Docker** (for containerized deployment):
   - Docker Engine 20.10 or later
   - Docker Compose (optional)

## 🛠️ Setup & Installation

### Option 1: Local Development Setup

1. **Clone the repository**:
   ```bash
   git clone <repository-url>
   cd MCP-OneNote-golang
   ```

2. **Set environment variables**:
   ```bash
   export ONENOTE_CLIENT_ID="your-azure-app-client-id"
   export ONENOTE_TENANT_ID="your-azure-tenant-id"
   export ONENOTE_REDIRECT_URI="http://localhost:8080/callback"
   export ONENOTE_DEFAULT_NOTEBOOK_NAME="My Notebook"  # Optional
   ```

3. **Build and run**:
   ```bash
   go build -o onenote-mcp-server ./cmd/onenote-mcp-server
   
   # Run in different modes:
   ./onenote-mcp-server                    # stdio mode (default)
   ./onenote-mcp-server -mode=streamable  # Streamable HTTP mode on port 8080
   ./onenote-mcp-server -mode=streamable -port=8081 # Streamable HTTP mode on custom port
   ```

### Option 2: Docker Deployment

1. **Create configuration file**:
   ```bash
   # Copy the example and customize with your Azure app details
   cp configs/example-config.json configs/config.json
   # Edit configs/config.json with your actual credentials
   ```

2. **Build and run with Docker**:
   ```bash
   docker build -t onenote-mcp-server .
   
   # Run in different modes:
   docker run -p 8080:8080 onenote-mcp-server                    # stdio mode (default)
   docker run -p 8080:8080 onenote-mcp-server -mode=streamable  # Streamable HTTP mode
   docker run -p 8081:8081 onenote-mcp-server -mode=streamable -port=8081 # Streamable HTTP mode on custom port
   ```

3. **Or use Docker Compose**:
   ```bash
   docker-compose up -d
   ```

## 🔧 Configuration

### Server Modes

The OneNote MCP Server supports two different modes for client communication:

#### 1. **stdio Mode (Default)**
- **Usage**: `./onenote-mcp-server` or `./onenote-mcp-server -mode=stdio`
- **Description**: Standard input/output communication for direct integration
- **Best for**: CLI tools, direct process communication, development

#### 2. **Streamable HTTP Mode**
- **Usage**: `./onenote-mcp-server -mode=streamable [-port=8080]`
- **Description**: HTTP-based communication with built-in Server-Sent Events (SSE) streaming for progress notifications
- **Best for**: HTTP clients, web applications, real-time updates, integration with HTTP-based systems
- **Endpoint**: `http://localhost:8080`
- **Features**: Includes SSE streaming for real-time progress updates during long-running operations

### Environment Variables

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| **Core Configuration** | | | |
| `ONENOTE_CLIENT_ID` | Azure App Registration Client ID | Yes | - |
| `ONENOTE_TENANT_ID` | Azure Tenant ID (use "common" for multi-tenant) | Yes | - |
| `ONENOTE_REDIRECT_URI` | OAuth2 redirect URI | Yes | - |
| `ONENOTE_DEFAULT_NOTEBOOK_NAME` | Default notebook name | No | - |
| `ONENOTE_TOOLSETS` | Comma-separated list of enabled toolsets | No | All |
| `ONENOTE_MCP_CONFIG` | Path to JSON configuration file | No | - |
| **QuickNote Configuration** | | | |
| `QUICKNOTE_NOTEBOOK_NAME` | Target notebook for quick notes | No | Falls back to default |
| `QUICKNOTE_PAGE_NAME` | Target page for quick notes | No | - |
| `QUICKNOTE_DATE_FORMAT` | Go time format for timestamps | No | Default format |
| **Authorization Configuration** | | | |
| `AUTHORIZATION_ENABLED` | Enable authorization system | No | false |
| `AUTHORIZATION_DEFAULT_MODE` | Global default permission | No | read |
| `AUTHORIZATION_CONFIG` | Path to authorization JSON config | No | - |
| **MCP Authentication (HTTP mode)** | | | |
| `MCP_AUTH_ENABLED` | Enable bearer token auth for HTTP mode | No | false |
| `MCP_BEARER_TOKEN` | Bearer token for HTTP authentication | No | - |
| **Logging Configuration** | | | |
| `LOG_LEVEL` | General logging level (DEBUG/INFO/WARN/ERROR) | No | INFO |
| `LOG_FORMAT` | Log format (text/json) | No | text |
| `MCP_LOG_FILE` | Path to log file | No | Console only |
| `CONTENT_LOG_LEVEL` | Content logging verbosity (DEBUG/INFO/WARN/ERROR/OFF) | No | INFO |

### Configuration File

Create `configs/config.json` with full feature support:
```json
{
  "client_id": "your-azure-app-client-id",
  "tenant_id": "your-azure-tenant-id", 
  "redirect_uri": "http://localhost:8080/callback",
  "notebook_name": "My Default Notebook",
  "toolsets": ["notebooks", "sections", "pages", "content"],
  
  "quicknote": {
    "notebook_name": "Personal Notes",
    "page_name": "Daily Journal", 
    "date_format": "January 2, 2006 - 3:04 PM"
  },
  
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
      "Personal Notes": "write",
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
  
  "mcp_auth": {
    "enabled": true,
    "bearer_token": "your-secret-bearer-token"
  },
  
  "log_level": "INFO",
  "log_format": "text",
  "content_log_level": "INFO"
}
```

## ⚡ QuickNote Tool

The `quickNote` tool provides rapid note-taking functionality with intelligent format detection and automatic timestamping:

### Features
- **Automatic Timestamping**: Each note gets an H3 timestamp heading in configurable format
- **Smart Format Detection**: Automatically detects and converts HTML, Markdown, or ASCII text  
- **Configurable Target**: Set specific notebook and page for quick notes via configuration
- **Performance Optimized**: Uses multi-layer caching for fast repeated operations
- **Append-Only**: New content is always appended, preserving existing page content

### Configuration
```json
{
  "quicknote": {
    "notebook_name": "Personal Notes",        // Target notebook (optional - falls back to default)
    "page_name": "Daily Journal",             // Target page (required)
    "date_format": "January 2, 2006 - 3:04 PM"  // Go time format (optional)
  }
}
```

### Usage Examples
```bash
# Plain text note
{"name": "quickNote", "arguments": {"content": "Had a great idea for the project"}}

# Markdown note  
{"name": "quickNote", "arguments": {"content": "# Meeting Notes\n\n- Action item: Review API\n- **Deadline**: EOW"}}

# HTML note
{"name": "quickNote", "arguments": {"content": "<p>This is <strong>formatted</strong> content</p>"}}
```

Each note automatically gets formatted with timestamp and proper HTML conversion:
```html
<h3>January 15, 2025 - 2:30 PM</h3>
<p>Your note content here...</p>
```

## 🔐 Authorization System

The server includes a comprehensive authorization system providing fine-grained access control over all operations:

### Permission Modes
- **`none`**: No access - operations are denied
- **`read`**: Read-only access - view operations only  
- **`write`**: Full read/write access - all operations allowed
- **`full`**: Administrative access (for tool permissions only)

### Hierarchical Permission Structure
1. **Tool-Level Permissions**: Control access to MCP tool categories (`auth_tools`, `notebook_management`, `page_write`, etc.)
2. **Resource-Level Permissions**: Specific permissions for notebooks, sections, and pages by name
3. **Default Permissions**: Fallback permissions for resource types
4. **Global Default**: Ultimate fallback permission level

### Security Features
- **Default-Deny Model**: Secure defaults with explicit permission grants
- **Principle of Least Privilege**: Minimum necessary permissions
- **Audit Logging**: All authorization decisions logged with context
- **OAuth Integration**: Works seamlessly with authentication flow

### Example Configurations

**Restrictive Setup** (Default-deny):
```json
{
  "authorization": {
    "enabled": true,
    "default_mode": "none",
    "tool_permissions": {"auth_tools": "full", "page_read": "read"},
    "notebook_permissions": {"Public Notes": "read"},
    "page_permissions": {"Daily Journal": "write"}
  }
}
```

**QuickNote-Focused Setup**:
```json
{
  "authorization": {
    "enabled": true,
    "default_mode": "read",
    "default_notebook_mode": "none",
    "tool_permissions": {"auth_tools": "full", "page_write": "write"},
    "notebook_permissions": {"Personal Notes": "write"},
    "page_permissions": {"Daily Journal": "write"}
  }
}
```

## 🎯 Available MCP Tools & Resources

### MCP Resources

The server provides hierarchical URI-based resources for data discovery:

#### Notebook Resources
- **`onenote://notebooks`**: Lists all available notebooks with metadata
- **`onenote://notebooks/{name}`**: Get specific notebook details by display name  
- **`onenote://notebooks/{name}/sections`**: Hierarchical view of sections and section groups within a notebook

#### Section Resources
- **`onenote://sections`**: Get all sections across all notebooks using global sections endpoint
- **`onenote://notebooks/{name}/sections`**: Get sections within a specific notebook

#### Page Resources  
- **`onenote://pages/{sectionIdOrName}`**: List all pages in a specific section by either section ID or display name
- **`onenote://page/{pageId}`**: Get HTML content for a specific page with data-id attributes for update operations

**Features**: Dynamic content generation from live OneNote data, progress notifications for long-running requests, proper URI encoding, and seamless MCP client integration.

### MCP Completions

The server includes completion infrastructure that is **prepared but not yet active**:
- **Notebook Name Completion**: Full implementation ready for notebook name autocomplete
- **Context-Aware Logic**: Fuzzy matching and relevance sorting algorithms implemented
- **Performance Optimized**: Cached results with intelligent filtering
- **⚠️ Status**: Waiting for completion support in the mcp-go library
- **Ready for Activation**: Complete handler implementation exists in `completions.go`

*Note: Completion functionality will be automatically enabled when the underlying mcp-go library adds completion support.*

### MCP Tools

### Notebook Operations
- **`listNotebooks`**: List all OneNote notebooks for the user
  - Returns: Array of notebook metadata (ID, display name)
  - **Note:** Automatically handles pagination to return all notebooks

### Section Operations
- **`listSections`**: List all sections in a notebook or section group
      - Parameters: `containerId` (optional) - notebook ID or section group ID. If not provided, uses the default notebook from config
  - Returns: Array of section metadata (ID, display name)
  - **Note:** Automatically handles pagination to return all sections

- **`createSection`**: Create a new section in a notebook or section group
    - Parameters: `containerId` (optional), `displayName` (required) - containerId can be notebook ID or section group ID. If not provided, uses the default notebook from config
    - Returns: Created section metadata
    - **Note:** Display name cannot contain: ?*\\/:<>|&#''%%~
    - **Container Hierarchy:** Sections can only be created inside notebooks or section groups, not inside other sections

### Section Group Operations
- **`listSectionGroups`**: List all section groups in a notebook or section group
    - Parameters: `containerId` (optional) - notebook ID or section group ID. If not provided, uses the default notebook from config
    - Returns: Array of section group metadata with parent information
    - **Note:** Each section group includes parent notebook, parent section group, or direct parent container details to show hierarchy
    - **Container Hierarchy:** Section groups can only be listed from notebooks or other section groups, not from sections

- **`createSectionGroup`**: Create a new section group in a notebook or section group
    - Parameters: `containerId` (optional), `displayName` (required) - containerId can be notebook ID or section group ID. If not provided, uses the default notebook from config
    - Returns: Created section group metadata
    - **Note:** Display name cannot contain: ?*\\/:<>|&#''%%~
    - **Container Hierarchy:** Section groups can only be created inside notebooks or other section groups, not inside sections

### Copy Operations
- **`copyPage`**: Copy a page to another section (asynchronous)
      - Parameters: `pageId` (required), `targetSectionId` (required)
  - Returns: Operation status and new page ID

### Page Operations
- **`listPages`**: List all pages in a section
  - Parameters: `sectionId` (required)
  - Returns: Array of page metadata (ID, title)
  - **Note:** Automatically handles pagination to return all pages

- **`createPage`**: Create a new page in a section
    - Parameters: `sectionId`, `title`, `content` (all required)
    - Returns: Created page metadata
    - **Note:** Title cannot contain: ?*\\/:<>|&#''%%~

- **`updatePageContent`**: Update page HTML content (simple replacement)
  - Parameters: `pageId`, `content` (all required)
  - Returns: Success confirmation
- **`updatePageContentAdvanced`**: Update specific parts of a page using advanced commands (preferred method)
  - Parameters: `pageId` (required), `commands` (required JSON array)
  - Returns: Success confirmation
  - **How to get data-id values:** Use `getPageContent` with `forUpdate=true` to retrieve HTML with `data-id` attributes. These values can be used as targets in your update commands.
  - **Preferred Usage:** Use this tool to add, change, or delete parts of a page. Only use a full page update if you intend to replace the entire page content.
  - **CRITICAL: Table Update Restrictions:** Tables must be updated as complete units. You CANNOT update individual table cells (td), headers (th), or rows (tr). Always target the entire table element and replace with complete table HTML to prevent layout corruption.
  - **How to use the data-tag attribute for built-in note tags:**
    - Use the `data-tag` attribute to add and update check boxes, stars, and other built-in note tags on a OneNote page.
    - To add or update a built-in note tag, just use the `data-tag` attribute on a supported element.
    - You can define a `data-tag` on the following elements: `p`, `ul`, `ol`, `li` (see more about note tags on lists), `img`, `h1`-`h6`, `title`.
    - Guidelines for lists:
      - Use `p` elements for to-do lists (no bullet/number, easier to update).
      - To create/update lists with the same note tag for all items, define `data-tag` on the `ul` or `ol`.
      - To create/update lists with unique note tags for some/all items, define `data-tag` on `li` elements and do not nest them in a `ul` or `ol`.
      - To update specific `li` elements, target them individually and define `data-tag` on the `li`.
    - Microsoft Graph rules:
      - The `data-tag` on a `ul` or `ol` overrides all child `li` elements.
      - Unique `data-tag` settings are honored for list items only if the `li` elements are not nested in a `ul` or `ol`, or if an `li` is individually addressed in an update.
      - Unnested `li` elements sent in input HTML are returned in a `ul` in the output HTML.
      - In output HTML, all data-tag list settings are defined on `span` elements on the list items.
    - **Possible data-tag values:**
      - shape[:status], to-do, to-do:completed, important, question, definition, highlight, contact, address, phone-number, web-site-to-visit, idea, password, critical, project-a, project-b, remember-for-later, movie-to-see, book-to-read, music-to-listen-to, source-for-article, remember-for-blog, discuss-with-person-a, discuss-with-person-a:completed, discuss-with-person-b, discuss-with-person-b:completed, discuss-with-manager, discuss-with-manager:completed, send-in-email, schedule-meeting, schedule-meeting:completed, call-back, call-back:completed, to-do-priority-1, to-do-priority-1:completed, to-do-priority-2, to-do-priority-2:completed, client-request, client-request:completed
  - **How to position elements absolutely on a OneNote page:**
    - The body element must specify `data-absolute-enabled="true"`. If omitted or set to false, all body content is rendered inside a _default absolute positioned div and all position settings are ignored.
    - Only `div`, `img`, and `object` elements can be absolute positioned elements.
    - Absolute positioned elements must specify `style="position:absolute"`.
    - Absolute positioned elements must be direct children of the body element. Any direct children of the body that aren't absolute positioned `div`, `img`, or `object` elements are rendered as static content inside the absolute positioned _default div.
    - Absolute positioned elements are positioned at their specified `top` and `left` coordinates, relative to the 0:0 starting position at the top, left corner of the page above the title area.
    - If an absolute positioned element omits the `top` or `left` coordinate, the missing coordinate is set to its default value: `top:120px` or `left:48px`.
    - Absolute positioned elements cannot be nested or contain positioned elements. The API ignores any position settings specified on nested elements inside an absolute positioned div, renders the nested content inside the absolute positioned parent div, and returns a warning in the api.diagnostics property in the response.
  - **Examples:**
    - Append content to a section:
      ```json
      [
        {"target": "data-id:section-123", "action": "append", "content": "<p>New content</p>"}
      ]
      ```
    - Replace the title:
      ```json
      [
        {"target": "title", "action": "replace", "content": "New Page Title"}
      ]
      ```
    - Delete a specific element:
      ```json
      [
        {"target": "data-id:item-456", "action": "delete"}
      ]
      ```
    - Insert content before an element:
      ```json
      [
        {"target": "data-id:section-789", "action": "insert", "position": "before", "content": "<div>Inserted above</div>"}
      ]
      ```
    - **Note:** For `append` actions, do not include the `position` parameter as it will be automatically excluded from the API request.

- **`deletePage`**: Delete a page by ID
  - Parameters: `pageId` (required)
  - Returns: Success confirmation

- **`searchPages`**: Search pages by title within a specific notebook
  - Parameters: `query` (required), `notebookId` (optional)
  - Returns: Array of matching page metadata with section context
  - **Note:** Recursively searches all sections and section groups in the notebook. If notebookId is not provided, uses the default notebook from config

- **`copyPage`**: Copy a page from one section to another using Microsoft Graph API (asynchronous)
  - Parameters: `pageId`, `targetSectionId` (both required)
  - Returns: New page ID and operation metadata
  - **Note:** Automatically handles asynchronous operation polling and completion

- **`getOnenoteOperation`**: Get status of asynchronous OneNote operations
  - Parameters: `operationId` (required)
  - Returns: Operation status and metadata
  - **Note:** Primarily used internally, but available for manual operation tracking

- **`movePage`**: Move a page from one section to another (copy then delete)
  - Parameters: `pageId`, `targetSectionId` (both required)
  - Returns: Moved page metadata with operation details

### Content Operations
- **`getPageContent`**: Get HTML content of a page
  - Parameters: `pageId` (required), `forUpdate` (optional string, set to 'true' to include data-id attributes for advanced updates)
  - Returns: HTML content as string
  - **Tip:** Use `forUpdate=true` to extract `data-id` values for use with advanced page updates (see below).

- **`listPageItems`**: List embedded items (images, files) in a page
  - Parameters: `pageId` (required)
  - Returns: Array of page item metadata with HTML attributes

- **`getPageItem`**: Get complete page item data (content + metadata)
  - Parameters: `pageId`, `pageItemId` (both required)
  - Returns: JSON with base64-encoded content and metadata

### Special Tools
- **`quickNote`**: Rapid note-taking with automatic timestamping and format detection
  - Parameters: `content` (required) - Text content in HTML, Markdown, or plain text format
  - Returns: Success confirmation with formatted content preview
  - **Features:**
    - Automatic timestamp header in configurable format
    - Smart format detection and conversion (HTML/Markdown/ASCII)
    - Configurable target notebook and page via configuration
    - Append-only operation preserving existing content
    - Multi-layer caching for fast repeated operations
  - **Configuration Required:** Set `quicknote.notebook_name` and `quicknote.page_name` in config
  - **Note:** Falls back to default notebook if `quicknote.notebook_name` not configured

### Authentication Operations
- **`getAuthStatus`**: Get current authentication status and token information
  - Parameters: None
  - Returns: Authentication state, token expiry, refresh availability
  - **Note:** Never exposes actual token values, only metadata

- **`initiateAuth`**: Start new OAuth authentication flow  
  - Parameters: None
  - Returns: Browser URL and instructions for OAuth completion
  - **Note:** Starts temporary HTTP server on port 8080 for OAuth callback

- **`refreshToken`**: Manually refresh authentication tokens
  - Parameters: None  
  - Returns: Updated authentication status after refresh
  - **Note:** Requires valid refresh token to be available

- **`clearAuth`**: Clear stored authentication tokens (logout)
  - Parameters: None
  - Returns: Success confirmation
  - **Note:** Requires `initiateAuth` to re-authenticate after clearing

## 🔐 Authentication Flow

The server uses OAuth 2.0 PKCE (Proof Key for Code Exchange) flow with **non-blocking startup**:

### **Server Startup (Non-Blocking)**
1. **Quick Start**: Server starts immediately without waiting for authentication
2. **Any State**: Works with no tokens, expired tokens, or valid tokens
3. **User Choice**: Authentication happens through MCP tools when needed

### **Authentication via MCP Tools**
4 authentication tools are available:

- **`getAuthStatus`**: Check current authentication state and token information
- **`initiateAuth`**: Start OAuth flow - returns browser URL for user authentication  
- **`refreshToken`**: Manually refresh tokens to extend session
- **`clearAuth`**: Logout and clear all stored tokens

### **OAuth Flow Details**
1. **Call `initiateAuth`**: Server generates OAuth URL and starts local HTTP server on port 8080
2. **Browser Authentication**: User visits provided Microsoft login URL
3. **Automatic Callback**: Server receives authorization code and exchanges for tokens
4. **Token Storage**: Tokens saved locally and automatically refreshed before expiration

### **Example Workflow**
```bash
# 1. Start server (immediate, no blocking)
./onenote-mcp-server

# 2. Check authentication status  
mcp > getAuthStatus
{"authenticated": false, "message": "No authentication tokens found"}

# 3. Initiate authentication
mcp > initiateAuth  
{"authUrl": "https://login.microsoftonline.com/...", "instructions": "Visit this URL..."}

# 4. User visits URL in browser, completes OAuth flow

# 5. Check status again
mcp > getAuthStatus
{"authenticated": true, "tokenExpiresIn": "59 minutes", "authMethod": "OAuth2_PKCE"}

# 6. Use OneNote operations
mcp > listNotebooks
[{"id": "...", "displayName": "My Notebook"}]
```

## 📁 Project Structure

```
MCP-OneNote-golang/
├── cmd/onenote-mcp-server/         # Main application entry point
│   ├── main.go                     # Server initialization, mode selection, caching
│   ├── tools.go                    # Main MCP tool registration orchestrator
│   ├── AuthTools.go                # Authentication-related MCP tools
│   ├── NotebookTools.go            # Notebook and section management tools
│   ├── PageTools.go                # Page content and item management tools  
│   ├── TestTools.go                # Testing and diagnostic tools
│   ├── resources.go                # Main MCP resource registration orchestrator
│   ├── NotebookResources.go        # Notebook-related MCP resources
│   ├── SectionResources.go         # Section-related MCP resources
│   ├── PageResources.go            # Page-related MCP resources
│   └── completions.go              # MCP completion definitions
├── internal/                       # Domain-specific modules
│   ├── auth/                       # OAuth2 authentication and token management
│   │   └── auth.go                 # PKCE flow, token refresh, secure storage
│   ├── authorization/              # Granular permission system
│   │   └── authorization.go        # Permission evaluation and enforcement
│   ├── config/                     # Multi-source configuration management
│   │   └── config.go               # Environment vars, JSON files, defaults
│   ├── graph/                      # Microsoft Graph SDK integration
│   │   └── client.go               # HTTP client and authentication
│   ├── notebooks/                  # Notebook domain operations
│   │   └── notebooks.go            # Notebook CRUD and management
│   ├── pages/                      # Page domain operations
│   │   └── pages.go                # Page content, items, and formatting
│   ├── sections/                   # Section domain operations
│   │   └── sections.go             # Section and section group operations
│   ├── utils/                      # Shared utilities
│   │   ├── validation.go           # Input validation and sanitization
│   │   └── image.go                # Image processing and optimization
│   └── logging/                    # Structured logging system
│       └── logger.go               # Configurable logging with levels
├── configs/                        # Configuration files and examples
│   ├── example-config.json         # Example configuration template
│   └── authorization-examples.json # Authorization configuration examples
├── docker/                         # Production-ready containerization
│   ├── Dockerfile                  # Multi-stage security-first build
│   ├── docker-compose.yml          # Multi-mode orchestration
│   └── .env.example                # Environment configuration template
├── docs/                           # Comprehensive documentation
│   ├── api.md                      # API reference and examples
│   ├── setup.md                    # Detailed setup instructions
│   ├── authorization-integration.md # Authorization system guide
│   └── tool-to-resource-conversion.md
├── CLAUDE.md                       # AI development guidance and architecture
└── README.md                       # This file
```

### Architecture Highlights

**Modular MCP Server Design**: Clear separation between entry point (`cmd/`), domain logic (`internal/`), and configuration (`configs/`) following Go best practices.

**Client Composition Pattern**: Core `graph.Client` composed into specialized domain clients (`NotebookClient`, `PageClient`, `SectionClient`) sharing HTTP/auth functionality.

**Comprehensive Authorization**: Hierarchical permission system with tool-level, resource-level, and default permissions supporting fine-grained access control.

**Multi-Layer Caching**: Thread-safe caching system with page metadata, search results, and notebook lookup caches providing 5-minute expiration and automatic invalidation.

**Progressive Enhancement**: Starts with basic MCP functionality and adds enterprise features like authorization, caching, progress notifications, and intelligent content processing.

## 🚀 Usage Examples

### Basic Notebook Operations
```bash
# List all notebooks
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{"name": "listNotebooks", "arguments": {}}'

# List sections in a notebook
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{"name": "listSections", "arguments": {"notebookId": "notebook-id"}}'

# Create a new section
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{"name": "createSection", "arguments": {"containerId": "notebook-id", "displayName": "New Section"}}'
```

### QuickNote Usage
```bash
# Plain text quick note
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "quickNote",
    "arguments": {
      "content": "Had a breakthrough idea for the user interface design!"
    }
  }'

# Markdown quick note with formatting
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "quickNote", 
    "arguments": {
      "content": "# Meeting Summary\n\n- **Decision**: Use React for frontend\n- **Action Item**: Research component libraries\n- *Deadline*: Friday EOD"
    }
  }'

# HTML quick note  
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "quickNote",
    "arguments": {
      "content": "<p>Important: <strong>Server maintenance</strong> scheduled for <em>tonight at 11 PM</em></p>"
    }
  }'
```

### Page Management
```bash
# Create a new page
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "createPage",
    "arguments": {
      "sectionId": "section-id",
      "title": "My New Page",
      "content": "<h1>Hello World</h1><p>This is my new page.</p>"
    }
  }'

# Update page content
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "updatePageContent",
    "arguments": {
      "pageId": "page-id",
      "content": "<h1>Updated Content</h1><p>This content was updated.</p>"
    }
  }'

# Copy page to different section (asynchronous operation)
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "copyPage",
    "arguments": {
      "pageId": "page-id",
      "targetSectionId": "target-section-id"
    }
  }'

# Check operation status (optional)
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "getOnenoteOperation",
    "arguments": {
      "operationId": "operation-id-from-copy-response"
    }
  }'

# Move page to different section
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "movePage",
    "arguments": {
      "pageId": "page-id",
      "targetSectionId": "target-section-id"
    }
  }'

### Content Extraction
```bash
# Get page content
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{"name": "getPageContent", "arguments": {"pageId": "page-id"}}'

# List embedded items
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{"name": "listPageItems", "arguments": {"pageId": "page-id"}}'

# Get page item with content
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "getPageItem",
    "arguments": {
      "pageId": "page-id",
      "pageItemId": "item-id"
    }
  }'
```

## 🆕 Recent Improvements (v1.7.0+)

### Enhanced Container Validation and Error Handling
- **Improved Container Type Detection**: All section and section group operations now use `determineContainerType` for upfront validation
- **Clear Error Messages**: Better error messages that explain exactly why operations fail and what container types are allowed
- **OneNote Hierarchy Enforcement**: Strict validation ensures operations follow OneNote's container hierarchy rules
- **Enhanced Debug Logging**: Detailed logging for troubleshooting container type detection and API responses

### Container Hierarchy Rules
The server now enforces OneNote's strict container hierarchy:
- **Notebooks** can contain sections and section groups
- **Section Groups** can contain sections and other section groups
- **Sections** can contain pages, but NOT other sections or section groups

### Improved Tool Descriptions
- Updated tool descriptions to clearly explain container hierarchy restrictions
- Added examples of valid and invalid container types
- Enhanced system prompts with container hierarchy guidance

### Better Response Processing
- Enhanced validation of API responses
- Improved parent container information handling
- Better error handling for missing or malformed data

### Modular Code Architecture
- **Separated Concerns**: Split large files into focused modules for better maintainability
- **Utility Functions**: Created dedicated utility modules for image processing and validation
- **HTTP Client**: Extracted HTTP operations into dedicated client module
- **Better Organization**: Improved code structure and separation of responsibilities

### Streamable HTTP Support
- **New Server Mode**: Added `-mode=streamable` for streamable HTTP protocol support with built-in SSE streaming
- **Port Configuration**: Added `-port` flag for custom port configuration
- **Flexible Deployment**: Support for stdio and streamable HTTP modes

## 🚨 Version 1.4.0 Migration Note

**All section and section group operations now use `containerId` (notebook or section group) instead of `notebookId` or `sectionGroupId`.**

- `listSections`, `listSectionGroups`, `createSection`, and `createSectionGroup` now take a `containerId` parameter.
- See [CHANGELOG.md](CHANGELOG.md) for details.

## 🔧 Development

### Building from Source
```
```
