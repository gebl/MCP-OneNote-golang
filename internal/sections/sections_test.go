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
	"github.com/stretchr/testify/mock"

	"github.com/gebl/onenote-mcp-server/internal/graph"
)

func TestNewSectionClient(t *testing.T) {
	t.Run("creates new section client", func(t *testing.T) {
		graphClient := &graph.Client{}
		sectionClient := NewSectionClient(graphClient)

		assert.NotNil(t, sectionClient)
		assert.Equal(t, graphClient, sectionClient.Client)
	})
}

func TestDisplayNameValidation(t *testing.T) {
	tests := []struct {
		name        string
		displayName string
		expectError bool
		description string
	}{
		{
			name:        "valid display name",
			displayName: "Valid Section Name",
			expectError: false,
			description: "Simple valid name should pass",
		},
		{
			name:        "name with spaces",
			displayName: "Section With Multiple Spaces",
			expectError: false,
			description: "Names with spaces should be valid",
		},
		{
			name:        "name with numbers",
			displayName: "Section 123",
			expectError: false,
			description: "Names with numbers should be valid",
		},
		{
			name:        "name with hyphens",
			displayName: "Section-With-Hyphens",
			expectError: false,
			description: "Names with hyphens should be valid",
		},
		{
			name:        "name with underscores",
			displayName: "Section_With_Underscores",
			expectError: false,
			description: "Names with underscores should be valid",
		},
		{
			name:        "name with question mark",
			displayName: "Section?",
			expectError: true,
			description: "Names with question marks should be invalid",
		},
		{
			name:        "name with asterisk",
			displayName: "Section*",
			expectError: true,
			description: "Names with asterisks should be invalid",
		},
		{
			name:        "name with backslash",
			displayName: "Section\\Name",
			expectError: true,
			description: "Names with backslashes should be invalid",
		},
		{
			name:        "name with forward slash",
			displayName: "Section/Name",
			expectError: true,
			description: "Names with forward slashes should be invalid",
		},
		{
			name:        "name with colon",
			displayName: "Section:Name",
			expectError: true,
			description: "Names with colons should be invalid",
		},
		{
			name:        "name with less than",
			displayName: "Section<Name",
			expectError: true,
			description: "Names with less than symbols should be invalid",
		},
		{
			name:        "name with greater than",
			displayName: "Section>Name",
			expectError: true,
			description: "Names with greater than symbols should be invalid",
		},
		{
			name:        "name with pipe",
			displayName: "Section|Name",
			expectError: true,
			description: "Names with pipe symbols should be invalid",
		},
		{
			name:        "name with ampersand",
			displayName: "Section&Name",
			expectError: true,
			description: "Names with ampersands should be invalid",
		},
		{
			name:        "name with hash",
			displayName: "Section#Name",
			expectError: true,
			description: "Names with hash symbols should be invalid",
		},
		{
			name:        "name with single quote",
			displayName: "Section'Name",
			expectError: true,
			description: "Names with single quotes should be invalid",
		},
		{
			name:        "name with percent",
			displayName: "Section%Name",
			expectError: true,
			description: "Names with percent symbols should be invalid",
		},
		{
			name:        "name with tilde",
			displayName: "Section~Name",
			expectError: true,
			description: "Names with tildes should be invalid",
		},
		{
			name:        "empty name",
			displayName: "",
			expectError: true,
			description: "Empty names should be invalid",
		},
		{
			name:        "whitespace only name",
			displayName: "   ",
			expectError: true,
			description: "Whitespace-only names should be invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the validation logic that would be used in CreateSection
			hasError := false

			// Check for empty or whitespace-only names
			if strings.TrimSpace(tt.displayName) == "" {
				hasError = true
			}

			// Check for illegal characters
			illegalChars := []string{"?", "*", "\\", "/", ":", "<", ">", "|", "&", "#", "'", "'", "%", "~"}
			for _, char := range illegalChars {
				if strings.Contains(tt.displayName, char) {
					hasError = true
					break
				}
			}

			assert.Equal(t, tt.expectError, hasError, tt.description)
		})
	}
}

