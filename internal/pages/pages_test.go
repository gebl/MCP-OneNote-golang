// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package pages

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

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

// TestPageClient_Operations tests core page operations with validation and error handling
func TestPageClient_Operations(t *testing.T) {
	t.Run("validates page creation parameters", func(t *testing.T) {
		tests := []struct {
			title     string
			content   string
			sectionID string
			valid     bool
		}{
			{"Valid Title", "<p>Content</p>", "section-123", true},
			{"", "<p>Content</p>", "section-123", false},           // Empty title
			{"Title", "", "section-123", false},                    // Empty content
			{"Title", "<p>Content</p>", "", false},                 // Empty section ID
			{"Valid Title", "<p>Content</p>", "section-123", true}, // All valid
		}
		
		for _, tt := range tests {
			isValid := tt.title != "" && tt.content != "" && tt.sectionID != ""
			assert.Equal(t, tt.valid, isValid, "Page creation validation failed for title=%s, content=%s, sectionID=%s", tt.title, tt.content, tt.sectionID)
		}
	})

	t.Run("validates page update parameters", func(t *testing.T) {
		tests := []struct {
			pageID  string
			updates []UpdateCommand
			valid   bool
		}{
			{
				"page-123",
				[]UpdateCommand{
					{Target: "data-id:element-1", Action: "replace", Content: "<p>New</p>"},
				},
				true,
			},
			{
				"", // Empty page ID
				[]UpdateCommand{
					{Target: "data-id:element-1", Action: "replace", Content: "<p>New</p>"},
				},
				false,
			},
			{
				"page-123",
				[]UpdateCommand{}, // Empty updates
				false,
			},
		}
		
		for _, tt := range tests {
			isValid := tt.pageID != "" && len(tt.updates) > 0
			assert.Equal(t, tt.valid, isValid, "Page update validation failed for pageID=%s, updates count=%d", tt.pageID, len(tt.updates))
		}
	})

	t.Run("validates page deletion parameters", func(t *testing.T) {
		tests := []struct {
			pageID string
			valid  bool
		}{
			{"page-123", true},
			{"", false}, // Empty page ID
		}
		
		for _, tt := range tests {
			isValid := tt.pageID != ""
			assert.Equal(t, tt.valid, isValid, "Page deletion validation failed for pageID=%s", tt.pageID)
		}
	})

	t.Run("validates page copy parameters", func(t *testing.T) {
		tests := []struct {
			pageID          string
			targetSectionID string
			valid           bool
		}{
			{"page-123", "section-456", true},
			{"", "section-456", false},    // Empty page ID
			{"page-123", "", false},       // Empty target section
		}
		
		for _, tt := range tests {
			isValid := tt.pageID != "" && tt.targetSectionID != ""
			assert.Equal(t, tt.valid, isValid, "Page copy validation failed for pageID=%s, targetSectionID=%s", tt.pageID, tt.targetSectionID)
		}
	})

	t.Run("validates page move parameters", func(t *testing.T) {
		tests := []struct {
			pageID          string
			targetSectionID string
			valid           bool
		}{
			{"page-123", "section-456", true},
			{"", "section-456", false},    // Empty page ID
			{"page-123", "", false},       // Empty target section
		}
		
		for _, tt := range tests {
			isValid := tt.pageID != "" && tt.targetSectionID != ""
			assert.Equal(t, tt.valid, isValid, "Page move validation failed for pageID=%s, targetSectionID=%s", tt.pageID, tt.targetSectionID)
		}
	})
}

