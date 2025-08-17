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

// MockNotebookCache implements the NotebookCache interface for testing
type MockNotebookCache struct {
	notebook     map[string]interface{}
	displayName  string
	notebookID   string
	sectionNames map[string]string
}

func NewMockNotebookCache() *MockNotebookCache {
	return &MockNotebookCache{
		sectionNames: make(map[string]string),
	}
}

func (m *MockNotebookCache) SetNotebook(notebook map[string]interface{}) {
	m.notebook = notebook
	if id, ok := notebook["id"].(string); ok {
		m.notebookID = id
	}
	if name, ok := notebook["displayName"].(string); ok {
		m.displayName = name
	}
}

func (m *MockNotebookCache) GetNotebook() (map[string]interface{}, bool) {
	return m.notebook, m.notebook != nil
}

func (m *MockNotebookCache) GetDisplayName() (string, bool) {
	return m.displayName, m.displayName != ""
}

func (m *MockNotebookCache) GetNotebookID() (string, bool) {
	return m.notebookID, m.notebookID != ""
}

func (m *MockNotebookCache) GetSectionName(sectionID string) (string, bool) {
	name, exists := m.sectionNames[sectionID]
	return name, exists
}

func (m *MockNotebookCache) GetSectionNameWithProgress(ctx context.Context, sectionID string, mcpServer interface{}, progressToken string, graphClient interface{}) (string, bool) {
	// For testing purposes, this behaves the same as GetSectionName
	// In real implementation, this would make API calls if cache misses
	return m.GetSectionName(sectionID)
}

func (m *MockNotebookCache) SetSectionName(sectionID, name string) {
	m.sectionNames[sectionID] = name
}

// MockQuickNoteConfig implements the QuickNoteConfig interface for testing
type MockQuickNoteConfig struct {
	notebookName    string
	defaultNotebook string
	pageName        string
}

func NewMockQuickNoteConfig() *MockQuickNoteConfig {
	return &MockQuickNoteConfig{}
}

func (m *MockQuickNoteConfig) GetNotebookName() string {
	return m.notebookName
}

func (m *MockQuickNoteConfig) GetDefaultNotebook() string {
	return m.defaultNotebook
}

func (m *MockQuickNoteConfig) GetPageName() string {
	return m.pageName
}

func (m *MockQuickNoteConfig) SetNotebookName(name string) {
	m.notebookName = name
}

func (m *MockQuickNoteConfig) SetDefaultNotebook(name string) {
	m.defaultNotebook = name
}

func (m *MockQuickNoteConfig) SetPageName(name string) {
	m.pageName = name
}

func TestAuthorizationConfig_Basic(t *testing.T) {
	// Test creating a new authorization config
	config := NewAuthorizationConfig()
	assert.NotNil(t, config)
	assert.False(t, config.Enabled)
	assert.Equal(t, PermissionRead, config.DefaultMode)
	assert.NotNil(t, config.ToolPermissions)
	assert.NotNil(t, config.NotebookPermissions)
	assert.NotNil(t, config.SectionPermissions)
}

func TestAuthorizationConfig_CompileMatchers(t *testing.T) {
	config := NewAuthorizationConfig()
	
	// Add some test patterns
	config.NotebookPermissions = map[string]PermissionLevel{
		"Work*":           PermissionWrite,
		"Personal Notes":  PermissionRead,
		"Archive*":        PermissionRead,
		"Private*":        PermissionNone,
	}
	
	config.SectionPermissions = map[string]PermissionLevel{
		"*/Confidential":  PermissionNone,
		"Work*/Draft*":    PermissionRead,
		"Personal Notes/Public": PermissionWrite,
	}
	
	// Compile matchers
	err := config.CompileMatchers()
	require.NoError(t, err)
	
	// Verify matchers were created
	assert.Len(t, config.notebookMatchers, 4)
	assert.Len(t, config.sectionMatchers, 3)
}

