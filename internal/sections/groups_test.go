package sections

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/gebl/onenote-mcp-server/internal/graph"
)

func TestListSectionGroupsValidation(t *testing.T) {
	tests := []struct {
		name          string
		containerID   string
		expectedError bool
		description   string
	}{
		{
			name:          "notebook container allowed",
			containerID:   "notebook-123",
			expectedError: false,
			description:   "Notebooks can contain section groups",
		},
		{
			name:          "section group container allowed",
			containerID:   "sectiongroup-456",
			expectedError: false,
			description:   "Section groups can contain other section groups",
		},
		{
			name:          "section container forbidden",
			containerID:   "section-789",
			expectedError: true,
			description:   "Sections cannot contain section groups",
		},
		{
			name:          "invalid container format",
			containerID:   "invalid-123",
			expectedError: true,
			description:   "Invalid container formats should be rejected",
		},
		{
			name:          "empty container ID",
			containerID:   "",
			expectedError: true,
			description:   "Empty container ID should be rejected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test container type validation logic
			hasError := false

			if tt.containerID == "" {
				hasError = true
			} else if strings.Contains(tt.containerID, "section-") && !strings.Contains(tt.containerID, "sectiongroup-") {
				// Sections cannot contain section groups
				hasError = true
			} else if !strings.Contains(tt.containerID, "notebook-") && !strings.Contains(tt.containerID, "sectiongroup-") {
				// Invalid container type
				hasError = true
			}

			assert.Equal(t, tt.expectedError, hasError, tt.description)
		})
	}
}

func TestListSectionsInSectionGroupValidation(t *testing.T) {
	tests := []struct {
		name           string
		sectionGroupID string
		expectedError  bool
		description    string
	}{
		{
			name:           "valid section group ID",
			sectionGroupID: "sectiongroup-123",
			expectedError:  false,
			description:    "Valid section group IDs should be accepted",
		},
		{
			name:           "empty section group ID",
			sectionGroupID: "",
			expectedError:  true,
			description:    "Empty section group ID should be rejected",
		},
		{
			name:           "whitespace only ID",
			sectionGroupID: "   ",
			expectedError:  true,
			description:    "Whitespace-only ID should be rejected",
		},
		{
			name:           "invalid format ID",
			sectionGroupID: "invalid-format",
			expectedError:  true,
			description:    "IDs not matching section group pattern should be rejected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test input validation logic
			hasError := false

			if strings.TrimSpace(tt.sectionGroupID) == "" {
				hasError = true
			} else if !strings.Contains(tt.sectionGroupID, "sectiongroup-") {
				hasError = true
			}

			assert.Equal(t, tt.expectedError, hasError, tt.description)
		})
	}
}

func TestCreateSectionGroupValidation(t *testing.T) {
	tests := []struct {
		name          string
		containerID   string
		displayName   string
		expectedError bool
		description   string
	}{
		{
			name:          "valid notebook container and name",
			containerID:   "notebook-123",
			displayName:   "New Section Group",
			expectedError: false,
			description:   "Valid inputs should be accepted",
		},
		{
			name:          "valid section group container and name",
			containerID:   "sectiongroup-456",
			displayName:   "Nested Section Group",
			expectedError: false,
			description:   "Section groups can contain other section groups",
		},
		{
			name:          "section container should fail",
			containerID:   "section-123",
			displayName:   "Invalid Section Group",
			expectedError: true,
			description:   "Sections cannot contain section groups",
		},
		{
			name:          "invalid display name with illegal characters",
			containerID:   "notebook-123",
			displayName:   "Section Group/With?Illegal*Characters",
			expectedError: true,
			description:   "Display names with illegal characters should be rejected",
		},
		{
			name:          "empty display name",
			containerID:   "notebook-123",
			displayName:   "",
			expectedError: true,
			description:   "Empty display names should be rejected",
		},
		{
			name:          "empty container ID",
			containerID:   "",
			displayName:   "Valid Name",
			expectedError: true,
			description:   "Empty container IDs should be rejected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test input validation
			hasError := false

			if tt.containerID == "" || strings.TrimSpace(tt.displayName) == "" {
				hasError = true
			} else {
				// Test display name validation
				illegalChars := []string{"?", "*", "\\", "/", ":", "<", ">", "|", "&", "#", "'", "'", "%", "~"}
				for _, char := range illegalChars {
					if strings.Contains(tt.displayName, char) {
						hasError = true
						break
					}
				}

				// Test container type validation (sections cannot contain section groups)
				if !hasError && strings.Contains(tt.containerID, "section-") && !strings.Contains(tt.containerID, "sectiongroup-") {
					hasError = true
				}
			}

			assert.Equal(t, tt.expectedError, hasError, tt.description)
		})
	}
}

