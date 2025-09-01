// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package authorization

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimplifiedAuthorizationConfig_ValidateConfig(t *testing.T) {
	config := NewAuthorizationConfig()
	config.NotebookPermissions = map[string]PermissionLevel{
		"Work Notes":     PermissionWrite,
		"Archive Notes":  PermissionRead,
		"Private Notes":  PermissionNone,
	}

	err := config.ValidateConfig()
	require.NoError(t, err)
}

func TestSimplifiedAuthorizationConfig_SetCurrentNotebook(t *testing.T) {
	config := NewAuthorizationConfig()
	config.Enabled = true
	config.DefaultNotebookPermissions = PermissionRead
	config.NotebookPermissions = map[string]PermissionLevel{
		"Work Notes": PermissionWrite,
		"Archive":    PermissionNone,
	}
	
	err := config.ValidateConfig()
	require.NoError(t, err)

	// Test allowed notebook with write permission
	err = config.SetCurrentNotebook("Work Notes")
	assert.NoError(t, err)
	assert.Equal(t, "Work Notes", config.GetCurrentNotebook())

	// Test blocked notebook with none permission
	err = config.SetCurrentNotebook("Archive")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "access denied")

	// Test default fallback (read permission)
	err = config.SetCurrentNotebook("Other Notebook")
	assert.NoError(t, err)
	assert.Equal(t, "Other Notebook", config.GetCurrentNotebook())
}

func TestSimplifiedAuthorizationConfig_IsAuthorized(t *testing.T) {
	config := NewAuthorizationConfig()
	config.Enabled = true
	config.DefaultNotebookPermissions = PermissionRead
	
	err := config.ValidateConfig()
	require.NoError(t, err)
	
	err = config.SetCurrentNotebook("Test Notebook")
	require.NoError(t, err)

	ctx := context.Background()
	req := mcp.CallToolRequest{}

	// Test auth tools (always allowed)
	resourceContext := ResourceContext{
		Operation: OperationRead,
	}
	err = config.IsAuthorized(ctx, "getAuthStatus", req, resourceContext)
	assert.NoError(t, err)

	// Test non-auth tool with read permission on read-only notebook
	resourceContext = ResourceContext{
		NotebookName: "Test Notebook",
		Operation:    OperationRead,
	}
	err = config.IsAuthorized(ctx, "listPages", req, resourceContext)
	assert.NoError(t, err)

	// Test non-auth tool with write permission (should fail with read-only notebook)
	resourceContext = ResourceContext{
		NotebookName: "Test Notebook",
		Operation:    OperationWrite,
	}
	err = config.IsAuthorized(ctx, "createPage", req, resourceContext)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "access denied")
}

func TestSimplifiedAuthorizationConfig_GetNotebookPermission(t *testing.T) {
	config := NewAuthorizationConfig()
	config.DefaultNotebookPermissions = PermissionRead
	config.NotebookPermissions = map[string]PermissionLevel{
		"Work Notes":    PermissionWrite,
		"Archive Notes": PermissionRead,
		"Private Notes": PermissionNone,
	}

	// Test exact match
	assert.Equal(t, PermissionWrite, config.GetNotebookPermission("Work Notes"))
	assert.Equal(t, PermissionRead, config.GetNotebookPermission("Archive Notes"))
	assert.Equal(t, PermissionNone, config.GetNotebookPermission("Private Notes"))

	// Test default fallback
	assert.Equal(t, PermissionRead, config.GetNotebookPermission("Unknown Notebook"))
}

func TestSimplifiedAuthorizationConfig_FilterNotebooks(t *testing.T) {
	config := NewAuthorizationConfig()
	config.Enabled = true
	config.DefaultNotebookPermissions = PermissionRead
	config.NotebookPermissions = map[string]PermissionLevel{
		"Work Notes":    PermissionWrite,
		"Private Notes": PermissionNone,
	}

	notebooks := []map[string]interface{}{
		{"displayName": "Work Notes", "id": "1"},
		{"displayName": "Personal Notes", "id": "2"}, // Uses default (read)
		{"displayName": "Private Notes", "id": "3"},  // Should be filtered out
	}

	filtered := config.FilterNotebooks(notebooks)
	
	assert.Len(t, filtered, 2)
	assert.Equal(t, "Work Notes", filtered[0]["displayName"])
	assert.Equal(t, "Personal Notes", filtered[1]["displayName"])
}

func TestSimplifiedAuthorizationConfig_DisabledAuthorization(t *testing.T) {
	config := NewAuthorizationConfig()
	config.Enabled = false // Disabled authorization

	ctx := context.Background()
	req := mcp.CallToolRequest{}
	resourceContext := ResourceContext{
		NotebookName: "Any Notebook",
		Operation:    OperationWrite,
	}

	// Should allow all operations when disabled
	err := config.IsAuthorized(ctx, "createPage", req, resourceContext)
	assert.NoError(t, err)

	// Should allow any notebook selection when disabled
	err = config.SetCurrentNotebook("Any Notebook")
	assert.NoError(t, err)

	// Should not filter notebooks when disabled
	notebooks := []map[string]interface{}{
		{"displayName": "Work Notes", "id": "1"},
		{"displayName": "Private Notes", "id": "2"},
	}
	filtered := config.FilterNotebooks(notebooks)
	assert.Len(t, filtered, 2) // No filtering when disabled
}

func TestSimplifiedAuthorizationConfig_CrossNotebookAccess(t *testing.T) {
	config := NewAuthorizationConfig()
	config.Enabled = true
	config.DefaultNotebookPermissions = PermissionWrite
	
	err := config.SetCurrentNotebook("Current Notebook")
	require.NoError(t, err)

	ctx := context.Background()
	req := mcp.CallToolRequest{}

	// Test cross-notebook access attempt (should be blocked)
	resourceContext := ResourceContext{
		NotebookName: "Different Notebook",
		Operation:    OperationRead,
	}
	err = config.IsAuthorized(ctx, "listPages", req, resourceContext)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot access notebook 'Different Notebook'")
}