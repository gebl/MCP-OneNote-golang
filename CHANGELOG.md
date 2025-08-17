# Changelog

All notable changes to the OneNote MCP Server project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed
- **listNotebooks Tool**: Fixed blank notebook IDs in response
  - Corrected key mismatch between notebook client (`notebookId`) and tool implementation (`id`)
  - Notebook IDs now properly returned in `listNotebooks` tool response

### Security
- **Git History Sanitization**: Completely removed sensitive configuration data from git history
  - **Credential Purge**: Used `git filter-repo` to remove Azure client ID and other sensitive data from all commits
  - **History Cleanup**: Git repository history now contains no sensitive credentials or tokens
  - **Enhanced .gitignore**: Added comprehensive patterns to prevent future commits of sensitive configuration files
  - **Sanitized Example Config**: Replaced real Azure client ID with placeholder values in example configuration
  - **Documentation Updates**: Updated README.md and documentation to reflect security hardening measures

### Added
- **Major Architecture Refactor**: Complete reorganization of MCP tool structure for better maintainability and performance
  - **Modular Tool Architecture**: Split monolithic `tools.go` into specialized modules (`AuthTools.go`, `NotebookTools.go`, `PageTools.go`)
  - **MCP Resources Support**: Added `resources.go` providing `notebooks` resource for data discovery and client integration
  - **Global Notebook Cache**: Implemented thread-safe notebook cache system with automatic initialization and management
  - **Default Notebook Initialization**: Server automatically selects and caches a default notebook on startup when authenticated
- **Enhanced MCP Capabilities**: Full support for MCP resources specification
  - **Resource Discovery**: Clients can discover available notebooks through MCP resources
  - **Performance Optimizations**: Cached notebook data for fast resource responses
- **Code Organization Improvements**: Better separation of concerns and maintainability
  - **Removed Deprecated Code**: Eliminated `prompts.go` file as prompts functionality is deprecated in current MCP specification
  - **Focused Tool Modules**: Each tool module handles a specific domain (auth, notebooks, pages)
  - **Improved Test Coverage**: Updated test suite to cover new architecture and resource functionality
- **Enhanced Tool Descriptions**: Significantly improved user-friendliness and clarity of all MCP tool descriptions
  - **Centralized Description System**: Created `internal/resources/descriptions.go` for consistent tool description management
  - **User-Friendly Language**: Replaced technical jargon with clear, accessible explanations
  - **HOW-TO Guidance**: Added concrete step-by-step instructions for complex workflows
  - **Prominent Warnings**: Critical requirements now appear at the top with warning symbols
  - **Real-World Examples**: Added folder/file analogies and practical use cases for better comprehension
  - **Workflow Integration**: Clear guidance on how tools work together in common scenarios

### Changed
- **Tool Registration Architecture**: Moved from single-file registration to modular registration system
  - **Specialized Modules**: Tools organized by domain instead of single monolithic file
  - **Better Error Handling**: Each module handles its own error scenarios with appropriate context
  - **Enhanced Logging**: Improved logging with module-specific context and debugging information
- **Server Initialization**: Enhanced startup process with notebook cache initialization
  - **Non-Blocking Startup**: Server starts immediately even without authentication
  - **Automatic Notebook Selection**: Default notebook automatically selected when authentication is available
  - **Configuration Priority**: Respects `ONENOTE_DEFAULT_NOTEBOOK_NAME` configuration for default selection
- **Updated Documentation**: Comprehensive updates to CLAUDE.md reflecting new architecture
  - **Architecture Overview**: Updated to reflect modular tool organization
  - **MCP Features Section**: New section documenting resources capabilities
  - **Testing Strategy**: Updated test documentation to reflect new architecture

### Removed
- **Deprecated Prompts System**: Removed `prompts.go` file as prompts are deprecated in current MCP specification
  - **Focus on Tools**: Resources provide better integration than prompts
  - **Cleaner Architecture**: Removes unused functionality and reduces complexity
  - **Better MCP Compliance**: Aligns with current MCP specification standards
