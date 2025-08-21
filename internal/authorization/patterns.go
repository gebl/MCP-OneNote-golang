// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package authorization

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/gebl/onenote-mcp-server/internal/logging"
)

// CompiledPattern represents a compiled permission pattern with precedence
type CompiledPattern struct {
	Original    string          // Original pattern string
	Permission  PermissionLevel // Permission level for this pattern
	Regex       *regexp.Regexp  // Compiled regex (nil for exact/prefix patterns)
	Precedence  int             // Lower = higher precedence (0 = highest)
	IsExact     bool            // True for exact matches (no wildcards)
	IsPrefix    bool            // True for simple prefix patterns (ends with *)
	IsSuffix    bool            // True for simple suffix patterns (starts with *)
	IsRecursive bool            // True for /** patterns
	PathSegments int            // Number of path segments (for /a/b/c counting)
	LiteralChars int            // Number of literal characters (more = higher precedence)
}

// PatternEngine handles pattern compilation and matching with precedence
type PatternEngine struct {
	compiledPatterns []CompiledPattern
}

// NewPatternEngine creates a new pattern engine
func NewPatternEngine() *PatternEngine {
	return &PatternEngine{
		compiledPatterns: make([]CompiledPattern, 0),
	}
}

// CompilePatterns compiles a map of patterns to a sorted list by precedence
func (pe *PatternEngine) CompilePatterns(patterns map[string]PermissionLevel) error {
	pe.compiledPatterns = make([]CompiledPattern, 0, len(patterns))
	
	for pattern, permission := range patterns {
		compiled, err := pe.compilePattern(pattern, permission)
		if err != nil {
			return fmt.Errorf("failed to compile pattern '%s': %v", pattern, err)
		}
		pe.compiledPatterns = append(pe.compiledPatterns, compiled)
	}
	
	// Sort by precedence (lower number = higher precedence)
	sort.Slice(pe.compiledPatterns, func(i, j int) bool {
		return pe.compiledPatterns[i].Precedence < pe.compiledPatterns[j].Precedence
	})
	
	logging.AuthorizationLogger.Debug("Compiled and sorted patterns by precedence",
		"pattern_count", len(pe.compiledPatterns),
		"patterns", pe.getPatternSummary())
	
	return nil
}

// compilePattern compiles a single pattern with precedence calculation
func (pe *PatternEngine) compilePattern(pattern string, permission PermissionLevel) (CompiledPattern, error) {
	compiled := CompiledPattern{
		Original:   pattern,
		Permission: permission,
	}
	
	// Normalize pattern (trim leading/trailing slashes for consistency)
	normalizedPattern := strings.Trim(pattern, "/")
	
	// Calculate base metrics
	compiled.PathSegments = strings.Count(normalizedPattern, "/") + 1
	if normalizedPattern == "" {
		compiled.PathSegments = 0
	}
	
	// Check for wildcards
	hasSingleWildcard := strings.Contains(pattern, "*") && !strings.Contains(pattern, "**")
	hasRecursiveWildcard := strings.Contains(pattern, "**")
	
	// Count literal characters (excluding wildcards)
	literalPattern := strings.ReplaceAll(pattern, "**", "")
	literalPattern = strings.ReplaceAll(literalPattern, "*", "")
	compiled.LiteralChars = len(literalPattern)
	
	if !hasSingleWildcard && !hasRecursiveWildcard {
		// Exact match - highest precedence
		compiled.IsExact = true
		compiled.Precedence = 0 + compiled.PathSegments // Exact matches sorted by path depth
		
	} else if hasRecursiveWildcard {
		// Recursive patterns (**) - lower precedence
		compiled.IsRecursive = true
		compiled.Precedence = 1000 + (20 - compiled.LiteralChars) + (10 - compiled.PathSegments)
		
		// Convert /** patterns to regex
		regexPattern := strings.ReplaceAll(regexp.QuoteMeta(pattern), "\\*\\*", ".*")
		regex, err := regexp.Compile("^" + regexPattern + "$")
		if err != nil {
			return compiled, fmt.Errorf("invalid recursive pattern regex: %v", err)
		}
		compiled.Regex = regex
		
	} else if hasSingleWildcard {
		// Single wildcard patterns - medium precedence
		if strings.HasSuffix(pattern, "*") && !strings.Contains(strings.TrimSuffix(pattern, "*"), "*") {
			// Simple prefix pattern like "public_*"
			compiled.IsPrefix = true
			compiled.Precedence = 100 + (20 - compiled.LiteralChars) + compiled.PathSegments
			
		} else if strings.HasPrefix(pattern, "*") && !strings.Contains(strings.TrimPrefix(pattern, "*"), "*") {
			// Simple suffix pattern like "*_notes"
			compiled.IsSuffix = true
			compiled.Precedence = 150 + (20 - compiled.LiteralChars) + compiled.PathSegments
			
		} else {
			// Complex single wildcard patterns - need regex
			compiled.Precedence = 200 + (20 - compiled.LiteralChars) + compiled.PathSegments
			
			regexPattern := strings.ReplaceAll(regexp.QuoteMeta(pattern), "\\*", "[^/]*") // * matches anything except /
			regex, err := regexp.Compile("^" + regexPattern + "$")
			if err != nil {
				return compiled, fmt.Errorf("invalid wildcard pattern regex: %v", err)
			}
			compiled.Regex = regex
		}
	}
	
	logging.AuthorizationLogger.Debug("Compiled pattern",
		"pattern", pattern,
		"permission", permission,
		"precedence", compiled.Precedence,
		"is_exact", compiled.IsExact,
		"is_prefix", compiled.IsPrefix,
		"is_suffix", compiled.IsSuffix,
		"is_recursive", compiled.IsRecursive,
		"path_segments", compiled.PathSegments,
		"literal_chars", compiled.LiteralChars)
	
	return compiled, nil
}

