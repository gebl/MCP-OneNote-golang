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
	md "github.com/JohannesKaufmann/html-to-markdown"
	htmlparser "golang.org/x/net/html"
	
	"github.com/gebl/onenote-mcp-server/internal/logging"
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
	originalLength := len(content)
	content = strings.TrimSpace(content)
	trimmedLength := len(content)
	
	logging.UtilsLogger.Debug("Starting text format detection",
		"original_length", originalLength,
		"trimmed_length", trimmedLength,
		"content_preview", truncateString(content, 100))
	
	if content == "" {
		logging.UtilsLogger.Debug("Content is empty, returning ASCII format")
		return FormatASCII
	}

	// Check for HTML tags first
	logging.UtilsLogger.Debug("Checking for HTML tags")
	if hasHTMLTags(content) {
		logging.UtilsLogger.Debug("HTML tags detected, returning HTML format")
		return FormatHTML
	}

	// Check for Markdown syntax
	logging.UtilsLogger.Debug("Checking for Markdown syntax")
	if hasMarkdownSyntax(content) {
		logging.UtilsLogger.Debug("Markdown syntax detected, returning Markdown format")
		return FormatMarkdown
	}

	// Default to ASCII if no specific format detected
	logging.UtilsLogger.Debug("No specific format detected, returning ASCII format")
	return FormatASCII
}

// hasHTMLTags checks if the content contains HTML tags
func hasHTMLTags(content string) bool {
	// Look for HTML tag patterns like <tag>, </tag>, <tag attr="value">
	htmlTagRegex := regexp.MustCompile(`<\/?[a-zA-Z][^>]*>`)
	matches := htmlTagRegex.FindAllString(content, -1)
	hasMatch := len(matches) > 0
	
	logging.UtilsLogger.Debug("HTML tag detection",
		"content_length", len(content),
		"regex_pattern", `<\/?[a-zA-Z][^>]*>`,
		"matches_found", len(matches),
		"has_html_tags", hasMatch,
		"detected_tags", matches)
	
	return hasMatch
}

// hasMarkdownSyntax checks if the content contains Markdown syntax at the beginning of lines
func hasMarkdownSyntax(content string) bool {
	lines := strings.Split(content, "\n")
	totalLines := len(lines)
	nonEmptyLines := 0
	markdownLines := []string{}
	
	logging.UtilsLogger.Debug("Starting Markdown syntax detection",
		"total_lines", totalLines,
		"content_length", len(content))
	
	for i, line := range lines {
		originalLine := line
		line = strings.TrimSpace(line)
		if line == "" {
			logging.UtilsLogger.Debug("Skipping empty line", "line_number", i+1)
			continue
		}
		
		nonEmptyLines++
		logging.UtilsLogger.Debug("Checking line for Markdown patterns",
			"line_number", i+1,
			"original_line", truncateString(originalLine, 50),
			"trimmed_line", truncateString(line, 50))
		
		// Check for common Markdown patterns at the beginning of lines
		if hasMarkdownLinePattern(line) {
			markdownLines = append(markdownLines, line)
			logging.UtilsLogger.Debug("Markdown pattern detected in line",
				"line_number", i+1,
				"line_content", truncateString(line, 50),
				"pattern_matched", true)
			logging.UtilsLogger.Debug("Markdown syntax confirmed, found patterns",
				"total_lines", totalLines,
				"non_empty_lines", nonEmptyLines,
				"markdown_lines_found", len(markdownLines),
				"detected_lines", markdownLines)
			return true
		}
	}
	
	logging.UtilsLogger.Debug("Markdown syntax detection completed",
		"total_lines", totalLines,
		"non_empty_lines", nonEmptyLines,
		"markdown_lines_found", len(markdownLines),
		"has_markdown_syntax", false)
	
	return false
}

// hasMarkdownLinePattern checks if a line starts with Markdown syntax
func hasMarkdownLinePattern(line string) bool {
	patterns := []struct {
		name    string
		regex   string
		pattern *regexp.Regexp
	}{
		{"headers", `^#{1,6}\s`, regexp.MustCompile(`^#{1,6}\s`)},
		{"unordered_lists", `^[-*+]\s`, regexp.MustCompile(`^[-*+]\s`)},
		{"ordered_lists", `^\d+\.\s`, regexp.MustCompile(`^\d+\.\s`)},
		{"blockquotes", `^>\s`, regexp.MustCompile(`^>\s`)},
		{"code_blocks", `^` + "```" + `|^~~~`, regexp.MustCompile(`^` + "```" + `|^~~~`)},
		{"horizontal_rules", `^(---+|\*\*\*+|___+)$`, regexp.MustCompile(`^(---+|\*\*\*+|___+)$`)},
		{"tables", `^\|.*\|`, regexp.MustCompile(`^\|.*\|`)},
	}
	
	logging.UtilsLogger.Debug("Checking line for Markdown patterns",
		"line_content", truncateString(line, 50),
		"patterns_to_check", len(patterns))
	
	for _, p := range patterns {
		if p.pattern.MatchString(line) {
			logging.UtilsLogger.Debug("Markdown pattern matched",
				"pattern_name", p.name,
				"pattern_regex", p.regex,
				"line_content", truncateString(line, 50),
				"matched", true)
			return true
		} else {
			logging.UtilsLogger.Debug("Markdown pattern not matched",
				"pattern_name", p.name,
				"pattern_regex", p.regex,
				"line_content", truncateString(line, 50),
				"matched", false)
		}
	}
	
	logging.UtilsLogger.Debug("No Markdown patterns matched for line",
		"line_content", truncateString(line, 50),
		"patterns_checked", len(patterns))
	
	return false
}

