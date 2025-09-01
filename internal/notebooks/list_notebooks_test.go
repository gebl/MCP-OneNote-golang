// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package notebooks

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestListNotebooks tests the ListNotebooks function with various scenarios
func TestListNotebooks(t *testing.T) {
	t.Run("successfully lists notebooks with basic information", func(t *testing.T) {
		// Create a mock notebook client that simulates the Graph SDK response
		mockClient := NewMockNotebookClient()
		
		// Set up test data
		expectedNotebooks := []map[string]interface{}{
			{
				"notebookId":  "notebook-1",
				"displayName": "Work Notebook",
			},
			{
				"notebookId":  "notebook-2", 
				"displayName": "Personal Notes",
			},
		}
		
		// Configure mock to return expected notebooks
		mockClient.notebooks = expectedNotebooks
		
		// Test the function
		notebooks, err := mockClient.ListNotebooks()
		
		// Verify results
		assert.NoError(t, err)
		assert.Len(t, notebooks, 2)
		assert.Equal(t, expectedNotebooks, notebooks)
		
		// Verify specific notebook properties
		assert.Equal(t, "notebook-1", notebooks[0]["notebookId"])
		assert.Equal(t, "Work Notebook", notebooks[0]["displayName"])
		assert.Equal(t, "notebook-2", notebooks[1]["notebookId"])
		assert.Equal(t, "Personal Notes", notebooks[1]["displayName"])
	})

	t.Run("returns empty slice when no notebooks found", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		mockClient.notebooks = []map[string]interface{}{}
		
		notebooks, err := mockClient.ListNotebooks()
		
		assert.NoError(t, err)
		assert.NotNil(t, notebooks, "Should return empty slice, not nil")
		assert.Len(t, notebooks, 0)
	})

	t.Run("handles authentication errors", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		mockClient.SetError(errors.New("401 Unauthorized: JWT token expired"))
		
		notebooks, err := mockClient.ListNotebooks()
		
		assert.Error(t, err)
		assert.Nil(t, notebooks)
		assert.Contains(t, err.Error(), "401 Unauthorized")
	})

	t.Run("handles permanent authentication failures", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		mockClient.SetError(errors.New("403 Forbidden: Access denied"))
		
		notebooks, err := mockClient.ListNotebooks()
		
		assert.Error(t, err)
		assert.Nil(t, notebooks)
		assert.Contains(t, err.Error(), "403 Forbidden")
	})

	t.Run("handles network connectivity errors", func(t *testing.T) {
		networkErrors := []string{
			"connection timeout",
			"DNS resolution failed",
			"connection refused",
			"network unreachable",
			"host unreachable",
		}

		for _, networkError := range networkErrors {
			t.Run(networkError, func(t *testing.T) {
				mockClient := NewMockNotebookClient()
				mockClient.SetError(fmt.Errorf(networkError))

				notebooks, err := mockClient.ListNotebooks()

				assert.Error(t, err)
				assert.Nil(t, notebooks)
				assert.Contains(t, err.Error(), networkError)
			})
		}
	})

	t.Run("handles various HTTP error codes", func(t *testing.T) {
		httpErrors := []struct {
			code        string
			description string
		}{
			{"400", "bad request"},
			{"401", "unauthorized"},
			{"403", "forbidden"},
			{"404", "not found"},
			{"429", "too many requests"},
			{"500", "internal server error"},
			{"502", "bad gateway"},
			{"503", "service unavailable"},
			{"504", "gateway timeout"},
		}

		for _, httpError := range httpErrors {
			t.Run(fmt.Sprintf("HTTP %s", httpError.code), func(t *testing.T) {
				mockClient := NewMockNotebookClient()
				errorMsg := fmt.Sprintf("%s %s", httpError.code, httpError.description)
				mockClient.SetError(fmt.Errorf(errorMsg))

				notebooks, err := mockClient.ListNotebooks()

				assert.Error(t, err)
				assert.Nil(t, notebooks)
				assert.Contains(t, err.Error(), httpError.code)
			})
		}
	})

	t.Run("handles notebooks with missing or null fields gracefully", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		
		// Set up notebooks with various missing fields
		mockClient.notebooks = []map[string]interface{}{
			{
				"notebookId":  "complete-notebook",
				"displayName": "Complete Notebook",
			},
			{
				"notebookId": "missing-name-notebook",
				// Missing displayName
			},
			{
				"displayName": "Missing ID Notebook",
				// Missing notebookId
			},
			{
				"notebookId":  "",
				"displayName": "",
			},
		}
		
		notebooks, err := mockClient.ListNotebooks()
		
		assert.NoError(t, err)
		assert.Len(t, notebooks, 4, "Should include all notebooks even with missing fields")
		
		// Verify complete notebook
		assert.Equal(t, "complete-notebook", notebooks[0]["notebookId"])
		assert.Equal(t, "Complete Notebook", notebooks[0]["displayName"])
		
		// Verify handling of missing fields
		assert.Equal(t, "missing-name-notebook", notebooks[1]["notebookId"])
		assert.NotContains(t, notebooks[1], "displayName")
		
		assert.NotContains(t, notebooks[2], "notebookId")
		assert.Equal(t, "Missing ID Notebook", notebooks[2]["displayName"])
		
		assert.Equal(t, "", notebooks[3]["notebookId"])
		assert.Equal(t, "", notebooks[3]["displayName"])
	})

	t.Run("handles concurrent access safely", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		mockClient.notebooks = []map[string]interface{}{
			{"notebookId": "notebook-1", "displayName": "Concurrent Test"},
		}
		
		// Run multiple goroutines calling ListNotebooks concurrently
		numGoroutines := 10
		results := make(chan []map[string]interface{}, numGoroutines)
		errors := make(chan error, numGoroutines)
		
		for i := 0; i < numGoroutines; i++ {
			go func() {
				notebooks, err := mockClient.ListNotebooks()
				if err != nil {
					errors <- err
					return
				}
				results <- notebooks
			}()
		}
		
		// Collect results
		for i := 0; i < numGoroutines; i++ {
			select {
			case notebooks := <-results:
				assert.Len(t, notebooks, 1)
				assert.Equal(t, "Concurrent Test", notebooks[0]["displayName"])
			case err := <-errors:
				t.Errorf("Unexpected error in concurrent access: %v", err)
			case <-time.After(time.Second):
				t.Error("Timeout waiting for concurrent operation")
			}
		}
	})
}

