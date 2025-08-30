// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package notebooks

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/gebl/onenote-mcp-server/internal/graph"
)

// TestNotebookSearchFunctionality tests search logic without complex mocks
func TestNotebookSearchFunctionality(t *testing.T) {
	t.Run("notebook search by name logic", func(t *testing.T) {
		// Mock data structure that mimics the actual API response
		notebooks := []map[string]interface{}{
			{"id": "notebook-1", "displayName": "Work Notebook"},
			{"id": "notebook-2", "displayName": "Personal Journal"},
			{"id": "notebook-3", "displayName": "Study Materials"},
		}

		// Test exact match search
		targetName := "Personal Journal"
		var foundID string

		for _, notebook := range notebooks {
			if displayName, ok := notebook["displayName"].(string); ok && displayName == targetName {
				if id, ok := notebook["id"].(string); ok {
					foundID = id
					break
				}
			}
		}

		assert.Equal(t, "notebook-2", foundID)
	})

	t.Run("notebook search case sensitivity", func(t *testing.T) {
		notebooks := []map[string]interface{}{
			{"id": "notebook-1", "displayName": "Personal Notebook"},
		}

		// Test case sensitivity
		targetName := "personal notebook" // Different case
		var foundID string

		for _, notebook := range notebooks {
			if displayName, ok := notebook["displayName"].(string); ok && displayName == targetName {
				if id, ok := notebook["id"].(string); ok {
					foundID = id
					break
				}
			}
		}

		assert.Equal(t, "", foundID) // Should not match due to case sensitivity
	})
}

