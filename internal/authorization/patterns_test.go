// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package authorization

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPatternEngine_CompilePatterns(t *testing.T) {
	tests := []struct {
		name     string
		patterns map[string]PermissionLevel
		wantErr  bool
	}{
		{
			name: "valid patterns",
			patterns: map[string]PermissionLevel{
				"exact_match":     PermissionRead,
				"prefix_*":        PermissionWrite,
				"*_suffix":        PermissionRead,
				"/path/**":        PermissionNone,
				"/path/*/file":    PermissionWrite,
			},
			wantErr: false,
		},
		{
			name: "empty patterns",
			patterns: map[string]PermissionLevel{},
			wantErr: false,
		},
		{
			name: "pattern with brackets treated as exact",
			patterns: map[string]PermissionLevel{
				"[section]": PermissionRead,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pe := NewPatternEngine()
			err := pe.CompilePatterns(tt.patterns)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, pe.compiledPatterns, len(tt.patterns))
			}
		})
	}
}

func TestPatternEngine_PrecedenceOrdering(t *testing.T) {
	patterns := map[string]PermissionLevel{
		"/**":                   PermissionNone,    // Lowest precedence
		"/Projects/**":          PermissionRead,    // Medium precedence 
		"/Projects/public_*":    PermissionWrite,   // Higher precedence
		"/Projects/Web App/public_*": PermissionFull, // Highest precedence
		"exact_match":           PermissionRead,    // Exact match
	}
	
	pe := NewPatternEngine()
	err := pe.CompilePatterns(patterns)
	require.NoError(t, err)
	
	// Verify precedence ordering (lower number = higher precedence)
	compiled := pe.GetAllPatterns()
	require.Len(t, compiled, 5)
	
	// Exact matches should have highest precedence (lowest number)
	assert.True(t, compiled[0].IsExact)
	assert.Equal(t, "exact_match", compiled[0].Original)
	
	// More specific path patterns should come before less specific ones
	var webAppIndex, projectsPublicIndex, projectsRecursiveIndex, globalRecursiveIndex int
	for i, pattern := range compiled {
		switch pattern.Original {
		case "/Projects/Web App/public_*":
			webAppIndex = i
		case "/Projects/public_*":
			projectsPublicIndex = i
		case "/Projects/**":
			projectsRecursiveIndex = i
		case "/**":
			globalRecursiveIndex = i
		}
	}
	
	// More specific patterns should have higher precedence (lower index)
	assert.True(t, webAppIndex < projectsPublicIndex, "Web App pattern should have higher precedence than Projects pattern")
	assert.True(t, projectsPublicIndex < projectsRecursiveIndex, "Projects public pattern should have higher precedence than Projects recursive")
	assert.True(t, projectsRecursiveIndex < globalRecursiveIndex, "Projects recursive should have higher precedence than global recursive")
}

func TestPatternEngine_Match(t *testing.T) {
	patterns := map[string]PermissionLevel{
		"Meeting Notes":              PermissionWrite,   // Exact match
		"public_*":                   PermissionRead,    // Prefix pattern
		"*_draft":                    PermissionWrite,   // Suffix pattern
		"/Projects/**":               PermissionRead,    // Recursive pattern
		"/Projects/public_*":         PermissionWrite,   // More specific pattern
		"/Projects/Web App/public_*": PermissionFull,    // Most specific pattern
		"/Archive/*":                 PermissionNone,    // Single level wildcard
	}
	
	pe := NewPatternEngine()
	err := pe.CompilePatterns(patterns)
	require.NoError(t, err)
	
	tests := []struct {
		name            string
		value           string
		expectedPerm    PermissionLevel
		expectedPattern string
		expectedMatch   bool
	}{
		{
			name:            "exact match",
			value:           "Meeting Notes",
			expectedPerm:    PermissionWrite,
			expectedPattern: "Meeting Notes",
			expectedMatch:   true,
		},
		{
			name:            "prefix match",
			value:           "public_document",
			expectedPerm:    PermissionRead,
			expectedPattern: "public_*",
			expectedMatch:   true,
		},
		{
			name:            "suffix match",
			value:           "my_draft",
			expectedPerm:    PermissionWrite,
			expectedPattern: "*_draft",
			expectedMatch:   true,
		},
		{
			name:            "most specific path pattern",
			value:           "/Projects/Web App/public_doc",
			expectedPerm:    PermissionFull,
			expectedPattern: "/Projects/Web App/public_*",
			expectedMatch:   true,
		},
		{
			name:            "less specific path pattern",
			value:           "/Projects/Mobile/public_spec",
			expectedPerm:    PermissionRead,
			expectedPattern: "/Projects/**",
			expectedMatch:   true,
		},
		{
			name:            "recursive pattern",
			value:           "/Projects/Mobile/Design/wireframes",
			expectedPerm:    PermissionRead,
			expectedPattern: "/Projects/**",
			expectedMatch:   true,
		},
		{
			name:            "single level pattern match",
			value:           "/Archive/old_notes",
			expectedPerm:    PermissionNone,
			expectedPattern: "/Archive/*",
			expectedMatch:   true,
		},
		{
			name:            "single level pattern no match (too deep)",
			value:           "/Archive/2024/old_notes",
			expectedPerm:    "",
			expectedPattern: "",
			expectedMatch:   false,
		},
		{
			name:            "no match",
			value:           "random_section",
			expectedPerm:    "",
			expectedPattern: "",
			expectedMatch:   false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			perm, pattern, match := pe.Match(tt.value)
			assert.Equal(t, tt.expectedMatch, match, "Match result mismatch")
			assert.Equal(t, tt.expectedPerm, perm, "Permission mismatch")
			assert.Equal(t, tt.expectedPattern, pattern, "Pattern mismatch")
		})
	}
}

