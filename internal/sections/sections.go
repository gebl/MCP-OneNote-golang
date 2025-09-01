// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

// sections.go - Section operations for the Microsoft Graph API client.
//
// This file contains all section-related operations including listing,
// creating, and managing sections within OneNote notebooks and section groups.
//
// Key Features:
// - List sections in notebooks, section groups, or sections
// - Create new sections
// - Comprehensive container type detection
//
// Operations Supported:
// - ListSections: List sections in a container (notebook/section group)
// - CreateSection: Create a new section in a container
// - Helper functions for processing responses and validation
//
// Usage Example:
//   sectionClient := sections.NewSectionClient(graphClient)
//   sections, err := sectionClient.ListSections(notebookID)
//   if err != nil {
//       logging.SectionLogger.Error("Failed to list sections", "error", err)
//   }
//
//   newSection, err := sectionClient.CreateSection(notebookID, "My New Section")
//   if err != nil {
//       logging.SectionLogger.Error("Failed to create section", "error", err)
//   }

package sections

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gebl/onenote-mcp-server/internal/graph"
	httputils "github.com/gebl/onenote-mcp-server/internal/http"
	"github.com/gebl/onenote-mcp-server/internal/logging"
	"github.com/gebl/onenote-mcp-server/internal/utils"
)

// Context keys are now imported from utils to avoid duplication
// (removed local constants since they're not used in this package anymore)

// SectionClient provides section-specific operations
type SectionClient struct {
	*graph.Client
}

// NewSectionClient creates a new SectionClient
func NewSectionClient(client *graph.Client) *SectionClient {
	return &SectionClient{Client: client}
}

// ListSections fetches immediate sections and section groups in a notebook or section group by ID using direct HTTP API calls.
// containerID: ID of the notebook or section group to list sections from.
// Returns a slice containing both sections and section groups metadata maps, and an error if any.
func (c *SectionClient) ListSections(containerID string) ([]map[string]interface{}, error) {
	logging.SectionLogger.Info("Listing immediate sections and section groups for container using direct HTTP API", "container_id", containerID)

	// Sanitize and validate the containerID to prevent injection attacks
	sanitizedContainerID, err := c.Client.SanitizeOneNoteID(containerID, "containerID")
	if err != nil {
		return nil, err
	}

	// Determine if this is a notebook or section group by checking the response from both endpoints
	// Try notebook endpoint first
	notebookURL := fmt.Sprintf("https://graph.microsoft.com/v1.0/me/onenote/notebooks/%s/sections", sanitizedContainerID)
	logging.SectionLogger.Debug("Trying notebook URL", "url", notebookURL)

	notebookResp, err := c.Client.MakeAuthenticatedRequest("GET", notebookURL, nil, nil)
	if err != nil {
		logging.SectionLogger.Debug("Notebook endpoint failed, trying section group endpoint", "error", err)
	} else {
		defer notebookResp.Body.Close()
		if notebookResp.StatusCode == 200 {
			logging.SectionLogger.Debug("Successfully found sections in notebook")
			// Get immediate sections from notebook
			directSections, errSections := c.processSectionsResponse(notebookResp, "ListSections")
			if errSections != nil {
				return nil, errSections
			}

			// Get immediate section groups from the notebook
			var allItems []map[string]interface{}
			allItems = append(allItems, directSections...)

			sectionGroups, groupsErr := c.ListSectionGroups(containerID)
			if groupsErr != nil {
				logging.SectionLogger.Debug("Failed to get section groups from notebook", "container_id", containerID, "error", groupsErr)
			} else {
				// Add section groups to the result (immediate children only)
				allItems = append(allItems, sectionGroups...)
			}

			return allItems, nil
		}
	}

	// Try section group endpoint
	sectionGroupURL := fmt.Sprintf("https://graph.microsoft.com/v1.0/me/onenote/sectionGroups/%s/sections", sanitizedContainerID)
	logging.SectionLogger.Debug("Trying section group URL", "url", sectionGroupURL)

	sectionGroupResp, err := c.Client.MakeAuthenticatedRequest("GET", sectionGroupURL, nil, nil)
	if err != nil {
		logging.SectionLogger.Debug("Section group endpoint also failed", "error", err)
		return nil, fmt.Errorf("container ID %s is not a valid notebook or section group ID: %v", containerID, err)
	}
	defer sectionGroupResp.Body.Close()

	if sectionGroupResp.StatusCode != 200 {
		return nil, fmt.Errorf("container ID %s is not a valid notebook or section group ID", containerID)
	}

	logging.SectionLogger.Debug("Successfully found sections in section group")
	// Get immediate sections from section group
	directSections, err := c.processSectionsResponse(sectionGroupResp, "ListSections")
	if err != nil {
		return nil, err
	}

	// Get immediate section groups from this section group
	var allItems []map[string]interface{}
	allItems = append(allItems, directSections...)

	nestedSectionGroups, err := c.ListSectionGroups(containerID)
	if err != nil {
		logging.SectionLogger.Debug("Failed to get nested section groups from section group", "container_id", containerID, "error", err)
	} else {
		// Add section groups to the result (immediate children only)
		allItems = append(allItems, nestedSectionGroups...)
	}

	return allItems, nil
}