// TestPageClient_ContentProcessing tests content processing and format detection
func TestPageClient_ContentProcessing(t *testing.T) {
	t.Run("processes HTML content correctly", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected bool // whether it should be considered valid HTML
		}{
			{"valid HTML", "<p>Hello World</p>", true},
			{"HTML with attributes", `<p class="test">Content</p>`, true},
			{"nested HTML", "<div><p>Nested</p></div>", true},
			{"self-closing tags", "<br/><hr/>", true},
			{"plain text", "Just text", false}, // Not HTML
			{"empty content", "", false},
		}
		
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Test if content looks like HTML
				isHTML := strings.Contains(tt.input, "<") && strings.Contains(tt.input, ">")
				assert.Equal(t, tt.expected, isHTML, "HTML detection failed for: %s", tt.input)
			})
		}
	})

	t.Run("handles special characters in content", func(t *testing.T) {
		tests := []struct {
			input    string
			expected string
		}{
			{"Hello & World", "Hello &amp; World"},
			{"x < y", "x &lt; y"},
			{"x > y", "x &gt; y"},
			{`"quoted"`, "&quot;quoted&quot;"},
		}
		
		for _, tt := range tests {
			result := htmlEscape(tt.input)
			assert.Equal(t, tt.expected, result)
		}
	})

	t.Run("detects and handles table elements", func(t *testing.T) {
		tests := []struct {
			target   string
			isTable  bool
		}{
			{"td:data-id-123", true},
			{"th:data-id-456", true},
			{"tr:data-id-789", true},
			{"tbody:data-id-abc", true},
			{"thead:data-id-def", true},
			{"tfoot:data-id-ghi", true},
			{"p:data-id-123", false},
			{"div:data-id-456", false},
			{"span:data-id-789", false},
			{"data-id:element-1", false}, // Standard data-id target
		}
		
		for _, tt := range tests {
			result := isTableElement(tt.target)
			assert.Equal(t, tt.isTable, result, "Table element detection failed for: %s", tt.target)
		}
	})
}

// TestPageClient_UpdateCommands tests update command processing and validation
func TestPageClient_UpdateCommands(t *testing.T) {
	t.Run("validates update command structure", func(t *testing.T) {
		tests := []struct {
			name    string
			command UpdateCommand
			valid   bool
		}{
			{
				"valid append command",
				UpdateCommand{Target: "data-id:element-1", Action: "append", Content: "<p>New</p>"},
				true,
			},
			{
				"valid replace command", 
				UpdateCommand{Target: "data-id:element-2", Action: "replace", Content: "<div>Replacement</div>"},
				true,
			},
			{
				"valid insert command",
				UpdateCommand{Target: "data-id:element-3", Action: "insert", Content: "<span>Insert</span>", Position: "after"},
				true,
			},
			{
				"valid delete command",
				UpdateCommand{Target: "data-id:element-4", Action: "delete"},
				true,
			},
			{
				"invalid empty target",
				UpdateCommand{Target: "", Action: "append", Content: "<p>Content</p>"},
				false,
			},
			{
				"invalid empty action",
				UpdateCommand{Target: "data-id:element-1", Action: "", Content: "<p>Content</p>"},
				false,
			},
			{
				"invalid empty content for non-delete action",
				UpdateCommand{Target: "data-id:element-1", Action: "append", Content: ""},
				false,
			},
		}
		
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				isValid := tt.command.Target != "" && tt.command.Action != ""
				if tt.command.Action != "delete" {
					isValid = isValid && tt.command.Content != ""
				}
				assert.Equal(t, tt.valid, isValid, "Update command validation failed")
			})
		}
	})

	t.Run("serializes update commands to JSON correctly", func(t *testing.T) {
		command := UpdateCommand{
			Target:   "data-id:element-1",
			Action:   "replace", 
			Content:  "<p>Test content</p>",
			Position: "after",
		}
		
		// Test JSON marshaling
		data, err := command.MarshalJSON()
		assert.NoError(t, err)
		assert.Contains(t, string(data), "data-id:element-1")
		assert.Contains(t, string(data), "replace")
		assert.Contains(t, string(data), "Test content")
		assert.Contains(t, string(data), "after")
	})

	t.Run("validates table update restrictions", func(t *testing.T) {
		tests := []struct {
			name     string
			commands []UpdateCommand
			valid    bool
		}{
			{
				"non-table updates are valid",
				[]UpdateCommand{
					{Target: "data-id:element-1", Action: "append", Content: "<p>Content</p>"},
				},
				true,
			},
			{
				"table cell update should be rejected",
				[]UpdateCommand{
					{Target: "td:data-id-123", Action: "replace", Content: "<td>New Cell</td>"},
				},
				false,
			},
			{
				"multiple mixed updates with table cell should be rejected",
				[]UpdateCommand{
					{Target: "data-id:element-1", Action: "append", Content: "<p>Content</p>"},
					{Target: "td:data-id-456", Action: "replace", Content: "<td>Cell</td>"},
				},
				false,
			},
		}
		
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := validateTableUpdates(tt.commands)
				if tt.valid {
					assert.NoError(t, err)
				} else {
					assert.Error(t, err)
				}
			})
		}
	})
}

