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

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
	
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

// hasMarkdownSyntax checks if the content contains Markdown syntax using goldmark parser
func hasMarkdownSyntax(content string) bool {
	logging.UtilsLogger.Debug("Starting Markdown syntax detection with goldmark",
		"content_length", len(content),
		"content_preview", truncateString(content, 100))
	
	// First check for common markdown patterns using regex (fast check)
	if hasCommonMarkdownPatterns(content) {
		logging.UtilsLogger.Debug("Common markdown patterns detected via regex")
		return true
	}
	
	// Create goldmark parser with extensions for more thorough parsing
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Table,
			extension.Strikethrough,
			extension.TaskList,
		),
	)
	
	// Parse the content
	reader := text.NewReader([]byte(content))
	doc := md.Parser().Parse(reader)
	
	// Count meaningful markdown nodes
	markdownNodes := 0
	detectedTypes := []string{}
	
	err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			switch n.Kind() {
			case ast.KindHeading:
				markdownNodes++
				detectedTypes = append(detectedTypes, "heading")
			case ast.KindList:
				markdownNodes++
				detectedTypes = append(detectedTypes, "list")
			case ast.KindEmphasis:
				markdownNodes++
				detectedTypes = append(detectedTypes, "emphasis")
			case ast.KindLink:
				markdownNodes++
				detectedTypes = append(detectedTypes, "link")
			case ast.KindCodeBlock:
				markdownNodes++
				detectedTypes = append(detectedTypes, "codeblock")
			case ast.KindFencedCodeBlock:
				markdownNodes++
				detectedTypes = append(detectedTypes, "fenced_codeblock")
			case ast.KindBlockquote:
				markdownNodes++
				detectedTypes = append(detectedTypes, "blockquote")
			case ast.KindThematicBreak:
				markdownNodes++
				detectedTypes = append(detectedTypes, "horizontal_rule")
			}
		}
		return ast.WalkContinue, nil
	})
	
	if err != nil {
		logging.UtilsLogger.Debug("Error walking AST", "error", err)
		return false
	}
	
	hasMarkdown := markdownNodes > 0
	
	logging.UtilsLogger.Debug("Markdown syntax detection completed with goldmark",
		"markdown_nodes_found", markdownNodes,
		"detected_types", detectedTypes,
		"has_markdown_syntax", hasMarkdown)
	
	return hasMarkdown
}

// hasCommonMarkdownPatterns checks for common markdown patterns using regex
func hasCommonMarkdownPatterns(content string) bool {
	patterns := []struct {
		name    string
		regex   *regexp.Regexp
	}{
		{"headers", regexp.MustCompile(`^#{1,6}\s`)},
		{"unordered_lists", regexp.MustCompile(`^[-*+]\s`)},
		{"ordered_lists", regexp.MustCompile(`^\d+\.\s`)},
		{"blockquotes", regexp.MustCompile(`^>\s`)},
		{"code_blocks", regexp.MustCompile("^```|^~~~")},
		{"horizontal_rules", regexp.MustCompile(`^(---+|\*\*\*+|___+)$`)},
		{"tables", regexp.MustCompile(`^\|.*\|.*$`)},
		{"strikethrough", regexp.MustCompile(`~~.*?~~`)},
		{"emphasis", regexp.MustCompile(`\*\*.*?\*\*|\*.*?\*|__.*?__|_.*?_`)},
		{"inline_code", regexp.MustCompile("`.*?`")},
		{"links", regexp.MustCompile(`\[.*?\]\(.*?\)`)},
	}
	
	lines := strings.Split(content, "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		// Check line-based patterns
		for _, p := range patterns {
			if p.name == "strikethrough" || p.name == "emphasis" || p.name == "inline_code" || p.name == "links" {
				// These can appear anywhere in the content, not just at line start
				if p.regex.MatchString(content) {
					logging.UtilsLogger.Debug("Markdown pattern detected via regex",
						"pattern_name", p.name,
						"content_preview", truncateString(content, 50))
					return true
				}
			} else {
				// These must appear at the start of lines
				if p.regex.MatchString(line) {
					logging.UtilsLogger.Debug("Markdown pattern detected via regex",
						"pattern_name", p.name,
						"line_content", truncateString(line, 50))
					return true
				}
			}
		}
	}
	
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

// convertMarkdownToHTML converts Markdown text to HTML using goldmark
func convertMarkdownToHTML(markdownText string) string {
	logging.UtilsLogger.Debug("Starting Markdown to HTML conversion with goldmark",
		"input_length", len(markdownText),
		"input_preview", truncateString(markdownText, 100))
	
	// Set up goldmark with extensions and options suitable for OneNote
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,           // GitHub Flavored Markdown
			extension.Table,         // Table support
			extension.Strikethrough, // Strikethrough support
			extension.TaskList,      // Task list support
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(), // Auto-generate heading IDs
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(), // Convert line breaks to <br>
			html.WithUnsafe(),    // Allow raw HTML (needed for OneNote)
		),
	)
	
	logging.UtilsLogger.Debug("Goldmark configured",
		"extensions", "GFM, Table, Strikethrough, TaskList",
		"options", "AutoHeadingID, HardWraps, Unsafe")
	
	// Convert markdown to HTML
	var buf strings.Builder
	err := md.Convert([]byte(markdownText), &buf)
	if err != nil {
		logging.UtilsLogger.Error("Markdown conversion failed", "error", err)
		// Fallback to treating as plain text
		return convertASCIIToHTML(markdownText)
	}
	
	resultHTML := buf.String()
	
	logging.UtilsLogger.Debug("Markdown to HTML conversion completed with goldmark",
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

// ConvertHTMLToText converts HTML content to plain text format using goquery
func ConvertHTMLToText(htmlContent string) (string, error) {
	logging.UtilsLogger.Debug("Starting HTML to plain text conversion with goquery",
		"input_length", len(htmlContent),
		"input_preview", truncateString(htmlContent, 100))

	// Parse the HTML with goquery
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		logging.UtilsLogger.Error("HTML parsing with goquery failed", "error", err)
		return "", fmt.Errorf("failed to parse HTML with goquery: %v", err)
	}

	var textBuilder strings.Builder

	// Process specific elements in order to avoid duplication
	processElement(doc.Find("body"), &textBuilder)

	plainText := textBuilder.String()

	// Clean up the text - remove excessive whitespace and normalize spacing
	cleanText := strings.TrimSpace(plainText)
	
	// Replace multiple consecutive spaces with single space
	re := regexp.MustCompile(`[ \t]+`)
	cleanText = re.ReplaceAllString(cleanText, " ")
	
	// Replace multiple consecutive newlines with just two (paragraph breaks)
	re = regexp.MustCompile(`\n{3,}`)
	cleanText = re.ReplaceAllString(cleanText, "\n\n")
	
	// Clean up spacing around newlines
	re = regexp.MustCompile(`\n[ \t]+`)
	cleanText = re.ReplaceAllString(cleanText, "\n")
	re = regexp.MustCompile(`[ \t]+\n`)
	cleanText = re.ReplaceAllString(cleanText, "\n")

	logging.UtilsLogger.Debug("HTML to plain text conversion completed with goquery",
		"input_length", len(htmlContent),
		"output_length", len(cleanText),
		"output_preview", truncateString(cleanText, 100))

	return cleanText, nil
}

