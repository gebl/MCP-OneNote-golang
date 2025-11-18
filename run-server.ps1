#!/usr/bin/env powershell
<#
.SYNOPSIS
    OneNote MCP Server - Build and Run Script (PowerShell)

.DESCRIPTION
    Builds and runs the OneNote MCP server with specified mode and port.

.PARAMETER Mode
    Server mode: "stdio" (default) or "http"

.PARAMETER Port
    Port number for HTTP mode (default: 8080)

.EXAMPLE
    .\run-server.ps1
    # Runs in stdio mode

.EXAMPLE
    .\run-server.ps1 -Mode http -Port 8081
    # Runs in HTTP mode on port 8081
#>

param(
    [string]$Mode = "stdio",
    [int]$Port = 8080
)

Write-Host "OneNote MCP Server - Build and Run" -ForegroundColor Cyan
Write-Host "=====================================" -ForegroundColor Cyan

Write-Host "Building server..." -ForegroundColor Yellow
try {
    & go build -o onenote-mcp-server.exe ./cmd/onenote-mcp-server
    if ($LASTEXITCODE -ne 0) {
        throw "Build failed with exit code $LASTEXITCODE"
    }
    Write-Host "Build successful!" -ForegroundColor Green
} catch {
    Write-Host "Build failed: $_" -ForegroundColor Red
    Read-Host "Press Enter to exit"
    exit 1
}

Write-Host ""

if ($Mode -eq "http") {
    Write-Host "Starting OneNote MCP Server in HTTP mode on port $Port..." -ForegroundColor Green
    Write-Host "Press Ctrl+C to stop the server" -ForegroundColor Yellow
    Write-Host ""
    & .\onenote-mcp-server.exe -mode=http "-port=$Port"
} else {
    Write-Host "Starting OneNote MCP Server in STDIO mode..." -ForegroundColor Green
    Write-Host "Use Ctrl+C to stop, or send MCP messages via stdin" -ForegroundColor Yellow
    Write-Host ""
    & .\onenote-mcp-server.exe
}

Write-Host ""
Read-Host "Press Enter to exit"