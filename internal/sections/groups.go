// groups.go - Section Group operations for the Microsoft Graph API client.
//
// This file contains all section group-related operations including listing,
// creating, and managing section groups within OneNote notebooks.
//
// Key Features:
// - List section groups in notebooks or other section groups
// - Create new section groups
// - List sections within section groups
// - Comprehensive container type detection
//
// Operations Supported:
// - ListSectionGroups: List section groups in a container (notebook/section group)
// - CreateSectionGroup: Create a new section group in a container
// - ListSectionsInSectionGroup: List sections within a section group
// - Helper functions for processing responses and validation
//
// Usage Example:
//   sectionClient := sections.NewSectionClient(graphClient)
//   sectionGroups, err := sectionClient.ListSectionGroups(notebookID)
//   if err != nil {
//       logging.SectionLogger.Error("Failed to list section groups", "error", err)
//   }
//
//   newSectionGroup, err := sectionClient.CreateSectionGroup(notebookID, "My New Section Group")
//   if err != nil {
//       logging.SectionLogger.Error("Failed to create section group", "error", err)
//   }

package sections

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gebl/onenote-mcp-server/internal/logging"
	"github.com/gebl/onenote-mcp-server/internal/utils"
)

const (
	containerTypeSection      = "section"
	containerTypeSectionGroup = "sectionGroup"
	containerTypeNotebook     = "notebook"
)

// ListSectionGroups fetches all section groups in a notebook or section group using direct HTTP API calls.
// containerID: ID of the notebook or section group to list section groups from.
// Returns a slice of section group metadata maps and an error, if any.
// Note: Section groups can only be listed from notebooks or other section groups, not from sections.
func (c *SectionClient) ListSectionGroups(containerID string) ([]map[string]interface{}, error) {
	logging.SectionLogger.Info("Listing section groups for container using direct HTTP API", "container_id", containerID)

	// Sanitize and validate the containerID to prevent injection attacks
	sanitizedContainerID, err := c.Client.SanitizeOneNoteID(containerID, "containerID")
	if err != nil {
		return nil, err
	}

	// Determine container type first for better error messages and logic flow
	containerType, err := c.determineContainerType(sanitizedContainerID)
	if err != nil {
		return nil, fmt.Errorf("failed to determine container type: %v", err)
	}

	logging.SectionLogger.Debug("Container type determined", "container_type", containerType)

	// Only allow listing section groups from notebooks or section groups
	// Sections cannot contain section groups according to OneNote hierarchy
	if containerType == containerTypeSection {
		return nil, fmt.Errorf("cannot list section groups from a section. Container ID %s is a section. Section groups can only be listed from notebooks or other section groups", containerID)
	}

	if containerType != containerTypeNotebook && containerType != containerTypeSectionGroup {
		return nil, fmt.Errorf("container ID %s is a %s. Section groups can only be listed from notebooks or section groups", containerID, containerType)
	}

	// Construct the appropriate URL based on container type
	var url string
	if containerType == "notebook" {
		url = fmt.Sprintf("https://graph.microsoft.com/v1.0/me/onenote/notebooks/%s/sectionGroups", sanitizedContainerID)
		logging.SectionLogger.Debug("Using notebook endpoint", "url", url)
	} else { // sectionGroup
		url = fmt.Sprintf("https://graph.microsoft.com/v1.0/me/onenote/sectionGroups/%s/sectionGroups", sanitizedContainerID)
		logging.SectionLogger.Debug("Using section group endpoint", "url", url)
	}

	// Make the API request
	resp, err := c.Client.MakeAuthenticatedRequest("GET", url, nil, nil)
	if err != nil {
		logging.SectionLogger.Debug("API request failed", "error", err)
		return nil, fmt.Errorf("failed to list section groups from %s: %v", containerType, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		logging.SectionLogger.Debug("API returned status code", "status_code", resp.StatusCode)
		return nil, fmt.Errorf("failed to list section groups from %s: HTTP %d", containerType, resp.StatusCode)
	}

	logging.SectionLogger.Debug("Successfully retrieved section groups", "container_type", containerType)
	return c.processSectionGroupsResponse(resp, "ListSectionGroups", containerID)
}

// processSectionGroupsResponse processes the HTTP response for section groups and returns filtered data
func (c *SectionClient) processSectionGroupsResponse(resp *http.Response, operation string, containerID string) ([]map[string]interface{}, error) {
	logging.SectionLogger.Debug("Received response", "status_code", resp.StatusCode, "headers", resp.Header)

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
		return nil, fmt.Errorf("failed to parse section groups response: %v", err)
	}

	// Extract section groups from the response
	sectionGroups, ok := result["value"].([]interface{})
	if !ok {
		logging.SectionLogger.Debug("No value field found in response")
		return nil, fmt.Errorf("no value field found in section groups response")
	}

	// Convert to map[string]interface{} format, preserving all raw data for hierarchical processing
	var resultSectionGroups []map[string]interface{}
	for i, sg := range sectionGroups {
		if sectionGroup, ok := sg.(map[string]interface{}); ok {
			// Extract the section group ID and name for logging
			sectionGroupID, hasID := sectionGroup["id"].(string)
			sectionGroupName, hasName := sectionGroup["displayName"].(string)

			if !hasID {
				logging.SectionLogger.Warn("Section group missing ID", "index", i)
				continue
			}

			if !hasName {
				logging.SectionLogger.Warn("Section group missing display name", "section_group_id", sectionGroupID)
				sectionGroupName = "Unnamed Section Group"
				// Set the displayName in the actual data
				sectionGroup["displayName"] = sectionGroupName
			}

			logging.SectionLogger.Debug("Processing section group", "id", sectionGroupID, "name", sectionGroupName)

			// Preserve the full raw section group data for hierarchical processing
			// This includes: id, displayName, createdDateTime, lastModifiedDateTime, parentNotebook, parentSectionGroup, etc.
			resultSectionGroups = append(resultSectionGroups, sectionGroup)
			logging.SectionLogger.Debug("Added section group to results", "name", sectionGroupName, "id", sectionGroupID)
		} else {
			logging.SectionLogger.Warn("Section group is not a valid map", "index", i)
		}
	}

	logging.SectionLogger.Info("Found total section groups", "count", len(resultSectionGroups))
	return resultSectionGroups, nil
}