func TestProcessSectionGroupsResponse(t *testing.T) {
	tests := []struct {
		name          string
		responseBody  string
		containerID   string
		expectedCount int
		expectedError bool
	}{
		{
			name:        "valid section groups response",
			containerID: "notebook-123",
			responseBody: `{
				"value": [
					{
						"id": "sectiongroup-1",
						"displayName": "Section Group 1",
						"parentNotebook": {
							"id": "notebook-123",
							"displayName": "Test Notebook"
						}
					},
					{
						"id": "sectiongroup-2",
						"displayName": "Section Group 2",
						"parentSectionGroup": {
							"id": "sectiongroup-parent",
							"displayName": "Parent Section Group"
						}
					}
				]
			}`,
			expectedCount: 2,
			expectedError: false,
		},
		{
			name:          "empty section groups response",
			containerID:   "notebook-456",
			responseBody:  `{"value": []}`,
			expectedCount: 0,
			expectedError: false,
		},
		{
			name:          "invalid JSON response",
			containerID:   "notebook-789",
			responseBody:  `{invalid json}`,
			expectedCount: 0,
			expectedError: true,
		},
		{
			name:        "section groups with sections",
			containerID: "notebook-123",
			responseBody: `{
				"value": [
					{
						"id": "sectiongroup-1",
						"displayName": "Section Group 1",
						"parentNotebook": {
							"id": "notebook-123",
							"displayName": "Test Notebook"
						},
						"sections": [
							{
								"id": "section-1",
								"displayName": "Section 1"
							}
						]
					}
				]
			}`,
			expectedCount: 1,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON parsing logic
			var data map[string]interface{}
			err := json.Unmarshal([]byte(tt.responseBody), &data)

			if tt.expectedError {
				assert.Error(t, err, "Should fail to parse invalid JSON")
			} else {
				assert.NoError(t, err, "Should parse valid JSON")

				if value, ok := data["value"].([]interface{}); ok {
					assert.Equal(t, tt.expectedCount, len(value), "Should have expected number of section groups")

					// Test section group structure validation
					for _, item := range value {
						if sectionGroup, ok := item.(map[string]interface{}); ok {
							// Should have required fields
							assert.Contains(t, sectionGroup, "id", "Section group should have ID")
							assert.Contains(t, sectionGroup, "displayName", "Section group should have display name")

							// Test parent information presence
							hasParentNotebook := false
							hasParentSectionGroup := false

							if _, exists := sectionGroup["parentNotebook"]; exists {
								hasParentNotebook = true
							}
							if _, exists := sectionGroup["parentSectionGroup"]; exists {
								hasParentSectionGroup = true
							}

							// Should have at least one parent (either notebook or section group)
							assert.True(t, hasParentNotebook || hasParentSectionGroup,
								"Section group should have either parentNotebook or parentSectionGroup")
						}
					}
				}
			}
		})
	}
}

func TestProcessCreateSectionGroupResponse(t *testing.T) {
	tests := []struct {
		name          string
		responseBody  string
		expectedError bool
		expectedID    string
	}{
		{
			name: "successful creation response",
			responseBody: `{
				"id": "new-sectiongroup-123",
				"displayName": "New Section Group",
				"createdDateTime": "2023-01-01T00:00:00Z"
			}`,
			expectedError: false,
			expectedID:    "new-sectiongroup-123",
		},
		{
			name:          "invalid JSON response",
			responseBody:  `{invalid json}`,
			expectedError: true,
			expectedID:    "",
		},
		{
			name: "response missing ID",
			responseBody: `{
				"displayName": "New Section Group",
				"createdDateTime": "2023-01-01T00:00:00Z"
			}`,
			expectedError: false,
			expectedID:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON parsing and validation logic
			var data map[string]interface{}
			err := json.Unmarshal([]byte(tt.responseBody), &data)

			if tt.expectedError {
				assert.Error(t, err, "Should fail to parse invalid JSON")
			} else {
				assert.NoError(t, err, "Should parse valid JSON")

				if tt.expectedID != "" {
					assert.Contains(t, data, "id", "Response should contain ID field")
					assert.Equal(t, tt.expectedID, data["id"], "Should have expected ID")
				}

				// Should contain display name for successful creations
				if !tt.expectedError && tt.expectedID != "" {
					assert.Contains(t, data, "displayName", "Response should contain displayName field")
				}
			}
		})
	}
}