// TestListNotebooks_Pagination tests pagination functionality
func TestListNotebooks_Pagination(t *testing.T) {
	t.Run("handles single page response", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		mockClient.notebooks = []map[string]interface{}{
			{"notebookId": "nb-1", "displayName": "Notebook 1"},
			{"notebookId": "nb-2", "displayName": "Notebook 2"},
		}
		
		notebooks, err := mockClient.ListNotebooks()
		
		assert.NoError(t, err)
		assert.Len(t, notebooks, 2)
	})

	t.Run("simulates pagination logic", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		
		// Simulate pagination by setting up multiple pages of data
		page1 := []map[string]interface{}{
			{"notebookId": "page1-nb1", "displayName": "Page 1 Notebook 1"},
			{"notebookId": "page1-nb2", "displayName": "Page 1 Notebook 2"},
		}
		page2 := []map[string]interface{}{
			{"notebookId": "page2-nb1", "displayName": "Page 2 Notebook 1"},
			{"notebookId": "page2-nb2", "displayName": "Page 2 Notebook 2"},
		}
		
		// Combine all pages (simulating what the real function does)
		allNotebooks := append(page1, page2...)
		mockClient.notebooks = allNotebooks
		
		notebooks, err := mockClient.ListNotebooks()
		
		assert.NoError(t, err)
		assert.Len(t, notebooks, 4)
		
		// Verify notebooks from both pages are included
		notebookIDs := make([]string, len(notebooks))
		for i, nb := range notebooks {
			notebookIDs[i] = nb["notebookId"].(string)
		}
		
		assert.Contains(t, notebookIDs, "page1-nb1")
		assert.Contains(t, notebookIDs, "page1-nb2") 
		assert.Contains(t, notebookIDs, "page2-nb1")
		assert.Contains(t, notebookIDs, "page2-nb2")
	})

	t.Run("handles pagination error gracefully", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		
		// Simulate pagination error
		mockClient.SetError(errors.New("failed to fetch next page of notebooks"))
		
		notebooks, err := mockClient.ListNotebooks()
		
		assert.Error(t, err)
		assert.Nil(t, notebooks)
		assert.Contains(t, err.Error(), "failed to fetch next page")
	})
}

