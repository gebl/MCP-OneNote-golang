// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package sections

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gebl/onenote-mcp-server/internal/graph"
)

// TestSectionClientFunctionality tests core section client functionality
func TestSectionClientFunctionality(t *testing.T) {
	t.Run("section client creation", func(t *testing.T) {
		graphClient := &graph.Client{}
		sectionClient := NewSectionClient(graphClient)

		assert.NotNil(t, sectionClient)
		assert.Equal(t, graphClient, sectionClient.Client)
		assert.IsType(t, &SectionClient{}, sectionClient)
	})
}

// TestContainerTypeDetection tests OneNote container type detection
func TestContainerTypeDetection(t *testing.T) {
	t.Run("determine container types from ID patterns", func(t *testing.T) {
		tests := []struct {
			name          string
			containerID   string
			expectedType  string
		}{
			{
				name:         "notebook container ID pattern",
				containerID:  "0-A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6!123",
				expectedType: "notebook",
			},
			{
				name:         "section group container ID pattern",
				containerID:  "0-A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6!123_456",
				expectedType: "sectionGroup",
			},
			{
				name:         "section container ID pattern",
				containerID:  "1-A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6!123",
				expectedType: "section",
			},
			{
				name:         "UUID format container ID",
				containerID:  "12345678-1234-1234-1234-123456789abc",
				expectedType: "uuid",
			},
			{
				name:         "simple container ID",
				containerID:  "simple-container-id",
				expectedType: "unknown",
			},
			{
				name:         "empty container ID",
				containerID:  "",
				expectedType: "empty",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				containerType := determineContainerTypeFromID(tt.containerID)
				assert.Equal(t, tt.expectedType, containerType)
			})
		}
	})
}

// TestSectionResponseProcessing tests section response processing
func TestSectionResponseProcessing(t *testing.T) {
	t.Run("process valid sections response", func(t *testing.T) {
		responseBody := `{
			"@odata.context": "https://graph.microsoft.com/v1.0/$metadata#users('me')/onenote/sections",
			"value": [
				{
					"id": "section-1",
					"displayName": "Work Notes",
					"self": "https://graph.microsoft.com/v1.0/users/me/onenote/sections/section-1",
					"pagesUrl": "https://graph.microsoft.com/v1.0/users/me/onenote/sections/section-1/pages",
					"createdDateTime": "2024-01-01T10:00:00Z",
					"lastModifiedDateTime": "2024-01-02T15:30:00Z",
					"parentNotebook": {
						"id": "notebook-1",
						"displayName": "My Notebook"
					}
				},
				{
					"id": "section-2",
					"displayName": "Personal Notes",
					"self": "https://graph.microsoft.com/v1.0/users/me/onenote/sections/section-2",
					"pagesUrl": "https://graph.microsoft.com/v1.0/users/me/onenote/sections/section-2/pages"
				}
			]
		}`

		var response struct {
			Context string                   `json:"@odata.context"`
			Value   []map[string]interface{} `json:"value"`
		}

		err := json.Unmarshal([]byte(responseBody), &response)
		require.NoError(t, err)

		assert.Len(t, response.Value, 2)

		// Verify first section
		section1 := response.Value[0]
		assert.Equal(t, "section-1", section1["id"])
		assert.Equal(t, "Work Notes", section1["displayName"])
		assert.Contains(t, section1, "self")
		assert.Contains(t, section1, "pagesUrl")
		assert.Contains(t, section1, "parentNotebook")

		// Verify second section (without parent notebook)
		section2 := response.Value[1]
		assert.Equal(t, "section-2", section2["id"])
		assert.Equal(t, "Personal Notes", section2["displayName"])
		_, hasParentNotebook := section2["parentNotebook"]
		assert.False(t, hasParentNotebook)
	})

	t.Run("handle empty sections response", func(t *testing.T) {
		responseBody := `{
			"@odata.context": "https://graph.microsoft.com/v1.0/$metadata#users('me')/onenote/sections",
			"value": []
		}`

		var response struct {
			Value []map[string]interface{} `json:"value"`
		}

		err := json.Unmarshal([]byte(responseBody), &response)
		require.NoError(t, err)

		assert.Len(t, response.Value, 0)
		assert.NotNil(t, response.Value) // Should be empty slice, not nil
	})

	t.Run("handle response without value field", func(t *testing.T) {
		responseBody := `{
			"@odata.context": "https://graph.microsoft.com/v1.0/$metadata#users('me')/onenote/sections"
		}`

		var response struct {
			Value []map[string]interface{} `json:"value"`
		}

		err := json.Unmarshal([]byte(responseBody), &response)
		require.NoError(t, err)

		assert.Len(t, response.Value, 0) // Works with both nil and empty slices
	})
}

