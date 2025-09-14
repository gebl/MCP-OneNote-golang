# OneNote MCP Server

**Created by Gabriel Lawrence**

## 🚀 Why This Project Exists

For the longest time, I've been dumping information into OneNote with the vision that **eventually AI will be able to organize and synthesize it into useful, searchable content**. This project represents the realization of that vision - giving AI direct access to OneNote through the **Model Context Protocol (MCP)** so it can finally help organize, search, and make sense of years of accumulated information.

### 🎯 Three Goals in One Project

1. **Solve a Real Problem**: Enable AI to organize and work with my OneNote knowledge base
2. **Learn MCP**: Experiment with the Model Context Protocol and understand how AI agents can interact with external systems
3. **Test AI-Assisted Development**: Try "vibe coding" with Claude, GitHub Copilot, and Cursor to see what AI can actually build

### 🤖 AI Coding Experience: The Good and The Challenging

Overall, I've been **pretty impressed with what AI can accomplish**. This entire codebase was primarily built through AI assistance, and the results exceeded expectations in many areas:

**What AI Excelled At:**
- Rapid prototyping and initial implementation
- Comprehensive test coverage generation
- Documentation writing and maintenance
- Understanding complex APIs (Microsoft Graph)
- Code organization and modular architecture

**Where Humans Were Still Essential:**
- **Security vulnerabilities**: AI frequently introduced IDOR (Insecure Direct Object Reference) bugs
- **Complex debugging**: Some bugs required human intuition to understand root causes
- **Authorization design**: AI struggled with secure permission models and kept over-engineering solutions
- **Performance optimization**: Understanding real-world usage patterns and bottlenecks

This project serves as both a **practical tool** for AI-powered knowledge management and a **learning playground** for understanding the current capabilities and limitations of AI-assisted development.

---

A Go-based Model Context Protocol (MCP) server that provides seamless integration with Microsoft OneNote via the Microsoft Graph API. This server enables AI assistants and other MCP clients to read, create, update, and manage OneNote notebooks, sections, pages, and embedded content with AI safety features including notebook-scoped authorization, caching, and intelligent content processing.

## ⚠️ Important Safety Notice

**This software is licensed under the MIT License and is provided "AS IS" without warranty of any kind.**

**🚨 EARLY/TESTING SOFTWARE WARNING**: This is early-stage software that **exposes your OneNote data to AI agents** who may decide to perform actions that could **corrupt, modify, or delete your data**. AI agents can autonomously create, update, move, or delete pages, sections, and content based on their interpretation of instructions.

**⚠️ MICROSOFT GRAPH API LIMITATIONS**: The OneNote Microsoft Graph API is **not fully complete** and has known limitations. It has been **observed to corrupt pages or entire notebooks** in certain scenarios, particularly with complex content structures, tables, or when making rapid successive updates. These corruption issues are inherent to the Microsoft Graph API itself, not this software.

**🛡️ CRITICAL SAFETY PRECAUTIONS**:
- **Backup your OneNote data regularly** before using this software
- **Use the authorization system** to restrict AI agent access to specific notebooks, sections, or pages
- **Carefully review and approve AI tool actions** before they are executed, especially write operations
- **Start with read-only permissions** and gradually expand access as you gain confidence
- **Test with non-critical data first** to understand how AI agents interact with your content
- **Monitor the activity logs** to track what changes are being made to your OneNote data

**Recommended Authorization Setup for Safety**:
```json
{
  "authorization": {
    "enabled": true,
    "default_notebook_permissions": "none",
    "notebook_permissions": {
      "AI Test Notebook": "write"
    }
  }
}
```

By using this software, you acknowledge that you understand these risks and have taken appropriate precautions to protect your data.

## 🚀 Features

### 🎯 Rapid Note-Taking
- **QuickNote Tool**: Intelligent rapid note-taking with automatic timestamping, format detection (HTML/Markdown/ASCII), and smart page lookup
- **Multi-Format Support**: Automatic conversion between Markdown, HTML, and plain text with proper formatting preservation
- **Configurable Templates**: Customizable date formats and target notebook/page configurations

