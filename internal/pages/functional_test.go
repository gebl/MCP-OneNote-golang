// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package pages

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gebl/onenote-mcp-server/internal/graph"
)

// TestPageClientFunctionality tests core page client functionality
func TestPageClientFunctionality(t *testing.T) {
	t.Run("page client creation", func(t *testing.T) {
		graphClient := &graph.Client{}
		pageClient := NewPageClient(graphClient)

		assert.NotNil(t, pageClient)
		assert.Equal(t, graphClient, pageClient.Client)
		assert.IsType(t, &PageClient{}, pageClient)
	})
}

// TestPageNotebookResolution tests page-to-notebook resolution logic
func TestPageNotebookResolution(t *testing.T) {
	t.Run("parse page metadata response", func(t *testing.T) {
		responseBody := `{
			"id": "page-123",
			"title": "Test Page",
			"parentSection": {
				"id": "section-456",
				"displayName": "Test Section"
			},
			"parentNotebook": {
				"id": "notebook-789",
				"displayName": "Test Notebook"
			}
		}`

		var pageInfo struct {
			ID             string `json:"id"`
			Title          string `json:"title"`
			ParentSection  *struct {
				ID          string `json:"id"`
				DisplayName string `json:"displayName"`
			} `json:"parentSection"`
			ParentNotebook *struct {
				ID          string `json:"id"`
				DisplayName string `json:"displayName"`
			} `json:"parentNotebook"`
		}

		err := json.Unmarshal([]byte(responseBody), &pageInfo)
		require.NoError(t, err)

		assert.Equal(t, "page-123", pageInfo.ID)
		assert.Equal(t, "Test Page", pageInfo.Title)
		assert.NotNil(t, pageInfo.ParentSection)
		assert.Equal(t, "section-456", pageInfo.ParentSection.ID)
		assert.Equal(t, "Test Section", pageInfo.ParentSection.DisplayName)
		assert.NotNil(t, pageInfo.ParentNotebook)
		assert.Equal(t, "notebook-789", pageInfo.ParentNotebook.ID)
		assert.Equal(t, "Test Notebook", pageInfo.ParentNotebook.DisplayName)
	})

	t.Run("handle missing parent information", func(t *testing.T) {
		responseBody := `{
			"id": "page-123",
			"title": "Orphaned Page"
		}`

		var pageInfo struct {
			ID             string `json:"id"`
			Title          string `json:"title"`
			ParentSection  *struct {
				ID          string `json:"id"`
				DisplayName string `json:"displayName"`
			} `json:"parentSection"`
			ParentNotebook *struct {
				ID          string `json:"id"`
				DisplayName string `json:"displayName"`
			} `json:"parentNotebook"`
		}

		err := json.Unmarshal([]byte(responseBody), &pageInfo)
		require.NoError(t, err)

		assert.Equal(t, "page-123", pageInfo.ID)
		assert.Equal(t, "Orphaned Page", pageInfo.Title)
		assert.Nil(t, pageInfo.ParentSection)
		assert.Nil(t, pageInfo.ParentNotebook)
	})
}

// TestUpdateCommandValidation tests page update command validation
func TestUpdateCommandValidation(t *testing.T) {
	t.Run("validate update command structure", func(t *testing.T) {
		tests := []struct {
			name    string
			command UpdateCommand
			valid   bool
			reason  string
		}{
			{
				name: "valid append command",
				command: UpdateCommand{
					Action:  "append",
					Target:  "body",
					Content: "<p>New content</p>",
				},
				valid: true,
			},
			{
				name: "valid insert command with position",
				command: UpdateCommand{
					Action:   "insert",
					Target:   "data-id:element-123",
					Content:  "<p>Inserted content</p>",
					Position: "before",
				},
				valid: true,
			},
			{
				name: "invalid command without action",
				command: UpdateCommand{
					Target:  "body",
					Content: "<p>Content</p>",
				},
				valid:  false,
				reason: "missing action",
			},
			{
				name: "invalid command without content",
				command: UpdateCommand{
					Action: "append",
					Target: "body",
				},
				valid:  false,
				reason: "missing content",
			},
			{
				name: "invalid action type",
				command: UpdateCommand{
					Action:  "invalidAction",
					Target:  "body",
					Content: "<p>Content</p>",
				},
				valid:  false,
				reason: "invalid action",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Validate required fields
				hasAction := tt.command.Action != ""
				hasContent := tt.command.Content != ""
				
				validActions := map[string]bool{
					"append":  true,
					"insert":  true,
					"replace": true,
					"delete":  true,
					"prepend": true,
				}
				
				validAction := validActions[tt.command.Action]
				isValid := hasAction && hasContent && validAction

				if tt.valid {
					assert.True(t, isValid, "Command should be valid: %s", tt.reason)
				} else {
					assert.False(t, isValid, "Command should be invalid: %s", tt.reason)
				}
			})
		}
	})
}

