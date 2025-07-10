# LM Studio MCP Configuration for OneNote Server

This guide shows how to configure LM Studio to connect to your OneNote MCP server running in SSE mode with bearer token authentication.

## Setup Steps

### 1. Configure Your OneNote MCP Server

First, set up your server environment variables. Create a `.env` file or set these in your environment:

```bash
# Required Azure Configuration
ONENOTE_CLIENT_ID=your-azure-app-client-id
ONENOTE_TENANT_ID=your-azure-tenant-id
ONENOTE_REDIRECT_URI=http://localhost:8080/callback

# Optional Configuration
ONENOTE_DEFAULT_NOTEBOOK_NAME=My Notebook
ONENOTE_TOOLSETS=notebooks,sections,pages,content

# MCP Authentication (for SSE mode)
MCP_AUTH_ENABLED=true
MCP_BEARER_TOKEN=your-secret-bearer-token-here

# Logging
LOG_LEVEL=INFO
LOG_FORMAT=text
CONTENT_LOG_LEVEL=INFO
```

### 2. Start the OneNote MCP Server in SSE Mode

```bash
# Start the server on port 8080 (or your preferred port)
./onenote-mcp-server.exe -mode=sse -port=8080
```

The server will start and be available with these endpoints:
- **SSE connection**: `http://localhost:8080/sse`
- **Message endpoint**: `http://localhost:8080/message`
- **Server info**: `http://localhost:8080/mcp` (for debugging)

### 3. Configure LM Studio

1. Open LM Studio
2. Go to Settings or MCP Configuration
3. Add the `mcp.json` configuration file to LM Studio's MCP servers directory, or configure it directly in the UI:

```json
{
  "servers": {
    "onenote": {
      "transport": {
        "type": "sse",
        "url": "http://localhost:8080/sse",
        "headers": {
          "Authorization": "your-secret-bearer-token-here"
        }
      }
    }
  }
}
```

**Important**: The SSE endpoint is `/sse`, not `/mcp`. The `/mcp` path is only for server information.

**Note:** The server accepts both authentication formats:
- Standard: `"Authorization": "Bearer your-token"`
- Simplified: `"Authorization": "your-token"` (for MCP client compatibility)

### 4. Authentication Setup

Before using the MCP tools, you'll need to authenticate with Microsoft:

1. Start a chat in LM Studio
2. Use the `getAuthStatus` tool to check authentication status
3. If not authenticated, use the `initiateAuth` tool to start OAuth flow
4. Visit the provided URL in your browser to complete authentication
5. The server will automatically receive the OAuth callback and store tokens

### 5. Available Tools

Once configured and authenticated, you'll have access to these OneNote tools:

**Notebook Management:**
- `listNotebooks` - List all your OneNote notebooks
- `searchPages` - Search for pages by title across notebooks

**Section Management:**
- `listSections` - List sections within a notebook or section group
- `createSection` - Create new sections

**Page Management:**
- `listPages` - List pages within a section
- `createPage` - Create new pages with HTML content
- `getPageContent` - Retrieve page content
- `updatePageContent` - Update page content
- `updatePageContentAdvanced` - Advanced page content updates
- `deletePage` - Delete pages
- `copyPage` - Copy pages between sections
- `movePage` - Move pages between sections

**Content Management:**
- `listPageItems` - List embedded items (images, files) in pages
- `getPageItem` - Get page item data with binary content

**Authentication Management:**
- `getAuthStatus` - Check authentication status
- `initiateAuth` - Start OAuth authentication
- `refreshToken` - Refresh authentication tokens
- `clearAuth` - Clear stored tokens

### 6. Security Considerations

- **Use HTTPS in production**: The bearer token should only be transmitted over HTTPS
- **Keep tokens secret**: Store bearer tokens securely and don't commit them to version control
- **Token rotation**: Consider rotating bearer tokens periodically
- **Network security**: If running on a server, ensure proper firewall configuration

### 7. Troubleshooting

**Connection Issues:**
- Verify the server is running on the correct port
- Check that the bearer token matches between server and client config
- Ensure no firewall is blocking the connection

**Authentication Issues:**
- Use `getAuthStatus` to check current authentication state
- Use `initiateAuth` to re-authenticate if tokens are expired
- Check the server logs for detailed authentication information

**Logging:**
- Set `LOG_LEVEL=DEBUG` for detailed server logs
- Set `CONTENT_LOG_LEVEL=DEBUG` to see actual OneNote content in logs
- Check the log file specified in `MCP_LOG_FILE` environment variable

### 8. Alternative Configuration (Without Authentication)

For development/testing, you can disable authentication:

```bash
# Disable authentication
MCP_AUTH_ENABLED=false
# No need to set MCP_BEARER_TOKEN
```

And update the LM Studio config:

```json
{
  "servers": {
    "onenote": {
      "transport": {
        "type": "sse",
        "url": "http://localhost:8080/sse"
      }
    }
  }
}
```

This removes the authentication requirement but should only be used in secure, local environments.