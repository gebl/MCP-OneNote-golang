@echo off
echo Testing new logging system...

echo.
echo ===== Testing DEBUG level =====
set LOG_LEVEL=DEBUG
timeout 3 onenote-mcp-server.exe 2>debug.log
echo Debug output saved to debug.log

echo.
echo ===== Testing INFO level =====
set LOG_LEVEL=INFO
timeout 3 onenote-mcp-server.exe 2>info.log
echo Info output saved to info.log

echo.
echo ===== Testing JSON format =====
set LOG_FORMAT=json
set LOG_LEVEL=INFO
timeout 3 onenote-mcp-server.exe 2>json.log
echo JSON output saved to json.log

echo.
echo ===== Testing file output =====
set LOG_FORMAT=text
set MCP_LOG_FILE=file.log
timeout 3 onenote-mcp-server.exe
echo File output saved to file.log

echo.
echo Testing complete! Check the generated log files.