- **Deprecated Section Copy Tool**: Removed `copySection` tool from NotebookTools.go and related functionality
  - **Code Simplification**: Eliminates complex asynchronous operation that was rarely used
  - **Focus on Core Operations**: Streamlines tool set to essential OneNote operations
  - **Better Maintainability**: Reduces codebase complexity and maintenance burden

### Technical Improvements
- **Thread-Safe Caching**: Robust notebook cache implementation with proper concurrency handling
- **Resource Generation**: Dynamic resource generation from live OneNote data
- **Error Recovery**: Enhanced error handling in tool modules with graceful degradation
- **Memory Management**: Improved memory usage through better data structure organization
- **Client Delegation Pattern**: Improved notebooks.go to properly delegate to section and page clients
  - **Better Separation**: Search functionality now properly uses specialized clients
  - **Method Consistency**: Fixed method calls to use proper client interfaces
  - **Test Updates**: Updated test suite to reflect new delegation patterns

### Fixed
- **Method Call Issues**: Fixed notebooks.go to use proper client methods instead of non-existent functions
  - **Search Functionality**: SearchPages now properly delegates to section and page clients
  - **ID Sanitization**: Fixed calls to use Client.SanitizeOneNoteID instead of local methods
  - **Test Compatibility**: Updated test files to work with new client structure
- **Page Extraction Methods**: Fixed pages.go method visibility and structure
  - **Method Visibility**: Made extractPageIDFromResourceLocation a proper client method
  - **Test Updates**: Updated page tests to use proper client methods
  - **Code Consistency**: Ensured all helper methods follow proper Go conventions

### Planned
- **Performance Optimizations**: Enhanced caching and connection pooling
- **Advanced Search Features**: Full-text search across page content
- **Batch Operations**: Support for bulk operations on multiple pages
- **Webhook Support**: Real-time notifications for content changes

## [1.7.0] - 2025-01-21

### Added
- **Enhanced Container Validation and Error Handling**: Improved section and section group operations with better container type detection
  - **Container Type Detection**: All section and section group operations now use `determineContainerType` for upfront validation
  - **Clear Error Messages**: Better error messages that explain exactly why operations fail and what container types are allowed
  - **OneNote Hierarchy Enforcement**: Strict validation ensures operations follow OneNote's container hierarchy rules
  - **Enhanced Debug Logging**: Detailed logging for troubleshooting container type detection and API responses
- **Streamable HTTP Support**: Added new server mode for HTTP-based communication
  - **New Mode**: `-mode=streamable` for streamable HTTP protocol support
  - **Port Configuration**: Added `-port` flag for custom port configuration
  - **Flexible Deployment**: Support for stdio, SSE, and streamable HTTP modes
  - **Documentation**: Comprehensive documentation for all server modes
- **Enhanced Tool Descriptions**: Added failure handling guidance to key tool descriptions
  - **Error Prevention**: Clear instructions to not create new pages as fallbacks when operations fail
  - **Workflow Guidance**: Tools now guide agents to report errors and stop workflows instead of creating fallback pages
  - **Affected Tools**: createPage, updatePageContent, updatePageContentAdvanced, createSection, createSectionGroup, copyPage, movePage, copySection
- **Improved Empty Result Handling**: Enhanced list tools to provide helpful messages when no results are found
  - **User-Friendly Messages**: Clear, informative responses when containers are empty
  - **Better UX**: Prevents confusion when no sections, pages, or notebooks are found
  - **Affected Tools**: listNotebooks, listSections, listPages, listSectionGroups, listSectionsInSectionGroup
- **Modular Code Architecture**: Major refactoring for better maintainability
  - **Separated Concerns**: Split large files into focused modules (notebooks.go, pages.go, sections.go, etc.)
  - **Utility Functions**: Created dedicated utility modules for image processing and validation
  - **HTTP Client**: Extracted HTTP operations into dedicated client module
  - **Better Organization**: Improved code structure and separation of responsibilities

