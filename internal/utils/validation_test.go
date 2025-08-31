// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package utils

import (
	"strings"
	"testing"
)

func TestDetectTextFormat(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected TextFormat
	}{
		{
			name:     "HTML content",
			content:  "<p>This is <strong>HTML</strong> content</p>",
			expected: FormatHTML,
		},
		{
			name:     "Markdown heading",
			content:  "# This is a heading",
			expected: FormatMarkdown,
		},
		{
			name:     "Markdown list",
			content:  "- Item 1\n- Item 2",
			expected: FormatMarkdown,
		},
		{
			name:     "Markdown emphasis",
			content:  "This is **bold** text",
			expected: FormatMarkdown,
		},
		{
			name:     "Markdown code block",
			content:  "```go\nfunc main() {}\n```",
			expected: FormatMarkdown,
		},
		{
			name:     "Plain ASCII text",
			content:  "This is just plain text",
			expected: FormatASCII,
		},
		{
			name:     "Empty content",
			content:  "",
			expected: FormatASCII,
		},
		{
			name:     "Whitespace only",
			content:  "   \n\t  \n  ",
			expected: FormatASCII,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectTextFormat(tt.content)
			if result != tt.expected {
				t.Errorf("DetectTextFormat(%q) = %v, want %v", tt.content, result, tt.expected)
			}
		})
	}
}

func TestConvertToHTML(t *testing.T) {
	tests := []struct {
		name            string
		content         string
		expectedFormat  TextFormat
		shouldContain   []string
		shouldNotContain []string
	}{
		{
			name:           "Markdown heading conversion",
			content:        "# Main Title\n\nThis is content.",
			expectedFormat: FormatMarkdown,
			shouldContain:  []string{"<h1", "Main Title", "<p>", "This is content"},
		},
		{
			name:           "Markdown list conversion",
			content:        "- First item\n- Second item",
			expectedFormat: FormatMarkdown,
			shouldContain:  []string{"<ul>", "<li>", "First item", "Second item"},
		},
		{
			name:           "Markdown emphasis conversion",
			content:        "This is **bold** and *italic* text",
			expectedFormat: FormatMarkdown,
			shouldContain:  []string{"<strong>", "bold", "<em>", "italic"},
		},
		{
			name:           "HTML passthrough",
			content:        "<p>This is <strong>HTML</strong></p>",
			expectedFormat: FormatHTML,
			shouldContain:  []string{"<p>", "<strong>", "HTML"},
		},
		{
			name:           "ASCII text conversion",
			content:        "This is plain text\nwith line breaks",
			expectedFormat: FormatASCII,
			shouldContain:  []string{"<p>", "This is plain text", "<br>", "with line breaks"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, format := ConvertToHTML(tt.content)
			
			if format != tt.expectedFormat {
				t.Errorf("ConvertToHTML(%q) format = %v, want %v", tt.content, format, tt.expectedFormat)
			}
			
			for _, expected := range tt.shouldContain {
				if !strings.Contains(result, expected) {
					t.Errorf("ConvertToHTML(%q) result should contain %q, got: %s", tt.content, expected, result)
				}
			}
			
			for _, notExpected := range tt.shouldNotContain {
				if strings.Contains(result, notExpected) {
					t.Errorf("ConvertToHTML(%q) result should not contain %q, got: %s", tt.content, notExpected, result)
				}
			}
		})
	}
}

func TestMarkdownAdvancedFeatures(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		shouldContain []string
	}{
		{
			name:          "Table conversion",
			content:       "| Header 1 | Header 2 |\n|----------|----------|\n| Cell 1   | Cell 2   |",
			shouldContain: []string{"<table>", "<thead>", "<th>", "Header 1", "Header 2", "<tbody>", "<td>", "Cell 1", "Cell 2"},
		},
		{
			name:          "Strikethrough",
			content:       "~~strikethrough text~~",
			shouldContain: []string{"<del>", "strikethrough text"},
		},
		{
			name:          "Task list",
			content:       "- [x] Completed task\n- [ ] Incomplete task",
			shouldContain: []string{"<input", "type=\"checkbox\"", "checked", "Completed task", "Incomplete task"},
		},
		{
			name:          "Fenced code block",
			content:       "```go\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n```",
			shouldContain: []string{"<pre>", "<code", "func main", "fmt.Println"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, format := ConvertToHTML(tt.content)
			
			if format != FormatMarkdown {
				t.Errorf("ConvertToHTML(%q) should detect as Markdown, got %v", tt.content, format)
			}
			
			for _, expected := range tt.shouldContain {
				if !strings.Contains(result, expected) {
					t.Errorf("ConvertToHTML(%q) result should contain %q, got: %s", tt.content, expected, result)
				}
			}
		})
	}
}