// Match finds the best matching pattern for a given value
func (pe *PatternEngine) Match(value string) (PermissionLevel, string, bool) {
	if len(pe.compiledPatterns) == 0 {
		return "", "", false
	}
	
	// Normalize value for comparison
	normalizedValue := strings.Trim(value, "/")
	
	// Try each pattern in precedence order (already sorted)
	for _, pattern := range pe.compiledPatterns {
		if pe.matchesPattern(normalizedValue, pattern) {
			logging.AuthorizationLogger.Debug("Pattern matched",
				"value", value,
				"normalized_value", normalizedValue,
				"pattern", pattern.Original,
				"permission", pattern.Permission,
				"precedence", pattern.Precedence,
				"match_type", pe.getMatchType(pattern))
			return pattern.Permission, pattern.Original, true
		}
	}
	
	logging.AuthorizationLogger.Debug("No pattern matched", "value", value, "normalized_value", normalizedValue)
	return "", "", false
}

// matchesPattern checks if a value matches a specific compiled pattern
func (pe *PatternEngine) matchesPattern(value string, pattern CompiledPattern) bool {
	normalizedPattern := strings.Trim(pattern.Original, "/")
	
	if pattern.IsExact {
		return value == normalizedPattern
		
	} else if pattern.IsPrefix {
		prefix := strings.TrimSuffix(normalizedPattern, "*")
		return strings.HasPrefix(value, prefix)
		
	} else if pattern.IsSuffix {
		suffix := strings.TrimPrefix(normalizedPattern, "*")
		return strings.HasSuffix(value, suffix)
		
	} else if pattern.Regex != nil {
		// For path patterns, test both with and without leading slash
		testValues := []string{value, "/" + value}
		if strings.HasPrefix(value, "/") {
			testValues = append(testValues, strings.TrimPrefix(value, "/"))
		}
		
		for _, testValue := range testValues {
			if pattern.Regex.MatchString(testValue) {
				return true
			}
		}
		return false
	}
	
	return false
}

// getMatchType returns a string describing the match type for logging
func (pe *PatternEngine) getMatchType(pattern CompiledPattern) string {
	if pattern.IsExact {
		return "exact"
	} else if pattern.IsPrefix {
		return "prefix"
	} else if pattern.IsSuffix {
		return "suffix"
	} else if pattern.IsRecursive {
		return "recursive"
	}
	return "regex"
}

// getPatternSummary returns a summary of compiled patterns for logging
func (pe *PatternEngine) getPatternSummary() []map[string]interface{} {
	summary := make([]map[string]interface{}, len(pe.compiledPatterns))
	for i, pattern := range pe.compiledPatterns {
		summary[i] = map[string]interface{}{
			"pattern":     pattern.Original,
			"permission":  pattern.Permission,
			"precedence":  pattern.Precedence,
			"type":        pe.getMatchType(pattern),
		}
	}
	return summary
}

// GetAllPatterns returns all compiled patterns (for testing/debugging)
func (pe *PatternEngine) GetAllPatterns() []CompiledPattern {
	return pe.compiledPatterns
}