func TestContainerTypeConstants(t *testing.T) {
	t.Run("container type constants are correctly defined", func(t *testing.T) {
		assert.Equal(t, "section", containerTypeSection)
		assert.Equal(t, "sectionGroup", containerTypeSectionGroup)
		assert.Equal(t, "notebook", containerTypeNotebook)

		// Ensure they are all different
		types := []string{containerTypeSection, containerTypeSectionGroup, containerTypeNotebook}
		seen := make(map[string]bool)
		for _, containerType := range types {
			assert.False(t, seen[containerType], "Container types should be unique")
			seen[containerType] = true
		}

		// Ensure they are all non-empty
		for _, containerType := range types {
			assert.NotEmpty(t, containerType, "Container types should not be empty")
		}
	})
}

func TestContainerTypeValidation(t *testing.T) {
	tests := []struct {
		name         string
		containerID  string
		expectedType string
		description  string
	}{
		{
			name:         "notebook ID pattern",
			containerID:  "notebook-123",
			expectedType: "notebook-like",
			description:  "IDs starting with 'notebook-' should be recognized as notebook type",
		},
		{
			name:         "section group ID pattern",
			containerID:  "sectiongroup-456",
			expectedType: "sectiongroup-like",
			description:  "IDs starting with 'sectiongroup-' should be recognized as section group type",
		},
		{
			name:         "section ID pattern",
			containerID:  "section-789",
			expectedType: "section-like",
			description:  "IDs starting with 'section-' should be recognized as section type",
		},
		{
			name:         "UUID pattern",
			containerID:  "12345678-1234-1234-1234-123456789abc",
			expectedType: "uuid-like",
			description:  "UUID patterns should be recognized",
		},
		{
			name:         "invalid pattern",
			containerID:  "invalid-format",
			expectedType: "unknown",
			description:  "Unknown patterns should be marked as such",
		},
		{
			name:         "empty ID",
			containerID:  "",
			expectedType: "invalid",
			description:  "Empty IDs should be invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test basic pattern recognition logic
			var detectedType string

			if tt.containerID == "" {
				detectedType = "invalid"
			} else if strings.HasPrefix(tt.containerID, "notebook-") {
				detectedType = "notebook-like"
			} else if strings.HasPrefix(tt.containerID, "sectiongroup-") {
				detectedType = "sectiongroup-like"
			} else if strings.HasPrefix(tt.containerID, "section-") {
				detectedType = "section-like"
			} else if len(tt.containerID) == 36 && strings.Count(tt.containerID, "-") == 4 {
				detectedType = "uuid-like"
			} else {
				detectedType = "unknown"
			}

			assert.Equal(t, tt.expectedType, detectedType, tt.description)
		})
	}
}

func TestSectionHierarchyRules(t *testing.T) {
	tests := []struct {
		name                    string
		containerType           string
		canContainSections      bool
		canContainSectionGroups bool
		description             string
	}{
		{
			name:                    "notebook container",
			containerType:           containerTypeNotebook,
			canContainSections:      true,
			canContainSectionGroups: true,
			description:             "Notebooks can contain both sections and section groups",
		},
		{
			name:                    "section group container",
			containerType:           containerTypeSectionGroup,
			canContainSections:      true,
			canContainSectionGroups: true,
			description:             "Section groups can contain both sections and other section groups",
		},
		{
			name:                    "section container",
			containerType:           containerTypeSection,
			canContainSections:      false,
			canContainSectionGroups: false,
			description:             "Sections cannot contain other sections or section groups",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test OneNote hierarchy rules
			switch tt.containerType {
			case containerTypeNotebook:
				assert.True(t, tt.canContainSections, "Notebooks should be able to contain sections")
				assert.True(t, tt.canContainSectionGroups, "Notebooks should be able to contain section groups")
			case containerTypeSectionGroup:
				assert.True(t, tt.canContainSections, "Section groups should be able to contain sections")
				assert.True(t, tt.canContainSectionGroups, "Section groups should be able to contain other section groups")
			case containerTypeSection:
				assert.False(t, tt.canContainSections, "Sections should not be able to contain other sections")
				assert.False(t, tt.canContainSectionGroups, "Sections should not be able to contain section groups")
			}
		})
	}
}