### Changed
- **`listSectionGroups` Tool**: Restructured to only work with notebooks and section groups as containers
  - **Container Restriction**: No longer accepts section IDs as containers (sections cannot contain section groups)
  - **Simplified Logic**: Removed confusing multiple endpoint attempts, now uses targeted API calls
  - **Better Error Messages**: Clear error messages when trying to list section groups from sections
- **`createSection` Tool**: Restructured to only work with notebooks and section groups as containers
  - **Container Restriction**: No longer accepts section IDs as containers (sections cannot contain other sections)
  - **Simplified Logic**: Removed confusing multiple endpoint attempts, now uses targeted API calls
  - **Better Error Messages**: Clear error messages when trying to create sections inside other sections
- **`createSectionGroup` Tool**: Enhanced with better container validation
  - **Container Restriction**: No longer accepts section IDs as containers (sections cannot contain section groups)
  - **Better Error Messages**: Clear error messages when trying to create section groups inside sections
- **Environment Variable Rename**: `ONENOTE_NOTEBOOK_NAME` has been renamed to `ONENOTE_DEFAULT_NOTEBOOK_NAME` for better clarity
  - **Backward Compatibility**: The JSON configuration file still uses `notebook_name` field
  - **Documentation**: Updated all documentation to reflect the new environment variable name
- **Advanced Page Update Documentation**: Added detailed documentation for the `data-tag` attribute (built-in note tags) and absolute positioning (`data-absolute-enabled`, `style`) to the advanced page update tool in both README and API docs. Includes rules, guidelines, and possible values for data-tag, and all requirements for absolute positioning.
- **Advanced Page Update Documentation**: Clarified that `updatePageContentAdvanced` is the preferred way to add, change, or delete parts of a page. Full page updates should only be used for replacing the entire page.
- **Data-id Extraction Guidance**: Updated documentation for both `getPageContent` and `updatePageContentAdvanced` to explain that `data-id` values can be obtained by using the `forUpdate=true` parameter with `getPageContent`.
- **Usage Examples**: Added concrete JSON examples for advanced page updates in both README and API documentation.

### Fixed
- **Wrong ID Issues**: Resolved issues where section and section group operations were returning wrong IDs
  - **Container Type Confusion**: Fixed confusion between different container types in API calls
  - **Parent Container Information**: Improved accuracy of parent container information in responses
  - **API Endpoint Selection**: Fixed incorrect API endpoint selection that was causing wrong results
- **Fixed JSON Marshaling Issues**: Resolved "Cannot convert undefined or null to object" errors
  - **Proper JSON Handling**: Updated stringifyNotebooks and stringifySections to use JSON marshaling
  - **Nil Value Protection**: Added null checks to prevent errors when processing empty results
  - **Consistent Output**: Ensures all list tools return properly formatted JSON responses
  - **Error Recovery**: Fallback to string formatting if JSON marshaling fails

## [1.6.0] - 2025-01-20

### Added
- **MCP Prompts Support**: Comprehensive prompt system providing contextual guidance for OneNote operations
  - **11 Specialized Prompts**: Cover all major OneNote operations with detailed guidance
  - **Contextual Instructions**: Each prompt provides step-by-step instructions and best practices
  - **Workflow Guidance**: Prompts include workflow recommendations for complex operations
  - **Error Handling Advice**: Built-in troubleshooting guidance for common issues
  - **Template Support**: Structured prompts for creating different types of content
- **Prompt Categories**:
  - **Navigation**: `explore-onenote-structure`, `navigate-to-section`
  - **Content Creation**: `create-structured-page`, `update-page-content`
  - **Search & Discovery**: `search-onenote-content`
  - **Organization**: `organize-onenote-structure`, `backup-and-migrate`
  - **Media Management**: `extract-media-content`
  - **Automation**: `create-content-workflow`, `batch-content-operations`
  - **Validation**: `validate-onenote-operation`