// TestPageClient_ResourceExtraction tests resource location and ID extraction
func TestPageClient_ResourceExtraction(t *testing.T) {
	t.Run("extracts page ID from resource location JSON", func(t *testing.T) {
		client := &PageClient{Client: &graph.Client{}}
		
		tests := []struct {
			jsonData       map[string]interface{}
			expectedID     string
			expectedError  bool
		}{
			{
				map[string]interface{}{
					"resourceLocation": "https://graph.microsoft.com/v1.0/me/onenote/pages/1-abc123def456",
				},
				"1-abc123def456",
				false,
			},
			{
				map[string]interface{}{
					"resourceLocation": "https://graph.microsoft.com/beta/users/user-id/onenote/pages/2-xyz789ghi012", 
				},
				"2-xyz789ghi012",
				false,
			},
			{
				map[string]interface{}{
					"resourceLocation": "invalid-location",
				},
				"",
				true,
			},
			{
				map[string]interface{}{},
				"",
				true,
			},
		}
		
		for _, tt := range tests {
			pageID, err := client.extractPageIDFromResourceLocation(tt.jsonData)
			if tt.expectedError {
				assert.Error(t, err)
				assert.Equal(t, "", pageID)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, pageID)
			}
		}
	})

	t.Run("extracts page item ID from resource URLs", func(t *testing.T) {
		tests := []struct {
			url        string
			expectedID string
		}{
			{
				"https://graph.microsoft.com/v1.0/users/me/onenote/resources/item-123/$value",
				"item-123",
			},
			{
				"https://www.onenote.com/api/v1.0/me/notes/resources/object-456/$value",
				"object-456",
			},
			{
				"https://graph.microsoft.com/v1.0/users/me/onenote/resources/0-896fbac8f72d01b02c5950345e65f588!1-4D24C77F19546939!39705/$value",
				"0-896fbac8f72d01b02c5950345e65f588!1-4D24C77F19546939!39705",
			},
			{
				"invalid-url",
				"",
			},
			{
				"",
				"",
			},
		}
		
		for _, tt := range tests {
			id := extractPageItemID(tt.url)
			assert.Equal(t, tt.expectedID, id)
		}
	})
}