func TestAuthorizationConfig_PermissionMatching(t *testing.T) {
	config := NewAuthorizationConfig()
	config.NotebookPermissions = map[string]PermissionLevel{
		"Work*":          PermissionWrite,
		"Personal Notes": PermissionRead,
		"Archive*":       PermissionRead,
		"Private*":       PermissionNone,
	}
	
	err := config.CompileMatchers()
	require.NoError(t, err)
	
	tests := []struct {
		notebook   string
		expected   PermissionLevel
		shouldFind bool
	}{
		{"Work Projects", PermissionWrite, true},
		{"Work Notes", PermissionWrite, true},
		{"Personal Notes", PermissionRead, true},
		{"Archive 2023", PermissionRead, true},
		{"Private Documents", PermissionNone, true},
		{"Unknown Notebook", "", false},
	}
	
	for _, test := range tests {
		t.Run(test.notebook, func(t *testing.T) {
			permission := config.matchPattern(test.notebook, config.notebookMatchers)
			if test.shouldFind {
				assert.Equal(t, test.expected, permission, "Pattern should match for notebook: %s", test.notebook)
			} else {
				assert.Equal(t, PermissionLevel(""), permission, "Pattern should not match for notebook: %s", test.notebook)
			}
		})
	}
}

func TestAuthorizationConfig_IsAuthorized(t *testing.T) {
	config := NewAuthorizationConfig()
	config.Enabled = true
	config.DefaultMode = PermissionRead
	config.ToolPermissions = map[ToolCategory]PermissionLevel{
		CategoryPageWrite: PermissionWrite,
		CategoryPageRead:  PermissionRead,
	}
	config.NotebookPermissions = map[string]PermissionLevel{
		"Work*":    PermissionWrite,
		"Private*": PermissionNone,
	}
	
	err := config.CompileMatchers()
	require.NoError(t, err)
	
	cache := NewMockNotebookCache()
	cache.SetNotebook(map[string]interface{}{
		"id":          "notebook-123",
		"displayName": "Work Projects",
	})
	
	ctx := context.Background()
	
	// Test authorized read operation
	req := createMockRequest("listPages", map[string]interface{}{
		"sectionID": "section-456",
	})
	
	resourceContext := ExtractResourceContext("listPages", req, cache)
	err = config.IsAuthorized(ctx, "listPages", req, resourceContext)
	assert.NoError(t, err, "Read operation should be authorized")
	
	// Test authorized write operation
	req = createMockRequest("createPage", map[string]interface{}{
		"sectionID": "section-456",
		"title":     "New Page",
		"content":   "Page content",
	})
	
	resourceContext = ExtractResourceContext("createPage", req, cache)
	err = config.IsAuthorized(ctx, "createPage", req, resourceContext)
	assert.NoError(t, err, "Write operation should be authorized for Work notebook")
	
	// Test unauthorized operation on private notebook using a non-filter tool
	cache.SetNotebook(map[string]interface{}{
		"id":          "notebook-789",
		"displayName": "Private Documents",
	})
	
	// Use getPageContent instead of listPages since listPages is now a filter tool
	reqGetPage := createMockRequest("getPageContent", map[string]interface{}{
		"pageID": "page-123",
	})
	
	resourceContext = ExtractResourceContext("getPageContent", reqGetPage, cache)
	err = config.IsAuthorized(ctx, "getPageContent", reqGetPage, resourceContext)
	assert.Error(t, err, "Operation should be denied for Private notebook")
}

func TestAuthorizationConfig_Disabled(t *testing.T) {
	config := NewAuthorizationConfig()
	config.Enabled = false // Authorization disabled
	
	cache := NewMockNotebookCache()
	ctx := context.Background()
	req := createMockRequest("deletePage", map[string]interface{}{
		"pageID": "page-123",
	})
	
	resourceContext := ExtractResourceContext("deletePage", req, cache)
	err := config.IsAuthorized(ctx, "deletePage", req, resourceContext)
	assert.NoError(t, err, "All operations should be allowed when authorization is disabled")
}