- **Enhanced User Experience**: Prompts provide intelligent guidance for OneNote operations
  - **Argument Validation**: Built-in parameter validation and suggestions
  - **Best Practices**: Industry best practices embedded in prompt responses
  - **Tool Integration**: Seamless integration with existing MCP tools
  - **Error Recovery**: Guidance for handling common OneNote operation errors

## [1.5.0] - 2025-01-19

### Added
- **Enhanced Search Functionality**: Completely redesigned `searchPages` tool for comprehensive notebook search
  - **Recursive Search**: Now searches through all sections and nested section groups within a notebook
  - **Notebook-Scoped Search**: Requires `notebookId` parameter to limit search scope and improve performance
  - **Rich Context Information**: Each result includes section name, ID, and full hierarchy path
  - **Client-Side Filtering**: Uses reliable client-side title filtering instead of unsupported Graph API search
  - **Comprehensive Coverage**: Automatically discovers and searches all containers in the notebook structure
- **Enhanced Page Content Retrieval**: Added optional `forUpdate` parameter to `getPageContent` tool
  - **Update Support**: When `forUpdate` is set to 'true', includes `includeIDs=true` parameter for update operations
  - **Backward Compatibility**: Default behavior unchanged when parameter is not provided
  - **Direct HTTP Integration**: Uses direct HTTP calls when `forUpdate` is true for better control
- **Advanced Page Content Updates**: Completely redesigned page update functionality
  - **Full Microsoft Graph API Support**: New `updatePageContentAdvanced` tool supports all OneNote update actions
  - **Command-Based Updates**: Uses JSON command arrays for precise content manipulation
  - **Multiple Actions**: Supports append, insert, prepend, and replace operations
  - **Target Flexibility**: Supports data-id, generated IDs, body, and title targets
  - **Backward Compatibility**: Original `updatePageContent` tool maintained for simple replacements
  - **Position Control**: Precise control over where content is inserted relative to target elements

### Changed
- **`searchPages` Tool**: Updated to require both `query` and `notebookId` parameters
  - **Parameter Change**: Now requires `notebookId` to specify which notebook to search
  - **Return Format**: Results now include `sectionName`, `sectionId`, and `sectionPath` for better context
  - **Search Algorithm**: Replaced Graph API search with recursive enumeration and client-side filtering
  - **Performance**: More efficient by limiting search scope to specific notebook

### Technical Improvements
- **Recursive Container Traversal**: New `searchPagesRecursively()` function for comprehensive search
- **Section Group Support**: Full support for searching nested section groups and their sections
- **Enhanced Debug Logging**: Comprehensive logging for search progress and container discovery
- **Error Resilience**: Continues searching other sections if individual sections fail
- **Path Tracking**: Maintains full hierarchy path for each result (e.g., "Notebook/Section Group/Section")

### Removed
- **Graph API Search**: Removed dependency on unsupported `$search` parameter
- **searchPagesInSection**: Removed helper function that used unsupported search API

## [1.4.0] - 2025-01-18

### Added
- **Enhanced Section Group Support**: All section and section group operations now support both notebook and section group containers
- **Smart Container Detection**: Functions automatically determine if an ID is for a notebook, section, or section group
- **Unified API Design**: Consistent parameter naming (`containerId`) across all functions
- **Direct HTTP API Integration**: Switched from Microsoft Graph SDK to direct HTTP API calls for better control and consistency
- **Name Validation**: Added validation for illegal characters in display names and titles

### Changed
- **`listSections`**: Now accepts `containerId` (notebook or section group) instead of `notebookId`
- **`listSectionGroups`**: Now accepts `containerId` (notebook, section, or section group) instead of `notebookId`
- **`createSection`**: Now accepts `containerId` (notebook or section group) instead of `notebookId`
- **`createSectionGroup`**: Now accepts `containerId` (notebook or section group) instead of `notebookId`
- **`copySection`**: Renamed from `copySectionToNotebook`, now accepts `targetContainerId` (notebook or section group)
- **Consolidated Functions**: Removed redundant `createSectionInSectionGroup` function