### 🔐 AI Safety & Authorization
- **Notebook-Scoped Authorization**: Simple permission system to prevent AI agents from accessing wrong notebooks
- **OAuth 2.0 PKCE Flow**: Secure authentication using Proof Key for Code Exchange with non-blocking startup
- **Safety-First Design**: Prevent AI agents from accidentally modifying sensitive notebooks
- **Audit Logging**: Authorization decision logging for AI agent safety monitoring
- **MCP Authentication Tools**: Built-in tools for authentication management (`auth_status`, `auth_initiate`, `auth_refresh`, `auth_clear`)

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
- **Streamlined MCP Protocol**: Focused on tools and resources following current MCP specification
- **Multi-Mode Support**: stdio and HTTP protocols with built-in SSE streaming

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
   ./onenote-mcp-server -mode=http        # HTTP mode on port 8080
   ./onenote-mcp-server -mode=http -port=8081 # HTTP mode on custom port
   ```

### Option 2: Docker Deployment

1. **Create configuration file**:
   ```bash
   # Copy the example and customize with your Azure app details
   cp configs/example-config.json docker/configs/config.json
   # Edit docker/configs/config.json with your actual credentials
   ```

2. **Build and run with Docker**:
   ```bash
   # Build from project root (not from docker/ subdirectory)
   docker build -f docker/Dockerfile -t onenote-mcp-server .

   # Run in different modes:
   docker run -p 8181:8181 onenote-mcp-server                    # stdio mode (default)
   docker run -p 8181:8181 onenote-mcp-server -mode=http -port=8181  # HTTP mode on port 8181
   ```

3. **Or use Docker Compose** (Recommended for HTTP mode):
   ```bash
   cd docker
   docker-compose build --no-cache  # Build fresh image
   docker-compose up -d              # Start in background
   docker-compose logs -f           # View logs
   docker-compose down              # Stop and remove containers
   ```

   The Docker Compose setup:
   - Runs on port 8181 by default
   - Mounts config from `docker/configs/config.json`
   - Supports Traefik labels for reverse proxy integration
   - Runs as non-root user (mcpuser) for security

## 🔧 Configuration

### Server Modes

The OneNote MCP Server supports two different modes for client communication:

#### 1. **stdio Mode (Default)**
- **Usage**: `./onenote-mcp-server` or `./onenote-mcp-server -mode=stdio`
- **Description**: Standard input/output communication for direct integration
- **Best for**: CLI tools, direct process communication, development

#### 2. **HTTP Mode**
- **Usage**: `./onenote-mcp-server -mode=http [-port=8181]`
- **Description**: HTTP-based communication with built-in Server-Sent Events (SSE) streaming for progress notifications
- **Best for**: HTTP clients, web applications, real-time updates, integration with HTTP-based systems
- **Endpoint**: `http://localhost:8181` (default port changed to 8181 in v2.0.0)
- **Features**:
  - SSE streaming for real-time progress updates during long-running operations
  - Optional Bearer token authentication (configure via `MCP_AUTH_ENABLED` and `MCP_BEARER_TOKEN`)
  - OAuth callback endpoint (`/callback`) automatically bypasses Bearer token requirement for browser-based auth flows

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
| `AUTHORIZATION_DEFAULT_NOTEBOOK_PERMISSIONS` | Default notebook permission level | No | read |
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
    "default_notebook_permissions": "read",
    "notebook_permissions": {
      "Personal Notes": "write",
      "Work Notes": "read",
      "AI Sandbox": "write",
      "Private Thoughts": "none"
    }
  },
  
  "mcp_auth": {
    "enabled": true,
    "bearer_token": "your-secret-bearer-token"
  },
  
  "log_level": "INFO",
  "log_format": "text",
  "log_file": "mcp-server.log",
  "content_log_level": "INFO"
}
```

## ⚡ QuickNote Tool

The `quick_note` tool provides rapid note-taking functionality with intelligent format detection and automatic timestamping:

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
{"name": "quick_note", "arguments": {"content": "Had a great idea for the project"}}

# Markdown note  
{"name": "quick_note", "arguments": {"content": "# Meeting Notes\n\n- Action item: Review API\n- **Deadline**: EOW"}}

# HTML note
{"name": "quick_note", "arguments": {"content": "<p>This is <strong>formatted</strong> content</p>"}}
```

Each note automatically gets formatted with timestamp and proper HTML conversion:
```html
<h3>January 15, 2025 - 2:30 PM</h3>
<p>Your note content here...</p>
```

## 🔐 Authorization System

The authorization system is designed as a **safety mechanism**, not a security feature. It prevents AI agents from accidentally accessing or modifying certain notebooks, acting as guardrails for autonomous AI operations.