// ListSectionGroupsWithContext fetches section groups with progress notification support
func (c *SectionClient) ListSectionGroupsWithContext(ctx context.Context, containerID string) ([]map[string]interface{}, error) {
	logging.SectionLogger.Info("Listing section groups with context", "container_id", containerID)

	// Send progress notification if available
	c.sendProgressNotification(ctx, "Fetching section groups from container: "+containerID)

	// Call the original ListSectionGroups method
	return c.ListSectionGroups(containerID)
}

// ListSectionsInSectionGroup fetches all sections in a section group by sectionGroupID using direct HTTP API calls.
// Returns a slice of section metadata maps and an error, if any.
func (c *SectionClient) ListSectionsInSectionGroup(sectionGroupID string) ([]map[string]interface{}, error) {
	logging.SectionLogger.Info("Listing sections for section group using direct HTTP API", "section_group_id", sectionGroupID)

	// Sanitize and validate the sectionGroupID to prevent injection attacks
	sanitizedSectionGroupID, err := c.Client.SanitizeOneNoteID(sectionGroupID, "sectionGroupID")
	if err != nil {
		return nil, err
	}

	// Construct the URL for listing sections in a section group
	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/me/onenote/sectionGroups/%s/sections", sanitizedSectionGroupID)
	logging.SectionLogger.Debug("Sections in section group URL", "url", url)

	// Make authenticated request
	resp, err := c.Client.MakeAuthenticatedRequest("GET", url, nil, nil)
	if err != nil {
		logging.SectionLogger.Debug("Authenticated request failed", "error", err)
		return nil, err
	}
	defer resp.Body.Close()

	logging.SectionLogger.Debug("Received response", "status_code", resp.StatusCode, "headers", resp.Header)

	// Handle HTTP response
	if errHandle := c.Client.HandleHTTPResponse(resp, "ListSectionsInSectionGroup"); errHandle != nil {
		logging.SectionLogger.Debug("HTTP response handling failed", "error", errHandle)
		return nil, errHandle
	}

	// Read response body
	content, err := c.Client.ReadResponseBody(resp, "ListSectionsInSectionGroup")
	if err != nil {
		logging.SectionLogger.Debug("Failed to read response body", "error", err)
		return nil, err
	}

	logging.SectionLogger.Debug("Response body", "content", string(content))

	// Parse the response JSON
	var result map[string]interface{}
	if err := json.Unmarshal(content, &result); err != nil {
		logging.SectionLogger.Debug("Failed to unmarshal response", "error", err)
		return nil, fmt.Errorf("failed to parse sections in section group response: %v", err)
	}

	// Extract sections from the response
	sections, ok := result["value"].([]interface{})
	if !ok {
		logging.SectionLogger.Debug("No value field found in response")
		return nil, fmt.Errorf("no value field found in sections in section group response")
	}

	// Convert to map[string]interface{} format with only essential information
	var resultSections []map[string]interface{}
	for _, s := range sections {
		if section, ok := s.(map[string]interface{}); ok {
			// Extract only the essential information
			filteredSection := map[string]interface{}{
				"id":   section["id"],
				"name": section["displayName"],
			}

			// Add parent information
			if parentNotebook, exists := section["parentNotebook"].(map[string]interface{}); exists {
				filteredSection["parentNotebook"] = map[string]interface{}{
					"id":   parentNotebook["id"],
					"name": parentNotebook["displayName"],
				}
			}

			if parentSectionGroup, exists := section["parentSectionGroup"].(map[string]interface{}); exists && parentSectionGroup != nil {
				filteredSection["parentSectionGroup"] = map[string]interface{}{
					"id":   parentSectionGroup["id"],
					"name": parentSectionGroup["displayName"],
				}
			}

			resultSections = append(resultSections, filteredSection)
		}
	}

	logging.SectionLogger.Info("Found total sections in section group", "count", len(resultSections))
	return resultSections, nil
}