### Technical Improvements
- **Container Type Detection**: New `determineContainerType()` helper function for smart endpoint selection
- **Response Processing**: Shared helper functions for consistent JSON response handling
- **Error Handling**: Enhanced error messages with container type validation
- **API Endpoints**: Updated to use beta endpoints for section copying operations
- **Polling Optimization**: Improved retry logic with exponential backoff
- **Input Validation**: Added `validateDisplayName()` function to check for illegal characters

### API Endpoints
- **Section Operations**: 
  - `GET /me/onenote/notebooks/{id}/sections` - List sections in notebook
  - `GET /me/onenote/sectionGroups/{id}/sections` - List sections in section group
  - `POST /me/onenote/notebooks/{id}/sections` - Create section in notebook
  - `POST /me/onenote/sectionGroups/{id}/sections` - Create section in section group
- **Section Group Operations**:
  - `GET /me/onenote/notebooks/{id}/sectionGroups` - List section groups in notebook
  - `GET /me/onenote/sections/{id}/sectionGroups` - List section groups in section
  - `GET /me/onenote/sectionGroups/{id}/sectionGroups` - List nested section groups
  - `POST /me/onenote/notebooks/{id}/sectionGroups` - Create section group in notebook
  - `POST /me/onenote/sectionGroups/{id}/sectionGroups` - Create nested section group
- **Copy Operations**:
  - `POST /me/onenote/sections/{id}/copyToNotebook` - Copy section to notebook
  - `POST /me/onenote/sections/{id}/copyToSectionGroup` - Copy section to section group (beta)

### Validation Rules
- **Illegal Characters**: The following characters are not allowed in display names and titles: `?*\\/:<>|&#''%%~`
- **Affected Operations**: `createPage`, `createSection`, `createSectionGroup`
- **Error Messages**: Clear error messages indicate which illegal character was found

## [1.3.0] - 2025-01-18

### Added
- **Asynchronous Operation Support**: Enhanced `copyPage` tool with proper async operation handling
  - Validates 202 status code for accepted asynchronous operations
  - Extracts operation ID and status from initial response
  - Implements polling mechanism with random delays (1-3 seconds)
  - Supports up to 30 polling attempts with timeout protection
  - Handles operation statuses: "Completed", "Failed", and intermediate states
  - Extracts new page ID from `resourceLocation` when operation completes
- **Operation Status Tracking**: New `getOnenoteOperation` tool for monitoring async operations
  - Retrieves status of asynchronous OneNote operations
  - Supports operation ID validation and sanitization
  - Returns detailed operation metadata and status information
  - Integrates with copy operations for status monitoring
- **Page ID Extraction**: New `extractPageIDFromResourceLocation` utility function
  - Extracts page ID from Microsoft Graph API resourceLocation URLs
  - Supports URL patterns like `/onenote/pages/{pageId}`
  - Validates extracted page IDs using existing sanitization logic
  - Handles various URL formats and edge cases
- **Enhanced Copy Operations**: Improved `copyPage` implementation using beta API endpoint
  - Uses `https://graph.microsoft.com/beta/me/onenote/pages/{pageId}/copyToSection`
  - Proper JSON request body with target section ID
  - Comprehensive error handling and logging
  - Returns structured response with new page ID and operation metadata

### Changed
- **CopyPage Implementation**: Updated to use beta API endpoint for better compatibility
- **Error Handling**: Enhanced error messages and validation for async operations
- **Logging**: Added comprehensive debug logging for operation tracking
- **Response Format**: Standardized response format for copy operations

### Technical Improvements
- **Random Delay Implementation**: Added `math/rand` for polling delays to prevent API throttling
- **Status Code Validation**: Specific 202 status code checking for async operations
- **JSON Response Parsing**: Enhanced parsing with field validation
- **Operation Polling**: Robust polling mechanism with timeout and retry logic

## [1.2.0] - 2024-01-25

