# OneNote MCP Server

A Go-based Model Context Protocol (MCP) server that provides seamless integration with Microsoft OneNote via the Microsoft Graph API. This server enables AI assistants and other MCP clients to read, create, update, and manage OneNote notebooks, sections, pages, and embedded content.

## 🚀 Features

### Core OneNote Operations
- **Notebook Management**: List all OneNote notebooks for the authenticated user
- **Section Operations**: List sections within notebooks
- **Page Operations**: Create, read, update, and delete OneNote pages
- **Content Management**: Full HTML content support with rich formatting
- **Page Item Handling**: Extract and manage embedded images, files, and objects
- **Search Capabilities**: Recursive search through all sections and section groups within a notebook
- **MCP Prompts**: 11 comprehensive prompts providing contextual guidance for OneNote operations

### Authentication & Security
- **OAuth 2.0 PKCE Flow**: Secure authentication using Proof Key for Code Exchange
- **Automatic Token Refresh**: Handles token expiration and renewal automatically
- **Input Validation**: Sanitizes and validates all OneNote IDs to prevent injection attacks
- **Secure Token Storage**: Local token persistence with automatic refresh
- **Security Hardened**: Git history cleaned of any sensitive credentials or tokens

### Content Processing
- **Image Optimization**: Automatic scaling of large images for better performance
- **Metadata Extraction**: Rich HTML metadata extraction from OneNote content
- **Content Type Detection**: Intelligent content type detection from HTML attributes
- **Binary Content Handling**: Full support for embedded files and media

### Developer Experience
- **Comprehensive Logging**: Detailed debug and operation logs
- **Error Handling**: Robust error handling with meaningful error messages
- **Docker Support**: Containerized deployment with Docker and Docker Compose
- **Configuration Management**: Flexible configuration via environment variables or JSON files

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
   ./onenote-mcp-server -mode=sse         # SSE mode on port 8080
   ./onenote-mcp-server -mode=streamable  # Streamable HTTP mode on port 8080
   ./onenote-mcp-server -mode=sse -port=8081        # SSE mode on custom port
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
   docker run -p 8080:8080 onenote-mcp-server -mode=sse         # SSE mode
   docker run -p 8080:8080 onenote-mcp-server -mode=streamable  # Streamable HTTP mode
   docker run -p 8081:8081 onenote-mcp-server -mode=sse -port=8081        # SSE mode on custom port
   docker run -p 8081:8081 onenote-mcp-server -mode=streamable -port=8081 # Streamable HTTP mode on custom port
   ```

3. **Or use Docker Compose**:
   ```bash
   docker-compose up -d
   ```

## 🔧 Configuration

### Server Modes

The OneNote MCP Server supports three different modes for client communication:

#### 1. **stdio Mode (Default)**
- **Usage**: `./onenote-mcp-server` or `./onenote-mcp-server -mode=stdio`
- **Description**: Standard input/output communication for direct integration
- **Best for**: CLI tools, direct process communication, development

#### 2. **SSE Mode (Server-Sent Events)**
- **Usage**: `./onenote-mcp-server -mode=sse [-port=8080]`
- **Description**: HTTP-based communication using Server-Sent Events
- **Best for**: Web applications, real-time updates, browser-based clients
- **Endpoint**: `http://localhost:8080/mcp`

#### 3. **Streamable HTTP Mode**
- **Usage**: `./onenote-mcp-server -mode=streamable [-port=8080]`
- **Description**: HTTP-based communication using streamable HTTP protocol
- **Best for**: HTTP clients, REST APIs, integration with HTTP-based systems
- **Endpoint**: `http://localhost:8080`

### Environment Variables

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `ONENOTE_CLIENT_ID` | Azure App Registration Client ID | Yes | - |
| `ONENOTE_TENANT_ID` | Azure Tenant ID (use "common" for multi-tenant) | Yes | - |
| `ONENOTE_REDIRECT_URI` | OAuth2 redirect URI | Yes | - |
| `ONENOTE_DEFAULT_NOTEBOOK_NAME` | Default notebook name | No | - |
| `ONENOTE_TOOLSETS` | Comma-separated list of enabled toolsets | No | All |
| `MCP_LOG_FILE` | Path to log file | No | Console only |

### Configuration File

Create `configs/config.json`:
```json
{
  "client_id": "your-azure-app-client-id",
  "tenant_id": "your-azure-tenant-id",
  "redirect_uri": "http://localhost:8080/callback",
  "notebook_name": "My Notebook",  // Maps to ONENOTE_DEFAULT_NOTEBOOK_NAME
  "toolsets": ["notebooks", "sections", "pages", "content"]
}
```

## 🎯 Available MCP Tools & Prompts

### MCP Prompts
The server provides 11 contextual prompts that offer guidance for OneNote operations:

- **explore-onenote-structure**: Navigate and understand OneNote structure
- **navigate-to-section**: Find specific sections within notebooks
- **create-structured-page**: Create pages with proper templates and formatting
- **update-page-content**: Modify page content with precise targeting
- **search-onenote-content**: Advanced content search across notebooks
- **organize-onenote-structure**: Reorganize OneNote structure
- **backup-and-migrate**: Backup and move content between locations
- **extract-media-content**: Extract embedded media from pages
- **create-content-workflow**: Set up automated content workflows
- **batch-content-operations**: Perform operations on multiple items
- **validate-onenote-operation**: Troubleshoot and validate operations

Each prompt provides contextual guidance, best practices, and step-by-step instructions for OneNote operations.

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
├── cmd/onenote-mcp-server/     # Main application entry point
│   └── main.go                 # Server setup and tool registration
├── internal/
│   ├── auth/                   # OAuth2 authentication and token management
│   │   └── auth.go             # PKCE flow, token refresh, secure storage
│   ├── config/                 # Configuration management
│   │   └── config.go           # Environment and file-based config loading
│   └── graph/                  # Microsoft Graph API client
│       └── graph.go            # OneNote operations, content processing
├── configs/                    # Configuration files and examples
│   └── example-config.json     # Example configuration template
├── docker/                     # Containerization files
│   ├── Dockerfile              # Multi-stage Docker build
│   └── docker-compose.yml      # Docker Compose configuration
├── docs/                       # Documentation
│   ├── api.md                  # API reference
│   ├── setup.md                # Detailed setup instructions
│   └── tool-to-resource-conversion.md
└── README.md                   # This file
```

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
- **New Server Mode**: Added `-mode=streamable` for streamable HTTP protocol support
- **Port Configuration**: Added `-port` flag for custom port configuration
- **Flexible Deployment**: Support for stdio, SSE, and streamable HTTP modes

## 🚨 Version 1.4.0 Migration Note

**All section and section group operations now use `containerId` (notebook or section group) instead of `notebookId` or `sectionGroupId`.**

- `listSections`, `listSectionGroups`, `createSection`, and `createSectionGroup` now take a `containerId` parameter.
- See [CHANGELOG.md](CHANGELOG.md) for details.

## 🔧 Development

### Building from Source
```
```
