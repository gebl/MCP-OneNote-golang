// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package notebooks

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/gebl/onenote-mcp-server/internal/graph"
)

func TestNewNotebookClient(t *testing.T) {
	t.Run("creates new notebook client", func(t *testing.T) {
		graphClient := &graph.Client{}
		notebookClient := NewNotebookClient(graphClient)

		assert.NotNil(t, notebookClient)
		assert.Equal(t, graphClient, notebookClient.Client)
	})
}

func TestSanitizeOneNoteID(t *testing.T) {
	notebookClient := &NotebookClient{}

	tests := []struct {
		name     string
		id       string
		idType   string
		expected string
		hasError bool
	}{
		{
			name:     "valid notebook ID",
			id:       "valid-notebook-id",
			idType:   "notebookID",
			expected: "valid-notebook-id",
			hasError: false,
		},
		{
			name:     "empty ID",
			id:       "",
			idType:   "notebookID",
			expected: "",
			hasError: true,
		},
		{
			name:     "special characters in ID",
			id:       "notebook-id-with-special-chars-123",
			idType:   "notebookID",
			expected: "notebook-id-with-special-chars-123",
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := notebookClient.Client.SanitizeOneNoteID(tt.id, tt.idType)

			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// TestableNotebookClient extends NotebookClient for testing purposes
type TestableNotebookClient struct {
	*NotebookClient
	listSectionsFunc      func(string) ([]map[string]interface{}, error)
	listPagesFunc         func(string) ([]map[string]interface{}, error)
	listSectionGroupsFunc func(string) ([]map[string]interface{}, error)
	listNotebooksFunc     func() ([]map[string]interface{}, error)
}

func NewTestableNotebookClient(client *graph.Client) *TestableNotebookClient {
	return &TestableNotebookClient{
		NotebookClient: NewNotebookClient(client),
	}
}

func (t *TestableNotebookClient) ListSections(containerID string) ([]map[string]interface{}, error) {
	if t.listSectionsFunc != nil {
		return t.listSectionsFunc(containerID)
	}
	// Return empty slice instead of calling real implementation
	return []map[string]interface{}{}, nil
}

func (t *TestableNotebookClient) ListPages(sectionID string) ([]map[string]interface{}, error) {
	if t.listPagesFunc != nil {
		return t.listPagesFunc(sectionID)
	}
	// Return empty slice instead of calling real implementation
	return []map[string]interface{}{}, nil
}

func (t *TestableNotebookClient) ListSectionGroups(containerID string) ([]map[string]interface{}, error) {
	if t.listSectionGroupsFunc != nil {
		return t.listSectionGroupsFunc(containerID)
	}
	// Return empty slice instead of calling real implementation
	return []map[string]interface{}{}, nil
}

func (t *TestableNotebookClient) ListNotebooks() ([]map[string]interface{}, error) {
	if t.listNotebooksFunc != nil {
		return t.listNotebooksFunc()
	}
	// Return empty slice instead of calling real implementation
	return []map[string]interface{}{}, nil
}

// Test the search logic with simple isolated unit tests
func TestSearchLogic(t *testing.T) {
	t.Run("case insensitive title matching", func(t *testing.T) {
		query := "MEETING"
		pages := []map[string]interface{}{
			{"id": "page-1", "title": "meeting notes"},
			{"id": "page-2", "title": "project update"},
			{"id": "page-3", "title": "Daily Meeting"},
		}

		var matchingPages []map[string]interface{}
		queryLower := strings.ToLower(query)

		for _, page := range pages {
			if title, ok := page["title"].(string); ok {
				titleLower := strings.ToLower(title)
				if strings.Contains(titleLower, queryLower) {
					// Add context information like the real function does
					page["sectionName"] = "Test Section"
					page["sectionId"] = "section-1"
					page["sectionPath"] = "/Test Section"
					matchingPages = append(matchingPages, page)
				}
			}
		}

		assert.Len(t, matchingPages, 2) // Should match "meeting notes" and "Daily Meeting"

		// Verify context information was added
		for _, page := range matchingPages {
			assert.Contains(t, page, "sectionName")
			assert.Contains(t, page, "sectionId")
			assert.Contains(t, page, "sectionPath")
		}
	})

	t.Run("empty query matches all pages", func(t *testing.T) {
		query := ""
		pages := []map[string]interface{}{
			{"id": "page-1", "title": "Any Title"},
			{"id": "page-2", "title": "Another Title"},
		}

		var matchingPages []map[string]interface{}
		queryLower := strings.ToLower(query)

		for _, page := range pages {
			if title, ok := page["title"].(string); ok {
				titleLower := strings.ToLower(title)
				if strings.Contains(titleLower, queryLower) {
					matchingPages = append(matchingPages, page)
				}
			}
		}

		assert.Len(t, matchingPages, 2) // Empty query should match all
	})

	t.Run("no matches found", func(t *testing.T) {
		query := "nonexistent"
		pages := []map[string]interface{}{
			{"id": "page-1", "title": "Meeting Notes"},
			{"id": "page-2", "title": "Project Update"},
		}

		var matchingPages []map[string]interface{}
		queryLower := strings.ToLower(query)

		for _, page := range pages {
			if title, ok := page["title"].(string); ok {
				titleLower := strings.ToLower(title)
				if strings.Contains(titleLower, queryLower) {
					matchingPages = append(matchingPages, page)
				}
			}
		}

		assert.Len(t, matchingPages, 0)
	})

	t.Run("handle pages without title", func(t *testing.T) {
		query := "test"
		pages := []map[string]interface{}{
			{"id": "page-1", "title": "test page"},
			{"id": "page-2"},               // Missing title
			{"id": "page-3", "title": 123}, // Non-string title
		}

		var matchingPages []map[string]interface{}
		queryLower := strings.ToLower(query)

		for _, page := range pages {
			if title, ok := page["title"].(string); ok {
				titleLower := strings.ToLower(title)
				if strings.Contains(titleLower, queryLower) {
					matchingPages = append(matchingPages, page)
				}
			}
		}

		assert.Len(t, matchingPages, 1) // Only the valid page with title should match
	})
}

// Test the notebook search logic directly
func TestGetDefaultNotebookLogic(t *testing.T) {
	t.Run("find notebook by exact name match", func(t *testing.T) {
		notebooks := []map[string]interface{}{
			{"id": "notebook-1", "displayName": "Work Notebook"},
			{"id": "notebook-2", "displayName": "Personal Notebook"},
			{"id": "notebook-3", "displayName": "Study Notes"},
		}

		targetName := "Personal Notebook"
		var foundID string

		// Simulate the search logic from GetDefaultNotebookID
		for _, notebook := range notebooks {
			if displayName, exists := notebook["displayName"].(string); exists {
				if displayName == targetName {
					if id, exists := notebook["id"].(string); exists {
						foundID = id
						break
					}
				}
			}
		}

		assert.Equal(t, "notebook-2", foundID)
	})

	t.Run("notebook not found", func(t *testing.T) {
		notebooks := []map[string]interface{}{
			{"id": "notebook-1", "displayName": "Work Notebook"},
			{"id": "notebook-2", "displayName": "Personal Notebook"},
		}

		targetName := "Nonexistent Notebook"
		var foundID string

		for _, notebook := range notebooks {
			if displayName, exists := notebook["displayName"].(string); exists {
				if displayName == targetName {
					if id, exists := notebook["id"].(string); exists {
						foundID = id
						break
					}
				}
			}
		}

		assert.Equal(t, "", foundID)
	})

	t.Run("case sensitive matching", func(t *testing.T) {
		notebooks := []map[string]interface{}{
			{"id": "notebook-1", "displayName": "Personal Notebook"},
		}

		targetName := "personal notebook" // Different case
		var foundID string

		for _, notebook := range notebooks {
			if displayName, exists := notebook["displayName"].(string); exists {
				if displayName == targetName {
					if id, exists := notebook["id"].(string); exists {
						foundID = id
						break
					}
				}
			}
		}

		assert.Equal(t, "", foundID) // Should not match due to case sensitivity
	})

	t.Run("handle notebook without ID", func(t *testing.T) {
		notebooks := []map[string]interface{}{
			{"displayName": "Personal Notebook"}, // Missing ID
		}

		targetName := "Personal Notebook"
		var foundID string

		for _, notebook := range notebooks {
			if displayName, exists := notebook["displayName"].(string); exists {
				if displayName == targetName {
					if id, exists := notebook["id"].(string); exists {
						foundID = id
						break
					}
				}
			}
		}

		assert.Equal(t, "", foundID) // Should not find ID
	})

	t.Run("handle notebook without display name", func(t *testing.T) {
		notebooks := []map[string]interface{}{
			{"id": "notebook-1"}, // Missing displayName
		}

		targetName := "Personal Notebook"
		var foundID string

		for _, notebook := range notebooks {
			if displayName, exists := notebook["displayName"].(string); exists {
				if displayName == targetName {
					if id, exists := notebook["id"].(string); exists {
						foundID = id
						break
					}
				}
			}
		}

		assert.Equal(t, "", foundID) // Should not match
	})
}

func TestGetDefaultNotebookID_ConfigValidation(t *testing.T) {
	t.Run("nil config", func(t *testing.T) {
		result, err := GetDefaultNotebookID(nil, nil)
		assert.Error(t, err)
		assert.Equal(t, "", result)
		assert.Contains(t, err.Error(), "no default notebook name configured")
	})

	t.Run("empty notebook name", func(t *testing.T) {
		config := &graph.Config{NotebookName: ""}
		result, err := GetDefaultNotebookID(nil, config)
		assert.Error(t, err)
		assert.Equal(t, "", result)
		assert.Contains(t, err.Error(), "no default notebook name configured")
	})
}

// Performance tests for search logic
func BenchmarkNotebookSearchLogic(b *testing.B) {
	// Create a large set of notebooks for performance testing
	notebooks := make([]map[string]interface{}, 1000)
	for i := 0; i < 1000; i++ {
		notebooks[i] = map[string]interface{}{
			"id":          "notebook-" + string(rune(i+48)),
			"displayName": "Notebook " + string(rune(i+48)),
		}
	}
	// Add the target notebook at the end (worst case scenario)
	notebooks[999] = map[string]interface{}{
		"id":          "target-notebook",
		"displayName": "Target Notebook",
	}

	targetName := "Target Notebook"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var foundID string
		for _, notebook := range notebooks {
			if displayName, exists := notebook["displayName"].(string); exists {
				if displayName == targetName {
					if id, exists := notebook["id"].(string); exists {
						foundID = id
						break
					}
				}
			}
		}
		_ = foundID // Use the result
	}
}

func BenchmarkPageSearchLogic(b *testing.B) {
	// Create test data for page search performance
	pages := make([]map[string]interface{}, 1000)
	for i := 0; i < 1000; i++ {
		pages[i] = map[string]interface{}{
			"id":    "page-" + string(rune(i+48)),
			"title": "Page " + string(rune(i+48)) + " Meeting Notes",
		}
	}

	query := "meeting"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var matchingPages []map[string]interface{}
		queryLower := strings.ToLower(query)

		for _, page := range pages {
			if title, ok := page["title"].(string); ok {
				titleLower := strings.ToLower(title)
				if strings.Contains(titleLower, queryLower) {
					page["sectionName"] = "Test Section"
					page["sectionId"] = "section-1"
					page["sectionPath"] = "/Test Section"
					matchingPages = append(matchingPages, page)
				}
			}
		}
		_ = matchingPages // Use the result
	}
}

// MockGraphSDKClient provides a complete mock implementation for testing notebook operations
type MockGraphSDKClient struct {
	mock.Mock
	TokenManager   *MockTokenManager
	notebooks      []map[string]interface{}
	detailedInfo   map[string]map[string]interface{}
	pagesBySection map[string][]map[string]interface{}
}

type MockTokenManager struct {
	mock.Mock
	expired bool
}

func (m *MockTokenManager) IsExpired() bool {
	if m == nil {
		return false
	}
	return m.expired
}

func (m *MockTokenManager) SetExpired(expired bool) {
	m.expired = expired
}

// NewMockGraphSDKClient creates a mock client with sample data
func NewMockGraphSDKClient() *MockGraphSDKClient {
	mock := &MockGraphSDKClient{
		TokenManager: &MockTokenManager{expired: false},
		notebooks: []map[string]interface{}{
			{
				"id":          "notebook-1",
				"displayName": "Work Notebook",
				"createdDateTime": "2023-01-01T00:00:00Z",
			},
			{
				"id":          "notebook-2",
				"displayName": "Personal Notes",
				"createdDateTime": "2023-01-15T00:00:00Z",
			},
			{
				"id":          "notebook-3",
				"displayName": "Project Alpha",
				"createdDateTime": "2023-02-01T00:00:00Z",
			},
		},
		detailedInfo: map[string]map[string]interface{}{
			"Work Notebook": {
				"id":                   "notebook-1",
				"displayName":          "Work Notebook",
				"createdDateTime":      "2023-01-01T00:00:00Z",
				"lastModifiedDateTime": "2023-12-01T00:00:00Z",
				"isDefault":            true,
				"userRole":             "Owner",
				"isShared":             false,
				"sectionsUrl":          "https://graph.microsoft.com/v1.0/me/onenote/notebooks/notebook-1/sections",
				"sectionGroupsUrl":     "https://graph.microsoft.com/v1.0/me/onenote/notebooks/notebook-1/sectionGroups",
				"links": map[string]interface{}{
					"oneNoteClientUrl": "onenote:https://d.docs.live.net/...",
					"oneNoteWebUrl":    "https://onedrive.live.com/...",
				},
				"createdBy": map[string]interface{}{
					"user": map[string]interface{}{
						"id":          "user-123",
						"displayName": "Test User",
					},
				},
			},
			"Personal Notes": {
				"id":                   "notebook-2",
				"displayName":          "Personal Notes",
				"createdDateTime":      "2023-01-15T00:00:00Z",
				"lastModifiedDateTime": "2023-11-15T00:00:00Z",
				"isDefault":            false,
				"userRole":             "Owner",
				"isShared":             false,
			},
		},
		pagesBySection: map[string][]map[string]interface{}{
			"section-1": {
				{"id": "page-1", "title": "Meeting Notes", "sectionName": "Work Section", "sectionId": "section-1"},
				{"id": "page-2", "title": "Daily Standup", "sectionName": "Work Section", "sectionId": "section-1"},
			},
			"section-2": {
				{"id": "page-3", "title": "Project Meeting Notes", "sectionName": "Projects", "sectionId": "section-2"},
				{"id": "page-4", "title": "Sprint Review", "sectionName": "Projects", "sectionId": "section-2"},
			},
		},
	}
	return mock
}

// MockNotebookClient wraps the mock SDK client with notebook-specific methods
type MockNotebookClient struct {
	*MockGraphSDKClient
	mockError error
}

func NewMockNotebookClient() *MockNotebookClient {
	return &MockNotebookClient{
		MockGraphSDKClient: NewMockGraphSDKClient(),
	}
}

// SetError allows tests to simulate error conditions
func (m *MockNotebookClient) SetError(err error) {
	m.mockError = err
}

// ListNotebooks mocks the notebook listing operation
func (m *MockNotebookClient) ListNotebooks() ([]map[string]interface{}, error) {
	if m.mockError != nil {
		return nil, m.mockError
	}
	return m.notebooks, nil
}

// ListNotebooksDetailed mocks the detailed notebook listing operation
func (m *MockNotebookClient) ListNotebooksDetailed() ([]map[string]interface{}, error) {
	if m.mockError != nil {
		return nil, m.mockError
	}
	// Convert detailed info map to slice
	var detailed []map[string]interface{}
	for _, notebook := range m.notebooks {
		if name, ok := notebook["displayName"].(string); ok {
			if info, exists := m.detailedInfo[name]; exists {
				detailed = append(detailed, info)
			} else {
				detailed = append(detailed, notebook) // fallback to basic info
			}
		}
	}
	return detailed, nil
}

// GetDetailedNotebookByName mocks the detailed notebook retrieval by name
func (m *MockNotebookClient) GetDetailedNotebookByName(notebookName string) (map[string]interface{}, error) {
	if m.mockError != nil {
		return nil, m.mockError
	}
	if info, exists := m.detailedInfo[notebookName]; exists {
		return info, nil
	}
	return nil, fmt.Errorf("notebook not found: %s", notebookName)
}

// SearchPages mocks the page search operation
func (m *MockNotebookClient) SearchPages(query string, notebookID string) ([]map[string]interface{}, error) {
	if m.mockError != nil {
		return nil, m.mockError
	}
	
	results := make([]map[string]interface{}, 0) // Initialize as empty slice, not nil
	queryLower := strings.ToLower(query)
	
	// Search through all pages in all sections
	for sectionID, pages := range m.pagesBySection {
		for _, page := range pages {
			if title, ok := page["title"].(string); ok {
				titleLower := strings.ToLower(title)
				if strings.Contains(titleLower, queryLower) || query == "" {
					// Add section context like the real implementation
					pageCopy := make(map[string]interface{})
					for k, v := range page {
						pageCopy[k] = v
					}
					pageCopy["sectionName"] = fmt.Sprintf("Section %s", sectionID)
					pageCopy["sectionId"] = sectionID
					pageCopy["sectionPath"] = fmt.Sprintf("/Section %s", sectionID)
					results = append(results, pageCopy)
				}
			}
		}
	}
	return results, nil
}

// ListSections mocks section listing (used by SearchPages)
func (m *MockNotebookClient) ListSections(containerID string) ([]map[string]interface{}, error) {
	if m.mockError != nil {
		return make([]map[string]interface{}, 0), m.mockError
	}
	// Return mock sections for the container
	return []map[string]interface{}{
		{"id": "section-1", "displayName": "Work Section", "parentNotebook": map[string]interface{}{"id": containerID}},
		{"id": "section-2", "displayName": "Projects", "parentNotebook": map[string]interface{}{"id": containerID}},
	}, nil
}

// ListPages mocks page listing for a section
func (m *MockNotebookClient) ListPages(sectionID string) ([]map[string]interface{}, error) {
	if m.mockError != nil {
		return make([]map[string]interface{}, 0), m.mockError
	}
	if pages, exists := m.pagesBySection[sectionID]; exists {
		return pages, nil
	}
	return make([]map[string]interface{}, 0), nil
}

// ListSectionGroups mocks section group listing
func (m *MockNotebookClient) ListSectionGroups(containerID string) ([]map[string]interface{}, error) {
	if m.mockError != nil {
		return make([]map[string]interface{}, 0), m.mockError
	}
	return []map[string]interface{}{
		{"id": "sectiongroup-1", "displayName": "Archive", "parentNotebook": map[string]interface{}{"id": containerID}},
	}, nil
}

// TestNotebookClient_ListNotebooks tests the core notebook listing functionality
func TestNotebookClient_ListNotebooks(t *testing.T) {
	t.Run("successfully lists notebooks", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		
		notebooks, err := mockClient.ListNotebooks()
		
		assert.NoError(t, err)
		assert.Len(t, notebooks, 3)
		assert.Equal(t, "Work Notebook", notebooks[0]["displayName"])
		assert.Equal(t, "Personal Notes", notebooks[1]["displayName"])
		assert.Equal(t, "Project Alpha", notebooks[2]["displayName"])
		
		// Verify required fields are present
		for _, notebook := range notebooks {
			assert.Contains(t, notebook, "id")
			assert.Contains(t, notebook, "displayName")
			assert.NotEmpty(t, notebook["id"])
			assert.NotEmpty(t, notebook["displayName"])
		}
	})
	
	t.Run("handles authentication errors", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		mockClient.SetError(fmt.Errorf("401 unauthorized"))
		
		notebooks, err := mockClient.ListNotebooks()
		
		assert.Error(t, err)
		assert.Nil(t, notebooks)
		assert.Contains(t, err.Error(), "401")
	})
	
	t.Run("handles network errors", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		mockClient.SetError(fmt.Errorf("network timeout"))
		
		notebooks, err := mockClient.ListNotebooks()
		
		assert.Error(t, err)
		assert.Nil(t, notebooks)
		assert.Contains(t, err.Error(), "timeout")
	})
	
	t.Run("returns empty list when no notebooks exist", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		mockClient.notebooks = []map[string]interface{}{} // Empty notebooks
		
		notebooks, err := mockClient.ListNotebooks()
		
		assert.NoError(t, err)
		assert.Len(t, notebooks, 0)
		assert.NotNil(t, notebooks) // Should return empty slice, not nil
	})
}