func TestExtractResourceContext(t *testing.T) {
	cache := NewMockNotebookCache()
	cache.SetNotebook(map[string]interface{}{
		"id":          "notebook-123",
		"displayName": "Test Notebook",
	})
	cache.SetSectionName("section-456", "Test Section")
	
	tests := []struct {
		name     string
		toolName string
		args     map[string]interface{}
		expected ResourceContext
	}{
		{
			name:     "listPages with section ID",
			toolName: "listPages",
			args:     map[string]interface{}{"sectionID": "section-456"},
			expected: ResourceContext{
				NotebookName: "Test Notebook",
				NotebookID:   "notebook-123",
				SectionName:  "Test Section",
				SectionID:    "section-456",
				Operation:    OperationRead,
			},
		},
		{
			name:     "createPage with section ID",
			toolName: "createPage",
			args:     map[string]interface{}{"sectionID": "section-456", "title": "New Page"},
			expected: ResourceContext{
				NotebookName: "Test Notebook",
				NotebookID:   "notebook-123",
				SectionName:  "Test Section",
				SectionID:    "section-456",
				Operation:    OperationWrite,
			},
		},
		{
			name:     "deletePage with page ID",
			toolName: "deletePage",
			args:     map[string]interface{}{"pageID": "page-789"},
			expected: ResourceContext{
				NotebookName: "Test Notebook",
				NotebookID:   "notebook-123",
				PageID:       "page-789",
				Operation:    OperationWrite,
			},
		},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := createMockRequest(test.toolName, test.args)
			context := ExtractResourceContext(test.toolName, req, cache)
			
			assert.Equal(t, test.expected.NotebookName, context.NotebookName)
			assert.Equal(t, test.expected.NotebookID, context.NotebookID)
			assert.Equal(t, test.expected.SectionName, context.SectionName)
			assert.Equal(t, test.expected.SectionID, context.SectionID)
			assert.Equal(t, test.expected.PageID, context.PageID)
			assert.Equal(t, test.expected.Operation, context.Operation)
		})
	}
}

func TestAuthorizedToolHandler(t *testing.T) {
	config := NewAuthorizationConfig()
	config.Enabled = true
	config.DefaultMode = PermissionRead
	config.ToolPermissions = map[ToolCategory]PermissionLevel{
		CategoryPageWrite: PermissionNone, // Block all write operations
	}
	
	err := config.CompileMatchers()
	require.NoError(t, err)
	
	cache := NewMockNotebookCache()
	quickNote := NewMockQuickNoteConfig()
	
	// Create a mock handler
	handlerCalled := false
	mockHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		handlerCalled = true
		return mcp.NewToolResultText("Success"), nil
	}
	
	// Wrap handler with authorization
	wrappedHandler := AuthorizedToolHandler("createPage", mockHandler, config, cache, quickNote)
	
	// Test blocked operation
	ctx := context.Background()
	req := createMockRequest("createPage", map[string]interface{}{
		"sectionID": "section-123",
		"title":     "New Page",
	})
	
	result, err := wrappedHandler(ctx, req)
	assert.NoError(t, err, "Handler should not return error")
	assert.NotNil(t, result)
	assert.False(t, handlerCalled, "Original handler should not be called when authorization fails")
	
	// Result should be an error result  
	if len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(mcp.TextContent); ok {
			assert.Contains(t, textContent.Text, "Authorization failed")
		}
	}
}

func TestValidateAuthorizationConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *AuthorizationConfig
		expectError bool
	}{
		{
			name:        "nil config is valid",
			config:      nil,
			expectError: false,
		},
		{
			name: "valid config",
			config: &AuthorizationConfig{
				Enabled:     true,
				DefaultMode: PermissionRead,
				ToolPermissions: map[ToolCategory]PermissionLevel{
					CategoryPageRead: PermissionRead,
				},
				NotebookPermissions: map[string]PermissionLevel{
					"Work*": PermissionWrite,
				},
			},
			expectError: false,
		},
		{
			name: "invalid default mode",
			config: &AuthorizationConfig{
				Enabled:     true,
				DefaultMode: PermissionLevel("invalid"),
			},
			expectError: true,
		},
		{
			name: "invalid tool permission",
			config: &AuthorizationConfig{
				Enabled:     true,
				DefaultMode: PermissionRead,
				ToolPermissions: map[ToolCategory]PermissionLevel{
					CategoryPageRead: PermissionLevel("invalid"),
				},
			},
			expectError: true,
		},
		{
			name: "invalid tool category",
			config: &AuthorizationConfig{
				Enabled:     true,
				DefaultMode: PermissionRead,
				ToolPermissions: map[ToolCategory]PermissionLevel{
					ToolCategory("invalid"): PermissionRead,
				},
			},
			expectError: true,
		},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateAuthorizationConfig(test.config)
			if test.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFilterNotebooks(t *testing.T) {
	config := NewAuthorizationConfig()
	config.Enabled = true
	config.DefaultMode = PermissionNone
	config.DefaultNotebookMode = PermissionNone // Explicitly set notebook default to none
	config.NotebookPermissions = map[string]PermissionLevel{
		"Work*":    PermissionWrite,
		"Public*":  PermissionRead,
		"Private*": PermissionNone,
	}
	
	err := config.CompileMatchers()
	require.NoError(t, err)
	
	// Create test notebooks
	notebooks := []map[string]interface{}{
		{"displayName": "Work Projects", "id": "work-1"},
		{"displayName": "Public Notes", "id": "public-1"},
		{"displayName": "Private Documents", "id": "private-1"},
		{"displayName": "Archive Stuff", "id": "archive-1"},
	}
	
	filtered := config.FilterNotebooks(notebooks)
	
	// Should include Work* and Public*, exclude Private*
	// Archive* falls back to default (none) so excluded
	assert.Len(t, filtered, 2)
	
	// Check specific notebooks are included
	var names []string
	for _, nb := range filtered {
		names = append(names, nb["displayName"].(string))
	}
	assert.Contains(t, names, "Work Projects")
	assert.Contains(t, names, "Public Notes")
	assert.NotContains(t, names, "Private Documents")
	assert.NotContains(t, names, "Archive Stuff")
}

func TestFilterSections(t *testing.T) {
	config := NewAuthorizationConfig()
	config.Enabled = true
	config.DefaultMode = PermissionRead
	config.NotebookPermissions = map[string]PermissionLevel{
		"Work*": PermissionWrite,
	}
	config.SectionPermissions = map[string]PermissionLevel{
		"Work*/Confidential": PermissionNone,
		"*/Public":           PermissionWrite,
	}
	
	err := config.CompileMatchers()
	require.NoError(t, err)
	
	// Create test sections
	sections := []map[string]interface{}{
		{"displayName": "General", "id": "section-1"},
		{"displayName": "Confidential", "id": "section-2"},
		{"displayName": "Public", "id": "section-3"},
		{"displayName": "Draft", "id": "section-4"},
	}
	
	filtered := config.FilterSections(sections, "Work Projects")
	
	// Should include General (inherit work write), Public (explicit write)
	// Should exclude Confidential (explicit none)
	// Should include Draft (inherit work write)
	assert.Len(t, filtered, 3)
	
	var names []string
	for _, section := range filtered {
		names = append(names, section["displayName"].(string))
	}
	assert.Contains(t, names, "General")
	assert.Contains(t, names, "Public") 
	assert.Contains(t, names, "Draft")
	assert.NotContains(t, names, "Confidential")
}

func TestFilterPages(t *testing.T) {
	config := NewAuthorizationConfig()
	config.Enabled = true
	config.DefaultMode = PermissionRead
	config.DefaultNotebookMode = PermissionRead // Explicitly set to allow "Work Projects"
	config.DefaultSectionMode = ""              // Empty so it falls back to notebook permissions
	config.NotebookPermissions = map[string]PermissionLevel{
		"Private*": PermissionNone,
	}
	
	err := config.CompileMatchers()
	require.NoError(t, err)
	
	// Create test pages
	pages := []map[string]interface{}{
		{"title": "Page 1", "id": "page-1"},
		{"title": "Page 2", "id": "page-2"},
		{"title": "Page 3", "id": "page-3"},
	}
	
	// Test with allowed notebook
	filteredAllowed := config.FilterPages(pages, "section-1", "General", "Work Projects")
	assert.Len(t, filteredAllowed, 3) // All pages allowed
	
	// Test with blocked notebook
	filteredBlocked := config.FilterPages(pages, "section-2", "Secret", "Private Notes")
	assert.Len(t, filteredBlocked, 0) // All pages blocked
}

func TestFilterPagesWithPagePermissions(t *testing.T) {
	config := NewAuthorizationConfig()
	config.Enabled = true
	config.DefaultMode = PermissionNone // Block by default
	config.NotebookPermissions = map[string]PermissionLevel{
		"My Notebook": PermissionWrite, // Allow access to My Notebook
	}
	config.PagePermissions = map[string]PermissionLevel{
		"Notes":           PermissionWrite, // Allow Notes page
		"Draft*":          PermissionRead,  // Allow Draft pages  
		"Secret*":         PermissionNone,  // Block Secret pages
		"Another*":        PermissionNone,  // Block Another pages
	}
	
	err := config.CompileMatchers()
	require.NoError(t, err)
	
	// Create test pages
	pages := []map[string]interface{}{
		{"title": "Notes", "id": "page-1"},
		{"title": "Draft Ideas", "id": "page-2"}, 
		{"title": "Draft Plans", "id": "page-3"},
		{"title": "Secret Document", "id": "page-4"},
		{"title": "Another Page", "id": "page-5"},
	}
	
	// Test page-level filtering
	filtered := config.FilterPages(pages, "section-1", "General", "My Notebook")
	
	// Should only return Notes and Draft pages
	assert.Len(t, filtered, 3)
	
	// Check the returned pages
	var titles []string
	for _, page := range filtered {
		if title, ok := page["title"].(string); ok {
			titles = append(titles, title)
		}
	}
	
	assert.Contains(t, titles, "Notes")
	assert.Contains(t, titles, "Draft Ideas")
	assert.Contains(t, titles, "Draft Plans")
	assert.NotContains(t, titles, "Secret Document")
	assert.NotContains(t, titles, "Another Page")
}

func TestFilterPagesWithoutPagePermissions(t *testing.T) {
	config := NewAuthorizationConfig()
	config.Enabled = true
	config.DefaultMode = PermissionRead
	config.NotebookPermissions = map[string]PermissionLevel{
		"My Notebook": PermissionWrite,
	}
	// No page permissions defined
	
	err := config.CompileMatchers()
	require.NoError(t, err)
	
	// Create test pages  
	pages := []map[string]interface{}{
		{"title": "Notes", "id": "page-1"},
		{"title": "Secret Document", "id": "page-2"},
		{"title": "Another Page", "id": "page-3"},
	}
	
	// Test without page permissions - should return all pages
	filtered := config.FilterPages(pages, "section-1", "General", "My Notebook")
	assert.Len(t, filtered, 3) // All pages allowed when no page permissions
}

func TestFilteringDisabled(t *testing.T) {
	config := NewAuthorizationConfig()
	config.Enabled = false // Disabled
	
	notebooks := []map[string]interface{}{
		{"displayName": "Test", "id": "test-1"},
	}
	
	filtered := config.FilterNotebooks(notebooks)
	assert.Equal(t, notebooks, filtered) // Should return unchanged
}

func TestGetSelectedNotebook(t *testing.T) {
	config := NewAuthorizationConfig()
	config.Enabled = true
	config.DefaultMode = PermissionNone
	config.ToolPermissions = map[ToolCategory]PermissionLevel{
		CategoryNotebookRead: PermissionRead, // Allow notebook read tools
	}
	config.NotebookPermissions = map[string]PermissionLevel{
		"My Notebook": PermissionRead,  // Allow access to My Notebook
		"Private*":    PermissionNone,  // Block private notebooks
	}
	
	err := config.CompileMatchers()
	require.NoError(t, err)
	
	cache := NewMockNotebookCache()
	ctx := context.Background()
	
	// Test with allowed notebook
	cache.SetNotebook(map[string]interface{}{
		"id":          "notebook-123",
		"displayName": "My Notebook",
	})
	
	req := createMockRequest("getSelectedNotebook", map[string]interface{}{})
	resourceContext := ExtractResourceContext("getSelectedNotebook", req, cache)
	err = config.IsAuthorized(ctx, "getSelectedNotebook", req, resourceContext)
	assert.NoError(t, err, "getSelectedNotebook should work for allowed notebook")
	
	// Test with blocked notebook
	cache.SetNotebook(map[string]interface{}{
		"id":          "notebook-789",
		"displayName": "Private Documents",
	})
	
	req = createMockRequest("getSelectedNotebook", map[string]interface{}{})
	resourceContext = ExtractResourceContext("getSelectedNotebook", req, cache)
	err = config.IsAuthorized(ctx, "getSelectedNotebook", req, resourceContext)
	assert.Error(t, err, "getSelectedNotebook should be blocked for private notebook")
	
	// Test with no notebook selected (should fail due to default_mode: none)
	cacheEmpty := NewMockNotebookCache() // Empty cache, no notebook set
	req = createMockRequest("getSelectedNotebook", map[string]interface{}{})
	resourceContext = ExtractResourceContext("getSelectedNotebook", req, cacheEmpty)
	err = config.IsAuthorized(ctx, "getSelectedNotebook", req, resourceContext)
	assert.Error(t, err, "getSelectedNotebook should be blocked when no notebook is selected and default_mode is none")
}

func TestFilterToolsBypassResourceChecks(t *testing.T) {
	config := NewAuthorizationConfig()
	config.Enabled = true
	config.DefaultMode = PermissionNone
	config.ToolPermissions = map[ToolCategory]PermissionLevel{
		CategoryNotebookRead: PermissionRead, // Allow notebook read tools
	}
	config.NotebookPermissions = map[string]PermissionLevel{
		"Private*": PermissionNone, // Block private notebooks
	}
	
	err := config.CompileMatchers()
	require.NoError(t, err)
	
	cache := NewMockNotebookCache()
	cache.SetNotebook(map[string]interface{}{
		"id":          "notebook-789",
		"displayName": "Private Documents",
	})
	
	ctx := context.Background()
	
	// Test filter tool (should pass despite Private notebook being blocked)
	reqFilter := createMockRequest("listNotebooks", map[string]interface{}{})
	resourceContext := ExtractResourceContext("listNotebooks", reqFilter, cache)
	err = config.IsAuthorized(ctx, "listNotebooks", reqFilter, resourceContext)
	assert.NoError(t, err, "Filter tool should bypass resource-level checks")
	
	// Test non-filter tool (should fail for Private notebook)
	reqNonFilter := createMockRequest("getPageContent", map[string]interface{}{
		"pageID": "page-123",
	})
	resourceContext = ExtractResourceContext("getPageContent", reqNonFilter, cache)
	err = config.IsAuthorized(ctx, "getPageContent", reqNonFilter, resourceContext)
	assert.Error(t, err, "Non-filter tool should be blocked for Private notebook")
}

// Helper function to create mock MCP requests
func createMockRequest(toolName string, args map[string]interface{}) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: args,
		},
	}
}

