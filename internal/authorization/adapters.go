// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package authorization

import (
	"context"
)

// QuickNoteConfigAdapter adapts Config and QuickNoteConfig to the QuickNoteConfig interface
type QuickNoteConfigAdapter struct {
	QuickNote       QuickNoteConfigProvider
	DefaultNotebook string
}

// QuickNoteConfigProvider defines the interface for getting quicknote configuration
type QuickNoteConfigProvider interface {
	GetNotebookName() string
	GetPageName() string
}

// NewQuickNoteConfigAdapter creates a new adapter
func NewQuickNoteConfigAdapter(quickNote QuickNoteConfigProvider, defaultNotebook string) *QuickNoteConfigAdapter {
	return &QuickNoteConfigAdapter{
		QuickNote:       quickNote,
		DefaultNotebook: defaultNotebook,
	}
}

// GetNotebookName returns the quicknote-specific notebook name
func (qca *QuickNoteConfigAdapter) GetNotebookName() string {
	if qca.QuickNote != nil {
		return qca.QuickNote.GetNotebookName()
	}
	return ""
}

// GetDefaultNotebook returns the default notebook name as fallback
func (qca *QuickNoteConfigAdapter) GetDefaultNotebook() string {
	return qca.DefaultNotebook
}

// GetPageName returns the target page name
func (qca *QuickNoteConfigAdapter) GetPageName() string {
	if qca.QuickNote != nil {
		return qca.QuickNote.GetPageName()
	}
	return ""
}

// NotebookCacheAdapter adapts the main NotebookCache to the authorization interface
type NotebookCacheAdapter struct {
	Cache MainNotebookCache
}

// MainNotebookCache defines the interface for the main notebook cache
type MainNotebookCache interface {
	GetNotebook() (map[string]interface{}, bool)
	GetDisplayName() (string, bool)
	GetNotebookID() (string, bool)
	// Add method to get section name if available
	// This might need to be implemented in the main cache
}

// NewNotebookCacheAdapter creates a new cache adapter
func NewNotebookCacheAdapter(cache MainNotebookCache) *NotebookCacheAdapter {
	return &NotebookCacheAdapter{
		Cache: cache,
	}
}

// GetNotebook returns the current notebook
func (nca *NotebookCacheAdapter) GetNotebook() (map[string]interface{}, bool) {
	if nca.Cache != nil {
		return nca.Cache.GetNotebook()
	}
	return nil, false
}

// GetDisplayName returns the notebook display name
func (nca *NotebookCacheAdapter) GetDisplayName() (string, bool) {
	if nca.Cache != nil {
		return nca.Cache.GetDisplayName()
	}
	return "", false
}

// GetNotebookID returns the notebook ID
func (nca *NotebookCacheAdapter) GetNotebookID() (string, bool) {
	if nca.Cache != nil {
		return nca.Cache.GetNotebookID()
	}
	return "", false
}

// GetSectionName returns the section name for a given section ID
func (nca *NotebookCacheAdapter) GetSectionName(sectionID string) (string, bool) {
	if nca.Cache != nil {
		// Try to call GetSectionName if the cache supports it
		// We need to check if the cache has this method using type assertion
		if cacheWithSectionName, ok := nca.Cache.(interface {
			GetSectionName(string) (string, bool)
		}); ok {
			return cacheWithSectionName.GetSectionName(sectionID)
		}
	}
	return "", false
}

// GetSectionNameWithProgress returns the section name with progress notification support
func (nca *NotebookCacheAdapter) GetSectionNameWithProgress(ctx context.Context, sectionID string, mcpServer interface{}, progressToken string, graphClient interface{}) (string, bool) {
	if nca.Cache != nil {
		// Try to call GetSectionNameWithProgress if the cache supports it
		// We need to check if the cache has this method using type assertion
		if cacheWithProgressMethod, ok := nca.Cache.(interface {
			GetSectionNameWithProgress(context.Context, string, interface{}, string, interface{}) (string, bool)
		}); ok {
			return cacheWithProgressMethod.GetSectionNameWithProgress(ctx, sectionID, mcpServer, progressToken, graphClient)
		}
		
		// Fallback to regular GetSectionName if progress method is not available
		if cacheWithSectionName, ok := nca.Cache.(interface {
			GetSectionName(string) (string, bool)
		}); ok {
			return cacheWithSectionName.GetSectionName(sectionID)
		}
	}
	return "", false
}

// GetPageName returns the page name for a given page ID
func (nca *NotebookCacheAdapter) GetPageName(pageID string) (string, bool) {
	if nca.Cache != nil {
		// Try to call GetPageName if the cache supports it
		// We need to check if the cache has this method using type assertion
		if cacheWithPageName, ok := nca.Cache.(interface {
			GetPageName(string) (string, bool)
		}); ok {
			return cacheWithPageName.GetPageName(pageID)
		}
	}
	return "", false
}

// GetPageNameWithProgress returns the page name with progress notification support
func (nca *NotebookCacheAdapter) GetPageNameWithProgress(ctx context.Context, pageID string, mcpServer interface{}, progressToken string, graphClient interface{}) (string, bool) {
	if nca.Cache != nil {
		// Try to call GetPageNameWithProgress if the cache supports it
		// We need to check if the cache has this method using type assertion
		if cacheWithProgressMethod, ok := nca.Cache.(interface {
			GetPageNameWithProgress(context.Context, string, interface{}, string, interface{}) (string, bool)
		}); ok {
			return cacheWithProgressMethod.GetPageNameWithProgress(ctx, pageID, mcpServer, progressToken, graphClient)
		}
		
		// Fallback to regular GetPageName if progress method is not available
		if cacheWithPageName, ok := nca.Cache.(interface {
			GetPageName(string) (string, bool)
		}); ok {
			return cacheWithPageName.GetPageName(pageID)
		}
	}
	return "", false
}