# Setup Guide: OneNote MCP Server

## 1. Register an Azure AD Application

1. Go to [Azure Portal](https://portal.azure.com/)
2. Navigate to **Azure Active Directory > App registrations**
3. Click **New registration**
4. Name: `OneNote MCP Server`
5. Redirect URI: `http://localhost:8080/callback` (or your chosen URI)
6. Register and note the **Application (client) ID** and **Directory (tenant) ID**
7. Go to **Certificates & secrets** and create a new client secret
8. Under **API permissions**, add:
   - `Notes.ReadWrite.All`
   - `offline_access`

## 2. Set Environment Variables

```
ONENOTE_CLIENT_ID=your-client-id
ONENOTE_CLIENT_SECRET=your-client-secret
ONENOTE_TENANT_ID=your-tenant-id
ONENOTE_REDIRECT_URI=http://localhost:8080/callback
ONENOTE_TOOLSETS=notebooks,sections,pages,content
```

## 3. Build and Run

```
go build -o onenote-mcp-server ./cmd/onenote-mcp-server
./onenote-mcp-server
```

Or with Docker:

```
docker-compose up --build
```

## 4. Configuration File (Optional)

You can also use a JSON config file and set `ONENOTE_MCP_CONFIG=path/to/config.json`.

See `configs/example-config.json` for an example. 