func TestValidateDisplayName(t *testing.T) {
	tests := []struct {
		name        string
		displayName string
		expectError bool
		expectedChar string
	}{
		{
			name:        "Valid display name",
			displayName: "My Valid Section",
			expectError: false,
		},
		{
			name:        "Display name with question mark",
			displayName: "What is this?",
			expectError: true,
			expectedChar: "?",
		},
		{
			name:        "Display name with asterisk",
			displayName: "Section*Notes",
			expectError: true,
			expectedChar: "*",
		},
		{
			name:        "Display name with backslash",
			displayName: "Section\\Notes",
			expectError: true,
			expectedChar: "\\",
		},
		{
			name:        "Empty display name",
			displayName: "",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDisplayName(tt.displayName)
			
			if tt.expectError && err == nil {
				t.Error("Expected an error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			
			if tt.expectError && err != nil {
				errorMsg := err.Error()
				if !strings.Contains(errorMsg, tt.expectedChar) {
					t.Errorf("Expected error message to contain '%s', got: %s", tt.expectedChar, errorMsg)
				}
				if !strings.Contains(errorMsg, "illegal character") {
					t.Error("Error message should mention 'illegal character'")
				}
			}
		})
	}
}

func TestSuggestValidName(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		illegalChar string
		expected    string
	}{
		{
			name:        "Replace question mark with period",
			input:       "What is this?",
			illegalChar: "?",
			expected:    "What is this.",
		},
		{
			name:        "Replace asterisk with period",
			input:       "Section*Notes",
			illegalChar: "*",
			expected:    "Section.Notes",
		},
		{
			name:        "Replace ampersand with 'and'",
			input:       "Section&Notes",
			illegalChar: "&",
			expected:    "SectionandNotes",
		},
		{
			name:        "Unknown illegal character - no change",
			input:       "Section@Notes",
			illegalChar: "@",
			expected:    "Section@Notes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SuggestValidName(tt.input, tt.illegalChar)
			if result != tt.expected {
				t.Errorf("SuggestValidName(%q, %q) = %q, want %q", tt.input, tt.illegalChar, result, tt.expected)
			}
		})
	}
}

func TestGetReplacementChar(t *testing.T) {
	tests := []struct {
		name        string
		illegalChar string
		expected    string
	}{
		{"Question mark", "?", "."},
		{"Asterisk", "*", "."},
		{"Ampersand", "&", "and"},
		{"Unknown character", "@", "-"}, // Default fallback
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetReplacementChar(tt.illegalChar)
			if result != tt.expected {
				t.Errorf("GetReplacementChar(%q) = %q, want %q", tt.illegalChar, result, tt.expected)
			}
		})
	}
}

func TestConvertHTMLToMarkdown(t *testing.T) {
	tests := []struct {
		name          string
		htmlContent   string
		shouldContain []string
		expectError   bool
	}{
		{
			name:          "Simple paragraph",
			htmlContent:   "<p>This is a paragraph.</p>",
			shouldContain: []string{"This is a paragraph."},
			expectError:   false,
		},
		{
			name:          "Heading conversion",
			htmlContent:   "<h1>Main Title</h1>",
			shouldContain: []string{"# Main Title"},
			expectError:   false,
		},
		{
			name:          "Empty HTML",
			htmlContent:   "",
			shouldContain: []string{},
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertHTMLToMarkdown(tt.htmlContent)
			
			if tt.expectError && err == nil {
				t.Error("Expected an error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			
			if !tt.expectError {
				for _, expected := range tt.shouldContain {
					if !strings.Contains(result, expected) {
						t.Errorf("Result should contain %q, got: %s", expected, result)
					}
				}
			}
		})
	}
}

func TestConvertHTMLToText(t *testing.T) {
	tests := []struct {
		name          string
		htmlContent   string
		shouldContain []string
		expectError   bool
	}{
		{
			name:          "Simple paragraph",
			htmlContent:   "<p>This is a paragraph.</p>",
			shouldContain: []string{"This is a paragraph."},
			expectError:   false,
		},
		{
			name:          "Multiple paragraphs",
			htmlContent:   "<p>First.</p><p>Second.</p>",
			shouldContain: []string{"First.", "Second."},
			expectError:   false,
		},
		{
			name:          "Empty HTML",
			htmlContent:   "",
			shouldContain: []string{},
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertHTMLToText(tt.htmlContent)
			
			if tt.expectError && err == nil {
				t.Error("Expected an error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			
			if !tt.expectError {
				for _, expected := range tt.shouldContain {
					if !strings.Contains(result, expected) {
						t.Errorf("Result should contain %q, got: %s", expected, result)
					}
				}
			}
		})
	}
}