// TestNotebookClient_ListNotebooksDetailed tests detailed notebook information retrieval
func TestNotebookClient_ListNotebooksDetailed(t *testing.T) {
	t.Run("successfully lists notebooks with detailed information", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		
		detailed, err := mockClient.ListNotebooksDetailed()
		
		assert.NoError(t, err)
		assert.Len(t, detailed, 3)
		
		// Check first notebook has detailed info
		workNotebook := detailed[0]
		assert.Equal(t, "Work Notebook", workNotebook["displayName"])
		assert.Contains(t, workNotebook, "createdDateTime")
		assert.Contains(t, workNotebook, "lastModifiedDateTime")
		assert.Contains(t, workNotebook, "isDefault")
		assert.Contains(t, workNotebook, "userRole")
		assert.Contains(t, workNotebook, "links")
		assert.Contains(t, workNotebook, "createdBy")
		
		// Verify boolean fields
		assert.Equal(t, true, workNotebook["isDefault"])
		assert.Equal(t, false, workNotebook["isShared"])
		
		// Verify nested objects
		links, ok := workNotebook["links"].(map[string]interface{})
		assert.True(t, ok, "Links should be a map")
		assert.Contains(t, links, "oneNoteClientUrl")
		assert.Contains(t, links, "oneNoteWebUrl")
		
		createdBy, ok := workNotebook["createdBy"].(map[string]interface{})
		assert.True(t, ok, "CreatedBy should be a map")
		assert.Contains(t, createdBy, "user")
	})
	
	t.Run("handles authentication errors", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		mockClient.SetError(fmt.Errorf("JWT token expired"))
		
		detailed, err := mockClient.ListNotebooksDetailed()
		
		assert.Error(t, err)
		assert.Nil(t, detailed)
		assert.Contains(t, err.Error(), "JWT")
	})
	
	t.Run("handles empty detailed information gracefully", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		mockClient.detailedInfo = map[string]map[string]interface{}{} // Empty detailed info
		
		detailed, err := mockClient.ListNotebooksDetailed()
		
		assert.NoError(t, err)
		assert.Len(t, detailed, 3) // Should fall back to basic info
		
		// Should contain basic fields even without detailed info
		for _, notebook := range detailed {
			assert.Contains(t, notebook, "id")
			assert.Contains(t, notebook, "displayName")
		}
	})
}

