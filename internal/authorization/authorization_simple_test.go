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

func TestSimplifiedAuthorizationConfig_CompilePatterns(t *testing.T) {
	config := NewAuthorizationConfig()
	config.NotebookPermissions = map[string]PermissionLevel{
		"Work Notes": PermissionWrite,
		"temp_*":     PermissionRead,
	}
	config.SectionPermissions = map[string]PermissionLevel{
		"/Projects/**": PermissionRead,
		"public_*":     PermissionWrite,
	}
	config.PagePermissions = map[string]PermissionLevel{
		"draft_*":     PermissionWrite,
		"*_archive":   PermissionNone,
	}

	err := config.CompilePatterns()
	require.NoError(t, err)
	assert.NotNil(t, config.notebookEngine)
	assert.NotNil(t, config.sectionEngine)
	assert.NotNil(t, config.pageEngine)
}

func TestSimplifiedAuthorizationConfig_SetCurrentNotebook(t *testing.T) {
	config := NewAuthorizationConfig()
	config.Enabled = true
	config.DefaultNotebookPermissions = PermissionRead
	config.NotebookPermissions = map[string]PermissionLevel{
		"Work Notes": PermissionWrite,
		"Archive":    PermissionNone,
	}
	
	err := config.CompilePatterns()
	require.NoError(t, err)

	// Test allowed notebook
	err = config.SetCurrentNotebook("Work Notes")
	assert.NoError(t, err)
	assert.Equal(t, "Work Notes", config.GetCurrentNotebook())

	// Test blocked notebook
	err = config.SetCurrentNotebook("Archive")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "access denied")

	// Test default fallback
	err = config.SetCurrentNotebook("Other Notebook")
	assert.NoError(t, err)
	assert.Equal(t, "Other Notebook", config.GetCurrentNotebook())
}

func TestSimplifiedAuthorizationConfig_IsAuthorized(t *testing.T) {
	config := NewAuthorizationConfig()
	config.Enabled = true
	config.DefaultNotebookPermissions = PermissionRead
	
	err := config.CompilePatterns()
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

	// Test non-auth tool with read permission
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
	assert.Contains(t, err.Error(), "write")
}

func TestSimplifiedAuthorizationConfig_CrossNotebookAccess(t *testing.T) {
	config := NewAuthorizationConfig()
	config.Enabled = true
	config.DefaultNotebookPermissions = PermissionWrite
	
	err := config.CompilePatterns()
	require.NoError(t, err)
	
	err = config.SetCurrentNotebook("Current Notebook")
	require.NoError(t, err)

	ctx := context.Background()
	req := mcp.CallToolRequest{}

	// Test cross-notebook access (should be blocked)
	resourceContext := ResourceContext{
		NotebookName: "Different Notebook",
		Operation:    OperationRead,
	}
	err = config.IsAuthorized(ctx, "listPages", req, resourceContext)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot access notebook")
}

func TestSimplifiedAuthorizationConfig_PatternMatching(t *testing.T) {
	config := NewAuthorizationConfig()
	config.Enabled = true
	config.DefaultNotebookPermissions = PermissionRead
	config.SectionPermissions = map[string]PermissionLevel{
		"public_*":      PermissionWrite,
		"/Projects/**":  PermissionRead,
		"*_archive":     PermissionNone,
	}
	config.PagePermissions = map[string]PermissionLevel{
		"draft_*":       PermissionWrite,
		"temp_*":        PermissionWrite,
	}
	
	err := config.CompilePatterns()
	require.NoError(t, err)
	
	err = config.SetCurrentNotebook("Test Notebook")
	require.NoError(t, err)

	ctx := context.Background()
	req := mcp.CallToolRequest{}

	// Test section pattern matching
	resourceContext := ResourceContext{
		NotebookName: "Test Notebook",
		SectionName:  "public_docs",
		Operation:    OperationWrite,
	}
	err = config.IsAuthorized(ctx, "createPage", req, resourceContext)
	assert.NoError(t, err)

	// Test blocked section pattern
	resourceContext = ResourceContext{
		NotebookName: "Test Notebook",
		SectionName:  "old_archive",
		Operation:    OperationRead,
	}
	err = config.IsAuthorized(ctx, "listPages", req, resourceContext)
	assert.Error(t, err)

	// Test page pattern matching
	resourceContext = ResourceContext{
		NotebookName: "Test Notebook",
		SectionName:  "Documents",
		PageName:     "draft_document",
		Operation:    OperationWrite,
	}
	err = config.IsAuthorized(ctx, "updatePage", req, resourceContext)
	assert.NoError(t, err)
}

func TestSimplifiedAuthorizationConfig_SectionIDWithoutName_Security(t *testing.T) {
	config := NewAuthorizationConfig()
	config.Enabled = true
	config.DefaultNotebookPermissions = PermissionRead
	config.NotebookPermissions = map[string]PermissionLevel{
		"Clippings.io": PermissionWrite,
	}
	config.SectionPermissions = map[string]PermissionLevel{
		"Clippings.io/Books": PermissionRead, // Read-only section
	}
	
	err := config.CompilePatterns()
	require.NoError(t, err)
	
	err = config.SetCurrentNotebook("Clippings.io")
	require.NoError(t, err)

	ctx := context.Background()
	req := mcp.CallToolRequest{}

	// Test case 1: When section name is properly resolved, section permission should apply
	resourceContext := ResourceContext{
		NotebookName: "Clippings.io",
		SectionName:  "Books",  // Section name resolved correctly
		Operation:    OperationWrite,
	}
	err = config.IsAuthorized(ctx, "createPage", req, resourceContext)
	assert.Error(t, err) // Should be denied due to read-only section permission
	assert.Contains(t, err.Error(), "write")

	// Test case 2: SECURITY BUG - When section ID exists but section name is empty (resolution failed)
	// This simulates the exact scenario from the bug report
	resourceContext = ResourceContext{
		NotebookName: "Clippings.io",
		SectionID:    "0-4D24C77F19546939!s4073fa30fe08431cb33cc50a7af3fec3", // Section ID present
		SectionName:  "",  // Section name resolution failed (empty)
		Operation:    OperationWrite,
	}
	err = config.IsAuthorized(ctx, "createPage", req, resourceContext)
	assert.Error(t, err) // Should be denied due to fail-closed security model
	assert.Contains(t, err.Error(), "permission but only 'none' is granted") // Security model blocks access

	// Test case 3: Verify that notebook-level operations still work normally
	resourceContext = ResourceContext{
		NotebookName: "Clippings.io",
		Operation:    OperationWrite,
	}
	err = config.IsAuthorized(ctx, "createSection", req, resourceContext)
	assert.NoError(t, err) // Should be allowed (notebook has write permission)
}