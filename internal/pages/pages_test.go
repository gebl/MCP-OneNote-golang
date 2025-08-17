// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package pages

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/gebl/onenote-mcp-server/internal/graph"
)

func TestNewPageClient(t *testing.T) {
	t.Run("creates new page client", func(t *testing.T) {
		graphClient := &graph.Client{}
		pageClient := NewPageClient(graphClient)

		assert.NotNil(t, pageClient)
		assert.Equal(t, graphClient, pageClient.Client)
	})
}

func TestHTMLEscape(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "escape ampersand",
			input:    "Tom & Jerry",
			expected: "Tom &amp; Jerry",
		},
		{
			name:     "escape less than",
			input:    "x < y",
			expected: "x &lt; y",
		},
		{
			name:     "escape greater than",
			input:    "x > y",
			expected: "x &gt; y",
		},
		{
			name:     "escape double quotes",
			input:    `Say "Hello"`,
			expected: "Say &quot;Hello&quot;",
		},
		{
			name:     "escape single quotes",
			input:    "Don't worry",
			expected: "Don&#39;t worry",
		},
		{
			name:     "escape multiple characters",
			input:    `<title>Tom & Jerry's "Adventures"</title>`,
			expected: "&lt;title&gt;Tom &amp; Jerry&#39;s &quot;Adventures&quot;&lt;/title&gt;",
		},
		{
			name:     "no escaping needed",
			input:    "Simple Title",
			expected: "Simple Title",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := htmlEscape(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateTableUpdates(t *testing.T) {
	tests := []struct {
		name        string
		commands    []UpdateCommand
		expectError bool
		description string
	}{
		{
			name: "valid non-table commands",
			commands: []UpdateCommand{
				{Target: "body", Action: "replace", Content: "<p>Test</p>"},
				{Target: "title", Action: "replace", Content: "New Title"},
				{Target: "data-id:element-123", Action: "append", Content: "<p>Added</p>"},
			},
			expectError: false,
			description: "Non-table commands should be allowed",
		},
		{
			name: "valid table command targeting whole table",
			commands: []UpdateCommand{
				{Target: "table:data-id-123", Action: "replace", Content: "<table><tr><td>Cell</td></tr></table>"},
			},
			expectError: false,
			description: "Targeting whole table should be allowed",
		},
		{
			name: "invalid table cell command",
			commands: []UpdateCommand{
				{Target: "td:data-id-456", Action: "replace", Content: "<td>New Content</td>"},
			},
			expectError: true,
			description: "Targeting individual table cells should be blocked",
		},
		{
			name: "invalid table header command",
			commands: []UpdateCommand{
				{Target: "th:data-id-789", Action: "replace", Content: "<th>New Header</th>"},
			},
			expectError: true,
			description: "Targeting individual table headers should be blocked",
		},
		{
			name: "invalid table row command",
			commands: []UpdateCommand{
				{Target: "tr:data-id-101", Action: "replace", Content: "<tr><td>Row</td></tr>"},
			},
			expectError: true,
			description: "Targeting individual table rows should be blocked",
		},
		{
			name: "mixed commands with table cell violation",
			commands: []UpdateCommand{
				{Target: "body", Action: "replace", Content: "<p>Valid</p>"},
				{Target: "td:data-id-456", Action: "replace", Content: "<td>Invalid</td>"},
				{Target: "title", Action: "replace", Content: "Also Valid"},
			},
			expectError: true,
			description: "Mixed commands with table violations should be blocked",
		},
		{
			name: "multiple table element violations",
			commands: []UpdateCommand{
				{Target: "td:cell-1", Action: "replace", Content: "<td>Cell 1</td>"},
				{Target: "th:header-1", Action: "replace", Content: "<th>Header 1</th>"},
				{Target: "tr:row-1", Action: "replace", Content: "<tr><td>Row 1</td></tr>"},
			},
			expectError: true,
			description: "Multiple table element violations should be blocked",
		},
		{
			name:        "empty commands array",
			commands:    []UpdateCommand{},
			expectError: false,
			description: "Empty commands should be allowed (validation passes)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTableUpdates(tt.commands)

			if tt.expectError {
				assert.Error(t, err, tt.description)
				if err != nil {
					// Check that error message contains guidance about table restrictions
					assert.Contains(t, err.Error(), "TABLE UPDATE RESTRICTION", "Error should explain table restrictions")
					assert.Contains(t, err.Error(), "complete units", "Error should mention complete table updates")
				}
			} else {
				assert.NoError(t, err, tt.description)
			}
		})
	}
}

func TestUpdateCommandMarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		command     UpdateCommand
		expectField func(string) bool
		description string
	}{
		{
			name: "append action excludes position",
			command: UpdateCommand{
				Target:   "body",
				Action:   "append",
				Position: "after", // This should be excluded
				Content:  "<p>Test</p>",
			},
			expectField: func(json string) bool {
				return !strings.Contains(json, "position") // Position should NOT be present
			},
			description: "Append actions should exclude position field",
		},
		{
			name: "insert action includes position",
			command: UpdateCommand{
				Target:   "data-id:element-123",
				Action:   "insert",
				Position: "before",
				Content:  "<p>Test</p>",
			},
			expectField: func(json string) bool {
				return strings.Contains(json, "position") // Position should be present
			},
			description: "Insert actions should include position field",
		},
		{
			name: "replace action includes position",
			command: UpdateCommand{
				Target:   "title",
				Action:   "replace",
				Position: "after", // Should be included even though rarely used
				Content:  "New Title",
			},
			expectField: func(json string) bool {
				return strings.Contains(json, "position") // Position should be present
			},
			description: "Replace actions should include position field",
		},
		{
			name: "prepend action includes position",
			command: UpdateCommand{
				Target:   "body",
				Action:   "prepend",
				Position: "before",
				Content:  "<p>Prepended</p>",
			},
			expectField: func(json string) bool {
				return strings.Contains(json, "position") // Position should be present
			},
			description: "Prepend actions should include position field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBytes, err := tt.command.MarshalJSON()
			assert.NoError(t, err, "JSON marshaling should not fail")

			jsonString := string(jsonBytes)
			assert.True(t, tt.expectField(jsonString), tt.description)

			// Verify that other required fields are always present
			assert.Contains(t, jsonString, "target", "Target field should always be present")
			assert.Contains(t, jsonString, "action", "Action field should always be present")
			assert.Contains(t, jsonString, "content", "Content field should always be present")
		})
	}
}

