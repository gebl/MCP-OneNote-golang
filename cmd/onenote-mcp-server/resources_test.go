// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package main

import (
	"testing"
)

func TestExtractNotebookIdFromURI(t *testing.T) {
	testCases := []struct {
		name       string
		uri        string
		expectedID string
	}{
		{
			name:       "Valid notebook URI",
			uri:        "onenote://notebooks/1-abc123def456",
			expectedID: "1-abc123def456",
		},
		{
			name:       "Valid notebook URI with complex ID",
			uri:        "onenote://notebooks/1-abc123def456-789ghi",
			expectedID: "1-abc123def456-789ghi",
		},
		{
			name:       "Invalid URI - missing notebooks",
			uri:        "onenote://sections/1-abc123def456",
			expectedID: "",
		},
		{
			name:       "Invalid URI - missing ID",
			uri:        "onenote://notebooks/",
			expectedID: "",
		},
		{
			name:       "Invalid URI - wrong scheme",
			uri:        "http://notebooks/1-abc123def456",
			expectedID: "",
		},
		{
			name:       "Empty URI",
			uri:        "",
			expectedID: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractNotebookIDFromURI(tc.uri)
			if result != tc.expectedID {
				t.Errorf("Expected ID '%s', got '%s'", tc.expectedID, result)
			}
		})
	}
}

func TestExtractNotebookNameFromURI(t *testing.T) {
	testCases := []struct {
		name         string
		uri          string
		expectedName string
	}{
		{
			name:         "Valid notebook name URI",
			uri:          "onenote://notebooks/Work",
			expectedName: "Work",
		},
		{
			name:         "Valid notebook name URI with spaces",
			uri:          "onenote://notebooks/My%20Personal%20Notebook",
			expectedName: "My Personal Notebook",
		},
		{
			name:         "Invalid URI - with sections path",
			uri:          "onenote://notebooks/Work/sections",
			expectedName: "Work",
		},
		{
			name:         "Invalid URI - missing name",
			uri:          "onenote://notebooks/",
			expectedName: "",
		},
		{
			name:         "Invalid URI - wrong scheme",
			uri:          "http://notebooks/name/Work",
			expectedName: "",
		},
		{
			name:         "Empty URI",
			uri:          "",
			expectedName: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractNotebookNameFromURI(tc.uri)
			if result != tc.expectedName {
				t.Errorf("Expected name '%s', got '%s'", tc.expectedName, result)
			}
		})
	}
}