### 🛡️ Safety vs Security Philosophy

**This is NOT a security feature** - it's a safety mechanism to prevent AI agents from going crazy and accessing the wrong notebooks. Real security comes from:

- **Microsoft Graph API native security**: You're logged in as a specific user account with specific permissions
- **OAuth 2.0 token scope**: API access limited to `Notes.ReadWrite` for your account
- **Azure Active Directory**: Enterprise identity and access management

The authorization system simply provides **AI agent safety guardrails** to:
- Prevent accidental access to sensitive notebooks when AI agents act autonomously
- Allow you to designate "safe" notebooks for AI experimentation
- Provide read-only access to reference materials while protecting active work
- Give you peace of mind when letting AI agents operate independently

### 🎯 Design Evolution

I tried several different versions of this authorization system:

1. **Version 1**: Complex pattern matching with wildcards (`Work*`, `Archive/**`) and granular section/page permissions
2. **Version 2**: Hierarchical permissions with tool-level, resource-level, and default permissions  
3. **Version 3**: Simple notebook-scoped permissions (current)

The earlier versions with granular section and page filtering had significant **performance impact** - every operation required multiple permission checks, pattern matching, and recursive filtering. The complexity wasn't worth it for a safety mechanism.

**Current approach**: Focus security at the **notebook level** where it matters most. Once you trust an AI agent with a notebook, it has access to all sections and pages within it. This provides the right balance of safety and performance.

### 📋 Permission Levels (Simplified)

- **`none`**: Notebook not accessible/selectable by AI agents
- **`read`**: Read-only access - prevents create/update/delete operations  
- **`write`**: Full access including create/update/delete operations

### 🔧 Configuration

**Recommended Safety Setup**:
```json
{
  "authorization": {
    "enabled": true,
    "default_notebook_permissions": "read",
    "notebook_permissions": {
      "AI Sandbox": "write",           // Safe for AI experimentation  
      "Personal Notes": "write",       // Trusted AI access
      "Work Archive": "read",          // Reference only
      "Private Thoughts": "none"       // Completely off-limits
    }
  }
}
```

**Conservative Setup** (Safest):
```json
{
  "authorization": {
    "enabled": true,
    "default_notebook_permissions": "none",    // Block access to unlisted notebooks
    "notebook_permissions": {
      "AI Test Notebook": "write"              // Only one notebook allowed
    }
  }
}
```

### 🚀 How It Works

1. **`notebooks`** - Only shows notebooks with `read` or `write` permissions (filters out `none`)
2. **`notebook_select("Work Notebook")`** - Validates notebook has non-`none` permission  
3. **All subsequent operations** inherit the selected notebook's permission level
4. **Write operations blocked** on read-only notebooks (create/update/delete)
5. **Cross-notebook access prevented** - all operations scoped to selected notebook

### ⚠️ Important Notes

- **Performance-focused**: No complex pattern matching or granular filtering  
- **Notebook-scoped**: Once selected, AI has access to all content within that notebook
- **Safety-first**: Designed to prevent accidents, not provide enterprise security
- **Microsoft Graph dependency**: Real security comes from your Microsoft account permissions

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

### MCP Tools

### Notebook Operations
- **`notebooks`**: List all OneNote notebooks for the user
  - Parameters: None
  - Returns: Array of notebook metadata with ID, name, isAPIDefault, and isConfigDefault flags
  - **Note:** Automatically handles pagination to return all notebooks

- **`notebook_select`**: Select a notebook by name or ID to use as the active notebook
  - Parameters: `identifier` (required) - Notebook name or ID to select as active
  - Returns: Success confirmation with selected notebook metadata
  - **Note:** Sets the notebook in the global cache for use by other operations

- **`notebook_current`**: Get currently selected notebook metadata from cache
  - Parameters: None
  - Returns: Currently selected notebook metadata
  - **Note:** Returns error if no notebook is currently selected

- **`sections`**: Get hierarchical sections and section groups from selected notebook
  - Parameters: None (uses currently selected notebook)
  - Returns: Notebook root with nested sections and section groups as children
  - **Features:**
    - Multi-layer caching with 5-minute expiration
    - Real-time progress notifications for long-running operations
    - Cache-aware progress updates (differentiates between cached and API operations)
    - Hierarchical tree structure showing complete notebook organization
  - **Note:** Requires a notebook to be selected first via `notebook_select`