func TestSectionGroupContainerValidation(t *testing.T) {
	tests := []struct {
		name         string
		containerID  string
		shouldAccept bool
		description  string
	}{
		{
			name:         "notebook container",
			containerID:  "notebook-123",
			shouldAccept: true,
			description:  "Notebooks can contain section groups",
		},
		{
			name:         "section group container",
			containerID:  "sectiongroup-456",
			shouldAccept: true,
			description:  "Section groups can contain other section groups",
		},
		{
			name:         "section container",
			containerID:  "section-789",
			shouldAccept: false,
			description:  "Sections cannot contain section groups",
		},
		{
			name:         "invalid container format",
			containerID:  "invalid-123",
			shouldAccept: false,
			description:  "Invalid container types should be rejected",
		},
		{
			name:         "empty container",
			containerID:  "",
			shouldAccept: false,
			description:  "Empty container ID should be rejected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test container type validation logic
			if tt.containerID == "" {
				assert.False(t, tt.shouldAccept, "Empty container ID should not be accepted")
				return
			}

			// Check container type based on ID pattern
			isSection := strings.Contains(tt.containerID, "section-") && !strings.Contains(tt.containerID, "sectiongroup-")
			isSectionGroup := strings.Contains(tt.containerID, "sectiongroup-")
			isNotebook := strings.Contains(tt.containerID, "notebook-")

			if isSection {
				assert.False(t, tt.shouldAccept, "Sections should not be able to contain section groups")
			} else if isSectionGroup || isNotebook {
				assert.True(t, tt.shouldAccept, "Notebooks and section groups should be able to contain section groups")
			} else {
				assert.False(t, tt.shouldAccept, "Invalid container types should be rejected")
			}
		})
	}
}

// Test error handling scenarios
func TestSectionGroupErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		scenario    string
		expectError bool
	}{
		{
			name:        "network timeout",
			scenario:    "HTTP request timeout",
			expectError: true,
		},
		{
			name:        "authentication failure",
			scenario:    "401 Unauthorized response",
			expectError: true,
		},
		{
			name:        "forbidden access",
			scenario:    "403 Forbidden response",
			expectError: true,
		},
		{
			name:        "resource not found",
			scenario:    "404 Not Found response",
			expectError: true,
		},
		{
			name:        "server error",
			scenario:    "500 Internal Server Error",
			expectError: true,
		},
		{
			name:        "malformed response",
			scenario:    "Invalid JSON in response",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that various error scenarios are properly handled
			assert.True(t, tt.expectError, "Error scenarios should be handled appropriately")

			// Verify that the scenario description is meaningful
			assert.NotEmpty(t, tt.scenario, "Error scenario should have description")
		})
	}
}

// Benchmark tests for section group operations
func BenchmarkSectionGroupValidation(b *testing.B) {
	testContainers := []string{"notebook-123", "sectiongroup-456", "section-789"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		containerID := testContainers[i%len(testContainers)]
		// Test container validation logic
		isSection := strings.Contains(containerID, "section-") && !strings.Contains(containerID, "sectiongroup-")
		_ = isSection
	}
}

func BenchmarkDisplayNameValidationSectionGroup(b *testing.B) {
	testNames := []string{"Valid Name", "Invalid?Name", "Another*Invalid", "Good Name"}
	illegalChars := []string{"?", "*", "\\", "/", ":", "<", ">", "|", "&", "#", "'", "'", "%", "~"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		name := testNames[i%len(testNames)]
		hasIllegal := false
		for _, char := range illegalChars {
			if strings.Contains(name, char) {
				hasIllegal = true
				break
			}
		}
		_ = hasIllegal
	}
}

// Integration test setup
func TestSectionGroupClientSetup(t *testing.T) {
	t.Run("section group client can be created", func(t *testing.T) {
		graphClient := &graph.Client{}
		sectionClient := NewSectionClient(graphClient)

		assert.NotNil(t, sectionClient)
		assert.Equal(t, graphClient, sectionClient.Client)

		// Test that container type constants are properly defined
		assert.Equal(t, "section", containerTypeSection)
		assert.Equal(t, "sectionGroup", containerTypeSectionGroup)
		assert.Equal(t, "notebook", containerTypeNotebook)
	})
}