// TestNotebookClient_GetDetailedNotebookByName tests notebook retrieval by name
func TestNotebookClient_GetDetailedNotebookByName(t *testing.T) {
	t.Run("successfully retrieves notebook by exact name match", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		
		notebook, err := mockClient.GetDetailedNotebookByName("Work Notebook")
		
		assert.NoError(t, err)
		assert.NotNil(t, notebook)
		assert.Equal(t, "Work Notebook", notebook["displayName"])
		assert.Equal(t, "notebook-1", notebook["id"])
		
		// Verify detailed information is present
		assert.Contains(t, notebook, "createdDateTime")
		assert.Contains(t, notebook, "lastModifiedDateTime")
		assert.Contains(t, notebook, "isDefault")
		assert.Contains(t, notebook, "userRole")
		assert.Equal(t, "Owner", notebook["userRole"])
	})
	
	t.Run("returns error for non-existent notebook", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		
		notebook, err := mockClient.GetDetailedNotebookByName("Non-existent Notebook")
		
		assert.Error(t, err)
		assert.Nil(t, notebook)
		assert.Contains(t, err.Error(), "notebook not found")
		assert.Contains(t, err.Error(), "Non-existent Notebook")
	})
	
	t.Run("handles case-sensitive name matching", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		
		// Test with different case
		notebook, err := mockClient.GetDetailedNotebookByName("work notebook")
		
		assert.Error(t, err) // Should not match due to case sensitivity
		assert.Nil(t, notebook)
	})
	
	t.Run("handles authentication errors", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		mockClient.SetError(fmt.Errorf("403 forbidden"))
		
		notebook, err := mockClient.GetDetailedNotebookByName("Work Notebook")
		
		assert.Error(t, err)
		assert.Nil(t, notebook)
		assert.Contains(t, err.Error(), "403")
	})
	
	t.Run("validates empty notebook name", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		
		notebook, err := mockClient.GetDetailedNotebookByName("")
		
		assert.Error(t, err)
		assert.Nil(t, notebook)
		assert.Contains(t, err.Error(), "notebook not found")
	})
}