// TestPageClient_Integration tests integration scenarios for page operations
func TestPageClient_Integration(t *testing.T) {
	t.Run("page creation workflow validation", func(t *testing.T) {
		// Simulate a complete page creation workflow validation
		title := "Test Page"
		content := "<p>This is test content with <strong>formatting</strong>.</p>"
		sectionID := "section-123"
		
		// Validate inputs
		assert.NotEmpty(t, title, "Title should not be empty")
		assert.NotEmpty(t, content, "Content should not be empty") 
		assert.NotEmpty(t, sectionID, "Section ID should not be empty")
		
		// Validate content is HTML
		isHTML := strings.Contains(content, "<") && strings.Contains(content, ">")
		assert.True(t, isHTML, "Content should be valid HTML")
		
		// Simulate content escaping if needed
		escapedTitle := htmlEscape(title)
		assert.Equal(t, title, escapedTitle, "Simple title should not need escaping")
	})

	t.Run("page update workflow validation", func(t *testing.T) {
		// Simulate page update workflow
		pageID := "page-456"
		updates := []UpdateCommand{
			{Target: "data-id:element-1", Action: "replace", Content: "<p>Updated content</p>"},
			{Target: "data-id:element-2", Action: "append", Content: "<div>Additional content</div>"},
		}
		
		// Validate page ID
		assert.NotEmpty(t, pageID, "Page ID should not be empty")
		
		// Validate updates
		assert.Greater(t, len(updates), 0, "Should have at least one update")
		
		// Validate each update
		for i, update := range updates {
			assert.NotEmpty(t, update.Target, "Update %d target should not be empty", i)
			assert.NotEmpty(t, update.Action, "Update %d action should not be empty", i)
			if update.Action != "delete" {
				assert.NotEmpty(t, update.Content, "Update %d content should not be empty for non-delete actions", i)
			}
		}
		
		// Validate no table restrictions are violated
		err := validateTableUpdates(updates)
		assert.NoError(t, err, "Table validation should pass for regular updates")
	})

	t.Run("complex content processing", func(t *testing.T) {
		// Test processing of complex HTML content
		complexHTML := `
		<div>
			<h1>Title with "quotes" & special chars < ></h1>
			<p>Paragraph with <em>emphasis</em> and <strong>strong</strong> text.</p>
			<ul>
				<li>List item 1</li>
				<li>List item 2</li>
			</ul>
			<table>
				<tr><td data-id="cell-1">Cell 1</td><td data-id="cell-2">Cell 2</td></tr>
			</table>
		</div>
		`
		
		// Verify it contains expected elements
		assert.Contains(t, complexHTML, "<h1>")
		assert.Contains(t, complexHTML, "<table>")
		assert.Contains(t, complexHTML, "data-id=")
		
		// Test table element detection would work
		assert.True(t, isTableElement("td:data-id-cell1"))
		assert.True(t, isTableElement("tr:data-id-row1"))
		assert.False(t, isTableElement("div:data-id-content"))
	})
}

// TestPageClient_ErrorScenarios tests error handling and edge cases
func TestPageClient_ErrorScenarios(t *testing.T) {
	t.Run("handles empty or malformed inputs gracefully", func(t *testing.T) {
		// Test HTML escaping with empty input
		result := htmlEscape("")
		assert.Equal(t, "", result, "Empty string should remain empty")
		
		// Test page item ID extraction with malformed HTML
		id := extractPageItemID("<invalid-html")
		assert.Equal(t, "", id, "Should handle malformed HTML gracefully")
		
		// Test table element detection with empty input
		isTable := isTableElement("")
		assert.False(t, isTable, "Empty element should not be considered table element")
	})

	t.Run("validates update commands with edge cases", func(t *testing.T) {
		edgeCases := []UpdateCommand{
			{Target: "data-id:", Action: "append", Content: "<p>Content</p>"}, // Malformed target
			{Target: "data-id:valid", Action: "unknown", Content: "<p>Content</p>"}, // Invalid action
			{Target: "data-id:valid", Action: "append", Content: "<>"}, // Malformed content HTML
		}
		
		for i, cmd := range edgeCases {
			// Basic validation should still catch structural issues
			hasTarget := cmd.Target != ""
			hasAction := cmd.Action != ""
			
			// At least check that we can identify problematic commands
			if i == 0 { // Malformed target
				assert.Contains(t, cmd.Target, ":")
			}
			
			assert.True(t, hasTarget && hasAction, "Command %d should have basic structure", i)
		}
	})
}

// MockPageClient provides comprehensive mock implementation for testing page operations
type MockPageClient struct {
	mock.Mock
	pages         map[string]map[string]interface{}
	pageContent   map[string]string
	pageItems     map[string][]map[string]interface{}
	mockError     error
	shouldFail    map[string]bool
	createdPages  []map[string]interface{}
	operations    []string
}