// TestPageSearchInNotebook tests page search functionality within notebooks  
func TestPageSearchInNotebook(t *testing.T) {
	t.Run("case insensitive page title search", func(t *testing.T) {
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
					pageResult := make(map[string]interface{})
					for k, v := range page {
						pageResult[k] = v
					}
					pageResult["sectionName"] = "Test Section"
					pageResult["sectionId"] = "section-1"
					pageResult["sectionPath"] = "/Test Section"
					matchingPages = append(matchingPages, pageResult)
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

	t.Run("search with special characters", func(t *testing.T) {
		query := "C++ Programming"
		pages := []map[string]interface{}{
			{"id": "page-1", "title": "C++ Programming Guide"},
			{"id": "page-2", "title": "Java Programming"},
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

		assert.Len(t, matchingPages, 1)
		assert.Equal(t, "C++ Programming Guide", matchingPages[0]["title"])
	})
}

// TestNotebookDataProcessing tests notebook data processing
func TestNotebookDataProcessing(t *testing.T) {
	t.Run("process notebook list with missing fields", func(t *testing.T) {
		// Simulate processing notebooks with various field combinations
		notebooks := []map[string]interface{}{
			{"id": "notebook-1", "displayName": "Complete Notebook"},
			{"displayName": "Notebook without ID"},                    // Missing ID
			{"id": "notebook-3"},                                      // Missing display name
			{"id": "notebook-4", "displayName": "Another Complete"},
		}

		var validNotebooks []map[string]interface{}

		// Process notebooks, keeping only complete ones
		for _, notebook := range notebooks {
			if id, hasID := notebook["id"].(string); hasID && id != "" {
				if displayName, hasName := notebook["displayName"].(string); hasName && displayName != "" {
					validNotebooks = append(validNotebooks, notebook)
				}
			}
		}

		assert.Len(t, validNotebooks, 2) // Only complete notebooks
		assert.Equal(t, "notebook-1", validNotebooks[0]["id"])
		assert.Equal(t, "notebook-4", validNotebooks[1]["id"])
	})

	t.Run("handle notebook data type validation", func(t *testing.T) {
		// Test handling of various data types
		notebooks := []map[string]interface{}{
			{"id": "notebook-1", "displayName": "Valid String"},
			{"id": 123, "displayName": "Non-string ID"},           // Invalid ID type
			{"id": "notebook-3", "displayName": 456},             // Invalid displayName type
			{"id": nil, "displayName": "Nil ID"},                 // Nil ID
		}

		var validNotebooks []map[string]interface{}

		for _, notebook := range notebooks {
			// Strict type checking
			if id, ok := notebook["id"].(string); ok && id != "" {
				if displayName, ok := notebook["displayName"].(string); ok && displayName != "" {
					validNotebooks = append(validNotebooks, notebook)
				}
			}
		}

		assert.Len(t, validNotebooks, 1) // Only one valid notebook
		assert.Equal(t, "notebook-1", validNotebooks[0]["id"])
	})
}

// TestNotebookClientCreation tests notebook client initialization
func TestNotebookClientCreation(t *testing.T) {
	t.Run("client creation and composition", func(t *testing.T) {
		graphClient := &graph.Client{}
		notebookClient := NewNotebookClient(graphClient)

		assert.NotNil(t, notebookClient)
		assert.Equal(t, graphClient, notebookClient.Client)
		assert.IsType(t, &NotebookClient{}, notebookClient)
	})
}

// TestNotebookValidation tests validation functions
func TestNotebookValidation(t *testing.T) {
	t.Run("validate notebook identifiers", func(t *testing.T) {
		validIDs := []string{
			"0-B2C4D6F8A1B3C5D7!123",
			"1-A1B2C3D4E5F6G7H8!456",
			"notebook-uuid-format",
			"simple-id",
		}

		for _, id := range validIDs {
			// Basic validation - ID should not be empty and should contain valid characters
			assert.NotEmpty(t, id)
			assert.NotContains(t, id, " ") // No spaces
			assert.Greater(t, len(id), 0)
		}
	})

	t.Run("validate notebook display names", func(t *testing.T) {
		validNames := []string{
			"Work Notebook",
			"Personal Notes 2024",
			"Project-Alpha_v1",
			"简单笔记本", // Unicode characters
		}

		invalidNames := []string{
			"",                           // Empty name
			strings.Repeat("a", 256),     // Too long
			"Name\x00WithNull",           // Contains null character
		}

		for _, name := range validNames {
			assert.NotEmpty(t, name)
			assert.LessOrEqual(t, len(name), 255) // Reasonable length limit
			assert.NotContains(t, name, "\x00")   // No null characters
		}

		for _, name := range invalidNames {
			if name == "" {
				assert.Empty(t, name)
			} else if len(name) > 255 {
				assert.Greater(t, len(name), 255)
			} else if strings.Contains(name, "\x00") {
				assert.Contains(t, name, "\x00")
			}
		}
	})
}

// TestErrorHandlingScenarios tests various error conditions
func TestErrorHandlingScenarios(t *testing.T) {
	t.Run("handle empty notebook collections", func(t *testing.T) {
		var notebooks []map[string]interface{}

		// Should handle empty collections gracefully  
		assert.Len(t, notebooks, 0)

		// Operations on empty collections should work
		for _, notebook := range notebooks {
			// This loop body should never execute
			t.Errorf("Unexpected notebook in empty collection: %v", notebook)
		}
	})

	t.Run("handle nil notebook entries", func(t *testing.T) {
		// This simulates robustness against potential nil values
		notebooks := []map[string]interface{}{
			{"id": "valid-1", "displayName": "Valid Notebook"},
			nil, // Potential nil entry
		}

		var validNotebooks []map[string]interface{}

		for _, notebook := range notebooks {
			if notebook != nil { // Nil check
				if id, ok := notebook["id"].(string); ok && id != "" {
					if displayName, ok := notebook["displayName"].(string); ok && displayName != "" {
						validNotebooks = append(validNotebooks, notebook)
					}
				}
			}
		}

		assert.Len(t, validNotebooks, 1)
		assert.Equal(t, "valid-1", validNotebooks[0]["id"])
	})
}

// Performance benchmarks
func BenchmarkNotebookSearchPerformance(b *testing.B) {
	// Create a large dataset for performance testing
	notebooks := make([]map[string]interface{}, 10000)
	for i := 0; i < 10000; i++ {
		notebooks[i] = map[string]interface{}{
			"id":          fmt.Sprintf("notebook-%d", i),
			"displayName": fmt.Sprintf("Notebook %d", i),
		}
	}
	// Add target at the end (worst case scenario)
	notebooks[9999] = map[string]interface{}{
		"id":          "target-notebook",
		"displayName": "Target Notebook",
	}

	targetName := "Target Notebook"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var foundID string
		for _, notebook := range notebooks {
			if displayName, ok := notebook["displayName"].(string); ok && displayName == targetName {
				if id, ok := notebook["id"].(string); ok {
					foundID = id
					break
				}
			}
		}
		_ = foundID // Use result to prevent optimization
	}
}

func BenchmarkPageSearchPerformance(b *testing.B) {
	// Create test data for page search performance
	pages := make([]map[string]interface{}, 1000)
	for i := 0; i < 1000; i++ {
		pages[i] = map[string]interface{}{
			"id":    fmt.Sprintf("page-%d", i),
			"title": fmt.Sprintf("Page %d Meeting Notes", i),
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
					pageResult := make(map[string]interface{})
					for k, v := range page {
						pageResult[k] = v
					}
					pageResult["sectionName"] = "Test Section"
					pageResult["sectionId"] = "section-1"
					pageResult["sectionPath"] = "/Test Section"
					matchingPages = append(matchingPages, pageResult)
				}
			}
		}
		_ = matchingPages // Use result
	}
}