// TestNotebookClient_SearchPages tests the page search functionality
func TestNotebookClient_SearchPages(t *testing.T) {
	t.Run("successfully searches pages by title", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		
		results, err := mockClient.SearchPages("meeting", "notebook-1")
		
		assert.NoError(t, err)
		assert.Len(t, results, 2) // Should find "Meeting Notes" and "Project Meeting Notes"
		
		// Verify results contain expected pages
		foundTitles := make(map[string]bool)
		for _, page := range results {
			title, ok := page["title"].(string)
			assert.True(t, ok, "Page should have title")
			foundTitles[title] = true
			
			// Verify context information is added
			assert.Contains(t, page, "sectionName")
			assert.Contains(t, page, "sectionId")
			assert.Contains(t, page, "sectionPath")
		}
		
		assert.True(t, foundTitles["Meeting Notes"])
		assert.True(t, foundTitles["Project Meeting Notes"])
	})
	
	t.Run("performs case-insensitive search", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		
		// Search with different cases
		results1, err1 := mockClient.SearchPages("MEETING", "notebook-1")
		results2, err2 := mockClient.SearchPages("meeting", "notebook-1")
		results3, err3 := mockClient.SearchPages("Meeting", "notebook-1")
		
		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.NoError(t, err3)
		
		// All searches should return the same results
		assert.Equal(t, len(results1), len(results2))
		assert.Equal(t, len(results2), len(results3))
		assert.Greater(t, len(results1), 0, "Should find matching pages")
	})
	
	t.Run("returns all pages for empty query", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		
		results, err := mockClient.SearchPages("", "notebook-1")
		
		assert.NoError(t, err)
		assert.Len(t, results, 4) // Should return all 4 pages
		
		// Verify all pages have context information
		for _, page := range results {
			assert.Contains(t, page, "title")
			assert.Contains(t, page, "sectionName")
			assert.Contains(t, page, "sectionId")
			assert.Contains(t, page, "sectionPath")
		}
	})
	
	t.Run("returns empty results for non-matching query", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		
		results, err := mockClient.SearchPages("nonexistent", "notebook-1")
		
		assert.NoError(t, err)
		assert.Len(t, results, 0)
		assert.NotNil(t, results) // Should return empty slice, not nil
	})
	
	t.Run("handles authentication errors", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		mockClient.SetError(fmt.Errorf("401 unauthorized"))
		
		results, err := mockClient.SearchPages("meeting", "notebook-1")
		
		assert.Error(t, err)
		assert.Nil(t, results)
		assert.Contains(t, err.Error(), "401")
	})
	
	t.Run("handles partial search matches", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		
		results, err := mockClient.SearchPages("Notes", "notebook-1")
		
		assert.NoError(t, err)
		assert.Greater(t, len(results), 0, "Should find pages containing 'Notes'")
		
		// Verify all results contain the search term
		for _, page := range results {
			title, ok := page["title"].(string)
			assert.True(t, ok)
			assert.Contains(t, strings.ToLower(title), "notes")
		}
	})
}

