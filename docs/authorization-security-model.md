# OneNote MCP Server - Authorization Security Model

## Overview

The OneNote MCP Server implements a comprehensive hierarchical authorization system that enforces strict container-based access control. This security model ensures that users cannot bypass permissions by using resource IDs and that child resources cannot have more permissions than their parent containers.

## Security Rules Implemented

### 1. Container Hierarchy Enforcement

**Rule**: Notebooks contain sections, sections contain pages. You must have read/write permission on the parent container to access child resources.

- **Notebooks** → **Sections** → **Pages**
- No access to notebooks with `none` permission
- No access to sections/pages in notebooks with `none` permission
- No access to pages in sections with `none` permission

### 2. Permission Inheritance

**Rule**: Child resources inherit permissions from their parent containers unless explicitly overridden, but can never exceed parent permissions.

- Pages inherit section permissions by default
- Sections inherit notebook permissions by default
- Explicit permissions are constrained by parent permissions (e.g., a page with `write` permission in a `read-only` section gets `read` permission)

### 3. ID-Based Access Control

**Rule**: Resource IDs cannot be used to bypass permission checks.

- Page IDs without resolved names use section/notebook permissions
- Section IDs without resolved names use notebook permissions
- Unresolved resource contexts fall back to parent container permissions

### 4. Operation-Level Authorization

**Rule**: Read operations require `read` or higher permission, write operations require `write` or `full` permission.

- `none`: No access
- `read`: View operations only
- `write`/`full`: All operations

## Critical Security Fix: Cross-Notebook Page Access Prevention

**VULNERABILITY FIXED**: The authorization system previously allowed users to access pages from unauthorized notebooks by using page IDs when they had access to any notebook. This was a critical security flaw that has been resolved.

**Root Cause**: The authorization system was using cached notebook context instead of verifying which notebook actually contains the requested page ID.

**Fix Implemented**: Added strict validation that page ID access requires verified notebook context. The system now blocks any page ID access where the notebook context cannot be properly established.

**Security Rule Added**: `page ID access requires validated notebook ownership` - prevents cross-notebook access attempts.

## Security Test Results

The system passes 15 comprehensive security tests:

### Notebook-Level Access Control
- ✅ **NB-001**: Access allowed for notebooks with appropriate permissions
- ✅ **NB-002**: Access denied for notebooks with `none` permission

### Section-Level Access Control
- ✅ **SEC-001**: Access allowed for sections in accessible notebooks
- ✅ **SEC-002**: Access denied for sections in private notebooks
- ✅ **SEC-003**: Access denied for sections with explicit `none` permission

### Page-Level Access Control
- ✅ **PAGE-001**: Access allowed for pages in accessible sections
- ✅ **PAGE-002**: Access denied for pages in private notebooks
- ✅ **PAGE-003**: Access denied for pages with explicit `none` permission

### ID-Based Bypass Prevention
- ✅ **ID-001**: Page ID access blocked when notebook/section permissions deny access
- ✅ **ID-002**: Section ID access blocked when notebook permissions deny access
- ✅ **ID-003**: Cross-notebook page access via page ID blocked (CRITICAL SECURITY FIX)

### Permission Inheritance and Constraints
- ✅ **INH-001**: Section permissions properly constrained by notebook permissions
- ✅ **INH-002**: Page permissions properly constrained by section permissions

### Operation-Level Security
- ✅ **OP-001**: Write operations denied in read-only contexts
- ✅ **OP-002**: Read operations allowed in read-only contexts

## Configuration Example

```json
{
  "authorization": {
    "enabled": true,
    "default_mode": "none",
    "default_notebook_mode": "none",
    "default_section_mode": "none",
    "default_page_mode": "none",
    "tool_permissions": {
      "auth_tools": "full",
      "notebook_read": "read",
      "notebook_write": "write",
      "page_read": "read",
      "page_write": "write"
    },
    "notebook_permissions": {
      "Public Notebook": "write",
      "Read Only Notebook": "read",
      "Private Notebook": "none"
    },
    "section_permissions": {
      "Public Notebook/Secret Section": "none",
      "Read Only Notebook/Special Section": "write"
    },
    "page_permissions": {
      "Secret Document": "none",
      "Public Page": "write"
    }
  }
}
```

## Security Features

### Default-Deny Security Model
- All permissions default to `none` unless explicitly granted
- Users must be explicitly granted access to resources
- Principle of least privilege enforced

### Hierarchical Permission Resolution
1. Check explicit resource permission
2. Apply parent container constraints
3. Fall back to parent container permissions
4. Apply default permissions (with parent constraints)

### Filter Tool Bypass Protection
- Filter tools (like `notebooks`, `listPages`) bypass resource checks but filter results
- Non-filter tools enforce full resource-level authorization
- Prevents enumeration of restricted resources

### Comprehensive Logging
- All authorization decisions logged with detailed context
- Security rule violations logged at INFO level
- Permission inheritance and constraints logged for audit trails

## Testing the Security Model

Run the comprehensive security validation:

```bash
# Windows
scripts\test-security-model.bat

# Linux/macOS
scripts/test-security-model.sh

# Or run directly with Go
go test ./internal/authorization -v -run="TestSecurityModelComprehensive"
```

The test suite validates all security rules and provides a detailed report of the authorization system's behavior.

## Implementation Details

### Core Components

- **`authorization.go`**: Main authorization logic with hierarchical permission resolution
- **`context.go`**: Resource context extraction with ID resolution fallback
- **`security_test.go`**: Comprehensive test suite validating all security rules
- **`wrapper.go`**: Authorization wrapper for MCP tools

### Key Functions

- `getResourcePermission()`: Hierarchical permission resolution with security enforcement
- `getEffectivePermission()`: Permission constraint calculation
- `IsAuthorized()`: Main authorization check with tool and resource validation
- `FilterNotebooks/Sections/Pages()`: Result filtering for authorized resources

The security model ensures that the OneNote MCP Server provides robust, enterprise-grade access control while maintaining usability and performance.