// validation.go - Validation utilities for OneNote operations.
//
// This file contains validation functions for OneNote display names and other
// input validation utilities used across the application.
//
// Key Features:
// - Display name validation for OneNote sections, section groups, and pages
// - Illegal character detection and replacement suggestions
// - User-friendly error messages with alternatives
//
// Usage Example:
//   err := validateDisplayName("My Section")
//   if err != nil {
//       // Handle validation error
//   }
//
//   suggestedName := suggestValidName("My Section?", "?")
//   // Returns "My Section."

package utils

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

// ValidateDisplayName validates that a display name doesn't contain illegal characters.
// displayName: The display name to validate.
// Returns an error if the name contains illegal characters.
func ValidateDisplayName(displayName string) error {
	illegalChars := []string{"?", "*", "\\", "/", ":", "<", ">", "|", "&", "#", "'", "'", "%", "~"}
	for _, char := range illegalChars {
		if strings.Contains(displayName, char) {
			suggestedName := SuggestValidName(displayName, char)
			return fmt.Errorf("display name contains illegal character '%s'. Illegal characters are: ?*\\/:<>|&#''%%~\n\nSuggestion: Try using '%s' instead of '%s'.\n\nSuggested valid name: '%s'", char, GetReplacementChar(char), char, suggestedName)
		}
	}
	return nil
}

// SuggestValidName suggests a valid name by replacing illegal characters with appropriate alternatives
func SuggestValidName(name, illegalChar string) string {
	replacements := map[string]string{
		"?":  ".",
		"*":  ".",
		"\\": "-",
		"/":  "-",
		":":  "-",
		"<":  "(",
		">":  ")",
		"|":  "-",
		"&":  "and",
		"#":  "number",
		"%":  "percent",
		"~":  "-",
	}

	if replacement, exists := replacements[illegalChar]; exists {
		return strings.ReplaceAll(name, illegalChar, replacement)
	}
	return name
}

// GetReplacementChar returns a suggested replacement character for the given illegal character
func GetReplacementChar(illegalChar string) string {
	replacements := map[string]string{
		"?":  ".",
		"*":  ".",
		"\\": "-",
		"/":  "-",
		":":  "-",
		"<":  "(",
		">":  ")",
		"|":  "-",
		"&":  "and",
		"#":  "number",
		"%":  "percent",
		"~":  "-",
	}

	if replacement, exists := replacements[illegalChar]; exists {
		return replacement
	}
	return "-"
}

// TextFormat represents the format of text content
type TextFormat int

const (
	FormatASCII TextFormat = iota
	FormatMarkdown
	FormatHTML
)

// String returns the string representation of TextFormat
func (f TextFormat) String() string {
	switch f {
	case FormatASCII:
		return "ASCII"
	case FormatMarkdown:
		return "Markdown"
	case FormatHTML:
		return "HTML"
	default:
		return "Unknown"
	}
}

// DetectTextFormat analyzes text content to determine if it's HTML, Markdown, or plain ASCII
func DetectTextFormat(content string) TextFormat {
	content = strings.TrimSpace(content)
	if content == "" {
		return FormatASCII
	}

	// Check for HTML tags first
	if hasHTMLTags(content) {
		return FormatHTML
	}

	// Check for Markdown syntax
	if hasMarkdownSyntax(content) {
		return FormatMarkdown
	}

	// Default to ASCII if no specific format detected
	return FormatASCII
}

// hasHTMLTags checks if the content contains HTML tags
func hasHTMLTags(content string) bool {
	// Look for HTML tag patterns like <tag>, </tag>, <tag attr="value">
	htmlTagRegex := regexp.MustCompile(`<\/?[a-zA-Z][^>]*>`)
	return htmlTagRegex.MatchString(content)
}

// hasMarkdownSyntax checks if the content contains Markdown syntax at the beginning of lines
func hasMarkdownSyntax(content string) bool {
	lines := strings.Split(content, "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		// Check for common Markdown patterns at the beginning of lines
		if hasMarkdownLinePattern(line) {
			return true
		}
	}
	
	return false
}

// hasMarkdownLinePattern checks if a line starts with Markdown syntax
func hasMarkdownLinePattern(line string) bool {
	// Headers: # ## ### #### ##### ######
	if regexp.MustCompile(`^#{1,6}\s`).MatchString(line) {
		return true
	}
	
	// Unordered lists: - * +
	if regexp.MustCompile(`^[-*+]\s`).MatchString(line) {
		return true
	}
	
	// Ordered lists: 1. 2. 3. etc.
	if regexp.MustCompile(`^\d+\.\s`).MatchString(line) {
		return true
	}
	
	// Blockquotes: >
	if regexp.MustCompile(`^>\s`).MatchString(line) {
		return true
	}
	
	// Code blocks: ``` or ~~~
	if regexp.MustCompile(`^` + "```" + `|^~~~`).MatchString(line) {
		return true
	}
	
	// Horizontal rules: --- *** ___
	if regexp.MustCompile(`^(---+|\*\*\*+|___+)$`).MatchString(line) {
		return true
	}
	
	// Tables: | column | column |
	if regexp.MustCompile(`^\|.*\|`).MatchString(line) {
		return true
	}
	
	return false
}

// ConvertToHTML converts text content to HTML based on its detected format
func ConvertToHTML(content string) (string, TextFormat) {
	format := DetectTextFormat(content)
	
	switch format {
	case FormatHTML:
		// Already HTML, return as-is
		return content, format
	case FormatMarkdown:
		// Convert Markdown to HTML
		return convertMarkdownToHTML(content), format
	case FormatASCII:
		// Wrap plain text in paragraph tags, preserving line breaks
		return convertASCIIToHTML(content), format
	default:
		// Fallback to ASCII conversion
		return convertASCIIToHTML(content), FormatASCII
	}
}

// convertMarkdownToHTML converts Markdown text to HTML using gomarkdown
func convertMarkdownToHTML(markdownText string) string {
	// Set up parser with common extensions
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	
	// Set up HTML renderer with options suitable for OneNote
	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)
	
	// Convert markdown to HTML
	htmlBytes := markdown.ToHTML([]byte(markdownText), p, renderer)
	
	return string(htmlBytes)
}

// convertASCIIToHTML converts plain ASCII text to HTML, preserving line breaks
func convertASCIIToHTML(text string) string {
	// Escape HTML special characters
	text = strings.ReplaceAll(text, "&", "&amp;")
	text = strings.ReplaceAll(text, "<", "&lt;")
	text = strings.ReplaceAll(text, ">", "&gt;")
	text = strings.ReplaceAll(text, "\"", "&quot;")
	text = strings.ReplaceAll(text, "'", "&#39;")
	
	// Convert line breaks to <br> tags and wrap in paragraph
	lines := strings.Split(text, "\n")
	var processedLines []string
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			processedLines = append(processedLines, line)
		}
	}
	
	// Join lines with <br> tags if multiple lines, otherwise just wrap in <p>
	if len(processedLines) == 0 {
		return "<p></p>"
	} else if len(processedLines) == 1 {
		return "<p>" + processedLines[0] + "</p>"
	} else {
		return "<p>" + strings.Join(processedLines, "<br>") + "</p>"
	}
}
