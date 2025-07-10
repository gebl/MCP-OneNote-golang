package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/gebl/onenote-mcp-server/internal/graph"
)

// MockNotebookClientForCompletion provides a mock for completion testing
type MockNotebookClientForCompletion struct {
	mock.Mock
}

func (m *MockNotebookClientForCompletion) ListNotebooks() ([]map[string]interface{}, error) {
	args := m.Called()
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func TestNotebookNameCompletion(t *testing.T) {
	// Create test notebooks
	testNotebooks := []map[string]interface{}{
		{"displayName": "Work Notebook", "notebookId": "1"},
		{"displayName": "Personal Notes", "notebookId": "2"},
		{"displayName": "Meeting Notes", "notebookId": "3"},
		{"displayName": "Project Alpha", "notebookId": "4"},
		{"displayName": "Homework", "notebookId": "5"},
	}

	tests := []struct {
		name              string
		request           CompletionRequest
		expectedValues    []string
		shouldProvideComp bool
	}{
		{
			name: "notebook name argument should provide completion",
			request: CompletionRequest{
				Ref: CompletionReference{
					Type: "ref/prompt",
					Name: "notebook_workflow",
				},
				Argument: CompletionArgument{
					Name:  "notebookName",
					Value: "",
				},
			},
			expectedValues:    []string{"Work Notebook", "Personal Notes", "Meeting Notes", "Project Alpha", "Homework"},
			shouldProvideComp: true,
		},
		{
			name: "partial value filters results",
			request: CompletionRequest{
				Ref: CompletionReference{
					Type: "ref/prompt",
					Name: "notebook_workflow",
				},
				Argument: CompletionArgument{
					Name:  "notebookName",
					Value: "work",
				},
			},
			expectedValues:    []string{"Work Notebook", "Homework"}, // "Homework" contains "work"
			shouldProvideComp: true,
		},
		{
			name: "case insensitive matching",
			request: CompletionRequest{
				Ref: CompletionReference{
					Type: "ref/prompt",
					Name: "notebook_workflow",
				},
				Argument: CompletionArgument{
					Name:  "notebookName",
					Value: "NOTES",
				},
			},
			expectedValues:    []string{"Personal Notes", "Meeting Notes"},
			shouldProvideComp: true,
		},
		{
			name: "word boundary matching",
			request: CompletionRequest{
				Ref: CompletionReference{
					Type: "ref/prompt",
					Name: "notebook_workflow",
				},
				Argument: CompletionArgument{
					Name:  "notebookName",
					Value: "meet",
				},
			},
			expectedValues:    []string{"Meeting Notes"},
			shouldProvideComp: true,
		},
		{
			name: "non-notebook argument should not provide completion",
			request: CompletionRequest{
				Ref: CompletionReference{
					Type: "ref/prompt",
					Name: "some_other_prompt",
				},
				Argument: CompletionArgument{
					Name:  "someOtherArg",
					Value: "",
				},
			},
			expectedValues:    []string{},
			shouldProvideComp: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock graph client
			mockGraphClient := &graph.Client{} // Simplified for testing

			// Create completion handler
			handler := NewNotebookNameCompletionHandler(mockGraphClient)

			// Test shouldProvideCompletion
			shouldProvide := handler.shouldProvideCompletion(tt.request)
			assert.Equal(t, tt.shouldProvideComp, shouldProvide, "shouldProvideCompletion mismatch")

			if !shouldProvide {
				return // Skip the rest of the test if completion shouldn't be provided
			}

			// Test fuzzy matching logic directly (without actual API calls)
			var results []string
			for _, notebook := range testNotebooks {
				if displayName, exists := notebook["displayName"].(string); exists {
					if handler.matchesPartialValue(displayName, tt.request.Argument.Value) {
						results = append(results, displayName)
					}
				}
			}

			// Sort by relevance
			results = handler.sortByRelevance(results, tt.request.Argument.Value)

			assert.Equal(t, tt.expectedValues, results, "Completion results mismatch")
		})
	}
}

func TestMatchesPartialValue(t *testing.T) {
	handler := &NotebookNameCompletionHandler{}

	tests := []struct {
		displayName  string
		partialValue string
		shouldMatch  bool
	}{
		{"Work Notebook", "", true},        // empty partial matches everything
		{"Work Notebook", "work", true},    // case insensitive substring
		{"Work Notebook", "WORK", true},    // case insensitive
		{"Work Notebook", "note", true},    // substring match
		{"Work Notebook", "book", true},    // substring match
		{"Work Notebook", "xyz", false},    // no match
		{"Personal Notes", "person", true}, // word prefix
		{"Personal Notes", "notes", true},  // word match
		{"Meeting Notes", "meet", true},    // word prefix
		{"Project Alpha", "alpha", true},   // word match
		{"Project Alpha", "proj", true},    // word prefix
		{"Something Else", "xyz", false},   // no match
	}

	for _, tt := range tests {
		t.Run(tt.displayName+"_"+tt.partialValue, func(t *testing.T) {
			result := handler.matchesPartialValue(tt.displayName, tt.partialValue)
			assert.Equal(t, tt.shouldMatch, result)
		})
	}
}

func TestSortByRelevance(t *testing.T) {
	handler := &NotebookNameCompletionHandler{}

	suggestions := []string{
		"Work Notebook",     // exact prefix for "work"
		"Homework",          // substring match for "work"
		"Working Directory", // exact prefix for "work"
		"Network",           // substring match for "work" (netWORK!)
	}

	tests := []struct {
		partialValue string
		expected     []string
	}{
		{
			partialValue: "work",
			expected:     []string{"Work Notebook", "Working Directory", "Homework", "Network"}, // exact prefix first, then substring
		},
		{
			partialValue: "",
			expected:     []string{"Work Notebook", "Homework", "Working Directory", "Network"}, // original order when no partial
		},
		{
			partialValue: "net",
			expected:     []string{"Network"}, // only matches Network
		},
	}

	for _, tt := range tests {
		t.Run("partial_"+tt.partialValue, func(t *testing.T) {
			// Filter suggestions first (simulate the matching logic)
			var filtered []string
			for _, suggestion := range suggestions {
				if handler.matchesPartialValue(suggestion, tt.partialValue) {
					filtered = append(filtered, suggestion)
				}
			}

			// Sort by relevance
			result := handler.sortByRelevance(filtered, tt.partialValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShouldProvideCompletion(t *testing.T) {
	handler := &NotebookNameCompletionHandler{}

	tests := []struct {
		name     string
		request  CompletionRequest
		expected bool
	}{
		{
			name: "notebookName argument",
			request: CompletionRequest{
				Argument: CompletionArgument{Name: "notebookName"},
			},
			expected: true,
		},
		{
			name: "notebook_name argument",
			request: CompletionRequest{
				Argument: CompletionArgument{Name: "notebook_name"},
			},
			expected: true,
		},
		{
			name: "notebook argument",
			request: CompletionRequest{
				Argument: CompletionArgument{Name: "notebook"},
			},
			expected: true,
		},
		{
			name: "name argument for notebook prompt",
			request: CompletionRequest{
				Ref:      CompletionReference{Name: "notebook_workflow"},
				Argument: CompletionArgument{Name: "name"},
			},
			expected: true,
		},
		{
			name: "random argument",
			request: CompletionRequest{
				Argument: CompletionArgument{Name: "randomArg"},
			},
			expected: false,
		},
		{
			name: "case insensitive matching",
			request: CompletionRequest{
				Argument: CompletionArgument{Name: "NotebookName"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.shouldProvideCompletion(tt.request)
			assert.Equal(t, tt.expected, result)
		})
	}
}