### Added
- **Section Creation**: New `createSection` tool for creating sections in notebooks
  - Creates new sections with custom display names
  - Uses Microsoft Graph SDK for reliable section creation
  - Returns complete section metadata including ID and creation timestamp
- **Page Movement**: New `movePage` tool for moving pages between sections
  - Performs copy-then-delete operation for reliable page movement
  - Handles partial failures gracefully (copy succeeds, delete fails)
  - Returns detailed operation metadata including success status
- **Pagination Support**: Automatic handling of paginated responses from Microsoft Graph API
  - `listNotebooks`, `listSections`, `listPages`, and `searchPages` now handle `@odata.nextLink` automatically
  - All functions return complete results regardless of pagination
  - Enhanced logging for pagination tracking and debugging

### Changed
- **Improved Data Retrieval**: Replaced Microsoft Graph SDK calls with direct HTTP requests for better pagination control
- **Enhanced Logging**: Added detailed debug logging for pagination operations
- **Updated Documentation**: Added pagination notes to API documentation and README

## [1.1.0] - 2024-01-20

### Added
- **Page Copy Operation**: New `copyPage` tool using Microsoft Graph API's copyToSection endpoint
  - Asynchronous copy operations with polling support
  - Efficient copying without content reconstruction
  - Operation status tracking and error handling

### Changed
- **Replaced movePage with copyPage**: Updated to use native Microsoft Graph API instead of copy-then-delete
- **Enhanced Error Handling**: Better handling of asynchronous operations
- **Improved Documentation**: Updated API docs and examples for copyPage tool

## [1.0.0] - 2024-01-15

### Added
- **Initial Release**: Complete OneNote MCP Server implementation
- **OAuth 2.0 PKCE Authentication**: Secure authentication flow with Microsoft Graph API
- **Complete OneNote CRUD Operations**: 
  - List notebooks, sections, and pages
  - Create, read, update, and delete pages
  - Search pages by title
- **Content Management**:
  - HTML content support with rich formatting
  - Page item extraction (images, files, objects)
  - Binary content handling with base64 encoding
- **Advanced Features**:
  - Automatic token refresh with retry logic
  - Image optimization and scaling
  - HTML metadata extraction
  - Content type detection from multiple sources
- **Security Features**:
  - Input validation and sanitization
  - CSRF protection with state parameters
  - Secure token storage
- **Developer Experience**:
  - Comprehensive logging and error handling
  - Docker support with multi-stage builds
  - Configuration management via environment variables and JSON files
  - Detailed API documentation

### Technical Implementation
- **Microsoft Graph SDK Integration**: Uses official Go SDK for standard operations
- **Direct HTTP API Calls**: Custom implementation for advanced operations
- **Token Management**: Automatic expiry detection and refresh
- **Error Handling**: Comprehensive error categorization and recovery
- **Performance Optimizations**: Image scaling, connection pooling, efficient parsing

### Documentation
- **Comprehensive README**: Setup instructions, features, and usage examples
- **API Documentation**: Complete tool reference with examples
- **Setup Guide**: Step-by-step Azure app registration and deployment
- **Code Documentation**: Detailed comments in all Go files

## [0.9.0] - 2024-01-10

### Added
- **Core MCP Server Framework**: Basic server setup with tool registration
- **Authentication Foundation**: OAuth2 PKCE flow implementation
- **Microsoft Graph Client**: Basic client for OneNote operations
- **Configuration System**: Environment variable and file-based configuration

### Changed
- **Project Structure**: Organized into internal packages for better maintainability
- **Error Handling**: Improved error messages and logging

## [0.8.0] - 2024-01-05

### Added
- **Token Refresh Logic**: Automatic token refresh before expiration
- **Retry Mechanism**: Retry operations on authentication failures
- **Input Validation**: Sanitization of OneNote IDs and parameters

### Fixed
- **Authentication Issues**: Resolved token refresh problems causing malformed JWT errors
- **Content Processing**: Fixed HTML metadata extraction and attribute handling

## [0.7.0] - 2024-01-01

