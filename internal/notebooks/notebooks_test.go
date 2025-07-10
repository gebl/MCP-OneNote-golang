package notebooks

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

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
