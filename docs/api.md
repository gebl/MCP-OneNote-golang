# API Reference: OneNote MCP Server

## Toolsets

- **notebooks**: List, get, create, search notebooks
- **sections**: List, get, create, update, delete sections
- **pages**: List, get, create, update, delete, search pages
- **content**: Get/append page content, extract text

## Example Tool: create_page

```
{
  "name": "create_page",
  "description": "Create a new page in a OneNote section",
  "inputSchema": {
    "type": "object",
    "properties": {
      "section_id": {"type": "string", "description": "Section ID"},
      "title": {"type": "string", "description": "Page title"},
      "content": {"type": "string", "description": "HTML content"}
    },
    "required": ["section_id", "title"]
  }
}
```

## See `onenote_mcp_prd.md` for full tool and resource specifications. 