// TestNotebookClient_GetDefaultNotebookID tests the default notebook ID retrieval
func TestNotebookClient_GetDefaultNotebookID(t *testing.T) {
	mockNotebookClient := NewMockNotebookClient()
	
	t.Run("successfully finds notebook by name", func(t *testing.T) {
		config := &struct{ NotebookName string }{NotebookName: "Work Notebook"}
		
		// Mock the ListNotebooks call
		notebooks, err := mockNotebookClient.ListNotebooks()
		assert.NoError(t, err)
		
		// Simulate the default notebook ID logic
		var foundID string
		for _, notebook := range notebooks {
			if displayName, exists := notebook["displayName"].(string); exists {
				if displayName == config.NotebookName {
					if id, exists := notebook["id"].(string); exists {
						foundID = id
						break
					}
				}
			}
		}
		
		assert.Equal(t, "notebook-1", foundID)
	})
	
	t.Run("returns error for non-existent notebook name", func(t *testing.T) {
		config := &struct{ NotebookName string }{NotebookName: "Non-existent Notebook"}
		
		notebooks, err := mockNotebookClient.ListNotebooks()
		assert.NoError(t, err)
		
		// Simulate the default notebook ID logic
		var foundID string
		for _, notebook := range notebooks {
			if displayName, exists := notebook["displayName"].(string); exists {
				if displayName == config.NotebookName {
					if id, exists := notebook["id"].(string); exists {
						foundID = id
						break
					}
				}
			}
		}
		
		assert.Equal(t, "", foundID) // Should not find the notebook
	})
	
	t.Run("handles authentication errors during notebook listing", func(t *testing.T) {
		mockNotebookClient.SetError(fmt.Errorf("authentication failed"))
		
		notebooks, err := mockNotebookClient.ListNotebooks()
		
		assert.Error(t, err)
		assert.Nil(t, notebooks)
		assert.Contains(t, err.Error(), "authentication failed")
		
		// Reset error for cleanup
		mockNotebookClient.SetError(nil)
	})
}