// CreateSectionGroup creates a new section group in a notebook or section group using direct HTTP API calls.
// containerID: ID of the notebook or section group to create the section group in.
// displayName: Display name for the new section group.
// Returns the created section group metadata and an error, if any.
// Note: Section groups can only be created inside notebooks or other section groups, not inside sections.
func (c *SectionClient) CreateSectionGroup(containerID, displayName string) (map[string]interface{}, error) {
	logging.SectionLogger.Info("Creating section group in container using direct HTTP API", "display_name", displayName, "container_id", containerID)

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

	// Validate that we can create section groups in this container type
	if containerType == containerTypeSection {
		return nil, fmt.Errorf("cannot create a section group inside a section. Container ID %s is a section. Section groups can only be created inside notebooks or other section groups", containerID)
	}

	if containerType != containerTypeNotebook && containerType != containerTypeSectionGroup {
		return nil, fmt.Errorf("container ID %s is a %s. Section groups can only be created inside notebooks or other section groups", containerID, containerType)
	}

	logging.SectionLogger.Debug("Container type confirmed", "container_type", containerType)

	// Create the request body
	requestBody := map[string]interface{}{
		"displayName": displayName,
	}

	// Marshal the request body to JSON
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		logging.SectionLogger.Debug("Failed to marshal request body", "error", err)
		return nil, fmt.Errorf("failed to marshal request body: %v", err)
	}

	logging.SectionLogger.Debug("Request body", "content", string(jsonBody))

	// Use conditional logic to call the correct endpoint based on container type
	var url string
	var endpointType string
	if containerType == containerTypeNotebook {
		url = fmt.Sprintf("https://graph.microsoft.com/v1.0/me/onenote/notebooks/%s/sectionGroups", sanitizedContainerID)
		endpointType = "notebook"
	} else { // containerTypeSectionGroup
		url = fmt.Sprintf("https://graph.microsoft.com/v1.0/me/onenote/sectionGroups/%s/sectionGroups", sanitizedContainerID)
		endpointType = "section group"
	}

	logging.SectionLogger.Debug("Using correct endpoint based on container type", "url", url, "container_type", containerType, "endpoint_type", endpointType)

	// Make the API request to the correct endpoint
	resp, err := c.Client.MakeAuthenticatedRequest("POST", url, bytes.NewBuffer(jsonBody), map[string]string{"Content-Type": "application/json"})
	if err != nil {
		logging.SectionLogger.Debug("API request failed", "error", err, "endpoint_type", endpointType)
		return nil, fmt.Errorf("failed to create section group in %s: %v", endpointType, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		logging.SectionLogger.Debug("API returned non-201 status", "status_code", resp.StatusCode, "endpoint_type", endpointType)
		return nil, fmt.Errorf("failed to create section group in %s: HTTP %d", endpointType, resp.StatusCode)
	}

	logging.SectionLogger.Debug("Successfully created section group", "endpoint_type", endpointType)
	return c.processCreateSectionGroupResponse(resp, "CreateSectionGroup")
}

// processCreateSectionGroupResponse processes the HTTP response for creating section groups and returns the result
func (c *SectionClient) processCreateSectionGroupResponse(resp *http.Response, operation string) (map[string]interface{}, error) {
	logging.SectionLogger.Debug("Received response", "status_code", resp.StatusCode, "headers", resp.Header)

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
		return nil, fmt.Errorf("failed to parse create section group response: %v", err)
	}

	logging.SectionLogger.Info("Successfully created section group", "id", result["id"])
	return result, nil
}
