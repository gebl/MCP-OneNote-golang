#!/bin/bash

echo "OneNote MCP Server - Authorization Security Model Test"
echo "===================================================="
echo
echo "Running comprehensive security model validation..."
echo

# Get the directory of this script and navigate to the project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

# Run the security model test
go test ./internal/authorization -v -run="TestSecurityModelComprehensive"

if [ $? -eq 0 ]; then
    echo
    echo "✅ SECURITY MODEL VALIDATION PASSED"
    echo "All authorization rules are working correctly."
    exit 0
else
    echo
    echo "❌ SECURITY MODEL VALIDATION FAILED"
    echo "Review the authorization implementation."
    exit 1
fi