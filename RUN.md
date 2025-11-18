# Quick Start Guide

This directory contains scripts to easily build and run the OneNote MCP Server after the SDK migration.

## Scripts Available

### Windows
- **`run-server.bat`** - Windows Batch script
- **`run-server.ps1`** - PowerShell script (recommended)

### Linux/macOS
- **`run-server.sh`** - Bash shell script

## Usage

### Default (STDIO mode)
```bash
# Windows
run-server.bat
# or
.\run-server.ps1

# Linux/macOS
./run-server.sh
```

### HTTP mode
```bash
# Windows
run-server.bat http
run-server.bat http 8081
# or
.\run-server.ps1 -Mode http -Port 8081

# Linux/macOS
./run-server.sh http
./run-server.sh http 8081
```

## What the Scripts Do

1. **Build** the server using `go build -o onenote-mcp-server[.exe] ./cmd/onenote-mcp-server`
2. **Run** the server in the specified mode:
   - **STDIO mode** (default): For use with MCP clients via stdin/stdout
   - **HTTP mode**: Runs a web server for HTTP-based MCP communication

## Configuration

The server will automatically load configuration from:
- Environment variables (ONENOTE_CLIENT_ID, etc.)
- `test-config.json` file (if present)
- Default values

See the main README.md for full configuration details.

## Authentication

On first run, you may need to authenticate:
1. Start the server
2. Use the `auth_initiate` MCP tool to begin authentication
3. Follow the authentication URL provided

## Migration Notes

✅ **Migration Complete**: This server has been successfully migrated from `mark3labs/mcp-go` to the official `github.com/modelcontextprotocol/go-sdk`.

- All tools and resources are functional
- Progress notifications are temporarily disabled (non-critical feature)
- Server maintains backward compatibility