// ListSectionsWithContext fetches sections with progress notification support
func (c *SectionClient) ListSectionsWithContext(ctx context.Context, containerID string) ([]map[string]interface{}, error) {
	logging.SectionLogger.Info("Listing sections with context", "container_id", containerID)

	// Note: Progress notifications handled at the NotebookTools layer

	// Sanitize and validate the containerID to prevent injection attacks
	sanitizedContainerID, err := c.Client.SanitizeOneNoteID(containerID, "containerID")
	if err != nil {
		return nil, err
	}

	// Determine if this is a notebook or section group by checking the response from both endpoints
	// Try notebook endpoint first
	notebookURL := fmt.Sprintf("https://graph.microsoft.com/v1.0/me/onenote/notebooks/%s/sections", sanitizedContainerID)
	logging.SectionLogger.Debug("Trying notebook URL", "url", notebookURL)

	// Send progress notification before notebook API call
	c.sendProgressNotification(ctx, "Making notebook sections API call...")

	notebookResp, err := c.Client.MakeAuthenticatedRequest("GET", notebookURL, nil, nil)
	if err != nil {
		logging.SectionLogger.Debug("Notebook endpoint failed, trying section group endpoint", "error", err)
	} else {
		defer notebookResp.Body.Close()
		if notebookResp.StatusCode == 200 {
			logging.SectionLogger.Debug("Successfully found sections in notebook")
			
			// Send progress notification after successful API call
			c.sendProgressNotification(ctx, "Processing notebook sections response...")
			
			// Get immediate sections from notebook
			directSections, errSections := c.processSectionsResponse(notebookResp, "ListSections")
			if errSections != nil {
				return nil, errSections
			}

			// Get immediate section groups from the notebook with context
			var allItems []map[string]interface{}
			allItems = append(allItems, directSections...)

			// Send progress notification before section groups API call
			c.sendProgressNotification(ctx, "Fetching section groups...")
			
			sectionGroups, groupsErr := c.ListSectionGroupsWithContext(ctx, containerID)
			if groupsErr != nil {
				logging.SectionLogger.Debug("Failed to get section groups from notebook", "container_id", containerID, "error", groupsErr)
			} else {
				// Add section groups to the result (immediate children only)
				allItems = append(allItems, sectionGroups...)
			}

			return allItems, nil
		}
	}

	// Try section group endpoint
	sectionGroupURL := fmt.Sprintf("https://graph.microsoft.com/v1.0/me/onenote/sectionGroups/%s/sections", sanitizedContainerID)
	logging.SectionLogger.Debug("Trying section group URL", "url", sectionGroupURL)

	sectionGroupResp, err := c.Client.MakeAuthenticatedRequest("GET", sectionGroupURL, nil, nil)
	if err != nil {
		logging.SectionLogger.Debug("Section group endpoint also failed", "error", err)
		return nil, fmt.Errorf("container ID %s is not a valid notebook or section group ID: %v", containerID, err)
	}
	defer sectionGroupResp.Body.Close()

	if sectionGroupResp.StatusCode != 200 {
		return nil, fmt.Errorf("container ID %s is not a valid notebook or section group ID", containerID)
	}

	logging.SectionLogger.Debug("Successfully found sections in section group")
	// Get immediate sections from section group
	directSections, err := c.processSectionsResponse(sectionGroupResp, "ListSections")
	if err != nil {
		return nil, err
	}

	// Get immediate section groups from this section group with context
	var allItems []map[string]interface{}
	allItems = append(allItems, directSections...)

	// Note: Progress notifications handled at the NotebookTools layer
	nestedSectionGroups, err := c.ListSectionGroupsWithContext(ctx, containerID)
	if err != nil {
		logging.SectionLogger.Debug("Failed to get nested section groups from section group", "container_id", containerID, "error", err)
	} else {
		// Add section groups to the result (immediate children only)
		allItems = append(allItems, nestedSectionGroups...)
	}

	return allItems, nil
}