func TestExtractPageIDFromResourceLocation(t *testing.T) {
	// Create a client for testing
	graphClient := &graph.Client{}
	pageClient := NewPageClient(graphClient)

	tests := []struct {
		name        string
		jsonData    map[string]interface{}
		expectedID  string
		expectError bool
		description string
	}{
		{
			name: "valid graph API URL",
			jsonData: map[string]interface{}{
				"resourceLocation": "https://graph.microsoft.com/beta/users/12345/onenote/pages/1-ABC123DEF456!789",
			},
			expectedID:  "1-ABC123DEF456!789",
			expectError: false,
			description: "Should extract page ID from valid Graph API URL",
		},
		{
			name: "valid graph API URL with different format",
			jsonData: map[string]interface{}{
				"resourceLocation": "https://graph.microsoft.com/v1.0/me/onenote/pages/2-XYZ789ABC123!456",
			},
			expectedID:  "2-XYZ789ABC123!456",
			expectError: false,
			description: "Should extract page ID from v1.0 API URL",
		},
		{
			name: "missing resourceLocation field",
			jsonData: map[string]interface{}{
				"status": "Completed",
				"id":     "operation-123",
			},
			expectedID:  "",
			expectError: true,
			description: "Should error when resourceLocation field is missing",
		},
		{
			name: "non-string resourceLocation",
			jsonData: map[string]interface{}{
				"resourceLocation": 12345,
			},
			expectedID:  "",
			expectError: true,
			description: "Should error when resourceLocation is not a string",
		},
		{
			name: "invalid URL format",
			jsonData: map[string]interface{}{
				"resourceLocation": "https://invalid-url.com/not-onenote",
			},
			expectedID:  "",
			expectError: true,
			description: "Should error when URL doesn't contain onenote pages pattern",
		},
		{
			name: "empty resourceLocation",
			jsonData: map[string]interface{}{
				"resourceLocation": "",
			},
			expectedID:  "",
			expectError: true,
			description: "Should error when resourceLocation is empty",
		},
		{
			name: "URL with pages but no ID",
			jsonData: map[string]interface{}{
				"resourceLocation": "https://graph.microsoft.com/v1.0/me/onenote/pages/",
			},
			expectedID:  "",
			expectError: true,
			description: "Should error when URL has pages path but no ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the actual function now that it's a method
			result, err := pageClient.extractPageIDFromResourceLocation(tt.jsonData)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, result)
			}
		})
	}
}

