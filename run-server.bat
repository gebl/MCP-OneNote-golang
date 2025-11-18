@echo off
REM OneNote MCP Server - Build and Run Script (Windows)
REM Usage: run-server.bat [mode] [port]
REM   mode: stdio (default) or http
REM   port: port number for HTTP mode (default: 8080)

echo OneNote MCP Server - Build and Run
echo =====================================

REM Set default values
set MODE=%1
set PORT=%2
if "%MODE%"=="" set MODE=stdio
if "%PORT%"=="" set PORT=8080

echo Building server...
go build -o onenote-mcp-server.exe ./cmd/onenote-mcp-server
if %ERRORLEVEL% neq 0 (
    echo Build failed!
    pause
    exit /b 1
)

echo Build successful!
echo.

if "%MODE%"=="http" (
    echo Starting OneNote MCP Server in HTTP mode on port %PORT%...
    echo Press Ctrl+C to stop the server
    echo.
    onenote-mcp-server.exe -mode=http -port=%PORT%
) else (
    echo Starting OneNote MCP Server in STDIO mode...
    echo Use Ctrl+C to stop, or send MCP messages via stdin
    echo.
    onenote-mcp-server.exe
)

pause