// ListSectionsWithProgress fetches immediate sections and section groups with progress updates.
// Provides granular progress notifications during API calls to prevent client timeouts.
func (c *SectionClient) ListSectionsWithProgress(containerID string, progressCallback func(progress int, message string)) ([]map[string]interface{}, error) {
	logging.SectionLogger.Info("Listing immediate sections and section groups for container with progress", "container_id", containerID)

	if progressCallback != nil {
		progressCallback(0, "Initializing section listing...")
	}

	// Sanitize and validate the containerID to prevent injection attacks
	sanitizedContainerID, err := c.Client.SanitizeOneNoteID(containerID, "containerID")
	if err != nil {
		return nil, err
	}

	// Determine if this is a notebook or section group by checking the response from both endpoints
	// Try notebook endpoint first
	notebookURL := fmt.Sprintf("https://graph.microsoft.com/v1.0/me/onenote/notebooks/%s/sections", sanitizedContainerID)
	logging.SectionLogger.Debug("Trying notebook URL with progress", "url", notebookURL)

	if progressCallback != nil {
		progressCallback(10, "Checking if container is notebook...")
	}

	notebookResp, err := c.Client.MakeAuthenticatedRequest("GET", notebookURL, nil, nil)
	if err != nil {
		logging.SectionLogger.Debug("Notebook endpoint failed, trying section group endpoint", "error", err)
		if progressCallback != nil {
			progressCallback(20, "Not a notebook, checking if section group...")
		}
	} else {
		defer notebookResp.Body.Close()
		if notebookResp.StatusCode == 200 {
			logging.SectionLogger.Debug("Successfully found sections in notebook")
			if progressCallback != nil {
				progressCallback(30, "Container is notebook, getting sections...")
			}
			
			// Get immediate sections from notebook
			directSections, errSections := c.processSectionsResponse(notebookResp, "ListSectionsWithProgress")
			if errSections != nil {
				return nil, errSections
			}

			if progressCallback != nil {
				progressCallback(60, fmt.Sprintf("Found %d sections, getting section groups...", len(directSections)))
			}

			// Get immediate section groups from the notebook
			var allItems []map[string]interface{}
			allItems = append(allItems, directSections...)

			sectionGroups, groupsErr := c.ListSectionGroupsWithProgress(containerID, func(progress int, message string) {
				// Map section group progress from 60-90%
				adjustedProgress := 60 + (progress * 30 / 100)
				if progressCallback != nil {
					progressCallback(adjustedProgress, fmt.Sprintf("Section groups: %s", message))
				}
			})
			if groupsErr != nil {
				logging.SectionLogger.Debug("Failed to get section groups from notebook", "container_id", containerID, "error", groupsErr)
			} else {
				// Add section groups to the result (immediate children only)
				allItems = append(allItems, sectionGroups...)
			}

			if progressCallback != nil {
				progressCallback(100, fmt.Sprintf("Completed: %d sections and section groups", len(allItems)))
			}

			return allItems, nil
		}
	}

	// Try section group endpoint
	sectionGroupURL := fmt.Sprintf("https://graph.microsoft.com/v1.0/me/onenote/sectionGroups/%s/sections", sanitizedContainerID)
	logging.SectionLogger.Debug("Trying section group URL with progress", "url", sectionGroupURL)

	if progressCallback != nil {
		progressCallback(30, "Getting sections from section group...")
	}

	sectionGroupResp, err := c.Client.MakeAuthenticatedRequest("GET", sectionGroupURL, nil, nil)
	if err != nil {
		logging.SectionLogger.Debug("Section group endpoint also failed", "error", err)
		return nil, fmt.Errorf("container ID %s is not a valid notebook or section group ID: %v", containerID, err)
	}
	defer sectionGroupResp.Body.Close()

	if sectionGroupResp.StatusCode != 200 {
		return nil, fmt.Errorf("container ID %s is not a valid notebook or section group ID", containerID)
	}

	logging.SectionLogger.Debug("Successfully found sections in section group")
	if progressCallback != nil {
		progressCallback(50, "Processing section group sections...")
	}
	
	// Get immediate sections from section group
	directSections, err := c.processSectionsResponse(sectionGroupResp, "ListSectionsWithProgress")
	if err != nil {
		return nil, err
	}

	if progressCallback != nil {
		progressCallback(70, fmt.Sprintf("Found %d sections, getting nested section groups...", len(directSections)))
	}

	// Get immediate section groups from this section group
	var allItems []map[string]interface{}
	allItems = append(allItems, directSections...)

	nestedSectionGroups, err := c.ListSectionGroupsWithProgress(containerID, func(progress int, message string) {
		// Map nested section group progress from 70-95%
		adjustedProgress := 70 + (progress * 25 / 100)
		if progressCallback != nil {
			progressCallback(adjustedProgress, fmt.Sprintf("Nested groups: %s", message))
		}
	})
	if err != nil {
		logging.SectionLogger.Debug("Failed to get nested section groups from section group", "container_id", containerID, "error", err)
	} else {
		// Add section groups to the result (immediate children only)
		allItems = append(allItems, nestedSectionGroups...)
	}

	if progressCallback != nil {
		progressCallback(100, fmt.Sprintf("Completed: %d sections and section groups", len(allItems)))
	}

	return allItems, nil
}


