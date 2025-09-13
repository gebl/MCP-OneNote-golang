# Authorization System Integration Guide

This guide provides detailed instructions for integrating and configuring the authorization system in the MCP OneNote Server.

## Quick Start

### 1. Enable Authorization
Add the authorization configuration to your JSON config file:

```json
{
  "authorization": {
    "enabled": true,
    "default_mode": "read"
  }
}
```

### 2. Test Basic Functionality
Start the server and verify authorization is working:
```bash
./onenote-mcp-server.exe
```

Use an MCP client to test read operations (should work) and write operations (should be denied unless explicitly granted).

### 3. Configure Permissions
Add specific permissions for your use case:

```json
{
  "authorization": {
    "enabled": true,
    "default_mode": "read",
    "default_page_mode": "none",
    "tool_permissions": {
      "auth_tools": "full",
      "page_write": "write"
    },
    "notebook_permissions": {
      "My Notebook": "write"
    },
    "page_permissions": {
      "Daily Journal": "write"
    }
  }
}
```

## Configuration Patterns

### Pattern 1: QuickNote-Only Access
Perfect for dedicated note-taking setups:

```json
{
  "authorization": {
    "enabled": true,
    "default_mode": "read",
    "default_notebook_mode": "none",
    "default_page_mode": "none",
    "tool_permissions": {
      "auth_tools": "full",
      "page_write": "write"
    },
    "notebook_permissions": {
      "Personal Notes": "write"
    },
    "page_permissions": {
      "Daily Journal": "write",
      "Quick Notes": "write",
      "Meeting Notes": "write"
    }
  },
  "quicknote": {
    "page_name": "Daily Journal"
  }
}
```

**Benefits:**
- Restricts access to only specified notebooks and pages
- Allows QuickNote functionality 
- Prevents accidental modification of other content

### Pattern 2: Read-Heavy with Selective Write
Good for browsing with limited editing:

```json
{
  "authorization": {
    "enabled": true,
    "default_mode": "read",
    "default_tool_mode": "read",
    "tool_permissions": {
      "auth_tools": "full",
      "page_write": "write",
      "notebook_management": "read"
    },
    "notebook_permissions": {
      "Work Projects": "write",
      "Archive": "read",
      "Personal": "none"
    }
  }
}
```

**Benefits:**
- Allows browsing of most content
- Selective write access to work-related notebooks
- Protects personal content

### Pattern 3: Development Environment
Broad access with protected areas:

```json
{
  "authorization": {
    "enabled": true,
    "default_mode": "read",
    "default_tool_mode": "write",
    "tool_permissions": {
      "auth_tools": "full"
    },
    "notebook_permissions": {
      "Production Data": "none",
      "Customer Information": "none",
      "Development": "write",
      "Testing": "write"
    }
  }
}
```

**Benefits:**
- Allows broad development access
- Protects sensitive production and customer data
- Enables full testing capabilities

## Tool Permission Categories

### Core Tool Categories

#### `auth_tools` (Recommended: `"full"`)
Controls access to authentication tools:
- `auth_status`: Check authentication status
- `auth_initiate`: Start OAuth flow
- `auth_refresh`: Refresh authentication tokens
- `auth_clear`: Clear stored tokens

**Recommendation:** Always set to `"full"` to allow authentication management.

#### `page_write` (Recommended: `"write"` when needed)
Controls page creation and modification tools:
- `createPage`: Create new pages
- `updatePageContent`: Update page content
- `updatePageContentAdvanced`: Advanced content updates
- `deletePage`: Delete pages
- `quickNote`: Append to QuickNote page

**Use Cases:**
- Note-taking applications: `"write"`
- Read-only browsing: `"none"` or omit
- Content management: `"write"`

#### `page_read` (Recommended: `"read"`)
Controls page reading tools:
- `getPageContent`: Read page HTML content
- `listPages`: List pages in sections
- `getPageItem`: Get page images/attachments

**Recommendation:** Usually set to `"read"` for basic functionality.

#### `notebook_management` (Use carefully)
Controls notebook-level operations:
- `createNotebook`: Create new notebooks
- `setDefaultNotebook`: Change default notebook
- `getNotebooks`: List available notebooks

**Security Note:** Creating notebooks affects the user's OneNote structure.