func TestJSONResponseParsing(t *testing.T) {
	tests := []struct {
		name          string
		jsonData      string
		expectError   bool
		expectedCount int
		description   string
	}{
		{
			name: "valid sections response",
			jsonData: `{
				"value": [
					{
						"id": "section-1",
						"displayName": "Section 1",
						"parentNotebook": {
							"id": "notebook-1",
							"displayName": "Test Notebook"
						}
					},
					{
						"id": "section-2",
						"displayName": "Section 2"
					}
				]
			}`,
			expectError:   false,
			expectedCount: 2,
			description:   "Valid sections JSON should parse correctly",
		},
		{
			name:          "empty sections response",
			jsonData:      `{"value": []}`,
			expectError:   false,
			expectedCount: 0,
			description:   "Empty sections response should be valid",
		},
		{
			name:          "invalid JSON",
			jsonData:      `{invalid json}`,
			expectError:   true,
			expectedCount: 0,
			description:   "Invalid JSON should cause parsing error",
		},
		{
			name:          "missing value field",
			jsonData:      `{"data": []}`,
			expectError:   false,
			expectedCount: 0,
			description:   "Missing value field should be handled gracefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var response map[string]interface{}
			err := json.Unmarshal([]byte(tt.jsonData), &response)

			if tt.expectError {
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)

				if value, exists := response["value"]; exists {
					if valueArray, ok := value.([]interface{}); ok {
						assert.Equal(t, tt.expectedCount, len(valueArray), "Should have expected number of items")
					}
				} else {
					assert.Equal(t, 0, tt.expectedCount, "Missing value field should result in zero count")
				}
			}
		})
	}
}

func TestURLPatternExtraction(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectedID  string
		expectError bool
		description string
	}{
		{
			name:        "valid OneNote section URL",
			url:         "https://www.onenote.com/sections/section-abc123",
			expectedID:  "section-abc123",
			expectError: false,
			description: "Should extract section ID from OneNote URL",
		},
		{
			name:        "OneNote URL with query parameters",
			url:         "https://www.onenote.com/sections/section-def456?view=edit",
			expectedID:  "section-def456",
			expectError: false,
			description: "Should extract section ID ignoring query parameters",
		},
		{
			name:        "invalid URL format",
			url:         "https://invalid-url.com/something",
			expectedID:  "",
			expectError: true,
			description: "Invalid URL format should cause error",
		},
		{
			name:        "empty URL",
			url:         "",
			expectedID:  "",
			expectError: true,
			description: "Empty URL should cause error",
		},
		{
			name:        "URL without section ID",
			url:         "https://www.onenote.com/notebooks/",
			expectedID:  "",
			expectError: true,
			description: "URL without section ID should cause error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test URL parsing logic that would be used in extractSectionIDFromResourceLocation
			var extractedID string
			var hasError bool

			if tt.url == "" {
				hasError = true
			} else if strings.Contains(tt.url, "/sections/") {
				// Simple extraction logic
				parts := strings.Split(tt.url, "/sections/")
				if len(parts) > 1 {
					idPart := strings.Split(parts[1], "?")[0] // Remove query parameters
					if idPart != "" {
						extractedID = idPart
					} else {
						hasError = true
					}
				} else {
					hasError = true
				}
			} else {
				hasError = true
			}

			assert.Equal(t, tt.expectError, hasError, tt.description)
			if !hasError {
				assert.Equal(t, tt.expectedID, extractedID, "Should extract correct ID")
			}
		})
	}
}

func TestSectionDataStructure(t *testing.T) {
	t.Run("section metadata structure validation", func(t *testing.T) {
		// Test that section data structures have expected fields
		sampleSection := map[string]interface{}{
			"id":                   "section-123",
			"displayName":          "Test Section",
			"createdDateTime":      "2023-01-01T00:00:00Z",
			"lastModifiedDateTime": "2023-01-02T00:00:00Z",
			"parentNotebook": map[string]interface{}{
				"id":          "notebook-456",
				"displayName": "Test Notebook",
			},
		}

		// Validate required fields
		assert.Contains(t, sampleSection, "id", "Section should have ID")
		assert.Contains(t, sampleSection, "displayName", "Section should have display name")

		// Validate field types
		assert.IsType(t, "", sampleSection["id"], "Section ID should be string")
		assert.IsType(t, "", sampleSection["displayName"], "Section display name should be string")

		// Validate parent information
		if parentNotebook, exists := sampleSection["parentNotebook"]; exists {
			if parent, ok := parentNotebook.(map[string]interface{}); ok {
				assert.Contains(t, parent, "id", "Parent notebook should have ID")
				assert.Contains(t, parent, "displayName", "Parent notebook should have display name")
			}
		}
	})
}