// processElement recursively processes HTML elements to build text with proper structure
func processElement(selection *goquery.Selection, textBuilder *strings.Builder) {
	selection.Contents().Each(func(i int, s *goquery.Selection) {
		// If it's a text node, add it directly
		if goquery.NodeName(s) == "#text" {
			text := strings.TrimSpace(s.Text())
			if text != "" {
				if textBuilder.Len() > 0 && !strings.HasSuffix(textBuilder.String(), " ") && !strings.HasSuffix(textBuilder.String(), "\n") {
					textBuilder.WriteString(" ")
				}
				textBuilder.WriteString(text)
			}
			return
		}

		tagName := goquery.NodeName(s)
		
		// Handle different elements with appropriate formatting
		switch tagName {
		case "h1", "h2", "h3", "h4", "h5", "h6":
			// Headers: add spacing and the text
			if textBuilder.Len() > 0 {
				textBuilder.WriteString("\n\n")
			}
			textBuilder.WriteString(strings.TrimSpace(s.Text()))
			textBuilder.WriteString("\n")
			
		case "p":
			// Paragraphs: add spacing and process contents
			if textBuilder.Len() > 0 {
				textBuilder.WriteString("\n\n")
			}
			processElement(s, textBuilder)
			
		case "div":
			// Divs: process contents with spacing
			if textBuilder.Len() > 0 && !strings.HasSuffix(textBuilder.String(), "\n") {
				textBuilder.WriteString("\n\n")
			}
			processElement(s, textBuilder)
			
		case "br":
			// Line breaks
			textBuilder.WriteString("\n")
			
		case "ul", "ol":
			// Lists: add spacing and process list items
			if textBuilder.Len() > 0 {
				textBuilder.WriteString("\n")
			}
			processElement(s, textBuilder)
			
		case "li":
			// List items: add bullet point and process contents
			textBuilder.WriteString("\n- ")
			processElement(s, textBuilder)
			
		case "table":
			// Tables: add spacing and process table contents
			if textBuilder.Len() > 0 {
				textBuilder.WriteString("\n")
			}
			processElement(s, textBuilder)
			
		case "tr":
			// Table rows: add newline and process cells
			if textBuilder.Len() > 0 && !strings.HasSuffix(textBuilder.String(), "\n") {
				textBuilder.WriteString("\n")
			}
			processElement(s, textBuilder)
			
		case "td", "th":
			// Table cells: add tab separation and process contents
			if textBuilder.Len() > 0 && !strings.HasSuffix(textBuilder.String(), "\n") && !strings.HasSuffix(textBuilder.String(), "\t") {
				textBuilder.WriteString("\t")
			}
			processElement(s, textBuilder)
			
		default:
			// For other elements, just process their contents
			processElement(s, textBuilder)
		}
	})
}