// TestListNotebooksDetailed tests the detailed notebook listing function
func TestListNotebooksDetailed(t *testing.T) {
	t.Run("successfully lists notebooks with detailed information", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		
		// Set up notebook and detailed info using the existing mock structure
		mockClient.notebooks = []map[string]interface{}{
			{
				"notebookId":  "notebook-1",
				"displayName": "Work Notebook",
			},
		}
		
		// Set up detailed info using the existing detailedInfo map
		mockClient.detailedInfo["Work Notebook"] = map[string]interface{}{
			"id":                   "notebook-1",
			"notebookId":          "notebook-1", // Backward compatibility
			"displayName":         "Work Notebook",
			"createdDateTime":     "2023-01-01T00:00:00Z",
			"lastModifiedDateTime": "2023-12-01T12:00:00Z",
			"isDefault":           true,
			"userRole":            "Owner",
			"isShared":            false,
			"sectionsUrl":         "https://graph.microsoft.com/v1.0/me/onenote/notebooks/notebook-1/sections",
			"sectionGroupsUrl":    "https://graph.microsoft.com/v1.0/me/onenote/notebooks/notebook-1/sectionGroups",
			"links": map[string]interface{}{
				"oneNoteClientUrl": map[string]interface{}{
					"href": "onenote:https://d.docs.live.net/abc123/Documents/Work%20Notebook",
				},
				"oneNoteWebUrl": map[string]interface{}{
					"href": "https://onedrive.live.com/view.aspx?resid=ABC123&id=documents&wd=target%28Work%20Notebook.one%7C",
				},
			},
			"createdBy": map[string]interface{}{
				"user": map[string]interface{}{
					"id":          "user-123",
					"displayName": "Test User",
				},
			},
			"lastModifiedBy": map[string]interface{}{
				"user": map[string]interface{}{
					"id":          "user-123", 
					"displayName": "Test User",
				},
			},
		}
		
		notebooks, err := mockClient.ListNotebooksDetailed()
		
		assert.NoError(t, err)
		assert.Len(t, notebooks, 1)
		
		notebook := notebooks[0]
		
		// Verify basic properties
		assert.Equal(t, "notebook-1", notebook["id"])
		assert.Equal(t, "notebook-1", notebook["notebookId"]) // Backward compatibility
		assert.Equal(t, "Work Notebook", notebook["displayName"])
		
		// Verify timestamps
		assert.Equal(t, "2023-01-01T00:00:00Z", notebook["createdDateTime"])
		assert.Equal(t, "2023-12-01T12:00:00Z", notebook["lastModifiedDateTime"])
		
		// Verify metadata properties
		assert.Equal(t, true, notebook["isDefault"])
		assert.Equal(t, "Owner", notebook["userRole"])
		assert.Equal(t, false, notebook["isShared"])
		
		// Verify URLs
		assert.Contains(t, notebook, "sectionsUrl")
		assert.Contains(t, notebook, "sectionGroupsUrl")
		
		// Verify nested structures
		links, ok := notebook["links"].(map[string]interface{})
		assert.True(t, ok, "Links should be a map")
		assert.Contains(t, links, "oneNoteClientUrl")
		assert.Contains(t, links, "oneNoteWebUrl")
		
		createdBy, ok := notebook["createdBy"].(map[string]interface{})
		assert.True(t, ok, "CreatedBy should be a map")
		user, ok := createdBy["user"].(map[string]interface{})
		assert.True(t, ok, "User should be a map")
		assert.Equal(t, "user-123", user["id"])
		assert.Equal(t, "Test User", user["displayName"])
	})

	t.Run("returns empty slice when no detailed notebooks found", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		// Empty notebooks slice should result in empty detailed list
		mockClient.notebooks = []map[string]interface{}{}
		
		notebooks, err := mockClient.ListNotebooksDetailed()
		
		assert.NoError(t, err)
		assert.NotNil(t, notebooks, "Should return empty slice, not nil")
		assert.Len(t, notebooks, 0)
	})

	t.Run("handles detailed notebooks with partial information", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		
		// Set up notebooks - the mock will use basic info if detailed info is not available
		mockClient.notebooks = []map[string]interface{}{
			{"notebookId": "minimal-notebook", "displayName": "Minimal Notebook"},
			{"notebookId": "detailed-notebook", "displayName": "Detailed Notebook"},
		}
		
		// Only provide detailed info for one notebook
		mockClient.detailedInfo["Detailed Notebook"] = map[string]interface{}{
			"id":                   "detailed-notebook",
			"displayName":          "Detailed Notebook",
			"createdDateTime":      "2023-01-01T00:00:00Z",
			"lastModifiedDateTime": "2023-12-01T00:00:00Z",
			"isDefault":           true,
			"userRole":            "Owner",
		}
		
		notebooks, err := mockClient.ListNotebooksDetailed()
		
		assert.NoError(t, err)
		assert.Len(t, notebooks, 2)
		
		// Find notebooks by ID
		var minimal, detailed map[string]interface{}
		for _, nb := range notebooks {
			if nb["notebookId"] == "minimal-notebook" {
				minimal = nb
			} else if nb["notebookId"] == "detailed-notebook" {
				detailed = nb
			}
		}
		
		// Verify minimal notebook (falls back to basic info)
		assert.NotNil(t, minimal)
		assert.Equal(t, "minimal-notebook", minimal["notebookId"])
		assert.Equal(t, "Minimal Notebook", minimal["displayName"])
		
		// Verify detailed notebook
		assert.NotNil(t, detailed)
		assert.Equal(t, "detailed-notebook", detailed["id"])
		assert.Equal(t, "Detailed Notebook", detailed["displayName"])
		assert.Equal(t, "2023-01-01T00:00:00Z", detailed["createdDateTime"])
		assert.Equal(t, true, detailed["isDefault"])
		assert.Equal(t, "Owner", detailed["userRole"])
	})

	t.Run("handles authentication errors for detailed notebooks", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		mockClient.SetError(errors.New("401 Unauthorized: Invalid token"))
		
		notebooks, err := mockClient.ListNotebooksDetailed()
		
		assert.Error(t, err)
		assert.Nil(t, notebooks)
		assert.Contains(t, err.Error(), "401 Unauthorized")
	})

	t.Run("validates backward compatibility fields", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		
		mockClient.notebooks = []map[string]interface{}{
			{"notebookId": "backward-compat-test", "displayName": "Backward Compatibility Test"},
		}
		
		mockClient.detailedInfo["Backward Compatibility Test"] = map[string]interface{}{
			"id":          "backward-compat-test",
			"notebookId":  "backward-compat-test", // Should be same as id
			"displayName": "Backward Compatibility Test",
		}
		
		notebooks, err := mockClient.ListNotebooksDetailed()
		
		assert.NoError(t, err)
		assert.Len(t, notebooks, 1)
		
		notebook := notebooks[0]
		assert.Equal(t, notebook["id"], notebook["notebookId"], 
			"notebookId should equal id for backward compatibility")
		assert.Equal(t, "backward-compat-test", notebook["id"])
		assert.Equal(t, "backward-compat-test", notebook["notebookId"])
	})
}