### Added
- **Image Processing**: Automatic scaling of large images for better performance
- **Content Type Detection**: Intelligent detection from HTML attributes and HTTP headers
- **Enhanced Page Item Handling**: Rich metadata extraction from OneNote HTML

### Changed
- **getPageItem Tool**: Enhanced to use listPageItems for metadata extraction
- **Content Processing**: Improved HTML parsing and attribute handling

## [0.6.0] - 2023-12-28

### Added
- **Search Functionality**: Search pages by title using OData filters
- **Page Item Operations**: List and retrieve embedded items (images, files)
- **HTML Parsing**: Parse OneNote HTML to extract embedded content

### Changed
- **Tool Registration**: Removed getPageItemDetails tool in favor of enhanced getPageItem
- **Error Handling**: Improved error categorization and recovery

## [0.5.0] - 2023-12-20

### Added
- **Page Content Operations**: Get and update page HTML content
- **Page Management**: Create and delete pages with HTML support
- **Multipart Form Support**: Handle complex content updates

### Changed
- **API Structure**: Improved request/response handling
- **Logging**: Enhanced debug logging for troubleshooting

## [0.4.0] - 2023-12-15

### Added
- **Section Operations**: List sections within notebooks
- **Page Operations**: List pages within sections
- **Basic CRUD**: Foundation for Create, Read, Update, Delete operations

### Changed
- **Client Structure**: Improved Microsoft Graph client organization
- **Error Handling**: Better error messages and categorization

## [0.3.0] - 2023-12-10

### Added
- **Notebook Operations**: List all OneNote notebooks for the user
- **Microsoft Graph SDK Integration**: Use official SDK for standard operations
- **Authentication Provider**: Custom token provider for SDK integration

### Changed
- **Architecture**: Separated concerns between authentication and Graph operations
- **Configuration**: Improved configuration loading and validation

## [0.2.0] - 2023-12-05

### Added
- **OAuth2 PKCE Flow**: Complete authentication implementation
- **Token Management**: Secure storage and refresh of access tokens
- **Local HTTP Server**: Handle OAuth callback for authentication

### Changed
- **Authentication**: Moved from basic auth to OAuth2 PKCE
- **Security**: Enhanced security with PKCE and state parameter validation

## [0.1.0] - 2023-12-01

### Added
- **Project Foundation**: Initial Go project structure
- **MCP Server Framework**: Basic MCP protocol implementation
- **Configuration System**: Environment variable configuration
- **Basic Logging**: Console and file-based logging support

### Technical Details
- **Go Version**: 1.21+
- **Dependencies**: Microsoft Graph Go SDK, MCP Go framework
- **Architecture**: Modular design with internal packages
- **Testing**: Unit tests for core functionality

## Planned Features

### [1.2.0] - Upcoming
- **Page Movement**: Move pages between sections (✅ Implemented as copy-then-delete)
- **Batch Operations**: Support for bulk operations on multiple pages
- **Content Templates**: Pre-defined HTML templates for common use cases
- **Advanced Search**: Full-text search across page content
- **Webhook Support**: Real-time notifications for content changes

### [1.2.0] - Future
- **Collaboration Features**: Support for shared notebooks and permissions
- **Content Synchronization**: Offline support and conflict resolution
- **Advanced Media Handling**: Video and audio content support
- **Performance Monitoring**: Metrics and analytics dashboard

### [1.3.0] - Future
- **Plugin System**: Extensible architecture for custom tools
- **Multi-tenant Support**: Enhanced support for organizational deployments
- **API Rate Limiting**: Intelligent rate limiting and queuing
- **Backup and Recovery**: Automated backup and restore capabilities

## Contributing

We welcome contributions! Please see our contributing guidelines for details on:
- Code style and standards
- Testing requirements
- Pull request process
- Issue reporting

## Support

For support and questions:
- Create an issue in the GitHub repository
- Check the documentation in the `docs/` directory
- Review the troubleshooting section in README.md

## License

This project is licensed under the MIT License - see the LICENSE file for details. 