// sendProgressNotification sends a progress notification using the centralized utility
func (c *SectionClient) sendProgressNotification(ctx context.Context, message string) {
	utils.SendContextualMessage(ctx, message, logging.SectionLogger)
}

// processSectionsResponse processes the HTTP response for sections and returns filtered data
func (c *SectionClient) processSectionsResponse(resp *http.Response, operation string) ([]map[string]interface{}, error) {
	logging.SectionLogger.Debug("Received response", "status", resp.StatusCode, "headers", resp.Header)

	// Handle HTTP response
	if err := c.Client.HandleHTTPResponse(resp, operation); err != nil {
		logging.SectionLogger.Debug("HTTP response handling failed", "error", err)
		return nil, err
	}

	// Read response body
	content, err := c.Client.ReadResponseBody(resp, operation)
	if err != nil {
		logging.SectionLogger.Debug("Failed to read response body", "error", err)
		return nil, err
	}

	logging.SectionLogger.Debug("Response body", "content", string(content))

	// Parse the response JSON
	var result map[string]interface{}
	if err := json.Unmarshal(content, &result); err != nil {
		logging.SectionLogger.Debug("Failed to unmarshal response", "error", err)
		return nil, fmt.Errorf("failed to parse sections response: %v", err)
	}

	// Extract sections from the response
	sections, ok := result["value"].([]interface{})
	if !ok {
		logging.SectionLogger.Debug("No value field found in response")
		return nil, fmt.Errorf("no value field found in sections response")
	}

	// Convert to map[string]interface{} format, preserving raw data for hierarchical processing
	var resultSections []map[string]interface{}
	for _, s := range sections {
		if section, ok := s.(map[string]interface{}); ok {
			// Preserve the full raw section data for hierarchical processing
			// This includes: id, displayName, createdDateTime, lastModifiedDateTime, parentNotebook, parentSectionGroup, etc.
			resultSections = append(resultSections, section)
		}
	}

	logging.SectionLogger.Info("Found sections", "count", len(resultSections))

	// Ensure we always return a slice, even if empty
	if resultSections == nil {
		resultSections = []map[string]interface{}{}
	}
	return resultSections, nil
}