// ConvertToHTML converts text content to HTML based on its detected format
func ConvertToHTML(content string) (string, TextFormat) {
	originalLength := len(content)
	logging.UtilsLogger.Debug("Starting content conversion to HTML",
		"original_length", originalLength,
		"content_preview", truncateString(content, 100))
	
	format := DetectTextFormat(content)
	
	logging.UtilsLogger.Debug("Format detected, proceeding with conversion",
		"detected_format", format.String())
	
	var convertedHTML string
	
	switch format {
	case FormatHTML:
		// Already HTML, return as-is
		logging.UtilsLogger.Debug("Content is HTML, returning as-is")
		convertedHTML = content
	case FormatMarkdown:
		// Convert Markdown to HTML
		logging.UtilsLogger.Debug("Converting Markdown to HTML")
		convertedHTML = convertMarkdownToHTML(content)
	case FormatASCII:
		// Wrap plain text in paragraph tags, preserving line breaks
		logging.UtilsLogger.Debug("Converting ASCII to HTML")
		convertedHTML = convertASCIIToHTML(content)
	default:
		// Fallback to ASCII conversion
		logging.UtilsLogger.Debug("Unknown format, falling back to ASCII conversion")
		convertedHTML = convertASCIIToHTML(content)
		format = FormatASCII
	}
	
	convertedLength := len(convertedHTML)
	logging.UtilsLogger.Debug("Content conversion completed",
		"original_length", originalLength,
		"converted_length", convertedLength,
		"length_change", convertedLength-originalLength,
		"final_format", format.String(),
		"converted_preview", truncateString(convertedHTML, 100))
	
	return convertedHTML, format
}

// convertMarkdownToHTML converts Markdown text to HTML using gomarkdown
func convertMarkdownToHTML(markdownText string) string {
	logging.UtilsLogger.Debug("Starting Markdown to HTML conversion",
		"input_length", len(markdownText),
		"input_preview", truncateString(markdownText, 100))
	
	// Set up parser with common extensions
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	
	logging.UtilsLogger.Debug("Markdown parser configured",
		"extensions", "CommonExtensions|AutoHeadingIDs|NoEmptyLineBeforeBlock")
	
	// Set up HTML renderer with options suitable for OneNote
	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)
	
	logging.UtilsLogger.Debug("HTML renderer configured",
		"flags", "CommonFlags|HrefTargetBlank")
	
	// Convert markdown to HTML
	htmlBytes := markdown.ToHTML([]byte(markdownText), p, renderer)
	resultHTML := string(htmlBytes)
	
	logging.UtilsLogger.Debug("Markdown to HTML conversion completed",
		"input_length", len(markdownText),
		"output_length", len(resultHTML),
		"output_preview", truncateString(resultHTML, 100))
	
	return resultHTML
}

// convertASCIIToHTML converts plain ASCII text to HTML, preserving line breaks
func convertASCIIToHTML(text string) string {
	originalLength := len(text)
	logging.UtilsLogger.Debug("Starting ASCII to HTML conversion",
		"input_length", originalLength,
		"input_preview", truncateString(text, 100))
	
	// Escape HTML special characters
	escapedText := text
	replacements := map[string]string{
		"&":  "&amp;",
		"<":  "&lt;",
		">":  "&gt;",
		"\"": "&quot;",
		"'":  "&#39;",
	}
	
	for char, replacement := range replacements {
		beforeLength := len(escapedText)
		escapedText = strings.ReplaceAll(escapedText, char, replacement)
		afterLength := len(escapedText)
		if beforeLength != afterLength {
			logging.UtilsLogger.Debug("HTML character escaped",
				"character", char,
				"replacement", replacement,
				"before_length", beforeLength,
				"after_length", afterLength)
		}
	}
	
	// Convert line breaks to <br> tags and wrap in paragraph
	lines := strings.Split(escapedText, "\n")
	totalLines := len(lines)
	var processedLines []string
	
	logging.UtilsLogger.Debug("Processing lines for HTML conversion",
		"total_lines", totalLines)
	
	for i, line := range lines {
		originalLine := line
		line = strings.TrimSpace(line)
		if line != "" {
			processedLines = append(processedLines, line)
			logging.UtilsLogger.Debug("Line processed",
				"line_number", i+1,
				"original", truncateString(originalLine, 30),
				"trimmed", truncateString(line, 30),
				"kept", true)
		} else {
			logging.UtilsLogger.Debug("Line skipped (empty)",
				"line_number", i+1,
				"original", truncateString(originalLine, 30),
				"kept", false)
		}
	}
	
	var result string
	
	// Join lines with <br> tags if multiple lines, otherwise just wrap in <p>
	if len(processedLines) == 0 {
		result = "<p></p>"
		logging.UtilsLogger.Debug("No content lines, returning empty paragraph")
	} else if len(processedLines) == 1 {
		result = "<p>" + processedLines[0] + "</p>"
		logging.UtilsLogger.Debug("Single line, wrapping in paragraph")
	} else {
		result = "<p>" + strings.Join(processedLines, "<br>") + "</p>"
		logging.UtilsLogger.Debug("Multiple lines, joining with <br> tags",
			"lines_count", len(processedLines))
	}
	
	logging.UtilsLogger.Debug("ASCII to HTML conversion completed",
		"input_length", originalLength,
		"output_length", len(result),
		"processed_lines", len(processedLines),
		"output_preview", truncateString(result, 100))
	
	return result
}