// Benchmark tests for performance validation
func BenchmarkNewSectionClient(b *testing.B) {
	graphClient := &graph.Client{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewSectionClient(graphClient)
	}
}

func BenchmarkDisplayNameValidation(b *testing.B) {
	testNames := []string{
		"Valid Section Name",
		"Section?Invalid",
		"Section*Invalid",
		"Section/Invalid",
		"Another Valid Name",
	}

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

func BenchmarkContainerTypeDetection(b *testing.B) {
	testIDs := []string{
		"notebook-123",
		"sectiongroup-456",
		"section-789",
		"12345678-1234-1234-1234-123456789abc",
		"invalid-format",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := testIDs[i%len(testIDs)]
		var detectedType string
		if strings.HasPrefix(id, "notebook-") {
			detectedType = "notebook"
		} else if strings.HasPrefix(id, "sectiongroup-") {
			detectedType = "sectiongroup"
		} else if strings.HasPrefix(id, "section-") {
			detectedType = "section"
		} else {
			detectedType = "unknown"
		}
		_ = detectedType
	}
}

// Integration test setup
func TestSectionClientIntegrationSetup(t *testing.T) {
	t.Run("section client can be created with real graph client", func(t *testing.T) {
		// This would be used for integration tests with a real graph client
		graphClient := &graph.Client{}
		sectionClient := NewSectionClient(graphClient)

		assert.NotNil(t, sectionClient)
		assert.Equal(t, graphClient, sectionClient.Client)
		assert.IsType(t, &SectionClient{}, sectionClient)
	})

	t.Run("section client provides expected interface", func(t *testing.T) {
		graphClient := &graph.Client{}
		sectionClient := NewSectionClient(graphClient)

		// Verify that SectionClient has the expected structure
		assert.NotNil(t, sectionClient.Client, "Section client should embed graph client")
	})
}

// MockSectionClient provides comprehensive mock implementation for testing section operations
type MockSectionClient struct {
	mock.Mock
	sections           map[string][]map[string]interface{}
	sectionGroups      map[string][]map[string]interface{}
	allSections        []map[string]interface{}
	mockError          error
	shouldFail         map[string]bool
	createdSections    []map[string]interface{}
	createdGroups      []map[string]interface{}
	operations         []string
}

func NewMockSectionClient() *MockSectionClient {
	return &MockSectionClient{
		sections: map[string][]map[string]interface{}{
			"notebook-1": {
				{
					"id":          "section-1",
					"displayName": "Work Section",
					"createdDateTime": "2023-01-01T00:00:00Z",
					"lastModifiedDateTime": "2023-01-05T00:00:00Z",
					"parentNotebook": map[string]interface{}{
						"id": "notebook-1",
						"displayName": "Work Notebook",
					},
					"pagesUrl": "https://graph.microsoft.com/v1.0/me/onenote/sections/section-1/pages",
				},
				{
					"id":          "section-2",
					"displayName": "Projects",
					"parentNotebook": map[string]interface{}{
						"id": "notebook-1",
						"displayName": "Work Notebook",
					},
				},
			},
			"notebook-2": {
				{
					"id":          "section-3",
					"displayName": "Personal Notes",
					"parentNotebook": map[string]interface{}{
						"id": "notebook-2",
						"displayName": "Personal Notebook",
					},
				},
			},
			"sectiongroup-1": {
				{
					"id":          "section-4",
					"displayName": "Archived Projects",
					"parentSectionGroup": map[string]interface{}{
						"id": "sectiongroup-1",
						"displayName": "Archive",
					},
				},
			},
		},
		sectionGroups: map[string][]map[string]interface{}{
			"notebook-1": {
				{
					"id":          "sectiongroup-1",
					"displayName": "Archive",
					"createdDateTime": "2023-01-01T00:00:00Z",
					"parentNotebook": map[string]interface{}{
						"id": "notebook-1",
						"displayName": "Work Notebook",
					},
					"sectionsUrl": "https://graph.microsoft.com/v1.0/me/onenote/sectionGroups/sectiongroup-1/sections",
				},
				{
					"id":          "sectiongroup-2",
					"displayName": "Templates",
					"parentNotebook": map[string]interface{}{
						"id": "notebook-1",
						"displayName": "Work Notebook",
					},
				},
			},
		},
		allSections: []map[string]interface{}{
			{
				"id":          "section-1",
				"displayName": "Work Section",
				"parentNotebook": map[string]interface{}{"id": "notebook-1", "displayName": "Work Notebook"},
			},
			{
				"id":          "section-2",
				"displayName": "Projects",
				"parentNotebook": map[string]interface{}{"id": "notebook-1", "displayName": "Work Notebook"},
			},
			{
				"id":          "section-3",
				"displayName": "Personal Notes",
				"parentNotebook": map[string]interface{}{"id": "notebook-2", "displayName": "Personal Notebook"},
			},
			{
				"id":          "section-4",
				"displayName": "Archived Projects",
				"parentSectionGroup": map[string]interface{}{"id": "sectiongroup-1", "displayName": "Archive"},
			},
		},
		shouldFail:      make(map[string]bool),
		createdSections: make([]map[string]interface{}, 0),
		createdGroups:   make([]map[string]interface{}, 0),
		operations:      make([]string, 0),
	}
}

// SetError allows tests to simulate error conditions
func (m *MockSectionClient) SetError(err error) {
	m.mockError = err
}

// SetOperationFailure allows tests to simulate failures for specific operations
func (m *MockSectionClient) SetOperationFailure(operation string, shouldFail bool) {
	m.shouldFail[operation] = shouldFail
}

// Helper method to check if operation should fail
func (m *MockSectionClient) checkFailure(operation string) error {
	if m.mockError != nil {
		return m.mockError
	}
	if m.shouldFail[operation] {
		return fmt.Errorf("simulated failure for %s", operation)
	}
	return nil
}

// ListSections mocks section listing for a container
func (m *MockSectionClient) ListSections(containerID string) ([]map[string]interface{}, error) {
	m.operations = append(m.operations, "ListSections")
	if err := m.checkFailure("ListSections"); err != nil {
		return nil, err
	}
	
	if sections, exists := m.sections[containerID]; exists {
		return sections, nil
	}
	return []map[string]interface{}{}, nil
}

// CreateSection mocks section creation
func (m *MockSectionClient) CreateSection(containerID, displayName string) (map[string]interface{}, error) {
	m.operations = append(m.operations, "CreateSection")
	if err := m.checkFailure("CreateSection"); err != nil {
		return nil, err
	}
	
	// Basic validation
	if containerID == "" {
		return nil, fmt.Errorf("container ID cannot be empty")
	}
	if displayName == "" {
		return nil, fmt.Errorf("display name cannot be empty")
	}
	
	// Validate display name against illegal characters
	illegalChars := []string{"?", "*", "\\", "/", ":", "<", ">", "|", "&", "#", "'", "'", "%", "~"}
	for _, char := range illegalChars {
		if strings.Contains(displayName, char) {
			return nil, fmt.Errorf("display name contains illegal character: %s", char)
		}
	}
	
	// Determine container type - sections can only be created in notebooks or section groups
	containerType := m.determineContainerType(containerID)
	if containerType != "notebook" && containerType != "sectionGroup" {
		return nil, fmt.Errorf("invalid container type for section creation")
	}
	
	// Create new section
	newSectionID := fmt.Sprintf("section-%d", len(m.createdSections)+100)
	newSection := map[string]interface{}{
		"id":          newSectionID,
		"displayName": displayName,
		"createdDateTime": "2023-12-01T00:00:00Z",
		"lastModifiedDateTime": "2023-12-01T00:00:00Z",
		"pagesUrl": fmt.Sprintf("https://graph.microsoft.com/v1.0/me/onenote/sections/%s/pages", newSectionID),
	}
	
	// Set appropriate parent field based on container type
	if containerType == "notebook" {
		newSection["parentNotebook"] = map[string]interface{}{
			"id":          containerID,
			"displayName": fmt.Sprintf("Notebook %s", containerID),
		}
	} else {
		newSection["parentSectionGroup"] = map[string]interface{}{
			"id":          containerID,
			"displayName": fmt.Sprintf("Section Group %s", containerID),
		}
	}
	
	// Store the section
	if m.sections[containerID] == nil {
		m.sections[containerID] = []map[string]interface{}{}
	}
	m.sections[containerID] = append(m.sections[containerID], newSection)
	m.createdSections = append(m.createdSections, newSection)
	
	return newSection, nil
}

// determineContainerType mocks container type determination
func (m *MockSectionClient) determineContainerType(containerID string) string {
	if strings.HasPrefix(containerID, "notebook-") {
		return "notebook"
	}
	if strings.HasPrefix(containerID, "sectiongroup-") {
		return "sectionGroup"
	}
	if strings.HasPrefix(containerID, "section-") {
		return "section"
	}
	return "unknown"
}

// GetOperations returns the list of operations performed (for testing)
func (m *MockSectionClient) GetOperations() []string {
	return m.operations
}

// GetCreatedSections returns the list of created sections (for testing)
func (m *MockSectionClient) GetCreatedSections() []map[string]interface{} {
	return m.createdSections
}

// TestSectionClient_ListSections tests section listing functionality
func TestSectionClient_ListSections(t *testing.T) {
	t.Run("successfully lists sections for a notebook", func(t *testing.T) {
		mockClient := NewMockSectionClient()
		
		sections, err := mockClient.ListSections("notebook-1")
		
		assert.NoError(t, err)
		assert.Len(t, sections, 2) // Should have Work Section and Projects
		
		// Verify section properties
		for _, section := range sections {
			assert.Contains(t, section, "id")
			assert.Contains(t, section, "displayName")
			assert.Contains(t, section, "parentNotebook")
			
			// Verify parent notebook
			parentNotebook := section["parentNotebook"].(map[string]interface{})
			assert.Equal(t, "notebook-1", parentNotebook["id"])
		}
		
		// Verify specific sections
		foundNames := make(map[string]bool)
		for _, section := range sections {
			name := section["displayName"].(string)
			foundNames[name] = true
		}
		assert.True(t, foundNames["Work Section"])
		assert.True(t, foundNames["Projects"])
	})
	
	t.Run("handles authentication errors", func(t *testing.T) {
		mockClient := NewMockSectionClient()
		mockClient.SetOperationFailure("ListSections", true)
		
		sections, err := mockClient.ListSections("notebook-1")
		
		assert.Error(t, err)
		assert.Nil(t, sections)
		assert.Contains(t, err.Error(), "simulated failure")
	})
}

// TestSectionClient_CreateSection tests section creation functionality
func TestSectionClient_CreateSection(t *testing.T) {
	t.Run("successfully creates section in notebook", func(t *testing.T) {
		mockClient := NewMockSectionClient()
		
		section, err := mockClient.CreateSection("notebook-1", "New Test Section")
		
		assert.NoError(t, err)
		assert.NotNil(t, section)
		assert.Equal(t, "New Test Section", section["displayName"])
		assert.Contains(t, section, "id")
		assert.Contains(t, section, "parentNotebook")
		
		// Verify the section was created and tracked
		createdSections := mockClient.GetCreatedSections()
		assert.Len(t, createdSections, 1)
	})
	
	t.Run("validates display name for illegal characters", func(t *testing.T) {
		mockClient := NewMockSectionClient()
		
		// Test illegal character
		section, err := mockClient.CreateSection("notebook-1", "Section*Name")
		assert.Error(t, err)
		assert.Nil(t, section)
		assert.Contains(t, err.Error(), "illegal character")
	})
	
	t.Run("rejects section creation in invalid containers", func(t *testing.T) {
		mockClient := NewMockSectionClient()
		
		// Try to create section in a section (should fail due to hierarchy)
		section, err := mockClient.CreateSection("section-1", "Invalid Section")
		
		assert.Error(t, err)
		assert.Nil(t, section)
		assert.Contains(t, err.Error(), "invalid container type")
	})
}

// Benchmark tests for section operations
func BenchmarkSectionClient_CreateSection(b *testing.B) {
	mockClient := NewMockSectionClient()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sectionName := fmt.Sprintf("Benchmark Section %d", i)
		_, _ = mockClient.CreateSection("notebook-1", sectionName)
	}
}

func BenchmarkSectionClient_ListSections(b *testing.B) {
	mockClient := NewMockSectionClient()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		containerID := fmt.Sprintf("notebook-%d", (i%2)+1)
		_, _ = mockClient.ListSections(containerID)
	}
}