func TestPatternEngine_SpecificityPrecedence(t *testing.T) {
	// Test that more specific patterns take precedence over less specific ones
	patterns := map[string]PermissionLevel{
		"/Section A/**":        PermissionRead,  // Less specific
		"/Section A/public_*":  PermissionWrite, // More specific - should win
	}
	
	pe := NewPatternEngine()
	err := pe.CompilePatterns(patterns)
	require.NoError(t, err)
	
	// Test that the more specific pattern wins
	perm, pattern, match := pe.Match("/Section A/public_document")
	assert.True(t, match)
	assert.Equal(t, PermissionWrite, perm)
	assert.Equal(t, "/Section A/public_*", pattern)
}

func TestPatternEngine_PathNormalization(t *testing.T) {
	patterns := map[string]PermissionLevel{
		"/Projects/**": PermissionRead,
		"public_*":     PermissionWrite,
	}
	
	pe := NewPatternEngine()
	err := pe.CompilePatterns(patterns)
	require.NoError(t, err)
	
	tests := []struct {
		name            string
		value           string
		expectedPattern string
		expectedMatch   bool
	}{
		{
			name:            "with leading slash",
			value:           "/Projects/Web/file",
			expectedPattern: "/Projects/**",
			expectedMatch:   true,
		},
		{
			name:            "without leading slash",
			value:           "Projects/Web/file",
			expectedPattern: "/Projects/**",
			expectedMatch:   true,
		},
		{
			name:            "prefix pattern - no slash",
			value:           "public_doc",
			expectedPattern: "public_*",
			expectedMatch:   true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, pattern, match := pe.Match(tt.value)
			assert.Equal(t, tt.expectedMatch, match, "Match result mismatch")
			if match {
				assert.Equal(t, tt.expectedPattern, pattern, "Pattern mismatch")
			}
		})
	}
}

func TestPatternEngine_ComplexScenarios(t *testing.T) {
	// Real-world complex pattern matching scenarios
	patterns := map[string]PermissionLevel{
		// Global patterns
		"/**":                  PermissionRead,
		"temp_*":               PermissionWrite,
		"*_archive":            PermissionNone,
		
		// Section-specific patterns
		"/Personal/**":         PermissionWrite,
		"/Work/**":             PermissionRead,
		"/Work/Projects/**":    PermissionWrite,
		"/Work/Projects/*/Draft": PermissionFull,
		
		// Very specific patterns
		"/Work/Projects/WebApp/public_*": PermissionRead,
		"/Personal/Journal":              PermissionFull,
		
		// Edge cases
		"/Archive/*":           PermissionNone,  // Single level only
		"/Archive/*/*":         PermissionRead,  // Two levels deep
	}
	
	pe := NewPatternEngine()
	err := pe.CompilePatterns(patterns)
	require.NoError(t, err)
	
	tests := []struct {
		name            string
		value           string
		expectedPerm    PermissionLevel
		expectedPattern string
		description     string
	}{
		{
			name:            "exact override",
			value:           "/Personal/Journal",
			expectedPerm:    PermissionFull,
			expectedPattern: "/Personal/Journal",
			description:     "Exact match should override recursive pattern",
		},
		{
			name:            "specific path override",
			value:           "/Work/Projects/WebApp/public_doc",
			expectedPerm:    PermissionRead,
			expectedPattern: "/Work/Projects/WebApp/public_*",
			description:     "Most specific path pattern should win",
		},
		{
			name:            "draft pattern priority",
			value:           "/Work/Projects/MobileApp/Draft",
			expectedPerm:    PermissionFull,
			expectedPattern: "/Work/Projects/*/Draft",
			description:     "Specific draft pattern should override general project pattern",
		},
		{
			name:            "temp prefix global",
			value:           "temp_notes",
			expectedPerm:    PermissionWrite,
			expectedPattern: "temp_*",
			description:     "Global temp pattern should work",
		},
		{
			name:            "archive single level",
			value:           "/Archive/2024",
			expectedPerm:    PermissionNone,
			expectedPattern: "/Archive/*",
			description:     "Single level archive pattern should match",
		},
		{
			name:            "archive two levels - matches two-level pattern",
			value:           "/Archive/2024/January", 
			expectedPerm:    PermissionRead,
			expectedPattern: "/Archive/*/*",
			description:     "Two-level archive pattern should match two-level paths",
		},
		{
			name:            "archive three levels - matches global recursive",
			value:           "/Archive/2024/January/notes",
			expectedPerm:    PermissionRead,
			expectedPattern: "/**",
			description:     "Global recursive pattern should match deep archive paths",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			perm, pattern, match := pe.Match(tt.value)
			assert.True(t, match, "Should find a match: %s", tt.description)
			assert.Equal(t, tt.expectedPerm, perm, "Permission mismatch: %s", tt.description)
			assert.Equal(t, tt.expectedPattern, pattern, "Pattern mismatch: %s", tt.description)
		})
	}
}