// notebooks.go - Notebook operations for the Microsoft Graph API client.
//
// This file contains all notebook-related operations including listing notebooks,
// searching within notebooks, and other notebook management functions.
//
// Key Features:
// - List all OneNote notebooks for the authenticated user with pagination
// - Search pages within notebooks with recursive traversal
// - Comprehensive notebook structure exploration
// - Automatic pagination handling for large notebook collections
//
// Operations Supported:
// - ListNotebooks: List all notebooks with automatic pagination
// - SearchPages: Search pages by title within a specific notebook
// - searchPagesRecursively: Internal function for recursive search
// - extractNotebookList: Helper function to extract notebook data
//
// Usage Example:
//   notebooks, err := graphClient.ListNotebooks()
//   if err != nil {
//       logging.NotebookLogger.Error("Failed to list notebooks", "error", err)
//   }
//
//   results, err := graphClient.SearchPages("meeting", notebookID)
//   if err != nil {
//       logging.NotebookLogger.Error("Failed to search pages", "error", err)
//   }

package notebooks

import (
	"context"
	"fmt"
	"strings"

	abstractions "github.com/microsoft/kiota-abstractions-go"
	msgraphmodels "github.com/microsoftgraph/msgraph-sdk-go/models"

	"github.com/gebl/onenote-mcp-server/internal/graph"
	"github.com/gebl/onenote-mcp-server/internal/logging"
	"github.com/gebl/onenote-mcp-server/internal/pages"
	"github.com/gebl/onenote-mcp-server/internal/sections"
)

// NotebookClient provides notebook operations for the Graph API client
type NotebookClient struct {
	*graph.Client
}

// NewNotebookClient creates a new notebook client
func NewNotebookClient(client *graph.Client) *NotebookClient {
	return &NotebookClient{Client: client}
}