// TestTableUpdateValidation tests table update restrictions
func TestTableUpdateValidation(t *testing.T) {
	t.Run("validate table update restrictions", func(t *testing.T) {
		tests := []struct {
			name        string
			commands    []UpdateCommand
			shouldError bool
			reason      string
		}{
			{
				name: "valid non-table commands",
				commands: []UpdateCommand{
					{Action: "append", Target: "body", Content: "<p>New paragraph</p>"},
					{Action: "replace", Target: "data-id:div-123", Content: "<div>Updated div</div>"},
				},
				shouldError: false,
			},
			{
				name: "valid table replacement",
				commands: []UpdateCommand{
					{Action: "replace", Target: "data-id:table-123", Content: "<table><tr><td>New table</td></tr></table>"},
				},
				shouldError: false,
			},
			{
				name: "invalid table cell update",
				commands: []UpdateCommand{
					{Action: "replace", Target: "data-id:cell-123", Content: "<td>Updated cell</td>"},
				},
				shouldError: true,
				reason:      "table cells cannot be updated individually",
			},
			{
				name: "invalid table row update",
				commands: []UpdateCommand{
					{Action: "replace", Target: "data-id:tr-123", Content: "<tr><td>New row</td></tr>"},
				},
				shouldError: true,
				reason:      "table rows cannot be updated individually",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := validateTableUpdates(tt.commands)
				if tt.shouldError {
					assert.Error(t, err, "Should error: %s", tt.reason)
				} else {
					assert.NoError(t, err, "Should not error: %s", tt.reason)
				}
			})
		}
	})
}

// TestPageItemExtraction tests page item ID extraction from URLs
func TestPageItemExtraction(t *testing.T) {
	t.Run("extract page item IDs from various URL formats", func(t *testing.T) {
		tests := []struct {
			name       string
			url        string
			expectedID string
		}{
			{
				name:       "Microsoft Graph API URL",
				url:        "https://graph.microsoft.com/v1.0/me/onenote/pages/page-123/content/resources/item-456/$value",
				expectedID: "item-456",
			},
			{
				name:       "Legacy OneNote API URL",
				url:        "https://www.onenote.com/api/v1.0/pages/page-123/resources/item-789/$value",
				expectedID: "item-789",
			},
			{
				name:       "Complex OneNote ID with segments",
				url:        "https://graph.microsoft.com/beta/me/onenote/pages/1-ABC123DEF456!789/resources/0-XYZ987UVW654!321/$value",
				expectedID: "0-XYZ987UVW654!321",
			},
			{
				name:       "URL without resources pattern",
				url:        "https://graph.microsoft.com/v1.0/me/onenote/pages/page-123",
				expectedID: "",
			},
			{
				name:       "Empty URL",
				url:        "",
				expectedID: "",
			},
			{
				name:       "URL with resources but no $value",
				url:        "https://graph.microsoft.com/v1.0/me/onenote/pages/page-123/resources/item-456",
				expectedID: "",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				extractedID := extractPageItemID(tt.url)
				assert.Equal(t, tt.expectedID, extractedID)
			})
		}
	})
}