// TestSectionCreationValidation tests section creation validation
func TestSectionCreationValidation(t *testing.T) {
	t.Run("validate section creation parameters", func(t *testing.T) {
		tests := []struct {
			name         string
			containerID  string
			displayName  string
			shouldError  bool
			errorMessage string
		}{
			{
				name:        "valid notebook container and name",
				containerID: "0-A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6!123",
				displayName: "New Section",
				shouldError: false,
			},
			{
				name:        "valid section group container",
				containerID: "0-A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6!123_456",
				displayName: "New Section",
				shouldError: false,
			},
			{
				name:         "invalid section container",
				containerID:  "1-A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6!123",
				displayName:  "New Section",
				shouldError:  true,
				errorMessage: "sections cannot contain other sections",
			},
			{
				name:         "empty display name",
				containerID:  "0-A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6!123",
				displayName:  "",
				shouldError:  true,
				errorMessage: "display name cannot be empty",
			},
			{
				name:         "display name with illegal characters",
				containerID:  "0-A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6!123",
				displayName:  "Invalid?Name*With<Illegal>Characters",
				shouldError:  true,
				errorMessage: "contains illegal characters",
			},
			{
				name:         "empty container ID",
				containerID:  "",
				displayName:  "New Section",
				shouldError:  true,
				errorMessage: "container ID cannot be empty",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := validateSectionCreationParameters(tt.containerID, tt.displayName)

				if tt.shouldError {
					assert.Error(t, err)
					if tt.errorMessage != "" {
						assert.Contains(t, err.Error(), tt.errorMessage)
					}
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})
}

// TestSectionGroupOperations tests section group operations
func TestSectionGroupOperations(t *testing.T) {
	t.Run("process section groups response", func(t *testing.T) {
		responseBody := `{
			"@odata.context": "https://graph.microsoft.com/v1.0/$metadata#users('me')/onenote/notebooks('notebook-1')/sectionGroups",
			"value": [
				{
					"id": "sectionGroup-1",
					"displayName": "Project Alpha",
					"sectionsUrl": "https://graph.microsoft.com/v1.0/users/me/onenote/sectionGroups/sectionGroup-1/sections",
					"sectionGroupsUrl": "https://graph.microsoft.com/v1.0/users/me/onenote/sectionGroups/sectionGroup-1/sectionGroups",
					"createdDateTime": "2024-01-01T09:00:00Z",
					"lastModifiedDateTime": "2024-01-02T14:00:00Z",
					"parentNotebook": {
						"id": "notebook-1",
						"displayName": "Work Notebook"
					},
					"sections": [
						{
							"id": "section-1",
							"displayName": "Requirements"
						}
					],
					"sectionGroups": [
						{
							"id": "subGroup-1",
							"displayName": "Sub Group"
						}
					]
				}
			]
		}`

		var response struct {
			Value []map[string]interface{} `json:"value"`
		}

		err := json.Unmarshal([]byte(responseBody), &response)
		require.NoError(t, err)

		assert.Len(t, response.Value, 1)

		// Verify section group structure
		sg := response.Value[0]
		assert.Equal(t, "sectionGroup-1", sg["id"])
		assert.Equal(t, "Project Alpha", sg["displayName"])
		assert.Contains(t, sg, "sectionsUrl")
		assert.Contains(t, sg, "sectionGroupsUrl")

		// Verify nested sections
		if sections, ok := sg["sections"].([]interface{}); ok {
			assert.Len(t, sections, 1)
		}

		// Verify nested section groups
		if sectionGroups, ok := sg["sectionGroups"].([]interface{}); ok {
			assert.Len(t, sectionGroups, 1)
		}
	})

	t.Run("validate section group creation", func(t *testing.T) {
		tests := []struct {
			name        string
			containerID string
			displayName string
			valid       bool
		}{
			{
				name:        "valid notebook container",
				containerID: "0-A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6!123",
				displayName: "New Section Group",
				valid:       true,
			},
			{
				name:        "valid section group container",
				containerID: "0-A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6!123_456",
				displayName: "Nested Section Group",
				valid:       true,
			},
			{
				name:        "invalid section container",
				containerID: "1-A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6!123",
				displayName: "Invalid Container",
				valid:       false,
			},
			{
				name:        "empty display name",
				containerID: "0-A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6!123",
				displayName: "",
				valid:       false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := validateSectionGroupCreationParameters(tt.containerID, tt.displayName)

				if tt.valid {
					assert.NoError(t, err)
				} else {
					assert.Error(t, err)
				}
			})
		}
	})
}

// TestOneNoteHierarchyRules tests OneNote's container hierarchy validation
func TestOneNoteHierarchyRules(t *testing.T) {
	t.Run("validate OneNote container hierarchy rules", func(t *testing.T) {
		tests := []struct {
			name          string
			containerType string
			operation     string
			allowed       bool
		}{
			{
				name:          "sections in notebook allowed",
				containerType: "notebook",
				operation:     "create_section",
				allowed:       true,
			},
			{
				name:          "sections in section group allowed",
				containerType: "sectionGroup",
				operation:     "create_section",
				allowed:       true,
			},
			{
				name:          "sections in section forbidden",
				containerType: "section",
				operation:     "create_section",
				allowed:       false,
			},
			{
				name:          "section groups in notebook allowed",
				containerType: "notebook",
				operation:     "create_section_group",
				allowed:       true,
			},
			{
				name:          "section groups in section group allowed",
				containerType: "sectionGroup",
				operation:     "create_section_group",
				allowed:       true,
			},
			{
				name:          "section groups in section forbidden",
				containerType: "section",
				operation:     "create_section_group",
				allowed:       false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				allowed := isOperationAllowedInContainer(tt.containerType, tt.operation)
				assert.Equal(t, tt.allowed, allowed)
			})
		}
	})
}

// TestDisplayNameValidationFunctional tests OneNote display name validation functionally
func TestDisplayNameValidationFunctional(t *testing.T) {
	t.Run("validate display name constraints", func(t *testing.T) {
		validNames := []string{
			"Simple Section",
			"Section with Numbers 123",
			"Section-with-hyphens",
			"Section_with_underscores",
			"日本語セクション", // Unicode characters
		}

		invalidNames := []string{
			"",                                    // Empty
			"Section?with?question?marks",         // Question marks
			"Section*with*asterisks",              // Asterisks
			"Section\\with\\backslashes",          // Backslashes
			"Section/with/forward/slashes",        // Forward slashes
			"Section:with:colons",                 // Colons
			"Section<with<less<than",              // Less than
			"Section>with>greater>than",           // Greater than
			"Section|with|pipes",                  // Pipes
			"Section&with&ampersands",             // Ampersands
			"Section#with#hashes",                 // Hash symbols
			"Section'with'single'quotes",          // Single quotes
			"Section%with%percent",                // Percent symbols
			"Section~with~tildes",                 // Tildes
		}

		for _, name := range validNames {
			t.Run(fmt.Sprintf("valid_%s", name), func(t *testing.T) {
				err := validateDisplayName(name)
				assert.NoError(t, err, "Name should be valid: %s", name)
			})
		}

		for _, name := range invalidNames {
			t.Run(fmt.Sprintf("invalid_%s", strings.ReplaceAll(name, " ", "_")), func(t *testing.T) {
				err := validateDisplayName(name)
				assert.Error(t, err, "Name should be invalid: %s", name)
			})
		}
	})
}

// TestJSONResponseRobustness tests JSON parsing robustness
func TestJSONResponseRobustness(t *testing.T) {
	t.Run("parse sections with missing optional fields", func(t *testing.T) {
		responseBody := `{
			"value": [
				{
					"id": "section-1",
					"displayName": "Minimal Section"
				},
				{
					"id": "section-2",
					"displayName": "Section with Notebook",
					"parentNotebook": {
						"id": "notebook-1",
						"displayName": "Parent Notebook"
					}
				},
				{
					"id": "section-3",
					"displayName": "Section with Full Data",
					"self": "https://graph.microsoft.com/v1.0/sections/section-3",
					"pagesUrl": "https://graph.microsoft.com/v1.0/sections/section-3/pages",
					"createdDateTime": "2024-01-01T10:00:00Z",
					"lastModifiedDateTime": "2024-01-02T15:00:00Z"
				}
			]
		}`

		var response struct {
			Value []map[string]interface{} `json:"value"`
		}

		err := json.Unmarshal([]byte(responseBody), &response)
		require.NoError(t, err)

		assert.Len(t, response.Value, 3)

		// First section has minimal fields
		section1 := response.Value[0]
		assert.Equal(t, "section-1", section1["id"])
		assert.Equal(t, "Minimal Section", section1["displayName"])
		_, hasNotebook := section1["parentNotebook"]
		assert.False(t, hasNotebook)

		// Second section has parent notebook
		section2 := response.Value[1]
		assert.Contains(t, section2, "parentNotebook")

		// Third section has full data
		section3 := response.Value[2]
		assert.Contains(t, section3, "self")
		assert.Contains(t, section3, "pagesUrl")
		assert.Contains(t, section3, "createdDateTime")
	})

	t.Run("handle malformed JSON responses", func(t *testing.T) {
		malformedResponses := []string{
			`{"incomplete": json`,
			`{value: []}`, // Missing quotes
			`{"value": [{"id": "test"}`, // Unclosed array/object
		}

		for i, responseBody := range malformedResponses {
			t.Run(fmt.Sprintf("malformed_response_%d", i), func(t *testing.T) {
				var response struct {
					Value []map[string]interface{} `json:"value"`
				}

				err := json.Unmarshal([]byte(responseBody), &response)
				assert.Error(t, err, "Should fail to parse malformed JSON")
			})
		}
	})
}

// TestErrorScenarios tests various error handling scenarios  
func TestErrorScenarios(t *testing.T) {
	t.Run("handle empty and nil data", func(t *testing.T) {
		// Empty section list
		var sections []map[string]interface{}
		assert.Len(t, sections, 0) // Works with both nil and zero-length slices

		// Operations on empty list should work
		for _, section := range sections {
			t.Errorf("Unexpected section in empty list: %v", section)
		}
	})

	t.Run("handle sections with missing required fields", func(t *testing.T) {
		sections := []map[string]interface{}{
			{"id": "section-1", "displayName": "Complete Section"},
			{"displayName": "Section without ID"},          // Missing ID
			{"id": "section-3"},                           // Missing displayName
			{"id": "", "displayName": "Empty ID Section"}, // Empty ID
			{"id": "section-5", "displayName": ""},        // Empty displayName
		}

		var validSections []map[string]interface{}

		// Filter for sections with both required fields
		for _, section := range sections {
			if id, hasID := section["id"].(string); hasID && id != "" {
				if displayName, hasName := section["displayName"].(string); hasName && displayName != "" {
					validSections = append(validSections, section)
				}
			}
		}

		assert.Len(t, validSections, 1) // Only one complete section
		assert.Equal(t, "section-1", validSections[0]["id"])
	})
}

// Helper functions for validation

func determineContainerTypeFromID(containerID string) string {
	if containerID == "" {
		return "empty"
	}

	// UUID pattern
	if len(containerID) == 36 && strings.Count(containerID, "-") == 4 {
		return "uuid"
	}

	// OneNote ID patterns
	if strings.Contains(containerID, "_") {
		return "sectionGroup"
	}
	if strings.HasPrefix(containerID, "0-") {
		return "notebook"
	}
	if strings.HasPrefix(containerID, "1-") {
		return "section"
	}

	return "unknown"
}

func validateSectionCreationParameters(containerID, displayName string) error {
	if containerID == "" {
		return fmt.Errorf("container ID cannot be empty")
	}

	if displayName == "" {
		return fmt.Errorf("display name cannot be empty")
	}

	// Validate display name
	if err := validateDisplayName(displayName); err != nil {
		return err
	}

	// Check container hierarchy rules
	containerType := determineContainerTypeFromID(containerID)
	if containerType == "section" {
		return fmt.Errorf("sections cannot contain other sections")
	}

	return nil
}

func validateSectionGroupCreationParameters(containerID, displayName string) error {
	if containerID == "" {
		return fmt.Errorf("container ID cannot be empty")
	}

	if displayName == "" {
		return fmt.Errorf("display name cannot be empty")
	}

	// Validate display name
	if err := validateDisplayName(displayName); err != nil {
		return err
	}

	// Check container hierarchy rules
	containerType := determineContainerTypeFromID(containerID)
	if containerType == "section" {
		return fmt.Errorf("sections cannot contain section groups")
	}

	return nil
}

func validateDisplayName(displayName string) error {
	if displayName == "" {
		return fmt.Errorf("display name cannot be empty")
	}

	// OneNote illegal characters
	illegalChars := []string{"?", "*", "\\", "/", ":", "<", ">", "|", "&", "#", "'", "%", "~"}
	for _, char := range illegalChars {
		if strings.Contains(displayName, char) {
			return fmt.Errorf("display name contains illegal characters: %s", char)
		}
	}

	return nil
}

func isOperationAllowedInContainer(containerType, operation string) bool {
	switch containerType {
	case "notebook":
		return operation == "create_section" || operation == "create_section_group"
	case "sectionGroup":
		return operation == "create_section" || operation == "create_section_group"
	case "section":
		return false // Sections cannot contain other sections or section groups
	default:
		return false
	}
}

// Performance benchmarks

func BenchmarkContainerTypeDetectionPerformance(b *testing.B) {
	testIDs := []string{
		"0-A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6!123",
		"0-A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6!123_456",
		"1-A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6!789",
		"12345678-1234-1234-1234-123456789abc",
		"simple-container-id",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, id := range testIDs {
			_ = determineContainerTypeFromID(id)
		}
	}
}

func BenchmarkSectionResponseProcessing(b *testing.B) {
	// Create large sections response for benchmarking
	sections := make([]map[string]interface{}, 1000)
	for i := 0; i < 1000; i++ {
		sections[i] = map[string]interface{}{
			"id":          fmt.Sprintf("section-%d", i),
			"displayName": fmt.Sprintf("Section %d", i),
			"self":        fmt.Sprintf("https://graph.microsoft.com/v1.0/sections/section-%d", i),
			"pagesUrl":    fmt.Sprintf("https://graph.microsoft.com/v1.0/sections/section-%d/pages", i),
		}
	}

	responseData := map[string]interface{}{
		"value": sections,
	}

	jsonData, _ := json.Marshal(responseData)
	responseBody := string(jsonData)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var response struct {
			Value []map[string]interface{} `json:"value"`
		}
		_ = json.Unmarshal([]byte(responseBody), &response)
	}
}

func BenchmarkDisplayNameValidationFunctional(b *testing.B) {
	testNames := []string{
		"Valid Section Name",
		"Section with Numbers 123",
		"Section-with-hyphens",
		"Invalid?Name*With<Illegal>Characters",
		"Another/Invalid\\Name",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, name := range testNames {
			_ = validateDisplayName(name)
		}
	}
}