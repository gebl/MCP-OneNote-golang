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
	"strings"
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
