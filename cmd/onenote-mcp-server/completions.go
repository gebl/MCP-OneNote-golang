package main

import (
	"context"
	"strings"

	"github.com/mark3labs/mcp-go/server"

	"github.com/gebl/onenote-mcp-server/internal/graph"
	"github.com/gebl/onenote-mcp-server/internal/logging"
	"github.com/gebl/onenote-mcp-server/internal/notebooks"
)

// CompletionRequest represents the MCP completion request structure
type CompletionRequest struct {
	Ref      CompletionReference `json:"ref"`
	Argument CompletionArgument  `json:"argument"`
	Context  CompletionContext   `json:"context,omitempty"`
}

// CompletionReference represents the reference to prompt or resource
type CompletionReference struct {
	Type string `json:"type"` // "ref/prompt" or "ref/resource"
	Name string `json:"name"` // prompt name or resource URI
}

// CompletionArgument represents the argument being completed
type CompletionArgument struct {
	Name  string `json:"name"`  // argument name
	Value string `json:"value"` // current partial value
}

// CompletionContext provides additional context for completion
type CompletionContext struct {
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// CompletionResponse represents the MCP completion response structure
type CompletionResponse struct {
	Completion CompletionResult `json:"completion"`
}

// CompletionResult contains the completion suggestions
type CompletionResult struct {
	Values  []string `json:"values"`
	Total   *int     `json:"total,omitempty"`
	HasMore *bool    `json:"hasMore,omitempty"`
}

// NotebookNameCompletionHandler handles completion for notebook names
type NotebookNameCompletionHandler struct {
	graphClient *graph.Client
}

// NewNotebookNameCompletionHandler creates a new notebook name completion handler
func NewNotebookNameCompletionHandler(graphClient *graph.Client) *NotebookNameCompletionHandler {
	return &NotebookNameCompletionHandler{
		graphClient: graphClient,
	}
}

// HandleCompletion processes completion requests for notebook names
func (h *NotebookNameCompletionHandler) HandleCompletion(ctx context.Context, request CompletionRequest) (*CompletionResponse, error) {
	logging.MainLogger.Debug("Processing notebook name completion",
		"ref_type", request.Ref.Type,
		"ref_name", request.Ref.Name,
		"argument_name", request.Argument.Name,
		"argument_value", request.Argument.Value)

	// Only handle notebook name completions for relevant prompts and tools
	if !h.shouldProvideCompletion(request) {
		logging.MainLogger.Debug("Completion not applicable for this request")
		return &CompletionResponse{
			Completion: CompletionResult{
				Values: []string{},
			},
		}, nil
	}

	// Get notebook names
	notebookClient := notebooks.NewNotebookClient(h.graphClient)
	notebooks, err := notebookClient.ListNotebooks()
	if err != nil {
		logging.MainLogger.Error("Failed to get notebooks for completion", "error", err)
		return nil, err
	}

	// Extract and filter notebook names
	var suggestions []string
	partialValue := strings.ToLower(request.Argument.Value)

	for _, notebook := range notebooks {
		if displayName, exists := notebook["displayName"].(string); exists {
			// Apply fuzzy matching
			if h.matchesPartialValue(displayName, partialValue) {
				suggestions = append(suggestions, displayName)
			}
		}
	}

	// Sort by relevance (exact prefix matches first)
	suggestions = h.sortByRelevance(suggestions, request.Argument.Value)

	// Limit to maximum 100 suggestions as per MCP spec
	if len(suggestions) > 100 {
		hasMore := true
		total := len(suggestions)
		suggestions = suggestions[:100]

		return &CompletionResponse{
			Completion: CompletionResult{
				Values:  suggestions,
				Total:   &total,
				HasMore: &hasMore,
			},
		}, nil
	}

	logging.MainLogger.Debug("Notebook name completion results",
		"suggestions_count", len(suggestions),
		"partial_value", request.Argument.Value)

	return &CompletionResponse{
		Completion: CompletionResult{
			Values: suggestions,
		},
	}, nil
}

// shouldProvideCompletion determines if this handler should provide completion for the request
func (h *NotebookNameCompletionHandler) shouldProvideCompletion(request CompletionRequest) bool {
	// Check if this is a notebook name argument
	notebookNameArgs := []string{
		"notebookname", // lowercase for comparison
		"notebook_name",
		"notebook",
		"name", // for notebook-specific prompts
	}

	argumentName := strings.ToLower(request.Argument.Name)
	for _, argName := range notebookNameArgs {
		if argumentName == argName {
			return true
		}
	}

	// Check if this is for a notebook-related prompt or resource
	refName := strings.ToLower(request.Ref.Name)
	return strings.Contains(refName, "notebook")
}

// matchesPartialValue performs fuzzy matching against the partial value
func (h *NotebookNameCompletionHandler) matchesPartialValue(displayName, partialValue string) bool {
	if partialValue == "" {
		return true // empty partial matches everything
	}

	displayNameLower := strings.ToLower(displayName)
	partialValueLower := strings.ToLower(partialValue)

	// Exact substring match
	if strings.Contains(displayNameLower, partialValueLower) {
		return true
	}

	// Word boundary matching (matches beginning of words)
	words := strings.Fields(displayNameLower)
	for _, word := range words {
		if strings.HasPrefix(word, partialValueLower) {
			return true
		}
	}

	return false
}

// sortByRelevance sorts suggestions by relevance to the partial value
func (h *NotebookNameCompletionHandler) sortByRelevance(suggestions []string, partialValue string) []string {
	if len(suggestions) <= 1 || partialValue == "" {
		return suggestions
	}

	partialLower := strings.ToLower(partialValue)

	// Separate into categories for sorting
	var exactPrefixMatches []string
	var wordPrefixMatches []string
	var substringMatches []string

	for _, suggestion := range suggestions {
		suggestionLower := strings.ToLower(suggestion)

		if strings.HasPrefix(suggestionLower, partialLower) {
			exactPrefixMatches = append(exactPrefixMatches, suggestion)
		} else {
			// Check for word prefix matches
			words := strings.Fields(suggestionLower)
			hasWordPrefix := false
			for _, word := range words {
				if strings.HasPrefix(word, partialLower) {
					hasWordPrefix = true
					break
				}
			}

			if hasWordPrefix {
				wordPrefixMatches = append(wordPrefixMatches, suggestion)
			} else {
				substringMatches = append(substringMatches, suggestion)
			}
		}
	}

	// Combine in order of relevance
	var result []string
	result = append(result, exactPrefixMatches...)
	result = append(result, wordPrefixMatches...)
	result = append(result, substringMatches...)

	return result
}

// registerCompletions registers completion handlers with the MCP server
func registerCompletions(s *server.MCPServer, graphClient *graph.Client) {
	logging.MainLogger.Debug("Starting completion registration")

	// Create notebook name completion handler
	notebookHandler := NewNotebookNameCompletionHandler(graphClient)

	// Note: The mcp-go library doesn't currently support completion handlers directly.
	// This is a placeholder for when completion support is added to the library.
	// For now, we log that completion handlers are ready to be registered.

	logging.MainLogger.Info("Completion handlers prepared",
		"notebook_completion", "ready",
		"note", "Waiting for mcp-go library completion support")

	// TODO: When the mcp-go library adds completion support, register like this:
	// s.AddCompletionHandler("completion/complete", func(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	//     return notebookHandler.HandleCompletion(ctx, req)
	// })

	_ = notebookHandler // Prevent unused variable warning
	logging.MainLogger.Debug("Completion registration completed")
}