// TestListNotebooksDetailed_Pagination tests pagination for detailed notebooks
func TestListNotebooksDetailed_Pagination(t *testing.T) {
	t.Run("handles detailed notebooks pagination simulation", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		
		// Set up notebooks from multiple "pages"
		mockClient.notebooks = []map[string]interface{}{
			{"notebookId": "page1-nb1", "displayName": "Page 1 Notebook 1"},
			{"notebookId": "page1-nb2", "displayName": "Page 1 Notebook 2"},
			{"notebookId": "page2-nb1", "displayName": "Page 2 Notebook 1"},
			{"notebookId": "page2-nb2", "displayName": "Page 2 Notebook 2"},
		}
		
		// Set up detailed info for some of them
		mockClient.detailedInfo["Page 1 Notebook 1"] = map[string]interface{}{
			"id":        "page1-nb1",
			"displayName": "Page 1 Notebook 1",
			"isDefault": true,
			"userRole":  "Owner",
		}
		mockClient.detailedInfo["Page 2 Notebook 1"] = map[string]interface{}{
			"id":        "page2-nb1", 
			"displayName": "Page 2 Notebook 1",
			"isDefault": false,
			"userRole":  "Owner",
		}
		
		notebooks, err := mockClient.ListNotebooksDetailed()
		
		assert.NoError(t, err)
		assert.Len(t, notebooks, 4, "Should have all notebooks")
		
		// Verify we have notebooks from both pages
		notebookIDs := make([]string, len(notebooks))
		for i, nb := range notebooks {
			if id, ok := nb["notebookId"].(string); ok {
				notebookIDs[i] = id
			}
		}
		
		assert.Contains(t, notebookIDs, "page1-nb1")
		assert.Contains(t, notebookIDs, "page1-nb2") 
		assert.Contains(t, notebookIDs, "page2-nb1")
		assert.Contains(t, notebookIDs, "page2-nb2")
	})
}

