# OneNote MCP Server Setup Guide

This guide provides detailed step-by-step instructions for setting up the OneNote MCP Server, including Azure app registration, local development, and production deployment.

## Prerequisites

### Required Software
- **Go 1.21 or later** - [Download from golang.org](https://golang.org/dl/)
- **Git** - [Download from git-scm.com](https://git-scm.com/downloads)
- **Docker** (optional) - [Download from docker.com](https://www.docker.com/products/docker-desktop)

### Required Accounts
- **Microsoft Azure Account** - [Sign up at azure.microsoft.com](https://azure.microsoft.com/free/)
- **Microsoft 365 Account** with OneNote access

## Step 1: Azure App Registration

### 1.1 Create App Registration

1. **Sign in to Azure Portal**
   - Go to [portal.azure.com](https://portal.azure.com)
   - Sign in with your Microsoft account

2. **Navigate to App Registrations**
   - Search for "App registrations" in the search bar
   - Click "App registrations" in the results

3. **Create New Registration**
   - Click "New registration"
   - Fill in the registration details:
     - **Name**: `OneNote MCP Server` (or your preferred name)
     - **Supported account types**: 
       - Choose "Accounts in this organizational directory only" for single tenant
       - Choose "Accounts in any organizational directory" for multi-tenant
     - **Redirect URI**: 
       - Type: Web
       - URI: `http://localhost:8080/callback`
   - Click "Register"

### 1.2 Configure API Permissions

1. **Navigate to API Permissions**
   - In your app registration, click "API permissions" in the left menu

2. **Add Microsoft Graph Permissions**
   - Click "Add a permission"
   - Select "Microsoft Graph"
   - Choose "Delegated permissions"
   - Search for and select:
     - `Notes.ReadWrite` - Read and write OneNote notebooks
   - Click "Add permissions"

3. **Grant Admin Consent** (if needed)
   - If you're an admin, click "Grant admin consent for [your organization]"
   - This allows all users in your organization to use the app

### 1.3 Get Application Credentials

1. **Copy Application (Client) ID**
   - In your app registration, go to "Overview"
   - Copy the "Application (client) ID" - you'll need this for configuration

2. **Copy Directory (Tenant) ID**
   - Copy the "Directory (tenant) ID" - you'll need this for configuration

3. **Note Redirect URI**
   - The redirect URI you configured: `http://localhost:8080/callback`

## Step 2: Local Development Setup

### 2.1 Clone the Repository

```bash
git clone <repository-url>
cd MCP-OneNote-golang
```

### 2.2 Set Environment Variables

Create a `.env` file in the project root (optional) or set environment variables directly:

```bash
# Required Azure App Registration details
export ONENOTE_CLIENT_ID="your-application-client-id"
export ONENOTE_TENANT_ID="your-directory-tenant-id"
export ONENOTE_REDIRECT_URI="http://localhost:8080/callback"

# Optional configuration
export ONENOTE_NOTEBOOK_NAME="My Notebook"
export MCP_LOG_FILE="onenote-mcp-server.log"
```

**Replace the placeholder values:**
- `your-application-client-id`: The Application (Client) ID from Azure
- `your-directory-tenant-id`: The Directory (Tenant) ID from Azure

### 2.3 Build the Application

```bash
# Build for current platform
go build -o onenote-mcp-server ./cmd/onenote-mcp-server

# Or build with specific Go version
go build -ldflags="-s -w" -o onenote-mcp-server ./cmd/onenote-mcp-server
```

### 2.4 Run the Server

```bash
./onenote-mcp-server
```

**First Run Authentication Flow:**
1. The server will start and detect no valid tokens
2. It will start a local HTTP server on port 8080
3. You'll see a URL to visit in your browser
4. Click the URL or copy it to your browser
5. Sign in with your Microsoft account
6. Grant permissions to the application
7. You'll be redirected back and see "Authentication successful"
8. The server will save tokens and continue running

### 2.5 Verify Installation

Test the server by listing notebooks:

```bash
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{"name": "notebooks", "arguments": {}}'
```

You should see a JSON response with your OneNote notebooks.

## Step 3: Docker Deployment

### 3.1 Create Configuration File

Create `configs/config.json`:

```json
{
  "client_id": "your-application-client-id",
  "tenant_id": "your-directory-tenant-id",
  "redirect_uri": "http://localhost:8080/callback",
  "notebook_name": "My Notebook",
  "toolsets": ["notebooks", "sections", "pages", "content"]
}
```

### 3.2 Build Docker Image

```bash
docker build -t onenote-mcp-server .
```

### 3.3 Run with Docker

```bash
# Run with port mapping
docker run -p 8080:8080 \
  -v $(pwd)/configs:/app/configs \
  -v $(pwd)/tokens.json:/app/tokens.json \
  onenote-mcp-server

# Or use Docker Compose
docker-compose up -d
```

### 3.4 Docker Compose Configuration

Create `docker-compose.yml`:

```yaml
version: '3.8'
services:
  onenote-mcp-server:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - ./configs:/app/configs
      - ./tokens.json:/app/tokens.json
    environment:
      - ONENOTE_MCP_CONFIG=/app/configs/config.json
    restart: unless-stopped
```

## Step 4: Production Deployment

### 4.1 Security Considerations

1. **Use HTTPS in Production**
   - Configure reverse proxy (nginx, Apache) with SSL certificates
   - Update redirect URI to use HTTPS

2. **Secure Token Storage**
   - Use environment variables or secure key management
   - Don't commit tokens to version control

3. **Network Security**
   - Configure firewall rules
   - Use VPN or private networks if needed

### 4.2 Environment Configuration

Set production environment variables:

```bash
# Production Azure App Registration
export ONENOTE_CLIENT_ID="your-production-client-id"
export ONENOTE_TENANT_ID="your-production-tenant-id"
export ONENOTE_REDIRECT_URI="https://your-domain.com/callback"

# Production settings
export MCP_LOG_FILE="/var/log/onenote-mcp-server.log"
export ONENOTE_DEBUG=false
```

### 4.3 Systemd Service (Linux)

Create `/etc/systemd/system/onenote-mcp-server.service`:

```ini
[Unit]
Description=OneNote MCP Server
After=network.target

[Service]
Type=simple
User=onenote
WorkingDirectory=/opt/onenote-mcp-server
ExecStart=/opt/onenote-mcp-server/onenote-mcp-server
Restart=always
RestartSec=10
Environment=ONENOTE_CLIENT_ID=your-client-id
Environment=ONENOTE_TENANT_ID=your-tenant-id
Environment=ONENOTE_REDIRECT_URI=https://your-domain.com/callback
Environment=MCP_LOG_FILE=/var/log/onenote-mcp-server.log

[Install]
WantedBy=multi-user.target
```

Enable and start the service:

```bash
sudo systemctl daemon-reload
sudo systemctl enable onenote-mcp-server
sudo systemctl start onenote-mcp-server
sudo systemctl status onenote-mcp-server
```

## Step 5: Troubleshooting

### 5.1 Common Issues

**Authentication Errors:**
```bash
# Check if tokens are valid
cat tokens.json

# Delete tokens to force re-authentication
rm tokens.json
```

**Permission Errors:**
- Verify `Notes.ReadWrite` permission is granted in Azure
- Check if user has access to OneNote notebooks
- Ensure admin consent is granted (if required)

**Network Issues:**
```bash
# Test connectivity to Microsoft Graph API
curl -I https://graph.microsoft.com/v1.0/

# Check DNS resolution
nslookup graph.microsoft.com
```

**Port Conflicts:**
```bash
# Check if port 8080 is in use
netstat -tulpn | grep :8080

# Use different port
export ONENOTE_PORT=8081
```

### 5.2 Debug Mode

Enable debug logging:

```bash
export ONENOTE_DEBUG=true
export MCP_LOG_FILE="debug.log"
./onenote-mcp-server
```

### 5.3 Log Analysis

Check logs for errors:

```bash
# View real-time logs
tail -f onenote-mcp-server.log

# Search for errors
grep -i error onenote-mcp-server.log

# Search for authentication issues
grep -i auth onenote-mcp-server.log
```

## Step 6: Testing

### 6.1 Basic Functionality Test

Test all major operations:

```bash
# 1. List notebooks
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{"name": "notebooks", "arguments": {}}'

# 2. List sections (use notebook ID from step 1)
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "listSections",
    "arguments": {
      "containerId": "your-notebook-or-sectiongroup-id"
    }
  }'

# 3. Create a test page
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "createPage",
    "arguments": {
      "sectionId": "your-section-id",
      "title": "Test Page",
      "content": "<h1>Test Page</h1><p>This is a test page created by the MCP server.</p>"
    }
  }'
```

### 6.2 Integration Testing

Test with your MCP client:

```bash
# Example with curl
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "searchPages",
    "arguments": {
      "query": "test"
    }
  }'
```

## Step 7: Monitoring and Maintenance

### 7.1 Health Checks

Create a health check endpoint:

```bash
# Check server status
curl -I http://localhost:8080/health

# Check token validity
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{"name": "notebooks", "arguments": {}}'
```

### 7.2 Log Rotation

Configure log rotation in `/etc/logrotate.d/onenote-mcp-server`:

```
/var/log/onenote-mcp-server.log {
    daily
    missingok
    rotate 7
    compress
    delaycompress
    notifempty
    create 644 onenote onenote
    postrotate
        systemctl reload onenote-mcp-server
    endscript
}
```

### 7.3 Backup Strategy

Backup important files:

```bash
# Backup configuration
cp configs/config.json configs/config.json.backup

# Backup tokens (if using file storage)
cp tokens.json tokens.json.backup

# Backup logs
tar -czf logs-backup-$(date +%Y%m%d).tar.gz *.log
```

## Next Steps

After successful setup:

1. **Review the API Documentation** - See `docs/api.md` for detailed tool reference
2. **Test with Your MCP Client** - Integrate with your preferred MCP client
3. **Monitor Performance** - Set up monitoring and alerting
4. **Scale as Needed** - Consider load balancing for high-traffic deployments

For additional support, see the troubleshooting section in the main README.md file. 