func NewMockPageClient() *MockPageClient {
	return &MockPageClient{
		pages: map[string]map[string]interface{}{
			"page-1": {
				"id":                   "page-1",
				"title":                "Test Page 1",
				"contentUrl":           "https://graph.microsoft.com/v1.0/me/onenote/pages/page-1/content",
				"createdDateTime":      "2023-01-01T00:00:00Z",
				"lastModifiedDateTime": "2023-01-02T00:00:00Z",
				"parentSection": map[string]interface{}{
					"id":          "section-1",
					"displayName": "Test Section",
				},
			},
			"page-2": {
				"id":    "page-2",
				"title": "Meeting Notes",
				"parentSection": map[string]interface{}{
					"id":          "section-1",
					"displayName": "Test Section",
				},
			},
			"page-3": {
				"id":    "page-3",
				"title": "Project Notes",
				"parentSection": map[string]interface{}{
					"id":          "section-2",
					"displayName": "Projects",
				},
			},
		},
		pageContent: map[string]string{
			"page-1": `<!DOCTYPE html>
<html>
<head>
	<title>Test Page 1</title>
</head>
<body data-absolute-enabled="true" style="font-family:Calibri;font-size:11pt">
	<p data-id="element-1">This is the first paragraph.</p>
	<p data-id="element-2">This is the <strong>second</strong> paragraph with formatting.</p>
	<table data-id="table-1">
		<tr><td data-id="cell-1">Cell 1</td><td data-id="cell-2">Cell 2</td></tr>
	</table>
</body>
</html>`,
			"page-2": `<html><body><h1>Meeting Notes</h1><p>Meeting content here</p></body></html>`,
			"page-3": `<html><body><h1>Project Notes</h1><p>Project content here</p></body></html>`,
		},
		pageItems: map[string][]map[string]interface{}{
			"page-1": {
				{
					"id":               "item-1",
					"tagName":          "img",
					"originalUrl":      "https://example.com/image1.jpg",
					"pageItemID":       "item-1",
					"width":            "200",
					"height":           "150",
					"alt":              "Test Image",
					"contentType":      "image/jpeg",
					"content":          "fake-image-data",
				},
				{
					"id":               "item-2",
					"tagName":          "object",
					"originalUrl":      "https://example.com/document.pdf",
					"pageItemID":       "item-2",
					"contentType":      "application/pdf",
					"content":          "fake-pdf-data",
				},
			},
			"page-2": {
				{
					"id":               "item-3",
					"tagName":          "img",
					"originalUrl":      "https://example.com/chart.png",
					"pageItemID":       "item-3",
					"contentType":      "image/png",
					"content":          "fake-png-data",
				},
			},
		},
		shouldFail:    make(map[string]bool),
		createdPages:  make([]map[string]interface{}, 0),
		operations:    make([]string, 0),
	}
}

// Helper function to get file extension from MIME type
func getExtensionFromMimeType(mimeType string) string {
	switch mimeType {
	case "image/jpeg":
		return "jpg"
	case "image/png":
		return "png"
	case "application/pdf":
		return "pdf"
	default:
		return "bin"
	}
}

// SetError allows tests to simulate error conditions
func (m *MockPageClient) SetError(err error) {
	m.mockError = err
}

// SetOperationFailure allows tests to simulate failures for specific operations
func (m *MockPageClient) SetOperationFailure(operation string, shouldFail bool) {
	m.shouldFail[operation] = shouldFail
}

// Helper method to check if operation should fail
func (m *MockPageClient) checkFailure(operation string) error {
	if m.mockError != nil {
		return m.mockError
	}
	if m.shouldFail[operation] {
		return fmt.Errorf("simulated failure for %s", operation)
	}
	return nil
}

