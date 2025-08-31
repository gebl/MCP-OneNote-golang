// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetToolDescription(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		wantErr  bool
	}{
		{
			name:     "existing tool - getAuthStatus",
			toolName: "getAuthStatus",
			wantErr:  false,
		},
		{
			name:     "existing tool - listNotebooks",
			toolName: "listNotebooks",
			wantErr:  false,
		},
		{
			name:     "existing tool - createPage",
			toolName: "createPage",
			wantErr:  false,
		},
		{
			name:     "non-existent tool",
			toolName: "nonExistentTool",
			wantErr:  true,
		},
		{
			name:     "empty tool name",
			toolName: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desc, err := GetToolDescription(tt.toolName)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, desc)
				assert.Contains(t, err.Error(), "description not found for tool")
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, desc)
				assert.NotEmpty(t, desc)
			}
		})
	}
}

func TestMustGetToolDescription(t *testing.T) {
	t.Run("existing tool", func(t *testing.T) {
		desc := MustGetToolDescription("getAuthStatus")
		assert.NotEmpty(t, desc)
		assert.Contains(t, desc, "authentication")
	})

	t.Run("non-existent tool panics", func(t *testing.T) {
		assert.Panics(t, func() {
			MustGetToolDescription("nonExistentTool")
		})
	})
}

func TestGetAllDescriptions(t *testing.T) {
	all := GetAllDescriptions()
	
	// Verify we have all expected tools
	expectedTools := []string{
		"getAuthStatus", "refreshToken", "initiateAuth", "clearAuth",
		"listNotebooks", "createSection", "createSectionGroup", 
		"getSelectedNotebook", "selectNotebook", "getNotebookSections",
		"clearCache", "listPages", "getPageContent", "createPage",
		"updatePageContentAdvanced", "deletePage", "getPageItemContent",
		"listPageItems", "copyPage", "movePage", "updatePageContent",
	}
	
	for _, tool := range expectedTools {
		desc, exists := all[tool]
		assert.True(t, exists, "Missing tool description for: %s", tool)
		assert.NotEmpty(t, desc, "Empty description for tool: %s", tool)
	}
	
	// Verify returned map is a copy (modifications don't affect original)
	originalLen := len(all)
	all["testTool"] = "test description"
	
	all2 := GetAllDescriptions()
	assert.Equal(t, originalLen, len(all2))
	assert.NotContains(t, all2, "testTool")
}

func TestToolDescriptionsContent(t *testing.T) {
	// Test specific tool descriptions for key content
	tests := []struct {
		tool            string
		shouldContain   []string
		shouldNotContain []string
	}{
		{
			tool:          "listNotebooks",
			shouldContain: []string{"JSON array", "notebook ID", "name", "isAPIDefault"},
		},
		{
			tool:          "createPage",
			shouldContain: []string{"DO NOT CONVERT CONTENT", "PASS AS-IS", "Markdown", "HTML"},
		},
		{
			tool:          "updatePageContentAdvanced", 
			shouldContain: []string{"command-based targeting", "data-id", "append", "replace"},
		},
		{
			tool:          "getPageContent",
			shouldContain: []string{"HTML", "Markdown", "Text", "forUpdate"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.tool, func(t *testing.T) {
			desc, err := GetToolDescription(tt.tool)
			require.NoError(t, err)
			
			for _, content := range tt.shouldContain {
				assert.Contains(t, desc, content, 
					"Tool %s description should contain '%s'", tt.tool, content)
			}
			
			for _, content := range tt.shouldNotContain {
				assert.NotContains(t, desc, content,
					"Tool %s description should not contain '%s'", tt.tool, content)
			}
		})
	}
}

func TestToolDescriptionsMap(t *testing.T) {
	// Test that the internal map contains expected number of tools
	assert.Greater(t, len(toolDescriptions), 15, "Should have more than 15 tool descriptions")
	
	// Verify all descriptions are non-empty strings
	for toolName, desc := range toolDescriptions {
		assert.NotEmpty(t, desc, "Description for tool %s should not be empty", toolName)
		assert.IsType(t, "", desc, "Description for tool %s should be a string", toolName)
	}
}