// determineContainerType determines if a container ID is a notebook or section group by making test requests
func (c *SectionClient) determineContainerType(containerID string) (string, error) {
	logging.SectionLogger.Debug("Determining container type", "container_id", containerID)

	// Define endpoints to try in order
	endpoints := []httputils.HTTPRequestSpec{
		{
			Method:  "GET",
			URL:     fmt.Sprintf("https://graph.microsoft.com/v1.0/me/onenote/notebooks/%s", containerID),
			Headers: nil,
			Body:    nil,
		},
		{
			Method:  "GET", 
			URL:     fmt.Sprintf("https://graph.microsoft.com/v1.0/me/onenote/sectionGroups/%s", containerID),
			Headers: nil,
			Body:    nil,
		},
		{
			Method:  "GET",
			URL:     fmt.Sprintf("https://graph.microsoft.com/v1.0/me/onenote/sections/%s", containerID),
			Headers: nil,
			Body:    nil,
		},
	}

	containerTypes := []string{"notebook", "sectionGroup", "section"}

	for i, endpoint := range endpoints {
		logging.SectionLogger.Debug("Testing endpoint", "url", endpoint.URL, "expected_type", containerTypes[i])
		
		err := httputils.SafeRequest(
			c.Client.MakeAuthenticatedRequest,
			func(resp *http.Response, operation string) error {
				if resp.StatusCode == 200 {
					logging.SectionLogger.Debug("Container type detected", "type", containerTypes[i])
					return nil // Success case
				}
				return fmt.Errorf("HTTP %d", resp.StatusCode) // Non-200 status
			},
			endpoint.Method, endpoint.URL, endpoint.Body, endpoint.Headers,
			fmt.Sprintf("determineContainerType_%s", containerTypes[i]),
		)
		
		if err == nil {
			// Found the correct type
			return containerTypes[i], nil
		}
		
		logging.SectionLogger.Debug("Endpoint failed", "url", endpoint.URL, "error", err)
	}

	return "", fmt.Errorf("container ID %s is not a valid notebook, section group, or section", containerID)
}


// CreateSection creates a new section in a notebook or section group using direct HTTP API calls.
// containerID: ID of the notebook or section group to create the section in.
// displayName: Display name for the new section.
// Returns the created section metadata and an error, if any.
// Note: Sections can only be created inside notebooks or section groups, not inside other sections.
func (c *SectionClient) CreateSection(containerID, displayName string) (map[string]interface{}, error) {
	logging.SectionLogger.Info("Creating section in container using direct HTTP API", "display_name", displayName, "container_id", containerID)

	// Validate display name
	if err := utils.ValidateDisplayName(displayName); err != nil {
		return nil, err
	}

	// Sanitize and validate the containerID to prevent injection attacks
	sanitizedContainerID, err := c.Client.SanitizeOneNoteID(containerID, "containerID")
	if err != nil {
		return nil, err
	}

	// Determine container type first for better error messages
	containerType, err := c.determineContainerType(sanitizedContainerID)
	if err != nil {
		return nil, fmt.Errorf("failed to determine container type: %v", err)
	}

	// Validate that we can create sections in this container type
	if containerType == "section" {
		return nil, fmt.Errorf("cannot create a section inside another section. Container ID %s is a section. Sections can only be created inside notebooks or section groups", containerID)
	}

	if containerType != "notebook" && containerType != "sectionGroup" {
		return nil, fmt.Errorf("container ID %s is a %s. Sections can only be created inside notebooks or section groups", containerID, containerType)
	}

	logging.SectionLogger.Debug("Container type confirmed", "container_type", containerType)

	// Create the request body
	requestBody := map[string]interface{}{
		"displayName": displayName,
	}

	// Marshal the request body to JSON
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false) // Prevent escaping
	err = encoder.Encode(requestBody)
	if err != nil {
		logging.SectionLogger.Debug("Failed to marshal request body", "error", err)
		return nil, fmt.Errorf("failed to marshal request body: %v", err)
	}

	logging.SectionLogger.Debug("Request body", "body", buf.String())

	// Construct the appropriate URL based on container type
	var url string
	if containerType == "notebook" {
		url = fmt.Sprintf("https://graph.microsoft.com/v1.0/me/onenote/notebooks/%s/sections", sanitizedContainerID)
		logging.SectionLogger.Debug("Using notebook endpoint", "url", url)
	} else { // sectionGroup
		url = fmt.Sprintf("https://graph.microsoft.com/v1.0/me/onenote/sectionGroups/%s/sections", sanitizedContainerID)
		logging.SectionLogger.Debug("Using section group endpoint", "url", url)
	}

	// Make the API request
	var result map[string]interface{}
	err = httputils.SafeRequestWithCustomHandler(
		c.Client.MakeAuthenticatedRequest,
		func(resp *http.Response) error {
			if resp.StatusCode != 201 {
				logging.SectionLogger.Debug("API returned status code", "status", resp.StatusCode)
				return fmt.Errorf("failed to create section in %s: HTTP %d", containerType, resp.StatusCode)
			}
			
			logging.SectionLogger.Debug("Successfully created section", "container_type", containerType)
			createResult, procErr := c.processCreateSectionResponse(resp, "CreateSection")
			if procErr != nil {
				return procErr
			}
			result = createResult
			return nil
		},
		"POST", url, &buf, map[string]string{"Content-Type": "application/json"},
	)
	if err != nil {
		logging.SectionLogger.Debug("API request failed", "error", err)
		return nil, fmt.Errorf("failed to create section in %s: %v", containerType, err)
	}

	return result, nil
}