### Section Operations
- **`section_create`**: Create a new section in a notebook or section group
  - Parameters: `containerID` (optional), `displayName` (required)
  - Returns: Created section metadata with success status and section ID
  - **Notes:** 
    - If containerID is not provided, uses the server's configured default notebook
    - Display name cannot contain: ?*\\/:<>|&#''%%~
    - Container Hierarchy: Sections can only be created inside notebooks or section groups, not inside other sections

- **`section_group_create`**: Create a new section group in a notebook or section group
  - Parameters: `containerID` (optional), `displayName` (required)
  - Returns: Created section group metadata with success status and section group ID
  - **Notes:**
    - If containerID is not provided, uses the server's configured default notebook
    - Display name cannot contain: ?*\\/:<>|&#''%%~
    - Container Hierarchy: Section groups can only be created inside notebooks or other section groups, not inside sections

### Page Operations
- **`pages`**: List all pages in a section
  - Parameters: `sectionID` (required) - MUST be actual section ID, NOT a section name
  - Returns: Structured response with pages array, cache status, and performance metrics
  - **Features:**
    - Multi-layer caching with 5-minute expiration and automatic invalidation
    - Authorization filtering integration
    - Progress notification support for long-running operations
    - Cache hit/miss status reporting with performance metrics
  - **Note:** Automatically handles pagination to return all pages

- **`page_create`**: Create a new page in a section
  - Parameters: `sectionID` (required), `title` (required), `content` (required)
  - Returns: Created page metadata
  - **Features:**
    - Intelligent text format detection (HTML, Markdown, ASCII)
    - Automatic HTML conversion with gomarkdown library support
    - Illegal character validation with suggested alternatives
  - **Note:** Title cannot contain: ?*\\/:<>|&#''%%~

- **`page_update`**: Update page HTML content (simple replacement)
  - Parameters: `pageID` (required), `content` (required)
  - Returns: Success confirmation
  - **Note:** Replaces entire page content. Use `page_update_advanced` for targeted updates

- **`page_update_advanced`**: Update specific parts of a page using advanced commands (preferred method)
  - Parameters: `pageID` (required), `commands` (required JSON array)
  - Returns: Success confirmation
  - **How to get data-id values:** Use `page_content` with `forUpdate=true` to retrieve HTML with `data-id` attributes
  - **Preferred Usage:** Use this tool to add, change, or delete parts of a page. Only use full page update if you intend to replace entire page content
  - **CRITICAL: Table Update Restrictions:** Tables must be updated as complete units. You CANNOT update individual table cells (td), headers (th), or rows (tr). Always target the entire table element and replace with complete table HTML to prevent layout corruption
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

- **`page_delete`**: Delete a page by ID
  - Parameters: `pageID` (required)
  - Returns: Success confirmation

- **`page_copy`**: Copy a page from one section to another using Microsoft Graph API (asynchronous)
  - Parameters: `pageID` (required), `targetSectionID` (required)
  - Returns: New page ID and operation metadata
  - **Note:** Automatically handles asynchronous operation polling and completion

- **`page_move`**: Move a page from one section to another (copy then delete)
  - Parameters: `pageID` (required), `targetSectionID` (required)
  - Returns: Moved page metadata with operation details
  - **Note:** Implements move as copy-then-delete for reliable operation

- **`getOnenoteOperation`**: Get status of asynchronous OneNote operations
  - Parameters: `operationID` (required)
  - Returns: Operation status and metadata
  - **Note:** Primarily used internally, but available for manual operation tracking

### Content Operations
- **`page_content`**: Get HTML content of a page
  - Parameters: `pageID` (required), `forUpdate` (optional string, set to 'true' to include data-id attributes for advanced updates)
  - Returns: HTML content as string
  - **Features:**
    - Optional data-id attribute inclusion for advanced update targeting
    - Content processing and conversion utilities
  - **Tip:** Use `forUpdate=true` to extract `data-id` values for use with advanced page updates

- **`page_items`**: List embedded items (images, files) in a page
  - Parameters: `pageID` (required)
  - Returns: Array of page item metadata with HTML attributes
  - **Note:** Identifies all embedded objects within page content

- **`page_item_content`**: Get complete page item data (content + metadata)
  - Parameters: `pageID` (required), `pageItemID` (required), `fullSize` (optional)
  - Returns: JSON with base64-encoded content and metadata
  - **Features:**
    - Automatic image optimization with configurable size limits
    - Smart filename generation with proper extensions
    - Content-Type detection from HTML attributes and HTTP headers
    - Optional full-size retrieval bypassing automatic scaling