// ListNotebooks lists all OneNote notebooks for the authenticated user.
// Returns an array of notebook objects with ID and display name.
// This function handles pagination using the SDK's nextLink mechanism.
func (c *NotebookClient) ListNotebooks() ([]map[string]interface{}, error) {
	logging.NotebookLogger.Info("Listing notebooks using Microsoft Graph SDK with paging")

	if c.TokenManager != nil && c.TokenManager.IsExpired() {
		logging.NotebookLogger.Debug("Token is expired, attempting refresh before SDK call")
		if err := c.RefreshTokenIfNeeded(); err != nil {
			logging.NotebookLogger.Debug("Failed to refresh token", "error", err)
			return nil, fmt.Errorf("token expired and refresh failed: %v", err)
		}
	}

	ctx := context.Background()
	result, err := c.GraphClient.Me().Onenote().Notebooks().Get(ctx, nil)
	logging.NotebookLogger.Debug("SDK result", "result", result)
	if err != nil {
		if strings.Contains(err.Error(), "JWT") || strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "403") {
			logging.NotebookLogger.Debug("Auth error detected in SDK call, attempting token refresh")
			if c.TokenManager != nil && c.OAuthConfig != nil {
				if refreshErr := c.RefreshTokenIfNeeded(); refreshErr != nil {
					logging.NotebookLogger.Debug("Token refresh failed", "error", refreshErr)
					return nil, fmt.Errorf("authentication failed and token refresh failed: %v", refreshErr)
				}
				result, err = c.GraphClient.Me().Onenote().Notebooks().Get(ctx, nil)
				if err != nil {
					return nil, err
				}
			} else {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	var notebooks []map[string]interface{}
	if result != nil {
		notebooks = append(notebooks, extractNotebookList(result)...)
	}
	nextLink := result.GetOdataNextLink()
	for nextLink != nil {
		logging.NotebookLogger.Debug("Fetching next page of notebooks", "next_link", *nextLink)
		requestInfo := abstractions.NewRequestInformation()
		requestInfo.UrlTemplate = *nextLink
		requestInfo.Method = abstractions.GET
		resp, err := c.GraphClient.GetAdapter().Send(ctx, requestInfo, msgraphmodels.CreateNotebookCollectionResponseFromDiscriminatorValue, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch next page of notebooks: %v", err)
		}
		pageResp := resp.(msgraphmodels.NotebookCollectionResponseable)
		notebooks = append(notebooks, extractNotebookList(pageResp)...)
		nextLink = pageResp.GetOdataNextLink()
	}
	logging.NotebookLogger.Debug("Parsed notebooks", "notebooks", notebooks)
	logging.NotebookLogger.Info("Found notebooks", "count", len(notebooks))

	// Ensure we always return a slice, even if empty
	if notebooks == nil {
		notebooks = []map[string]interface{}{}
	}
	return notebooks, nil
}

// ListNotebooksDetailed lists all OneNote notebooks for the authenticated user with full metadata.
// Returns an array of notebook objects with comprehensive details including timestamps, links, ownership, and metadata.
// This function handles pagination using the SDK's nextLink mechanism.
func (c *NotebookClient) ListNotebooksDetailed() ([]map[string]interface{}, error) {
	logging.NotebookLogger.Info("Listing notebooks with detailed information using Microsoft Graph SDK with paging")

	if c.TokenManager != nil && c.TokenManager.IsExpired() {
		logging.NotebookLogger.Debug("Token is expired, attempting refresh before SDK call")
		if err := c.RefreshTokenIfNeeded(); err != nil {
			logging.NotebookLogger.Debug("Failed to refresh token", "error", err)
			return nil, fmt.Errorf("token expired and refresh failed: %v", err)
		}
	}

	ctx := context.Background()
	result, err := c.GraphClient.Me().Onenote().Notebooks().Get(ctx, nil)
	logging.NotebookLogger.Debug("SDK result for detailed notebooks", "result", result)
	if err != nil {
		if strings.Contains(err.Error(), "JWT") || strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "403") {
			logging.NotebookLogger.Debug("Auth error detected in SDK call, attempting token refresh")
			if c.TokenManager != nil && c.OAuthConfig != nil {
				if refreshErr := c.RefreshTokenIfNeeded(); refreshErr != nil {
					logging.NotebookLogger.Debug("Token refresh failed", "error", refreshErr)
					return nil, fmt.Errorf("authentication failed and token refresh failed: %v", refreshErr)
				}
				result, err = c.GraphClient.Me().Onenote().Notebooks().Get(ctx, nil)
				if err != nil {
					return nil, err
				}
			} else {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	var notebooks []map[string]interface{}
	if result != nil {
		notebooks = append(notebooks, extractDetailedNotebookList(result)...)
	}
	nextLink := result.GetOdataNextLink()
	for nextLink != nil {
		logging.NotebookLogger.Debug("Fetching next page of detailed notebooks", "next_link", *nextLink)
		requestInfo := abstractions.NewRequestInformation()
		requestInfo.UrlTemplate = *nextLink
		requestInfo.Method = abstractions.GET
		resp, err := c.GraphClient.GetAdapter().Send(ctx, requestInfo, msgraphmodels.CreateNotebookCollectionResponseFromDiscriminatorValue, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch next page of detailed notebooks: %v", err)
		}
		pageResp := resp.(msgraphmodels.NotebookCollectionResponseable)
		notebooks = append(notebooks, extractDetailedNotebookList(pageResp)...)
		nextLink = pageResp.GetOdataNextLink()
	}
	logging.NotebookLogger.Debug("Parsed detailed notebooks", "notebooks", notebooks)
	logging.NotebookLogger.Info("Found detailed notebooks", "count", len(notebooks))

	// Ensure we always return a slice, even if empty
	if notebooks == nil {
		notebooks = []map[string]interface{}{}
	}
	return notebooks, nil
}

// GetDetailedNotebookByName retrieves comprehensive notebook information by display name.
// Returns all available attributes including timestamps, links, ownership, and metadata.
func (c *NotebookClient) GetDetailedNotebookByName(notebookName string) (map[string]interface{}, error) {
	logging.NotebookLogger.Info("Getting detailed notebook information by name", "notebook_name", notebookName)

	if c.TokenManager != nil && c.TokenManager.IsExpired() {
		logging.NotebookLogger.Debug("Token is expired, attempting refresh before SDK call")
		if err := c.RefreshTokenIfNeeded(); err != nil {
			logging.NotebookLogger.Debug("Failed to refresh token", "error", err)
			return nil, fmt.Errorf("token expired and refresh failed: %v", err)
		}
	}

	ctx := context.Background()
	result, err := c.GraphClient.Me().Onenote().Notebooks().Get(ctx, nil)
	if err != nil {
		if strings.Contains(err.Error(), "JWT") || strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "403") {
			logging.NotebookLogger.Debug("Auth error detected in SDK call, attempting token refresh")
			if c.TokenManager != nil && c.OAuthConfig != nil {
				if refreshErr := c.RefreshTokenIfNeeded(); refreshErr != nil {
					logging.NotebookLogger.Debug("Token refresh failed", "error", refreshErr)
					return nil, fmt.Errorf("authentication failed and token refresh failed: %v", refreshErr)
				}
				result, err = c.GraphClient.Me().Onenote().Notebooks().Get(ctx, nil)
				if err != nil {
					return nil, err
				}
			} else {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	// Process all pages of results to find the notebook by name
	var allNotebooks []msgraphmodels.Notebookable
	if result != nil {
		allNotebooks = append(allNotebooks, result.GetValue()...)
	}

	nextLink := result.GetOdataNextLink()
	for nextLink != nil {
		logging.NotebookLogger.Debug("Fetching next page of notebooks for detailed search", "next_link", *nextLink)
		requestInfo := abstractions.NewRequestInformation()
		requestInfo.UrlTemplate = *nextLink
		requestInfo.Method = abstractions.GET
		resp, err := c.GraphClient.GetAdapter().Send(ctx, requestInfo, msgraphmodels.CreateNotebookCollectionResponseFromDiscriminatorValue, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch next page of notebooks: %v", err)
		}
		pageResp := resp.(msgraphmodels.NotebookCollectionResponseable)
		allNotebooks = append(allNotebooks, pageResp.GetValue()...)
		nextLink = pageResp.GetOdataNextLink()
	}

	// Search for the notebook by name (case-insensitive)
	for _, nb := range allNotebooks {
		if nb.GetDisplayName() != nil && strings.EqualFold(*nb.GetDisplayName(), notebookName) {
			logging.NotebookLogger.Debug("Found notebook by name", "notebook_name", notebookName, "id", *nb.GetId())

			// Extract detailed information for this notebook
			detailedInfo := extractDetailedNotebookInfo(nb)

			logging.NotebookLogger.Info("Retrieved detailed notebook information", "notebook_name", notebookName, "attributes_count", len(detailedInfo))
			return detailedInfo, nil
		}
	}

	return nil, fmt.Errorf("notebook with name '%s' not found", notebookName)
}

// extractNotebookList extracts notebook info from a NotebookCollectionResponseable
func extractNotebookList(result msgraphmodels.NotebookCollectionResponseable) []map[string]interface{} {
	var notebooks []map[string]interface{}
	for _, nb := range result.GetValue() {
		m := map[string]interface{}{}
		if nb.GetId() != nil {
			m["notebookId"] = *nb.GetId()
		}
		if nb.GetDisplayName() != nil {
			m["displayName"] = *nb.GetDisplayName()
		}
		notebooks = append(notebooks, m)
	}
	return notebooks
}

// extractDetailedNotebookList extracts comprehensive notebook info with all available attributes
func extractDetailedNotebookList(result msgraphmodels.NotebookCollectionResponseable) []map[string]interface{} {
	var notebooks []map[string]interface{}
	for _, nb := range result.GetValue() {
		m := extractDetailedNotebookInfo(nb)
		notebooks = append(notebooks, m)
	}
	return notebooks
}

// extractDetailedNotebookInfo extracts all available attributes from a single notebook
func extractDetailedNotebookInfo(nb msgraphmodels.Notebookable) map[string]interface{} {
	m := map[string]interface{}{}

	// Basic properties
	if nb.GetId() != nil {
		m["id"] = *nb.GetId()
		m["notebookId"] = *nb.GetId() // Keep backward compatibility
	}
	if nb.GetDisplayName() != nil {
		m["displayName"] = *nb.GetDisplayName()
	}

	// Timestamps
	if nb.GetCreatedDateTime() != nil {
		m["createdDateTime"] = nb.GetCreatedDateTime().Format("2006-01-02T15:04:05Z")
	}
	if nb.GetLastModifiedDateTime() != nil {
		m["lastModifiedDateTime"] = nb.GetLastModifiedDateTime().Format("2006-01-02T15:04:05Z")
	}

	// Metadata properties
	if nb.GetIsDefault() != nil {
		m["isDefault"] = *nb.GetIsDefault()
	}
	if nb.GetIsShared() != nil {
		m["isShared"] = *nb.GetIsShared()
	}
	if nb.GetUserRole() != nil {
		m["userRole"] = nb.GetUserRole().String()
	}

	// Note: Links and API endpoints are excluded from the response

	// Identity information
	if createdBy := nb.GetCreatedBy(); createdBy != nil {
		createdByMap := extractIdentitySet(createdBy)
		if len(createdByMap) > 0 {
			m["createdBy"] = createdByMap
		}
	}
	if lastModifiedBy := nb.GetLastModifiedBy(); lastModifiedBy != nil {
		lastModifiedByMap := extractIdentitySet(lastModifiedBy)
		if len(lastModifiedByMap) > 0 {
			m["lastModifiedBy"] = lastModifiedByMap
		}
	}

	return m
}

// extractIdentitySet extracts identity information from an IdentitySet
func extractIdentitySet(identitySet msgraphmodels.IdentitySetable) map[string]interface{} {
	m := map[string]interface{}{}

	if user := identitySet.GetUser(); user != nil {
		userMap := map[string]interface{}{}
		if user.GetId() != nil {
			userMap["id"] = *user.GetId()
		}
		if user.GetDisplayName() != nil {
			userMap["displayName"] = *user.GetDisplayName()
		}
		if len(userMap) > 0 {
			m["user"] = userMap
		}
	}

	if application := identitySet.GetApplication(); application != nil {
		appMap := map[string]interface{}{}
		if application.GetId() != nil {
			appMap["id"] = *application.GetId()
		}
		if application.GetDisplayName() != nil {
			appMap["displayName"] = *application.GetDisplayName()
		}
		if len(appMap) > 0 {
			m["application"] = appMap
		}
	}

	if device := identitySet.GetDevice(); device != nil {
		deviceMap := map[string]interface{}{}
		if device.GetId() != nil {
			deviceMap["id"] = *device.GetId()
		}
		if device.GetDisplayName() != nil {
			deviceMap["displayName"] = *device.GetDisplayName()
		}
		if len(deviceMap) > 0 {
			m["device"] = deviceMap
		}
	}

	return m
}

// SearchPages searches for pages by title within a specific notebook.
// This function recursively searches through all sections and section groups.
// Returns an array of matching pages with section context information.
//
// Parameters:
// - query: the search string to match in page titles (case-insensitive)
// - notebookID: the ID of the notebook to search within
//
// Returns:
// - pageId: unique identifier for the page
// - pageTitle: title of the page
// - sectionId: ID of the section containing the page
// - sectionName: name of the section containing the page
// - sectionPath: full hierarchy path (e.g., "Notebook/Section Group/Section")
func (c *NotebookClient) SearchPages(query string, notebookID string) ([]map[string]interface{}, error) {
	logging.NotebookLogger.Debug("Starting SearchPages", "query", query, "notebook_id", notebookID)
	logging.NotebookLogger.Info("Searching pages", "query", query, "notebook_id", notebookID)

	// Check token status
	if c.TokenManager != nil {
		logging.NotebookLogger.Debug("Token manager available, checking token expiration")
		if c.TokenManager.IsExpired() {
			logging.NotebookLogger.Debug("Token is expired, attempting refresh before SDK call")
			if err := c.RefreshTokenIfNeeded(); err != nil {
				logging.NotebookLogger.Debug("Failed to refresh token", "error", err)
				return nil, fmt.Errorf("token expired and refresh failed: %v", err)
			}
			logging.NotebookLogger.Debug("Token refresh completed successfully")
		} else {
			logging.NotebookLogger.Debug("Token is still valid, no refresh needed")
		}
	} else {
		logging.NotebookLogger.Debug("No token manager available, using static token")
	}

	// Sanitize the notebook ID
	sanitizedNotebookID, err := c.Client.SanitizeOneNoteID(notebookID, "notebookID")
	if err != nil {
		logging.NotebookLogger.Debug("Invalid notebook ID", "error", err)
		return nil, err
	}

	// First, get all sections in the notebook
	logging.NotebookLogger.Debug("Fetching sections for notebook", "notebook_id", sanitizedNotebookID)
	sections, err := c.ListSections(sanitizedNotebookID)
	if err != nil {
		logging.NotebookLogger.Debug("Failed to fetch sections", "error", err)
		return nil, fmt.Errorf("failed to fetch sections for notebook %s: %v", notebookID, err)
	}

	logging.NotebookLogger.Debug("Found sections in notebook", "count", len(sections))

	// Recursively search all sections and section groups
	var allPages []map[string]interface{}

	// Start recursive search from the notebook
	notebookPages, err := c.searchPagesRecursively(sanitizedNotebookID, query, "notebook", "")
	if err != nil {
		logging.NotebookLogger.Debug("Failed to search notebook recursively", "error", err)
		return nil, err
	}

	allPages = append(allPages, notebookPages...)
	logging.NotebookLogger.Debug("Found total pages in notebook search", "count", len(allPages))

	logging.NotebookLogger.Info("Found pages matching query", "count", len(allPages), "query", query, "notebook_id", notebookID)

	return allPages, nil
}

// searchPagesRecursively searches for pages recursively through sections and section groups.
// This function traverses the entire container hierarchy, searching all sections and
// recursively exploring section groups to find pages that match the query.
//
// Parameters:
// - containerID: the ID of the container (notebook or section group) to search
// - query: the search string to match in page titles (case-insensitive)
// - containerType: human-readable type for logging ("notebook" or "section group")
// - parentPath: the hierarchy path leading to this container
//
// Returns a slice of page metadata maps with section context and an error, if any.
func (c *NotebookClient) searchPagesRecursively(containerID, query, containerType, parentPath string) ([]map[string]interface{}, error) {
	logging.NotebookLogger.Debug("Searching recursively", "container_type", containerType, "container_id", containerID, "path", parentPath)

	var allPages []map[string]interface{}

	// Get sections in this container
	sections, err := c.ListSections(containerID)
	if err != nil {
		logging.NotebookLogger.Debug("Failed to get sections", "container_type", containerType, "container_id", containerID, "error", err)
		return nil, err
	}

	logging.NotebookLogger.Debug("Found sections", "count", len(sections), "container_type", containerType, "container_id", containerID)

	// Search pages in each section
	for i, section := range sections {
		sectionID, ok := section["id"].(string)
		if !ok {
			logging.NotebookLogger.Debug("Section has no valid ID, skipping", "index", i+1)
			continue
		}

		sectionName, _ := section["displayName"].(string)
		sectionPath := parentPath + "/" + sectionName
		logging.NotebookLogger.Debug("Searching section", "name", sectionName, "id", sectionID, "path", sectionPath)

		// Get all pages in this section
		sectionPages, errPages := c.ListPages(sectionID)
		if errPages != nil {
			logging.NotebookLogger.Debug("Failed to get pages in section", "section_id", sectionID, "error", errPages)
			continue
		}

		logging.NotebookLogger.Debug("Found pages in section", "count", len(sectionPages), "section_name", sectionName)

		// Filter pages by title
		queryLower := strings.ToLower(query)
		for _, page := range sectionPages {
			if title, ok := page["title"].(string); ok {
				titleLower := strings.ToLower(title)
				if strings.Contains(titleLower, queryLower) {
					logging.NotebookLogger.Debug("Page matches query", "title", title, "section", sectionName)
					// Add context information
					page["sectionName"] = sectionName
					page["sectionId"] = sectionID
					page["sectionPath"] = sectionPath
					allPages = append(allPages, page)
				}
			}
		}
	}

	// Get section groups in this container
	sectionGroups, err := c.ListSectionGroups(containerID)
	if err != nil {
		logging.NotebookLogger.Debug("Failed to get section groups", "container_type", containerType, "container_id", containerID, "error", err)
		// Don't return error, just continue without section groups
	} else {
		logging.NotebookLogger.Debug("Found section groups", "count", len(sectionGroups), "container_type", containerType, "container_id", containerID)

		// Recursively search each section group
		for i, sectionGroup := range sectionGroups {
			sectionGroupID, ok := sectionGroup["id"].(string)
			if !ok {
				logging.NotebookLogger.Debug("Section group has no valid ID, skipping", "index", i+1)
				continue
			}

			sectionGroupName, _ := sectionGroup["displayName"].(string)
			sectionGroupPath := parentPath + "/" + sectionGroupName
			logging.NotebookLogger.Debug("Recursively searching section group",
				"section_group_name", sectionGroupName, "section_group_id", sectionGroupID, "section_group_path", sectionGroupPath)

			// Recursively search this section group
			sectionGroupPages, err := c.searchPagesRecursively(sectionGroupID, query, "section group", sectionGroupPath)
			if err != nil {
				logging.NotebookLogger.Debug("Failed to search section group", "section_group_id", sectionGroupID, "error", err)
				continue
			}

			logging.NotebookLogger.Debug("Found matching pages in section group", "count", len(sectionGroupPages), "section_group_name", sectionGroupName)
			allPages = append(allPages, sectionGroupPages...)
		}
	}

	logging.NotebookLogger.Debug("Recursive search completed",
		"container_type", containerType, "container_id", containerID, "pages_count", len(allPages))
	return allPages, nil
}

func (c *NotebookClient) ListSections(containerID string) ([]map[string]interface{}, error) {
	// Delegate to the sections client
	sectionClient := sections.NewSectionClient(c.Client)
	return sectionClient.ListSections(containerID)
}

func (c *NotebookClient) ListSectionGroups(containerID string) ([]map[string]interface{}, error) {
	// Delegate to the sections client
	sectionClient := sections.NewSectionClient(c.Client)
	return sectionClient.ListSectionGroups(containerID)
}

func (c *NotebookClient) ListPages(sectionID string) ([]map[string]interface{}, error) {
	// Delegate to the pages client
	pageClient := pages.NewPageClient(c.Client)
	return pageClient.ListPages(sectionID)
}

// GetDefaultNotebookID returns the ID of the default notebook specified in the config.
// If no default notebook is configured, it returns an error.
func GetDefaultNotebookID(client *graph.Client, config *graph.Config) (string, error) {
	if config == nil || config.NotebookName == "" {
		return "", fmt.Errorf("no default notebook name configured")
	}

	logging.NotebookLogger.Debug("Looking up default notebook", "name", config.NotebookName)

	// Create a notebook client to list notebooks
	notebookClient := NewNotebookClient(client)

	// List all notebooks to find the one with the matching name
	notebooks, err := notebookClient.ListNotebooks()
	if err != nil {
		return "", fmt.Errorf("failed to list notebooks: %v", err)
	}

	// Search for the notebook with the matching name
	for _, notebook := range notebooks {
		if displayName, exists := notebook["displayName"].(string); exists {
			if displayName == config.NotebookName {
				if id, exists := notebook["notebookId"].(string); exists {
					logging.NotebookLogger.Debug("Found default notebook", "name", displayName, "id", id)
					return id, nil
				}
			}
		}
	}

	return "", fmt.Errorf("default notebook '%s' not found", config.NotebookName)
}