// processCreateSectionResponse processes the HTTP response for creating sections and returns the result
func (c *SectionClient) processCreateSectionResponse(resp *http.Response, operation string) (map[string]interface{}, error) {
	logging.SectionLogger.Debug("Received response", "status", resp.StatusCode, "headers", resp.Header)

	// Handle HTTP response
	if err := c.Client.HandleHTTPResponse(resp, operation); err != nil {
		logging.SectionLogger.Debug("HTTP response handling failed", "error", err)
		return nil, err
	}

	// Read response body
	content, err := c.Client.ReadResponseBody(resp, operation)
	if err != nil {
		logging.SectionLogger.Debug("Failed to read response body", "error", err)
		return nil, err
	}

	logging.SectionLogger.Debug("Response body", "content", string(content))

	// Parse the response JSON
	var result map[string]interface{}
	if err := json.Unmarshal(content, &result); err != nil {
		logging.SectionLogger.Debug("Failed to unmarshal response", "error", err)
		return nil, fmt.Errorf("failed to parse create section response: %v", err)
	}

	// Validate the response contains required fields
	sectionID, hasID := result["id"].(string)
	if !hasID {
		logging.SectionLogger.Error("Created section response missing ID field")
		return nil, fmt.Errorf("created section response missing ID field")
	}

	sectionName, hasName := result["displayName"].(string)
	if !hasName {
		logging.SectionLogger.Warn("Created section missing display name", "section_id", sectionID)
		sectionName = "Unnamed Section"
	}

	logging.SectionLogger.Debug("Successfully created section", "section_id", sectionID, "section_name", sectionName)
	logging.SectionLogger.Info("Successfully created section", "section_id", sectionID)

	return result, nil
}

// ListAllSections fetches all sections across all notebooks using the global sections endpoint
// This is equivalent to https://graph.microsoft.com/v1.0/me/onenote/sections?$select=displayName,id
func (c *SectionClient) ListAllSections() ([]map[string]interface{}, error) {
	logging.SectionLogger.Info("Listing all sections across all notebooks using global endpoint")

	// Use the global sections endpoint with select query to get displayName and id
	sectionsURL := "https://graph.microsoft.com/v1.0/me/onenote/sections?$select=displayName,id"
	logging.SectionLogger.Debug("Fetching from global sections endpoint", "url", sectionsURL)

	var sections []map[string]interface{}
	err := httputils.SafeRequestWithCustomHandler(
		c.Client.MakeAuthenticatedRequest,
		func(resp *http.Response) error {
			if err := c.Client.HandleHTTPResponse(resp, "ListAllSections"); err != nil {
				return err
			}
			sectionsResult, procErr := c.processSectionsResponse(resp, "ListAllSections")
			if procErr != nil {
				return procErr
			}
			sections = sectionsResult
			return nil
		},
		"GET", sectionsURL, nil, nil,
	)
	if err != nil {
		logging.SectionLogger.Error("Failed to fetch sections from global endpoint", "error", err)
		return nil, fmt.Errorf("failed to fetch all sections: %v", err)
	}

	logging.SectionLogger.Info("Successfully listed all sections", "sections_count", len(sections))
	return sections, nil
}

// ListAllSectionsWithContext fetches all sections with progress notification support
func (c *SectionClient) ListAllSectionsWithContext(ctx context.Context) ([]map[string]interface{}, error) {
	logging.SectionLogger.Info("Listing all sections with context")

	// Send progress notification if available
	c.sendProgressNotification(ctx, "Fetching all sections from global endpoint...")

	return c.ListAllSections()
}