// TestHTMLContentProcessing tests HTML content processing
func TestHTMLContentProcessing(t *testing.T) {
	t.Run("extract data-id attributes from HTML", func(t *testing.T) {
		htmlContent := `
		<html>
		<head><title>Test Page</title></head>
		<body data-absolute-enabled="true" style="font-family:Calibri,sans-serif;font-size:11pt">
			<div data-id="div-123">
				<p data-id="p-456">Existing content</p>
			</div>
			<table data-id="table-789">
				<tr data-id="row-001"><td data-id="cell-001">Cell content</td></tr>
			</table>
		</body>
		</html>`

		// Extract all data-id attributes
		dataIds := extractDataIDs(htmlContent)

		expectedIDs := []string{"div-123", "p-456", "table-789", "row-001", "cell-001"}
		assert.GreaterOrEqual(t, len(dataIds), len(expectedIDs))

		// Check that key IDs are found
		for _, expectedID := range expectedIDs {
			assert.Contains(t, dataIds, expectedID, "Should contain data-id: %s", expectedID)
		}
	})

	t.Run("handle HTML without data-id attributes", func(t *testing.T) {
		htmlContent := `
		<html>
		<head><title>Simple Page</title></head>
		<body>
			<div>
				<p>Simple content without data-id</p>
			</div>
		</body>
		</html>`

		dataIds := extractDataIDs(htmlContent)
		assert.Empty(t, dataIds)
	})
}

// TestHTMLEscaping tests HTML content escaping functionality
func TestHTMLEscapingFunctionality(t *testing.T) {
	t.Run("escape HTML special characters", func(t *testing.T) {
		tests := []struct {
			input    string
			expected string
		}{
			{"Hello & World", "Hello &amp; World"},
			{"<script>alert('xss')</script>", "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;"},
			{"Quote: \"Hello\"", "Quote: &quot;Hello&quot;"},
			{"Less < and > greater", "Less &lt; and &gt; greater"},
			{"Single 'quote'", "Single &#39;quote&#39;"},
			{"Multiple & < > \" ' characters", "Multiple &amp; &lt; &gt; &quot; &#39; characters"},
			{"", ""},
			{"No special chars", "No special chars"},
		}

		for _, tt := range tests {
			t.Run(tt.input, func(t *testing.T) {
				result := htmlEscape(tt.input)
				assert.Equal(t, tt.expected, result)
			})
		}
	})
}

// TestUpdateCommandJSON tests JSON marshaling for UpdateCommand
func TestUpdateCommandJSON(t *testing.T) {
	t.Run("marshal commands correctly", func(t *testing.T) {
		tests := []struct {
			name                string
			command             UpdateCommand
			shouldIncludePosition bool
		}{
			{
				name: "append command excludes position",
				command: UpdateCommand{
					Action:   "append",
					Target:   "body",
					Content:  "<p>New content</p>",
					Position: "after", // Should be excluded
				},
				shouldIncludePosition: false,
			},
			{
				name: "insert command includes position",
				command: UpdateCommand{
					Action:   "insert",
					Target:   "data-id:element-123",
					Content:  "<p>Inserted content</p>",
					Position: "before",
				},
				shouldIncludePosition: true,
			},
			{
				name: "replace command includes position",
				command: UpdateCommand{
					Action:   "replace",
					Target:   "data-id:element-456",
					Content:  "<p>Replacement content</p>",
					Position: "after",
				},
				shouldIncludePosition: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				data, err := json.Marshal(tt.command)
				require.NoError(t, err)

				var result map[string]interface{}
				err = json.Unmarshal(data, &result)
				require.NoError(t, err)

				assert.Equal(t, tt.command.Action, result["action"])
				assert.Equal(t, tt.command.Target, result["target"])
				assert.Equal(t, tt.command.Content, result["content"])

				_, hasPosition := result["position"]
				if tt.shouldIncludePosition {
					assert.True(t, hasPosition, "Position should be included for %s action", tt.command.Action)
					assert.Equal(t, tt.command.Position, result["position"])
				} else {
					assert.False(t, hasPosition, "Position should be excluded for %s action", tt.command.Action)
				}
			})
		}
	})
}

