@echo off
echo OneNote MCP Server - Authorization Security Model Test
echo ====================================================
echo.
echo Running comprehensive security model validation...
echo.

cd /d "%~dp0.."
go test ./internal/authorization -v -run="TestSecurityModelComprehensive"

if %ERRORLEVEL% EQU 0 (
    echo.
    echo ✅ SECURITY MODEL VALIDATION PASSED
    echo All authorization rules are working correctly.
    exit /b 0
) else (
    echo.
    echo ❌ SECURITY MODEL VALIDATION FAILED
    echo Review the authorization implementation.
    exit /b 1
)