#### `section_management` (Use carefully) 
Controls section-level operations:
- `createSection`: Create sections
- `createSectionGroup`: Create section groups
- `deleteSection`: Delete sections
- `getNotebookSections`: List notebook sections

**Security Note:** Section management can reorganize OneNote structure.

#### `content_management` (Advanced)
Controls advanced content operations:
- Advanced HTML manipulation
- Bulk content operations
- Complex page restructuring

**Security Note:** These operations can significantly modify content.

## Permission Resolution Examples

### Example 1: QuickNote Configuration
```json
{
  "authorization": {
    "enabled": true,
    "default_mode": "read",
    "default_notebook_mode": "none",
    "default_page_mode": "none",
    "tool_permissions": {
      "auth_tools": "full",
      "page_write": "write"
    },
    "notebook_permissions": {
      "Personal Notes": "write"
    },
    "page_permissions": {
      "Daily Journal": "write"
    }
  }
}
```

**Permission Resolution for `quickNote` to "Daily Journal" page in "Personal Notes":**
1. **Tool Check**: `page_write` = `"write"` ✓ (passes)
2. **Page Check**: `"Daily Journal"` = `"write"` ✓ (passes)
3. **Notebook Check**: `"Personal Notes"` = `"write"` ✓ (passes)
4. **Result**: ALLOWED

**Permission Resolution for `createPage` in "Other Notebook":**
1. **Tool Check**: `page_write` = `"write"` ✓ (passes)
2. **Notebook Check**: `"Other Notebook"` = not specified, use `default_notebook_mode` = `"none"` ✗ (fails)
3. **Result**: DENIED

### Example 2: Read-Heavy Configuration
```json
{
  "authorization": {
    "enabled": true,
    "default_mode": "read",
    "default_tool_mode": "read",
    "default_notebook_mode": "read",
    "default_page_mode": "read",
    "tool_permissions": {
      "auth_tools": "full"
    },
    "notebook_permissions": {
      "Archive": "none"
    }
  }
}
```

**Permission Resolution for `getPageContent` in "Archive" notebook:**
1. **Tool Check**: `page_read` not specified, use `default_tool_mode` = `"read"` ✓ (passes)
2. **Notebook Check**: `"Archive"` = `"none"` ✗ (fails)
3. **Result**: DENIED

**Permission Resolution for `getPageContent` in "Work Notes" notebook:**
1. **Tool Check**: `page_read` not specified, use `default_tool_mode` = `"read"` ✓ (passes)
2. **Notebook Check**: `"Work Notes"` not specified, use `default_notebook_mode` = `"read"` ✓ (passes)
3. **Result**: ALLOWED (read-only)

## Troubleshooting

### Common Issues

#### Issue: "Permission denied" for QuickNote
**Symptoms:** QuickNote tool returns permission denied error

**Diagnosis:**
1. Check tool permission: Is `page_write` set to `"write"`?
2. Check notebook permission: Does the target notebook have `"write"` access?
3. Check page permission: Does the target page have `"write"` access?

**Solution:**
```json
{
  "tool_permissions": {
    "page_write": "write"
  },
  "notebook_permissions": {
    "Your Target Notebook": "write"
  },
  "page_permissions": {
    "Your Target Page": "write"
  }
}
```

#### Issue: Can read but cannot write
**Symptoms:** Read operations work, write operations fail

**Diagnosis:** 
- Tool has read permission but not write permission
- OR resource has read permission but not write permission

**Solution:** Add explicit write permissions:
```json
{
  "tool_permissions": {
    "page_write": "write"
  },
  "notebook_permissions": {
    "Target Notebook": "write"
  }
}
```

#### Issue: Authorization seems disabled
**Symptoms:** All operations work regardless of configuration

**Diagnosis:** 
- `authorization.enabled` is `false` or missing
- Configuration file not loading properly

**Solution:**
1. Verify `"enabled": true` in authorization config
2. Check server logs for configuration loading errors
3. Verify config file path and JSON syntax

### Debugging Authorization

#### Enable Authorization Logging
Set log level to DEBUG to see authorization decisions:

```json
{
  "log_level": "DEBUG",
  "content_log_level": "DEBUG"
}
```

#### Log Message Examples