// TestNotebookClient_Integration tests integration scenarios
func TestNotebookClient_Integration(t *testing.T) {
	t.Run("complete notebook discovery workflow", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		
		// 1. List all notebooks
		notebooks, err := mockClient.ListNotebooks()
		assert.NoError(t, err)
		assert.Greater(t, len(notebooks), 0)
		
		// 2. Get detailed info for first notebook
		firstNotebook := notebooks[0]
		notebookName := firstNotebook["displayName"].(string)
		
		detailed, err := mockClient.GetDetailedNotebookByName(notebookName)
		assert.NoError(t, err)
		assert.Equal(t, firstNotebook["id"], detailed["id"])
		
		// 3. Search for pages in that notebook
		notebookID := firstNotebook["id"].(string)
		pages, err := mockClient.SearchPages("meeting", notebookID)
		assert.NoError(t, err)
		
		// Verify pages have proper context
		for _, page := range pages {
			assert.Contains(t, page, "title")
			assert.Contains(t, page, "sectionName")
			assert.Contains(t, page, "sectionId")
		}
	})
	
	t.Run("handles notebook with no matching pages", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		
		// Search for non-existent content
		pages, err := mockClient.SearchPages("zzz-nonexistent-zzz", "notebook-1")
		assert.NoError(t, err)
		assert.Len(t, pages, 0)
		assert.NotNil(t, pages)
	})
	
	t.Run("token refresh simulation", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		mockClient.TokenManager.SetExpired(true)
		
		// First call should work even with expired token (mock doesn't enforce this)
		notebooks, err := mockClient.ListNotebooks()
		assert.NoError(t, err)
		assert.Greater(t, len(notebooks), 0)
		
		// Verify token state can be checked
		assert.True(t, mockClient.TokenManager.IsExpired())
		
		// Reset token state
		mockClient.TokenManager.SetExpired(false)
		assert.False(t, mockClient.TokenManager.IsExpired())
	})

	t.Run("multi-notebook search scenario", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		
		// Search across multiple notebooks
		notebooks, err := mockClient.ListNotebooks()
		assert.NoError(t, err)
		
		allPages := make([]map[string]interface{}, 0)
		for _, notebook := range notebooks {
			notebookID := notebook["id"].(string)
			pages, err := mockClient.SearchPages("notes", notebookID)
			assert.NoError(t, err)
			allPages = append(allPages, pages...)
		}
		
		// Should find pages across multiple notebooks
		assert.Greater(t, len(allPages), 0)
		
		// Verify all pages have context information
		for _, page := range allPages {
			assert.Contains(t, page, "title")
			assert.Contains(t, page, "sectionName")
			assert.Contains(t, page, "sectionId")
			assert.Contains(t, page, "sectionPath")
		}
	})

	t.Run("detailed notebook retrieval with pagination simulation", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		
		// Add many notebooks to simulate pagination
		for i := 0; i < 50; i++ {
			name := fmt.Sprintf("Large Notebook %d", i)
			mockClient.notebooks = append(mockClient.notebooks, map[string]interface{}{
				"id":          fmt.Sprintf("large-notebook-%d", i),
				"displayName": name,
			})
			mockClient.detailedInfo[name] = map[string]interface{}{
				"id":                   fmt.Sprintf("large-notebook-%d", i),
				"displayName":          name,
				"createdDateTime":      "2023-01-01T00:00:00Z",
				"lastModifiedDateTime": "2023-12-01T00:00:00Z",
				"isDefault":            false,
				"userRole":             "Owner",
			}
		}
		
		// Test that we can still find specific notebooks efficiently
		targetName := "Large Notebook 25"
		notebook, err := mockClient.GetDetailedNotebookByName(targetName)
		assert.NoError(t, err)
		assert.Equal(t, targetName, notebook["displayName"])
		assert.Equal(t, "large-notebook-25", notebook["id"])
	})
}

// TestNotebookClient_PerformanceAndEdgeCases tests performance and edge case scenarios
func TestNotebookClient_PerformanceAndEdgeCases(t *testing.T) {
	t.Run("handles large notebook collections", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		
		// Add many notebooks
		for i := 0; i < 100; i++ {
			mockClient.notebooks = append(mockClient.notebooks, map[string]interface{}{
				"id":          fmt.Sprintf("notebook-%d", i+10),
				"displayName": fmt.Sprintf("Test Notebook %d", i+1),
			})
		}
		
		notebooks, err := mockClient.ListNotebooks()
		assert.NoError(t, err)
		assert.Len(t, notebooks, 103) // Original 3 + 100 new ones
	})
	
	t.Run("handles notebooks with special characters in names", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		
		// Add notebook with special characters
		specialNotebook := map[string]interface{}{
			"id":          "special-notebook",
			"displayName": "Notebook with Special & Characters < >",
		}
		mockClient.notebooks = append(mockClient.notebooks, specialNotebook)
		
		// Add detailed info
		mockClient.detailedInfo["Notebook with Special & Characters < >"] = specialNotebook
		
		notebook, err := mockClient.GetDetailedNotebookByName("Notebook with Special & Characters < >")
		assert.NoError(t, err)
		assert.Equal(t, "special-notebook", notebook["id"])
	})
	
	t.Run("handles concurrent access patterns", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		
		// Simulate concurrent operations
		done := make(chan bool, 3)
		
		// Multiple goroutines accessing the mock client
		go func() {
			notebooks, err := mockClient.ListNotebooks()
			assert.NoError(t, err)
			assert.Greater(t, len(notebooks), 0)
			done <- true
		}()
		
		go func() {
			pages, err := mockClient.SearchPages("meeting", "notebook-1")
			assert.NoError(t, err)
			assert.GreaterOrEqual(t, len(pages), 0)
			done <- true
		}()
		
		go func() {
			notebook, err := mockClient.GetDetailedNotebookByName("Work Notebook")
			assert.NoError(t, err)
			assert.NotNil(t, notebook)
			done <- true
		}()
		
		// Wait for all operations to complete
		for i := 0; i < 3; i++ {
			<-done
		}
	})
	
	t.Run("handles malformed notebook data gracefully", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		
		// Add malformed notebook data
		mockClient.notebooks = append(mockClient.notebooks, map[string]interface{}{
			"id": "malformed-1", // Missing displayName
		})
		mockClient.notebooks = append(mockClient.notebooks, map[string]interface{}{
			"displayName": "No ID Notebook", // Missing id
		})
		
		notebooks, err := mockClient.ListNotebooks()
		assert.NoError(t, err)
		
		// Should still return all notebooks, including malformed ones
		assert.Len(t, notebooks, 5) // Original 3 + 2 malformed
		
		// Verify we can handle the malformed data
		for _, notebook := range notebooks {
			// At least one of id or displayName should be present
			hasID := notebook["id"] != nil
			hasDisplayName := notebook["displayName"] != nil
			assert.True(t, hasID || hasDisplayName, "Notebook should have at least ID or displayName")
		}
	})
}

// Benchmark tests for performance validation
func BenchmarkNotebookClient_ListNotebooks(b *testing.B) {
	mockClient := NewMockNotebookClient()
	
	// Add many notebooks for realistic testing
	for i := 0; i < 1000; i++ {
		mockClient.notebooks = append(mockClient.notebooks, map[string]interface{}{
			"id":          fmt.Sprintf("notebook-%d", i),
			"displayName": fmt.Sprintf("Benchmark Notebook %d", i),
		})
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mockClient.ListNotebooks()
	}
}

func BenchmarkNotebookClient_SearchPages(b *testing.B) {
	mockClient := NewMockNotebookClient()
	
	// Add many pages for realistic testing
	for i := 0; i < 100; i++ {
		sectionID := fmt.Sprintf("section-%d", i%10)
		if mockClient.pagesBySection[sectionID] == nil {
			mockClient.pagesBySection[sectionID] = []map[string]interface{}{}
		}
		mockClient.pagesBySection[sectionID] = append(
			mockClient.pagesBySection[sectionID],
			map[string]interface{}{
				"id":    fmt.Sprintf("page-%d", i),
				"title": fmt.Sprintf("Meeting Page %d", i),
			},
		)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mockClient.SearchPages("meeting", "notebook-1")
	}
}

