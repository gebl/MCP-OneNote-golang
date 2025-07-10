# TODO: Refactoring and Codebase Organization

## Goals
- Reduce the size and complexity of the codebase
- Improve maintainability and readability
- Make it easier to add new features and tools
- Ensure clear separation of concerns

## Refactoring Ideas

### 1. Modularize Internal Packages

**Current Analysis:**
- `internal/graph/graph.go` (419 lines): Core client, HTTP utilities, token management, ID sanitization
- `internal/graph/notebooks.go` (292 lines): Notebook listing, page search, recursive traversal
- `internal/graph/sections.go` (1068 lines): Section & section group operations, validation, container detection
- `internal/graph/page.go` (1132 lines): Page CRUD, content management, page items, async operations

**Proposed New Structure:**
```
internal/
├── graph/           # Core client and shared infrastructure
│   ├── client.go    # Client struct, constructors, token management
│   ├── http.go      # HTTP utilities, request/response handling
│   └── utils.go     # ID sanitization, content type, filename generation
├── notebooks/       # Notebook operations
│   └── notebooks.go # List notebooks, search pages, recursive traversal
├── sections/        # Section operations
│   ├── sections.go  # Section CRUD, validation, response processing
│   └── groups.go    # Section group operations, container detection
├── pages/           # Page operations
│   ├── pages.go     # Page CRUD, content management
│   ├── items.go     # Page item handling, HTML parsing
│   └── operations.go # Async operations (copy, move, polling)
└── utils/           # Shared utilities
    ├── validation.go # Display name validation, character replacement
    └── image.go     # Image processing, scaling utilities
```

**Step-by-Step Action Plan:**

1. **Create new package directories**
   - Create `internal/notebooks/`, `internal/sections/`, `internal/pages/`, `internal/utils/`
   - Keep `internal/graph/` for core infrastructure

2. **Extract shared utilities first**
   - Move HTTP utilities from `graph.go` to `internal/graph/http.go`
   - Move ID sanitization and content utilities to `internal/graph/utils.go`
   - Move validation functions to `internal/utils/validation.go`
   - Move image processing to `internal/utils/image.go`

3. **Move domain-specific code**
   - Move notebook operations to `internal/notebooks/notebooks.go`
   - Split sections.go: sections to `internal/sections/sections.go`, section groups to `internal/sections/groups.go`
   - Split page.go: pages to `internal/pages/pages.go`, page items to `internal/pages/items.go`, async operations to `internal/pages/operations.go`

4. **Update imports and dependencies**
   - Update all import statements throughout the codebase
   - Ensure proper package boundaries and minimal cross-package dependencies
   - Use interfaces for shared contracts where needed

5. **Test and validate**
   - Run tests after each move to ensure nothing breaks
   - Update any package-level documentation
   - Verify all functionality works as expected

**Benefits:**
- Clear separation of concerns
- Easier to find and maintain specific functionality
- Reduced file sizes (no more 1000+ line files)
- Better testability with focused packages
- Improved code organization and readability

### 2. Reduce File Size
- Break up large files (e.g., `sections.go`, `tools.go`) into smaller, focused files
- Group related functions and types together
- Use Go interfaces to abstract Graph API operations

### 3. Clarify API Boundaries
- Clearly separate MCP tool registration from Graph API logic
- Move prompt and tool registration to their own subpackages
- Use dependency injection for easier testing and mocking

### 4. Improve Error Handling
- Standardize error types and error wrapping
- Centralize error formatting and logging
- Add more context to errors for easier debugging

### 5. Documentation and Comments
- Add package-level docs to all internal packages
- Ensure all exported functions/types have clear comments
- Maintain up-to-date API and architecture docs in `docs/`

### 6. Testing
- Add more unit tests for internal logic
- Use mocks for Graph API and auth flows
- Add integration tests for MCP tool flows

### 7. Configuration
- Centralize configuration loading and validation
- Use structs and environment variable parsing libraries

### 8. Dependency Management
- Audit and minimize external dependencies
- Use Go modules for version pinning

## Completed Work ✅

### 1. Code Modularization - COMPLETED ✅
- **Created new package directories**: `internal/notebooks/`, `internal/sections/`, `internal/pages/`, `internal/utils/`
- **Extracted shared utilities**: HTTP utilities, ID sanitization, content utilities moved to appropriate packages
- **Moved domain-specific code**: 
  - Notebook operations → `internal/notebooks/notebooks.go`
  - Section operations → `internal/sections/sections.go` and `internal/sections/groups.go`
  - Page operations → `internal/pages/pages.go`
- **Updated imports and dependencies**: All packages now use proper modular structure
- **Tested and validated**: Build successful, all functionality preserved

### 2. File Size Reduction - COMPLETED ✅
- Broke up large files (1000+ lines) into focused, manageable packages
- `internal/graph/sections.go` (1068 lines) → Split into `internal/sections/sections.go` and `internal/sections/groups.go`
- `internal/graph/page.go` (1132 lines) → Moved to `internal/pages/pages.go`
- `internal/graph/notebooks.go` (292 lines) → Moved to `internal/notebooks/notebooks.go`
- Original files now contain only stub comments for backward compatibility

### 3. API Boundaries Clarification - COMPLETED ✅
- Clearly separated MCP tool registration from Graph API logic
- Updated `cmd/onenote-mcp-server/tools.go` to use new modularized clients
- Implemented proper client pattern with `NotebookClient`, `SectionClient`, `PageClient`
- Resolved circular dependency issues by moving `GetDefaultNotebookID` to notebooks package

### 4. Package Structure - COMPLETED ✅
```
internal/
├── graph/           # Core client and shared infrastructure ✅
│   ├── client.go    # Client struct, constructors, token management ✅
│   ├── http.go      # HTTP utilities, request/response handling ✅
│   ├── utils.go     # ID sanitization, content type, filename generation ✅
│   ├── page.go      # Stub file (content moved to pages package) ✅
│   ├── notebooks.go # Stub file (content moved to notebooks package) ✅
│   └── sections.go  # Stub file (content moved to sections package) ✅
├── notebooks/       # Notebook operations ✅
│   └── notebooks.go # List notebooks, search pages, recursive traversal ✅
├── sections/        # Section operations ✅
│   ├── sections.go  # Section CRUD, validation, response processing ✅
│   └── groups.go    # Section group operations, container detection ✅
├── pages/           # Page operations ✅
│   └── pages.go     # Page CRUD, content management, page items ✅
└── utils/           # Shared utilities ✅
    ├── validation.go # Display name validation, character replacement ✅
    └── image.go     # Image processing, scaling utilities ✅
```

## Remaining Work

### 5. Improve Error Handling
- Standardize error types and error wrapping
- Centralize error formatting and logging
- Add more context to errors for easier debugging

### 6. Documentation and Comments
- Add package-level docs to all internal packages
- Ensure all exported functions/types have clear comments
- Maintain up-to-date API and architecture docs in `docs/`

### 7. Testing
- Add more unit tests for internal logic
- Use mocks for Graph API and auth flows
- Add integration tests for MCP tool flows

### 8. Configuration
- Centralize configuration loading and validation
- Use structs and environment variable parsing libraries

### 9. Dependency Management
- Audit and minimize external dependencies
- Use Go modules for version pinning

## Next Steps
- Focus on testing and documentation improvements
- Consider adding more granular page item operations
- Review error handling patterns across packages
- Keep documentation in sync with code changes 