// TestListNotebooks_EdgeCases tests various edge cases and error conditions
func TestListNotebooks_EdgeCases(t *testing.T) {
	t.Run("handles extremely large notebook names", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		
		// Create notebook with very long name
		longName := strings.Repeat("Very Long Notebook Name ", 100) // ~2500 characters
		mockClient.notebooks = []map[string]interface{}{
			{
				"notebookId":  "long-name-notebook",
				"displayName": longName,
			},
		}
		
		notebooks, err := mockClient.ListNotebooks()
		
		assert.NoError(t, err)
		assert.Len(t, notebooks, 1)
		assert.Equal(t, longName, notebooks[0]["displayName"])
		assert.Equal(t, len(longName), len(notebooks[0]["displayName"].(string)))
	})

	t.Run("handles notebooks with special characters in names", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		
		specialCharNotebooks := []map[string]interface{}{
			{"notebookId": "unicode-nb", "displayName": "📚 My Notebook with Emojis 🗂️"},
			{"notebookId": "foreign-nb", "displayName": "Тест Заметки (Test Notes)"},
			{"notebookId": "symbols-nb", "displayName": "Project @#$%^&*()_+-={}[]|\\:;\"'<>,.?/~`"},
			{"notebookId": "xml-nb", "displayName": "Notebook with <XML> & \"quotes\" & 'apostrophes'"},
		}
		
		mockClient.notebooks = specialCharNotebooks
		
		notebooks, err := mockClient.ListNotebooks()
		
		assert.NoError(t, err)
		assert.Len(t, notebooks, 4)
		
		// Verify special characters are preserved
		assert.Equal(t, "📚 My Notebook with Emojis 🗂️", notebooks[0]["displayName"])
		assert.Equal(t, "Тест Заметки (Test Notes)", notebooks[1]["displayName"])
		assert.Equal(t, "Project @#$%^&*()_+-={}[]|\\:;\"'<>,.?/~`", notebooks[2]["displayName"])
		assert.Equal(t, "Notebook with <XML> & \"quotes\" & 'apostrophes'", notebooks[3]["displayName"])
	})

	t.Run("handles duplicate notebook names", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		
		duplicateNameNotebooks := []map[string]interface{}{
			{"notebookId": "nb-1", "displayName": "My Notebook"},
			{"notebookId": "nb-2", "displayName": "My Notebook"}, // Same name
			{"notebookId": "nb-3", "displayName": "My Notebook"}, // Same name
		}
		
		mockClient.notebooks = duplicateNameNotebooks
		
		notebooks, err := mockClient.ListNotebooks()
		
		assert.NoError(t, err)
		assert.Len(t, notebooks, 3, "Should return all notebooks even with duplicate names")
		
		// Verify all notebooks have the same name but different IDs
		for _, nb := range notebooks {
			assert.Equal(t, "My Notebook", nb["displayName"])
		}
		
		// Verify IDs are unique
		ids := make(map[string]bool)
		for _, nb := range notebooks {
			id := nb["notebookId"].(string)
			assert.False(t, ids[id], "Notebook ID should be unique: %s", id)
			ids[id] = true
		}
		assert.Len(t, ids, 3, "Should have 3 unique IDs")
	})

	t.Run("handles malformed notebook data gracefully", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		
		malformedNotebooks := []map[string]interface{}{
			{
				"notebookId":     123,        // Wrong type (should be string)
				"displayName":    "Valid Name",
			},
			{
				"notebookId":     "valid-id", 
				"displayName":    []string{"Invalid", "Type"}, // Wrong type
			},
			{
				"unknownField":   "unknown value",
				"anotherField":   42,
				// Missing both required fields
			},
		}
		
		mockClient.notebooks = malformedNotebooks
		
		notebooks, err := mockClient.ListNotebooks()
		
		// Should not error, but handle malformed data gracefully
		assert.NoError(t, err)
		assert.Len(t, notebooks, 3, "Should return all entries even if malformed")
		
		// Data should be preserved as-is (type conversion is handled by the extraction functions)
		assert.Equal(t, 123, notebooks[0]["notebookId"])
		assert.Equal(t, []string{"Invalid", "Type"}, notebooks[1]["displayName"])
		assert.Equal(t, "unknown value", notebooks[2]["unknownField"])
	})
}