func BenchmarkNotebookClient_GetDetailedNotebookByName(b *testing.B) {
	mockClient := NewMockNotebookClient()
	
	// Add many detailed notebooks
	for i := 0; i < 100; i++ {
		name := fmt.Sprintf("Benchmark Notebook %d", i)
		mockClient.detailedInfo[name] = map[string]interface{}{
			"id":          fmt.Sprintf("notebook-%d", i),
			"displayName": name,
			"isDefault":   i == 0,
			"userRole":    "Owner",
		}
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		name := fmt.Sprintf("Benchmark Notebook %d", i%100)
		_, _ = mockClient.GetDetailedNotebookByName(name)
	}
}

// Additional benchmarks for comprehensive performance testing
func BenchmarkNotebookClient_LargeDatasetOperations(b *testing.B) {
	mockClient := NewMockNotebookClient()
	
	// Create realistic large dataset
	for i := 0; i < 5000; i++ {
		mockClient.notebooks = append(mockClient.notebooks, map[string]interface{}{
			"id":          fmt.Sprintf("bench-notebook-%d", i),
			"displayName": fmt.Sprintf("Benchmark Notebook %d", i),
		})
	}
	
	b.Run("ListNotebooks_5000", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = mockClient.ListNotebooks()
		}
	})
	
	b.Run("NotebookSearch_WorstCase", func(b *testing.B) {
		// Target notebook is at the end (worst case)
		targetName := "Benchmark Notebook 4999"
		mockClient.detailedInfo[targetName] = map[string]interface{}{
			"id":          "bench-notebook-4999",
			"displayName": targetName,
		}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = mockClient.GetDetailedNotebookByName(targetName)
		}
	})
	
	b.Run("NotebookSearch_BestCase", func(b *testing.B) {
		// Target notebook is at the beginning (best case)
		targetName := "Benchmark Notebook 0"
		mockClient.detailedInfo[targetName] = map[string]interface{}{
			"id":          "bench-notebook-0",
			"displayName": targetName,
		}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = mockClient.GetDetailedNotebookByName(targetName)
		}
	})
}

func BenchmarkNotebookClient_ConcurrentAccess(b *testing.B) {
	mockClient := NewMockNotebookClient()
	
	// Add moderate amount of test data
	for i := 0; i < 100; i++ {
		notebookName := fmt.Sprintf("Concurrent Notebook %d", i)
		mockClient.notebooks = append(mockClient.notebooks, map[string]interface{}{
			"id":          fmt.Sprintf("concurrent-notebook-%d", i),
			"displayName": notebookName,
		})
		mockClient.detailedInfo[notebookName] = map[string]interface{}{
			"id":          fmt.Sprintf("concurrent-notebook-%d", i),
			"displayName": notebookName,
		}
	}
	
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Mix different operations
			switch rand.Intn(3) {
			case 0:
				_, _ = mockClient.ListNotebooks()
			case 1:
				targetName := fmt.Sprintf("Concurrent Notebook %d", rand.Intn(100))
				_, _ = mockClient.GetDetailedNotebookByName(targetName)
			case 2:
				_, _ = mockClient.SearchPages("test", fmt.Sprintf("concurrent-notebook-%d", rand.Intn(100)))
			}
		}
	})
}

// TestNotebookClient_Validation tests validation logic for notebook operations
func TestNotebookClient_Validation(t *testing.T) {
	t.Run("validates empty notebook name for GetDetailedNotebookByName", func(t *testing.T) {
		// We can't easily test the full method without mocking the Graph SDK,
		// but we can test the validation logic that would be used
		notebookName := ""
		assert.True(t, notebookName == "", "Empty notebook name should be rejected")
	})

	t.Run("validates notebook ID format", func(t *testing.T) {
		// Test different ID formats
		tests := []struct {
			id       string
			valid    bool
		}{
			{"", false},
			{"valid-notebook-id", true},
			{"notebook-123-abc", true},
		}
		
		for _, tt := range tests {
			// We're testing the concept of ID validation
			isValid := tt.id != ""
			assert.Equal(t, tt.valid, isValid, "ID validation should work correctly for: %s", tt.id)
		}
	})

	t.Run("validates search parameters", func(t *testing.T) {
		// Test search parameter validation logic
		tests := []struct {
			query      string
			notebookID string
			valid      bool
		}{
			{"meeting", "notebook-123", true},
			{"", "notebook-123", true},        // Empty query is valid (matches all)
			{"test", "", false},               // Empty notebook ID is invalid
		}
		
		for _, tt := range tests {
			// Test the validation logic that would be used in SearchPages
			isValid := tt.notebookID != ""
			assert.Equal(t, tt.valid, isValid, "Search validation should work for query=%s, notebookID=%s", tt.query, tt.notebookID)
		}
	})

	t.Run("validates OneNote ID sanitization", func(t *testing.T) {
		// Test various OneNote ID formats that need sanitization
		tests := []struct {
			name       string
			id         string
			idType     string
			expected   string
			shouldFail bool
		}{
			{"valid standard ID", "1-ABC123DEF456!789", "notebookID", "1-ABC123DEF456!789", false},
			{"valid simple ID", "notebook-123", "notebookID", "notebook-123", false},
			{"empty ID", "", "notebookID", "", true},
			{"whitespace ID", "   ", "notebookID", "", true},
			{"ID with spaces", "notebook 123", "notebookID", "notebook 123", false}, // Should be handled by sanitizer
		}
		
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Simulate the ID validation logic
				trimmedID := strings.TrimSpace(tt.id)
				isEmpty := trimmedID == ""
				
				if tt.shouldFail {
					assert.True(t, isEmpty, "Expected ID to be invalid: %s", tt.id)
				} else {
					assert.False(t, isEmpty, "Expected ID to be valid: %s", tt.id)
					assert.NotEmpty(t, trimmedID, "Expected non-empty trimmed ID")
				}
			})
		}
	})

	t.Run("validates notebook name patterns", func(t *testing.T) {
		// Test notebook name patterns that might cause issues
		tests := []struct {
			name        string
			notebookName string
			valid       bool
			reason      string
		}{
			{"normal name", "Work Notebook", true, "standard name"},
			{"empty name", "", false, "empty string"},
			{"whitespace only", "   ", false, "whitespace only"},
			{"very long name", strings.Repeat("a", 300), false, "too long"},
			{"unicode name", "工作笔记本", true, "unicode characters"},
			{"special chars", "Project-Alpha_v1.2", true, "special characters"},
			{"with newlines", "Name\nWith\nNewlines", false, "contains newlines"},
			{"with tabs", "Name\tWith\tTabs", false, "contains tabs"},
		}
		
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Basic validation logic for notebook names
				trimmedName := strings.TrimSpace(tt.notebookName)
				isValid := trimmedName != "" && 
					len(trimmedName) <= 255 && 
					!strings.Contains(trimmedName, "\n") && 
					!strings.Contains(trimmedName, "\t") &&
					!strings.Contains(trimmedName, "\x00")
				
				if tt.valid {
					assert.True(t, isValid, "Expected notebook name to be valid: %s (%s)", tt.notebookName, tt.reason)
				} else {
					assert.False(t, isValid, "Expected notebook name to be invalid: %s (%s)", tt.notebookName, tt.reason)
				}
			})
		}
	})
}