// GetSectionByID fetches a specific section by its ID
func (c *SectionClient) GetSectionByID(sectionID string) (map[string]interface{}, error) {
	logging.SectionLogger.Info("Getting section by ID", "section_id", sectionID)

	// Sanitize and validate the sectionID to prevent injection attacks
	sanitizedSectionID, err := c.Client.SanitizeOneNoteID(sectionID, "sectionID")
	if err != nil {
		return nil, err
	}

	// Use the specific section endpoint
	sectionURL := fmt.Sprintf("https://graph.microsoft.com/v1.0/me/onenote/sections/%s", sanitizedSectionID)
	logging.SectionLogger.Debug("Fetching section details", "url", sectionURL)

	content, err := httputils.SafeRequestWithBody(
		c.Client.MakeAuthenticatedRequest,
		c.Client.HandleHTTPResponse,
		c.Client.ReadResponseBody,
		"GET", sectionURL, nil, nil,
		"GetSectionByID",
	)
	if err != nil {
		logging.SectionLogger.Error("Failed to fetch section by ID", "section_id", sectionID, "error", err)
		return nil, fmt.Errorf("failed to fetch section %s: %v", sectionID, err)
	}

	logging.SectionLogger.Debug("Response body", "content", string(content))

	// Parse the response JSON
	var result map[string]interface{}
	if err := json.Unmarshal(content, &result); err != nil {
		logging.SectionLogger.Debug("Failed to unmarshal response", "error", err)
		return nil, fmt.Errorf("failed to parse section response: %v", err)
	}

	// Validate the response contains required fields
	if _, hasID := result["id"].(string); !hasID {
		logging.SectionLogger.Error("Section response missing ID field")
		return nil, fmt.Errorf("section response missing ID field")
	}

	if displayName, hasName := result["displayName"].(string); hasName {
		logging.SectionLogger.Debug("Successfully retrieved section", "section_id", sectionID, "section_name", displayName)
	} else {
		logging.SectionLogger.Debug("Successfully retrieved section (no display name)", "section_id", sectionID)
	}

	return result, nil
}

// ResolveSectionNotebook resolves which notebook owns a section by its ID
func (c *SectionClient) ResolveSectionNotebook(ctx context.Context, sectionID string) (notebookID string, notebookName string, err error) {
	logging.SectionLogger.Debug("Resolving notebook ownership for section",
		"section_id", sectionID)

	// Sanitize and validate the sectionID to prevent injection attacks
	sanitizedSectionID, err := c.Client.SanitizeOneNoteID(sectionID, "sectionID")
	if err != nil {
		return "", "", fmt.Errorf("invalid section ID: %v", err)
	}

	// Use Graph API to get section metadata including parent notebook
	// We use $expand to get the parent notebook information in the same call
	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/me/onenote/sections/%s?$expand=parentNotebook", sanitizedSectionID)
	
	response, err := c.Client.MakeAuthenticatedRequest("GET", url, nil, nil)
	if err != nil {
		logging.SectionLogger.Error("Failed to resolve section notebook via Graph API",
			"section_id", sectionID,
			"error", err.Error())
		return "", "", fmt.Errorf("failed to resolve section notebook: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		logging.SectionLogger.Error("Graph API returned error when resolving section notebook",
			"section_id", sectionID,
			"status_code", response.StatusCode)
		return "", "", fmt.Errorf("Graph API error %d when resolving section notebook", response.StatusCode)
	}

	var sectionInfo struct {
		ID             string `json:"id"`
		DisplayName    string `json:"displayName"`
		ParentNotebook *struct {
			ID          string `json:"id"`
			DisplayName string `json:"displayName"`
		} `json:"parentNotebook"`
	}

	if err := json.NewDecoder(response.Body).Decode(&sectionInfo); err != nil {
		logging.SectionLogger.Error("Failed to decode section notebook resolution response",
			"section_id", sectionID,
			"error", err.Error())
		return "", "", fmt.Errorf("failed to decode Graph API response: %w", err)
	}

	// Validate that we got the parent notebook information
	if sectionInfo.ParentNotebook == nil {
		logging.SectionLogger.Error("Section response missing parent notebook information",
			"section_id", sectionID)
		return "", "", fmt.Errorf("unable to determine parent notebook for section %s", sectionID)
	}

	notebookID = sectionInfo.ParentNotebook.ID
	notebookName = sectionInfo.ParentNotebook.DisplayName

	logging.SectionLogger.Debug("Successfully resolved section notebook ownership",
		"section_id", sectionID,
		"section_name", sectionInfo.DisplayName,
		"notebook_id", notebookID,
		"notebook_name", notebookName)

	return notebookID, notebookName, nil
}