### Special Tools
- **`quick_note`**: Rapid note-taking with automatic timestamping and format detection
  - Parameters: `content` (required) - Text content in HTML, Markdown, or plain text format
  - Returns: Success confirmation with formatted content preview
  - **Features:**
    - Automatic timestamp header in configurable format
    - Smart format detection and conversion (HTML/Markdown/ASCII)
    - Configurable target notebook and page via configuration
    - Append-only operation preserving existing content
    - Multi-layer caching for fast repeated operations
    - Performance optimized with cache-aware progress notifications
  - **Configuration Required:** Set `quicknote.notebook_name` and `quicknote.page_name` in config
  - **Note:** Falls back to default notebook if `quicknote.notebook_name` not configured

### Cache Management
- **`cache_clear`**: Clear all cached data (notebook sections and pages)
  - Parameters: None
  - Returns: Success confirmation with details of cleared cache layers
  - **Features:**
    - Clears notebook sections cache
    - Clears page metadata cache for all sections
    - Forces fresh data retrieval on next requests
    - System maintenance operation (no authorization required)

### Authentication Operations
- **`auth_status`**: Get current authentication status and token information
  - Parameters: None
  - Returns: Authentication state, token expiry, refresh availability
  - **Note:** Never exposes actual token values, only metadata

- **`auth_initiate`**: Start new OAuth authentication flow
  - Parameters: None
  - Returns: Browser URL and instructions for OAuth completion
  - **Note:** Starts temporary HTTP server on port 8080 for OAuth callback

- **`auth_refresh`**: Manually refresh authentication tokens
  - Parameters: None
  - Returns: Updated authentication status after refresh
  - **Note:** Requires valid refresh token to be available

- **`auth_clear`**: Clear stored authentication tokens (logout)
  - Parameters: None
  - Returns: Success confirmation
  - **Note:** Requires `auth_initiate` to re-authenticate after clearing

### Testing and Utilities
- **`testProgress`**: Test tool for progress notification functionality
  - Parameters: None
  - Returns: Success confirmation after completing progress test
  - **Features:**
    - Emits progress messages from 0 to 10 over 10 seconds
    - Tests progress notification system functionality
    - Useful for debugging progress streaming in different transport modes

## 🔐 Authentication Flow

The server uses OAuth 2.0 PKCE (Proof Key for Code Exchange) flow with **non-blocking startup**:

### **Server Startup (Non-Blocking)**
1. **Quick Start**: Server starts immediately without waiting for authentication
2. **Any State**: Works with no tokens, expired tokens, or valid tokens
3. **User Choice**: Authentication happens through MCP tools when needed

### **Authentication via MCP Tools**
4 authentication tools are available:

- **`auth_status`**: Check current authentication state and token information
- **`auth_initiate`**: Start OAuth flow - returns browser URL for user authentication
- **`auth_refresh`**: Manually refresh tokens to extend session
- **`auth_clear`**: Logout and clear all stored tokens

### **OAuth Flow Details**
1. **Call `auth_initiate`**: Server generates OAuth URL and starts local HTTP server on port 8080
2. **Browser Authentication**: User visits provided Microsoft login URL
3. **Automatic Callback**: Server receives authorization code and exchanges for tokens
4. **Token Storage**: Tokens saved locally and automatically refreshed before expiration