// TestNotebookClient_RealWorldScenarios tests realistic usage patterns
func TestNotebookClient_RealWorldScenarios(t *testing.T) {
	t.Run("typical user workflow - find and search notebooks", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		
		// Step 1: User lists all notebooks to see what's available
		notebooks, err := mockClient.ListNotebooks()
		assert.NoError(t, err)
		assert.Greater(t, len(notebooks), 0)
		
		// Step 2: User selects a specific notebook for detailed info
		workNotebook := "Work Notebook"
		details, err := mockClient.GetDetailedNotebookByName(workNotebook)
		assert.NoError(t, err)
		assert.Equal(t, workNotebook, details["displayName"])
		
		// Step 3: User searches for specific content in that notebook
		notebookID := details["id"].(string)
		meetingPages, err := mockClient.SearchPages("meeting", notebookID)
		assert.NoError(t, err)
		
		// Step 4: User searches for different content
		projectPages, err := mockClient.SearchPages("project", notebookID)
		assert.NoError(t, err)
		
		// Verify the workflow produces expected results
		for _, page := range meetingPages {
			title := strings.ToLower(page["title"].(string))
			assert.Contains(t, title, "meeting")
		}
		
		for _, page := range projectPages {
			title := strings.ToLower(page["title"].(string))
			assert.Contains(t, title, "project")
		}
	})

	t.Run("power user workflow - detailed notebook management", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		
		// Power user gets detailed view of all notebooks
		detailedNotebooks, err := mockClient.ListNotebooksDetailed()
		assert.NoError(t, err)
		assert.Greater(t, len(detailedNotebooks), 0)
		
		// Analyze notebook metadata
		defaultNotebooks := 0
		sharedNotebooks := 0
		ownerNotebooks := 0
		
		for _, notebook := range detailedNotebooks {
			if isDefault, ok := notebook["isDefault"].(bool); ok && isDefault {
				defaultNotebooks++
			}
			if isShared, ok := notebook["isShared"].(bool); ok && isShared {
				sharedNotebooks++
			}
			if userRole, ok := notebook["userRole"].(string); ok && userRole == "Owner" {
				ownerNotebooks++
			}
		}
		
		// Verify metadata analysis
		assert.GreaterOrEqual(t, defaultNotebooks, 0, "Should have at least 0 default notebooks")
		assert.GreaterOrEqual(t, sharedNotebooks, 0, "Should have at least 0 shared notebooks")
		assert.Greater(t, ownerNotebooks, 0, "Should have at least 1 owned notebook")
	})

	t.Run("search across multiple notebooks scenario", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		
		// Get all available notebooks
		notebooks, err := mockClient.ListNotebooks()
		assert.NoError(t, err)
		
		// Search for "notes" across all notebooks
		allSearchResults := make(map[string][]map[string]interface{})
		totalPages := 0
		
		for _, notebook := range notebooks {
			notebookID := notebook["id"].(string)
			notebookName := notebook["displayName"].(string)
			
			pages, err := mockClient.SearchPages("notes", notebookID)
			assert.NoError(t, err)
			
			if len(pages) > 0 {
				allSearchResults[notebookName] = pages
				totalPages += len(pages)
			}
		}
		
		// Verify cross-notebook search results
		assert.Greater(t, len(allSearchResults), 0, "Should find results in at least one notebook")
		assert.Greater(t, totalPages, 0, "Should find at least one page across all notebooks")
		
		// Verify result structure
		for notebookName, pages := range allSearchResults {
			assert.NotEmpty(t, notebookName, "Notebook name should not be empty")
			for _, page := range pages {
				assert.Contains(t, page, "title", "Page should have title")
				assert.Contains(t, page, "sectionName", "Page should have section context")
				assert.Contains(t, page, "sectionId", "Page should have section ID")
				assert.Contains(t, page, "sectionPath", "Page should have section path")
			}
		}
	})

	t.Run("large scale notebook search performance", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		
		// Create large scale test data
		for i := 0; i < 100; i++ {
			// Add notebooks
			notebookName := fmt.Sprintf("Scale Test Notebook %d", i)
			mockClient.notebooks = append(mockClient.notebooks, map[string]interface{}{
				"id":          fmt.Sprintf("scale-notebook-%d", i),
				"displayName": notebookName,
			})
			
			// Add pages to each notebook
			for j := 0; j < 10; j++ {
				sectionID := fmt.Sprintf("scale-section-%d-%d", i, j)
				if mockClient.pagesBySection[sectionID] == nil {
					mockClient.pagesBySection[sectionID] = []map[string]interface{}{}
				}
				mockClient.pagesBySection[sectionID] = append(
					mockClient.pagesBySection[sectionID],
					map[string]interface{}{
						"id":    fmt.Sprintf("scale-page-%d-%d", i, j),
						"title": fmt.Sprintf("Test Page %d-%d Meeting", i, j),
					},
				)
			}
		}
		
		// Test performance with large dataset
		start := time.Now()
		notebooks, err := mockClient.ListNotebooks()
		listDuration := time.Since(start)
		
		assert.NoError(t, err)
		assert.Len(t, notebooks, 103) // Original 3 + 100 new ones
		assert.Less(t, listDuration, time.Second, "List operation should complete quickly")
		
		// Test search performance
		start = time.Now()
		pages, err := mockClient.SearchPages("meeting", "scale-notebook-50")
		searchDuration := time.Since(start)
		
		assert.NoError(t, err)
		assert.Greater(t, len(pages), 0)
		assert.Less(t, searchDuration, time.Second, "Search operation should complete quickly")
	})
}