// truncateString truncates a string to a maximum length for logging purposes
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// ConvertHTMLToMarkdown converts HTML content to Markdown format
func ConvertHTMLToMarkdown(htmlContent string) (string, error) {
	logging.UtilsLogger.Debug("Starting HTML to Markdown conversion",
		"input_length", len(htmlContent),
		"input_preview", truncateString(htmlContent, 100))

	converter := md.NewConverter("", true, nil)
	
	markdownText, err := converter.ConvertString(htmlContent)
	if err != nil {
		logging.UtilsLogger.Error("HTML to Markdown conversion failed", "error", err)
		return "", fmt.Errorf("failed to convert HTML to Markdown: %v", err)
	}

	// Clean up the markdown - remove excessive whitespace
	cleanMarkdown := strings.TrimSpace(markdownText)

	logging.UtilsLogger.Debug("HTML to Markdown conversion completed",
		"input_length", len(htmlContent),
		"output_length", len(cleanMarkdown),
		"output_preview", truncateString(cleanMarkdown, 100))

	return cleanMarkdown, nil
}

// ConvertHTMLToText converts HTML content to plain text format
func ConvertHTMLToText(htmlContent string) (string, error) {
	logging.UtilsLogger.Debug("Starting HTML to plain text conversion",
		"input_length", len(htmlContent),
		"input_preview", truncateString(htmlContent, 100))

	// Parse the HTML
	doc, err := htmlparser.Parse(strings.NewReader(htmlContent))
	if err != nil {
		logging.UtilsLogger.Error("HTML parsing failed", "error", err)
		return "", fmt.Errorf("failed to parse HTML: %v", err)
	}

	// Extract text from the HTML document
	var textBuilder strings.Builder
	extractText(doc, &textBuilder)
	
	plainText := textBuilder.String()

	// Clean up the text - remove excessive whitespace
	cleanText := strings.TrimSpace(plainText)
	// Replace multiple consecutive spaces with single space
	re := regexp.MustCompile(`\s+`)
	cleanText = re.ReplaceAllString(cleanText, " ")
	// Replace multiple consecutive newlines with just two (paragraph breaks)
	re = regexp.MustCompile(`\n{3,}`)
	cleanText = re.ReplaceAllString(cleanText, "\n\n")

	logging.UtilsLogger.Debug("HTML to plain text conversion completed",
		"input_length", len(htmlContent),
		"output_length", len(cleanText),
		"output_preview", truncateString(cleanText, 100))

	return cleanText, nil
}

// extractText recursively extracts text from HTML nodes
func extractText(n *htmlparser.Node, textBuilder *strings.Builder) {
	if n.Type == htmlparser.TextNode {
		textBuilder.WriteString(n.Data)
	}
	
	// Add line breaks for certain block elements
	if n.Type == htmlparser.ElementNode {
		switch n.Data {
		case "br":
			textBuilder.WriteString("\n")
		case "p", "div", "h1", "h2", "h3", "h4", "h5", "h6":
			// Add spacing around block elements
			if textBuilder.Len() > 0 {
				textBuilder.WriteString("\n\n")
			}
		case "li":
			textBuilder.WriteString("\n- ")
		case "tr":
			textBuilder.WriteString("\n")
		case "td", "th":
			textBuilder.WriteString("\t")
		}
	}

	// Recursively process child nodes
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractText(c, textBuilder)
	}
	
	// Add closing line breaks for certain block elements
	if n.Type == htmlparser.ElementNode {
		switch n.Data {
		case "p", "div", "h1", "h2", "h3", "h4", "h5", "h6":
			textBuilder.WriteString("\n")
		}
	}
}