func TestNotebookCacheAdapter(t *testing.T) {
	// Create a mock cache that supports GetSectionName
	mockCache := NewMockNotebookCache()
	mockCache.SetSectionName("section-123", "Test Section")
	
	// Create adapter - cast to MainNotebookCache interface for the adapter
	adapter := NewNotebookCacheAdapter(mockCache)
	
	// Test GetSectionName
	sectionName, found := adapter.GetSectionName("section-123")
	assert.True(t, found, "Section should be found")
	assert.Equal(t, "Test Section", sectionName, "Section name should match")
	
	// Test non-existent section
	sectionName, found = adapter.GetSectionName("section-999")
	assert.False(t, found, "Non-existent section should not be found")
	assert.Equal(t, "", sectionName, "Section name should be empty for non-existent section")
}

// Benchmark tests
func BenchmarkAuthorizationCheck(b *testing.B) {
	config := NewAuthorizationConfig()
	config.Enabled = true
	config.DefaultMode = PermissionRead
	config.NotebookPermissions = map[string]PermissionLevel{
		"Work*":          PermissionWrite,
		"Personal*":      PermissionRead,
		"Archive*":       PermissionRead,
		"Private*":       PermissionNone,
		"Test Notebook":  PermissionWrite,
	}
	
	config.SectionPermissions = map[string]PermissionLevel{
		"*/Confidential": PermissionNone,
		"Work*/Draft*":   PermissionRead,
		"*/Public":       PermissionWrite,
	}
	
	err := config.CompileMatchers()
	require.NoError(b, err)
	
	cache := NewMockNotebookCache()
	cache.SetNotebook(map[string]interface{}{
		"id":          "notebook-123",
		"displayName": "Work Projects",
	})
	
	ctx := context.Background()
	req := createMockRequest("listPages", map[string]interface{}{
		"sectionID": "section-456",
	})
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		resourceContext := ExtractResourceContext("listPages", req, cache)
		_ = config.IsAuthorized(ctx, "listPages", req, resourceContext)
	}
}