func TestExtractPageItemID(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectedID  string
		description string
	}{
		{
			name:        "legacy OneNote API URL",
			url:         "https://www.onenote.com/api/v1.0/me/notes/resources/ABC123DEF456!789/$value",
			expectedID:  "ABC123DEF456!789",
			description: "Should extract ID from legacy OneNote API URL",
		},
		{
			name:        "Microsoft Graph API URL",
			url:         "https://graph.microsoft.com/v1.0/users/me/onenote/resources/XYZ789ABC123!456/$value",
			expectedID:  "XYZ789ABC123!456",
			description: "Should extract ID from Microsoft Graph API URL",
		},
		{
			name:        "complex OneNote ID with multiple segments",
			url:         "https://graph.microsoft.com/v1.0/me/onenote/resources/0-896fbac8f72d01b02c5950345e65f588!1-4D24C77F19546939!39705/$value",
			expectedID:  "0-896fbac8f72d01b02c5950345e65f588!1-4D24C77F19546939!39705",
			description: "Should extract complex OneNote ID with multiple exclamation marks",
		},
		{
			name:        "URL without resources pattern",
			url:         "https://graph.microsoft.com/v1.0/me/onenote/pages/ABC123",
			expectedID:  "",
			description: "Should return empty string for URLs without resources pattern",
		},
		{
			name:        "empty URL",
			url:         "",
			expectedID:  "",
			description: "Should return empty string for empty URL",
		},
		{
			name:        "URL with resources but no $value",
			url:         "https://graph.microsoft.com/v1.0/me/onenote/resources/ABC123",
			expectedID:  "",
			description: "Should return empty string for URLs without $value suffix",
		},
		{
			name:        "URL with resources and $value but no ID",
			url:         "https://graph.microsoft.com/v1.0/me/onenote/resources//$value",
			expectedID:  "",
			description: "Should return empty string for URLs with empty ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPageItemID(tt.url)
			assert.Equal(t, tt.expectedID, result, tt.description)
		})
	}
}

func TestPageItemInfoStructure(t *testing.T) {
	t.Run("PageItemInfo has expected fields", func(t *testing.T) {
		item := PageItemInfo{
			TagName:     "img",
			PageItemID:  "test-id-123",
			Attributes:  map[string]string{"alt": "Test Image", "width": "100"},
			OriginalURL: "https://example.com/image.jpg",
		}

		assert.Equal(t, "img", item.TagName)
		assert.Equal(t, "test-id-123", item.PageItemID)
		assert.Equal(t, "https://example.com/image.jpg", item.OriginalURL)
		assert.Contains(t, item.Attributes, "alt")
		assert.Equal(t, "Test Image", item.Attributes["alt"])
	})
}

func TestPageItemDataStructure(t *testing.T) {
	t.Run("PageItemData has expected fields", func(t *testing.T) {
		content := []byte("fake image content")
		item := PageItemData{
			ContentType: "image/jpeg",
			Filename:    "test.jpg",
			Size:        int64(len(content)),
			Content:     content,
			TagName:     "img",
			Attributes:  map[string]string{"src": "test.jpg"},
			OriginalURL: "https://example.com/test.jpg",
		}

		assert.Equal(t, "image/jpeg", item.ContentType)
		assert.Equal(t, "test.jpg", item.Filename)
		assert.Equal(t, int64(len(content)), item.Size)
		assert.Equal(t, content, item.Content)
		assert.Equal(t, "img", item.TagName)
		assert.Contains(t, item.Attributes, "src")
		assert.Equal(t, "https://example.com/test.jpg", item.OriginalURL)
	})
}

func TestUpdateCommandStructure(t *testing.T) {
	t.Run("UpdateCommand has expected fields", func(t *testing.T) {
		cmd := UpdateCommand{
			Target:   "data-id:element-123",
			Action:   "replace",
			Position: "after",
			Content:  "<p>New content</p>",
		}

		assert.Equal(t, "data-id:element-123", cmd.Target)
		assert.Equal(t, "replace", cmd.Action)
		assert.Equal(t, "after", cmd.Position)
		assert.Equal(t, "<p>New content</p>", cmd.Content)
	})
}

// Test constants
func TestConstants(t *testing.T) {
	t.Run("HTML tag constants are correctly defined", func(t *testing.T) {
		assert.Equal(t, "img", imgTag)
		assert.Equal(t, "object", objectTag)

		// Ensure constants are different
		assert.NotEqual(t, imgTag, objectTag)

		// Ensure constants are not empty
		assert.NotEmpty(t, imgTag)
		assert.NotEmpty(t, objectTag)
	})
}

// Performance benchmarks
func BenchmarkHTMLEscape(b *testing.B) {
	testString := `<title>Tom & Jerry's "Adventures" with lots of special chars</title>`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = htmlEscape(testString)
	}
}

func BenchmarkExtractPageItemID(b *testing.B) {
	testURL := "https://graph.microsoft.com/v1.0/me/onenote/resources/0-896fbac8f72d01b02c5950345e65f588!1-4D24C77F19546939!39705/$value"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = extractPageItemID(testURL)
	}
}

func BenchmarkValidateTableUpdates(b *testing.B) {
	commands := []UpdateCommand{
		{Target: "body", Action: "replace", Content: "<p>Test</p>"},
		{Target: "title", Action: "replace", Content: "New Title"},
		{Target: "data-id:element-123", Action: "append", Content: "<p>Added</p>"},
		{Target: "td:data-id-456", Action: "replace", Content: "<td>Cell</td>"}, // This will cause validation error
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validateTableUpdates(commands)
	}
}