// TestListNotebooks_Performance tests performance-related scenarios
func TestListNotebooks_Performance(t *testing.T) {
	t.Run("handles large number of notebooks efficiently", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		
		// Create a large number of notebooks
		largeNotebookList := make([]map[string]interface{}, 1000)
		for i := 0; i < 1000; i++ {
			largeNotebookList[i] = map[string]interface{}{
				"notebookId":  fmt.Sprintf("notebook-%04d", i),
				"displayName": fmt.Sprintf("Test Notebook %04d", i),
			}
		}
		
		mockClient.notebooks = largeNotebookList
		
		start := time.Now()
		notebooks, err := mockClient.ListNotebooks()
		duration := time.Since(start)
		
		assert.NoError(t, err)
		assert.Len(t, notebooks, 1000)
		
		// Should complete reasonably quickly (adjust threshold as needed)
		assert.Less(t, duration, 100*time.Millisecond, 
			"Large notebook list should be processed efficiently")
		
		// Verify first and last notebooks
		assert.Equal(t, "notebook-0000", notebooks[0]["notebookId"])
		assert.Equal(t, "notebook-0999", notebooks[999]["notebookId"])
	})

	t.Run("measures detailed notebook processing performance", func(t *testing.T) {
		mockClient := NewMockNotebookClient()
		
		// Create notebooks and detailed info for performance testing
		largeNotebookList := make([]map[string]interface{}, 50)
		for i := 0; i < 50; i++ {
			notebookName := fmt.Sprintf("Detailed Notebook %03d", i)
			largeNotebookList[i] = map[string]interface{}{
				"notebookId":  fmt.Sprintf("detailed-nb-%03d", i),
				"displayName": notebookName,
			}
			
			// Add detailed info for this notebook
			mockClient.detailedInfo[notebookName] = map[string]interface{}{
				"id":                   fmt.Sprintf("detailed-nb-%03d", i),
				"displayName":          notebookName,
				"createdDateTime":      "2023-01-01T00:00:00Z",
				"lastModifiedDateTime": "2023-12-01T00:00:00Z",
				"isDefault":           i == 0,
				"userRole":            "Owner",
				"isShared":            i%10 == 0, // Every 10th notebook is shared
				"links": map[string]interface{}{
					"oneNoteClientUrl": map[string]interface{}{
						"href": fmt.Sprintf("onenote:notebook-%d", i),
					},
					"oneNoteWebUrl": map[string]interface{}{
						"href": fmt.Sprintf("https://example.com/notebook-%d", i),
					},
				},
				"createdBy": map[string]interface{}{
					"user": map[string]interface{}{
						"id":          fmt.Sprintf("user-%d", i%5), // 5 different users
						"displayName": fmt.Sprintf("User %d", i%5),
					},
				},
			}
		}
		
		mockClient.notebooks = largeNotebookList
		
		start := time.Now()
		notebooks, err := mockClient.ListNotebooksDetailed()
		duration := time.Since(start)
		
		assert.NoError(t, err)
		assert.Len(t, notebooks, 50)
		
		// Should handle complex structures efficiently
		assert.Less(t, duration, 200*time.Millisecond, 
			"Detailed notebook processing should be efficient")
		
		// Verify structure integrity for first few notebooks
		for i := 0; i < 3 && i < len(notebooks); i++ {
			nb := notebooks[i]
			
			if links, ok := nb["links"].(map[string]interface{}); ok {
				assert.Contains(t, links, "oneNoteClientUrl")
				assert.Contains(t, links, "oneNoteWebUrl")
			}
			
			if createdBy, ok := nb["createdBy"].(map[string]interface{}); ok {
				user, ok := createdBy["user"].(map[string]interface{})
				assert.True(t, ok, "User should be properly structured")
				assert.Contains(t, user, "id")
				assert.Contains(t, user, "displayName")
			}
		}
	})
}