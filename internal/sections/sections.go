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
	"github.com/gebl/onenote-mcp-server/internal/logging"
	"github.com/gebl/onenote-mcp-server/internal/utils"
)

// Context keys for progress notification system
type contextKey string

const (
	mcpServerKey     contextKey = "mcpServer"
	progressTokenKey contextKey = "progressToken"
)

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

	// Send progress notification if available
	c.sendProgressNotification(ctx, "Fetching sections from container: "+containerID)

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

			// Get immediate section groups from the notebook with context
			var allItems []map[string]interface{}
			allItems = append(allItems, directSections...)

			c.sendProgressNotification(ctx, "Fetching section groups from notebook: "+containerID)
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

	c.sendProgressNotification(ctx, "Fetching nested section groups from section group: "+containerID)
	nestedSectionGroups, err := c.ListSectionGroupsWithContext(ctx, containerID)
	if err != nil {
		logging.SectionLogger.Debug("Failed to get nested section groups from section group", "container_id", containerID, "error", err)
	} else {
		// Add section groups to the result (immediate children only)
		allItems = append(allItems, nestedSectionGroups...)
	}

	return allItems, nil
}

// sendProgressNotification sends a progress notification if context contains MCP server info
func (c *SectionClient) sendProgressNotification(ctx context.Context, message string) {
	logging.SectionLogger.Debug("sendProgressNotification called",
		"message", message,
		"has_context", ctx != nil)

	// Extract MCP server and progress token from context
	var mcpServer interface{}
	var progressToken string

	if serverVal := ctx.Value(mcpServerKey); serverVal != nil {
		mcpServer = serverVal
		logging.SectionLogger.Debug("MCP server found in context",
			"message", message,
			"server_type", fmt.Sprintf("%T", mcpServer))
	} else {
		logging.SectionLogger.Debug("No MCP server in context", "message", message)
	}

	if tokenVal := ctx.Value(progressTokenKey); tokenVal != nil {
		progressToken, _ = tokenVal.(string)
		logging.SectionLogger.Debug("Progress token found in context",
			"message", message,
			"progress_token", progressToken)
	} else {
		logging.SectionLogger.Debug("No progress token in context", "message", message)
	}

	// If we have both server and token, send notification
	if mcpServer != nil && progressToken != "" {
		logging.SectionLogger.Debug("Progress notification context fully available - logging for debugging",
			"message", message,
			"progressToken", progressToken,
			"server_available", true)
		// Note: We can't directly call SendNotificationToClient here because we don't have
		// access to the server type. This is logged for debugging purposes.
		// The actual progress notification sending is handled in the NotebookTools layer.
	} else {
		logging.SectionLogger.Debug("Progress notification context incomplete",
			"message", message,
			"has_server", mcpServer != nil,
			"has_token", progressToken != "",
			"progressToken", progressToken)
	}
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

	// Try notebook endpoint first
	notebookURL := fmt.Sprintf("https://graph.microsoft.com/v1.0/me/onenote/notebooks/%s", containerID)
	logging.SectionLogger.Debug("Testing notebook endpoint", "url", notebookURL)

	notebookResp, err := c.Client.MakeAuthenticatedRequest("GET", notebookURL, nil, nil)
	if err != nil {
		logging.SectionLogger.Debug("Notebook endpoint failed", "error", err)
	} else {
		defer notebookResp.Body.Close()
		if notebookResp.StatusCode == 200 {
			logging.SectionLogger.Debug("Container is a notebook")
			return "notebook", nil
		}
	}

	// Try section group endpoint
	sectionGroupURL := fmt.Sprintf("https://graph.microsoft.com/v1.0/me/onenote/sectionGroups/%s", containerID)
	logging.SectionLogger.Debug("Testing section group endpoint", "url", sectionGroupURL)

	sectionGroupResp, err := c.Client.MakeAuthenticatedRequest("GET", sectionGroupURL, nil, nil)
	if err != nil {
		logging.SectionLogger.Debug("Section group endpoint failed", "error", err)
	} else {
		defer sectionGroupResp.Body.Close()
		if sectionGroupResp.StatusCode == 200 {
			logging.SectionLogger.Debug("Container is a section group")
			return "sectionGroup", nil
		}
	}

	// Try section endpoint as well (for completeness)
	sectionURL := fmt.Sprintf("https://graph.microsoft.com/v1.0/me/onenote/sections/%s", containerID)
	logging.SectionLogger.Debug("Testing section endpoint", "url", sectionURL)

	sectionResp, err := c.Client.MakeAuthenticatedRequest("GET", sectionURL, nil, nil)
	if err != nil {
		logging.SectionLogger.Debug("Section endpoint failed", "error", err)
	} else {
		defer sectionResp.Body.Close()
		if sectionResp.StatusCode == 200 {
			logging.SectionLogger.Debug("Container is a section")
			return "section", nil
		}
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
	resp, err := c.Client.MakeAuthenticatedRequest("POST", url, &buf, map[string]string{"Content-Type": "application/json"})
	if err != nil {
		logging.SectionLogger.Debug("API request failed", "error", err)
		return nil, fmt.Errorf("failed to create section in %s: %v", containerType, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		logging.SectionLogger.Debug("API returned status code", "status", resp.StatusCode)
		return nil, fmt.Errorf("failed to create section in %s: HTTP %d", containerType, resp.StatusCode)
	}

	logging.SectionLogger.Debug("Successfully created section", "container_type", containerType)
	return c.processCreateSectionResponse(resp, "CreateSection")
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

	resp, err := c.Client.MakeAuthenticatedRequest("GET", sectionsURL, nil, nil)
	if err != nil {
		logging.SectionLogger.Error("Failed to fetch sections from global endpoint", "error", err)
		return nil, fmt.Errorf("failed to fetch all sections: %v", err)
	}
	defer resp.Body.Close()

	sections, err := c.processSectionsResponse(resp, "ListAllSections")
	if err != nil {
		return nil, err
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