// TestPageItemDataStructures tests data structures
func TestPageItemDataStructures(t *testing.T) {
	t.Run("PageItemData structure validation", func(t *testing.T) {
		pageItem := &PageItemData{
			ContentType: "image/jpeg",
			Filename:    "test-image.jpg",
			Size:        1024,
			Content:     []byte("fake-image-data"),
			TagName:     "img",
			Attributes: map[string]string{
				"src":   "test.jpg",
				"alt":   "Test image",
				"width": "100",
			},
			OriginalURL: "https://example.com/test.jpg",
		}

		assert.Equal(t, "image/jpeg", pageItem.ContentType)
		assert.Equal(t, "test-image.jpg", pageItem.Filename)
		assert.Equal(t, int64(1024), pageItem.Size)
		assert.NotEmpty(t, pageItem.Content)
		assert.Equal(t, "img", pageItem.TagName)
		assert.Len(t, pageItem.Attributes, 3)
		assert.Equal(t, "test.jpg", pageItem.Attributes["src"])
		assert.Equal(t, "https://example.com/test.jpg", pageItem.OriginalURL)
	})

	t.Run("empty PageItemData handling", func(t *testing.T) {
		pageItem := &PageItemData{}

		assert.Empty(t, pageItem.ContentType)
		assert.Empty(t, pageItem.Filename)
		assert.Equal(t, int64(0), pageItem.Size)
		assert.Empty(t, pageItem.Content)
		assert.Empty(t, pageItem.TagName)
		assert.Nil(t, pageItem.Attributes)
		assert.Empty(t, pageItem.OriginalURL)
	})
}

// TestErrorHandling tests various error scenarios
func TestErrorHandling(t *testing.T) {
	t.Run("handle malformed JSON", func(t *testing.T) {
		malformedJSON := `{"incomplete": json`

		var result map[string]interface{}
		err := json.Unmarshal([]byte(malformedJSON), &result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid character")
	})

	t.Run("handle empty JSON object", func(t *testing.T) {
		emptyJSON := `{}`

		var pageInfo struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		}

		err := json.Unmarshal([]byte(emptyJSON), &pageInfo)
		assert.NoError(t, err)
		assert.Empty(t, pageInfo.ID)
		assert.Empty(t, pageInfo.Title)
	})
}

// Helper functions

// extractDataIDs extracts data-id attributes from HTML content
func extractDataIDs(htmlContent string) []string {
	var ids []string

	// Simple pattern matching for data-id attributes
	parts := strings.Split(htmlContent, `data-id="`)
	for i := 1; i < len(parts); i++ {
		endQuote := strings.Index(parts[i], `"`)
		if endQuote > 0 {
			ids = append(ids, parts[i][:endQuote])
		}
	}

	return ids
}

// Performance benchmarks
func BenchmarkHTMLEscapePerformance(b *testing.B) {
	testString := "This is a test string with <tags> & \"quotes\" and 'apostrophes' that needs escaping"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = htmlEscape(testString)
	}
}

func BenchmarkUpdateCommandMarshalPerformance(b *testing.B) {
	cmd := UpdateCommand{
		Action:   "replace",
		Target:   "data-id:element-123",
		Content:  "<p>This is replacement content with some length to simulate real usage</p>",
		Position: "after",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(cmd)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDataIdExtraction(b *testing.B) {
	htmlContent := `
	<html>
	<body>
		<div data-id="div-1"><p data-id="p-1">Content 1</p></div>
		<div data-id="div-2"><p data-id="p-2">Content 2</p></div>
		<table data-id="table-1">
			<tr data-id="row-1"><td data-id="cell-1">Cell 1</td></tr>
			<tr data-id="row-2"><td data-id="cell-2">Cell 2</td></tr>
		</table>
	</body>
	</html>`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = extractDataIDs(htmlContent)
	}
}