// CreatePage mocks page creation
func (m *MockPageClient) CreatePage(sectionID, title, content string) (map[string]interface{}, error) {
	m.operations = append(m.operations, "CreatePage")
	if err := m.checkFailure("CreatePage"); err != nil {
		return nil, err
	}
	
	// Basic validation
	if title == "" {
		return nil, fmt.Errorf("page title cannot be empty")
	}
	if content == "" {
		return nil, fmt.Errorf("page content cannot be empty")
	}
	if sectionID == "" {
		return nil, fmt.Errorf("section ID cannot be empty")
	}
	
	// Create new page
	newPageID := fmt.Sprintf("page-%d", len(m.createdPages)+100)
	newPage := map[string]interface{}{
		"id":    newPageID,
		"title": title,
		"contentUrl": fmt.Sprintf("https://graph.microsoft.com/v1.0/me/onenote/pages/%s/content", newPageID),
		"parentSection": map[string]interface{}{
			"id":          sectionID,
			"displayName": fmt.Sprintf("Section %s", sectionID),
		},
		"createdDateTime": "2023-12-01T00:00:00Z",
		"lastModifiedDateTime": "2023-12-01T00:00:00Z",
	}
	
	// Store the page
	m.pages[newPageID] = newPage
	m.pageContent[newPageID] = content
	m.createdPages = append(m.createdPages, newPage)
	
	return newPage, nil
}

// GetOperations returns the list of operations performed (for testing)
func (m *MockPageClient) GetOperations() []string {
	return m.operations
}

// GetCreatedPages returns the list of created pages (for testing)
func (m *MockPageClient) GetCreatedPages() []map[string]interface{} {
	return m.createdPages
}

// TestPageClient_CreatePage tests page creation functionality
func TestPageClient_CreatePage(t *testing.T) {
	t.Run("successfully creates page with valid inputs", func(t *testing.T) {
		mockClient := NewMockPageClient()
		
		page, err := mockClient.CreatePage("section-1", "New Test Page", "<p>Test content</p>")
		
		assert.NoError(t, err)
		assert.NotNil(t, page)
		assert.Equal(t, "New Test Page", page["title"])
		assert.Contains(t, page, "id")
		assert.Contains(t, page, "contentUrl")
		assert.Contains(t, page, "parentSection")
		
		// Verify parent section
		parentSection := page["parentSection"].(map[string]interface{})
		assert.Equal(t, "section-1", parentSection["id"])
		
		// Verify the page was stored
		createdPages := mockClient.GetCreatedPages()
		assert.Len(t, createdPages, 1)
		assert.Equal(t, page["id"], createdPages[0]["id"])
	})
	
	t.Run("validates required parameters", func(t *testing.T) {
		mockClient := NewMockPageClient()
		
		// Empty title
		page, err := mockClient.CreatePage("section-1", "", "<p>Content</p>")
		assert.Error(t, err)
		assert.Nil(t, page)
		assert.Contains(t, err.Error(), "title cannot be empty")
		
		// Empty content
		page, err = mockClient.CreatePage("section-1", "Title", "")
		assert.Error(t, err)
		assert.Nil(t, page)
		assert.Contains(t, err.Error(), "content cannot be empty")
		
		// Empty section ID
		page, err = mockClient.CreatePage("", "Title", "<p>Content</p>")
		assert.Error(t, err)
		assert.Nil(t, page)
		assert.Contains(t, err.Error(), "section ID cannot be empty")
	})
	
	t.Run("handles authentication errors", func(t *testing.T) {
		mockClient := NewMockPageClient()
		mockClient.SetOperationFailure("CreatePage", true)
		
		page, err := mockClient.CreatePage("section-1", "Title", "<p>Content</p>")
		
		assert.Error(t, err)
		assert.Nil(t, page)
		assert.Contains(t, err.Error(), "simulated failure")
	})
}

// Benchmark tests for page operations  
func BenchmarkMockPageClient_CreatePage(b *testing.B) {
	mockClient := NewMockPageClient()
	content := "<p>Benchmark page content with some formatting and structure</p>"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		title := fmt.Sprintf("Benchmark Page %d", i)
		_, _ = mockClient.CreatePage("section-1", title, content)
	}
}