### **Example Workflow**
```bash
# 1. Start server (immediate, no blocking)
./onenote-mcp-server

# 2. Check authentication status
mcp > auth_status
{"authenticated": false, "message": "No authentication tokens found"}

# 3. Initiate authentication
mcp > auth_initiate
{"authUrl": "https://login.microsoftonline.com/...", "instructions": "Visit this URL..."}

# 4. User visits URL in browser, completes OAuth flow

# 5. Check status again
mcp > auth_status
{"authenticated": true, "tokenExpiresIn": "59 minutes", "authMethod": "OAuth2_PKCE"}

# 6. Use OneNote operations
mcp > notebooks
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
│   └── PageResources.go            # Page-related MCP resources
├── internal/                       # Domain-specific modules
│   ├── auth/                       # OAuth2 authentication and token management
│   │   ├── auth.go                 # PKCE flow, token refresh, secure storage
│   │   ├── manager.go              # Authentication manager
│   │   └── middleware.go           # Authentication middleware
│   ├── authorization/              # Simplified notebook-scoped permission system
│   │   ├── authorization.go        # Permission evaluation and enforcement
│   │   ├── context.go              # Authorization context management
│   │   ├── wrapper.go              # Tool authorization wrapper
│   │   └── adapters.go             # Configuration adapters
│   ├── config/                     # Multi-source configuration management
│   │   └── config.go               # Environment vars, JSON files, defaults
│   ├── graph/                      # Microsoft Graph SDK integration
│   │   ├── client.go               # HTTP client and authentication
│   │   ├── http.go                 # HTTP utilities
│   │   └── utils.go                # Graph API utilities
│   ├── http/                       # Shared HTTP utilities
│   │   └── helpers.go              # HTTP helper functions
│   ├── notebooks/                  # Notebook domain operations
│   │   └── notebooks.go            # Notebook CRUD and management
│   ├── pages/                      # Page domain operations
│   │   └── pages.go                # Page content, items, and formatting
│   ├── sections/                   # Section domain operations
│   │   ├── sections.go             # Section and section group operations
│   │   └── groups.go               # Section group operations
│   ├── resources/                  # MCP resource descriptions
│   │   └── descriptions.go         # Resource documentation
│   ├── utils/                      # Shared utilities
│   │   ├── validation.go           # Input validation and sanitization
│   │   ├── image.go                # Image processing and optimization
│   │   ├── progress.go             # Progress notification utilities
│   │   └── tool_helpers.go         # MCP tool helper functions
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

**Notebook-Scoped Authorization**: Simplified permission system focused on notebook-level access control for AI agent safety.

**Multi-Layer Caching**: Thread-safe caching system with page metadata, search results, and notebook lookup caches providing 5-minute expiration and automatic invalidation.

**Progressive Enhancement**: Starts with basic MCP functionality and adds enterprise features like authorization, caching, progress notifications, and intelligent content processing.

## 🚀 Usage Examples

### Basic Notebook Operations
```bash
# List all notebooks
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{"name": "notebooks", "arguments": {}}'

# List sections in selected notebook
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{"name": "sections", "arguments": {}}'

# Create a new section
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{"name": "section_create", "arguments": {"containerId": "notebook-id", "displayName": "New Section"}}'
```

### QuickNote Usage
```bash
# Plain text quick note
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "quick_note",
    "arguments": {
      "content": "Had a breakthrough idea for the user interface design!"
    }
  }'

# Markdown quick note with formatting
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "quick_note", 
    "arguments": {
      "content": "# Meeting Summary\n\n- **Decision**: Use React for frontend\n- **Action Item**: Research component libraries\n- *Deadline*: Friday EOD"
    }
  }'

# HTML quick note  
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "quick_note",
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
    "name": "page_create",
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
    "name": "page_update",
    "arguments": {
      "pageId": "page-id",
      "content": "<h1>Updated Content</h1><p>This content was updated.</p>"
    }
  }'

# Copy page to different section (asynchronous operation)
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "page_copy",
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
    "name": "page_move",
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
  -d '{"name": "page_content", "arguments": {"pageId": "page-id"}}'

# List embedded items
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{"name": "page_items", "arguments": {"pageId": "page-id"}}'

# Get page item with content
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "page_item_content",
    "arguments": {
      "pageId": "page-id",
      "pageItemId": "item-id"
    }
  }'
```

## 🆕 Version 2.0.0 - Major Release

### OAuth Callback Improvements
- **Unified callback handling**: OAuth callbacks now use the main HTTP server in HTTP mode instead of spawning a separate server
- **Authentication bypass for callbacks**: `/callback` endpoint no longer requires Bearer token authentication
- **Improved Docker support**: Better layer caching and correct port configuration

### Previous Improvements (v1.7.0)

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
- **New Server Mode**: Added `-mode=http` for HTTP protocol support with built-in SSE streaming
- **Port Configuration**: Added `-port` flag for custom port configuration
- **Flexible Deployment**: Support for stdio and HTTP modes

## 🚨 Version 1.4.0 Migration Note

**All section and section group operations now use `containerId` (notebook or section group) instead of `notebookId` or `sectionGroupId`.**

- `listSections`, `listSectionGroups`, `createSection`, and `createSectionGroup` now take a `containerId` parameter.
- See [CHANGELOG.md](CHANGELOG.md) for details.

## 🔧 Development

### Building from Source
```
```