**Successful Authorization:**
```
[auth] Authorization granted: tool=page_write, resource=Daily Journal, permission=write, reason=explicit_page_permission
```

**Failed Authorization:**
```
[auth] Authorization denied: tool=page_write, resource=Archive Notes, permission=none, reason=default_notebook_mode
```

#### Trace Permission Resolution
Look for log messages showing the permission resolution path:
```
[auth] Permission resolution: tool_permission=write -> notebook_permission=none -> denied
[auth] Permission resolution: tool_permission=write -> page_permission=write -> granted
```

## Security Considerations

### Principle of Least Privilege
- Start with restrictive defaults (`"none"` or `"read"`)
- Grant only necessary permissions
- Regularly review and audit permissions

### Sensitive Data Protection
```json
{
  "notebook_permissions": {
    "Personal Information": "none",
    "Financial Data": "none",
    "Customer Records": "none"
  }
}
```

### Production Recommendations
1. **Always enable authorization in production**
2. **Use restrictive defaults**
3. **Document permission grants**
4. **Monitor authorization logs**
5. **Regular permission audits**

### Development vs Production
**Development:**
```json
{
  "authorization": {
    "enabled": true,
    "default_mode": "read",
    "default_tool_mode": "write"
  }
}
```

**Production:**
```json
{
  "authorization": {
    "enabled": true,
    "default_mode": "none",
    "default_tool_mode": "none",
    "default_notebook_mode": "none",
    "default_page_mode": "none"
  }
}
```

## Environment Variables

Authorization can also be configured via environment variables:

```bash
# Enable authorization
AUTHORIZATION_ENABLED=true

# Set default modes  
AUTHORIZATION_DEFAULT_MODE=read
AUTHORIZATION_DEFAULT_TOOL_MODE=read
AUTHORIZATION_DEFAULT_NOTEBOOK_MODE=none
AUTHORIZATION_DEFAULT_PAGE_MODE=none

# Tool permissions (JSON format)
AUTHORIZATION_TOOL_PERMISSIONS='{"auth_tools":"full","page_write":"write"}'

# Resource permissions (JSON format) 
AUTHORIZATION_NOTEBOOK_PERMISSIONS='{"My Notebook":"write"}'
AUTHORIZATION_PAGE_PERMISSIONS='{"Daily Journal":"write"}'
```

**Note:** JSON configuration takes precedence over environment variables.

## Migration Guide

### From Unprotected to Protected
1. **Start with permissive defaults:**
   ```json
   {
     "authorization": {
       "enabled": true,
       "default_mode": "read",
       "default_tool_mode": "read"
     }
   }
   ```

2. **Test all existing functionality**

3. **Gradually restrict permissions:**
   ```json
   {
     "authorization": {
       "enabled": true,
       "default_mode": "read",
       "default_notebook_mode": "none",
       "default_page_mode": "none"
     }
   }
   ```

4. **Add explicit permissions for required functionality**

### Rollback Strategy
Keep a backup configuration file:
```json
{
  "authorization": {
    "enabled": false
  }
}
```

This completely disables authorization if issues arise.

## Integration Examples

### Code Integration Pattern
When adding authorization to existing tools, use this pattern:

```go
// Original tool handler
func handleListPages(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // Tool implementation
    return mcp.NewToolResultText("result"), nil
}

// Authorization wrapper registration
wrappedHandler := authorization.AuthorizedToolHandler(
    "listPages", 
    handleListPages, 
    cfg.Authorization, 
    cacheAdapter, 
    quickNoteAdapter,
)

server.AddTool(listPagesTool, wrappedHandler)
```

### Resource Context Extraction
The authorization system automatically extracts context from tool arguments:

```go
// For tools that operate on notebooks
func extractNotebookContext(args map[string]interface{}) *ResourceContext {
    if notebookName, ok := args["notebookName"].(string); ok {
        return &ResourceContext{
            NotebookName: notebookName,
            Operation: "write", // or "read"
        }
    }
    return &ResourceContext{}
}
```

### Cache Integration
Implement section name lookup in your notebook cache:

```go
type NotebookCache interface {
    GetSectionName(sectionID string) (string, bool)
    GetNotebookName(notebookID string) (string, bool)
}
```

This enables authorization checks for section-based operations.