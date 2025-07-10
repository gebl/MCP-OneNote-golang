# OneNote MCP API Documentation

This document provides comprehensive documentation for all MCP tools available in the OneNote MCP Server.

## 🚨 Version 1.4.0 Migration Note

**All section and section group operations now use `containerId` (notebook or section group) instead of `notebookId` or `sectionGroupId`.**

- `listSections`, `listSectionGroups`, `createSection`, and `createSectionGroup` now take a `containerId` parameter.
- `copySection` replaces `copySectionToNotebook` and takes a `targetContainerId` (notebook or section group).
- **Name Validation**: Display names and titles cannot contain: `?*\\/:<>|&#''%%~`

## 🚨 Version 1.7.0 Container Hierarchy Enforcement

**The server now strictly enforces OneNote's container hierarchy rules:**

- **Notebooks** can contain sections and section groups
- **Section Groups** can contain sections and other section groups  
- **Sections** can contain pages, but NOT other sections or section groups

All section and section group operations now validate container types upfront and provide clear error messages when hierarchy rules are violated.

## Overview

The OneNote MCP Server provides 14 tools for interacting with Microsoft OneNote via the Microsoft Graph API. All tools support automatic token refresh and comprehensive error handling.

## Authentication

All tools require valid OAuth2 authentication. The server handles authentication automatically using PKCE flow and maintains access/refresh tokens locally.

## MCP Prompts

The OneNote MCP Server provides 11 comprehensive prompts that offer contextual guidance for OneNote operations. These prompts help users understand how to perform various tasks and provide best practices for working with OneNote content.

### Available Prompts

#### 1. explore-onenote-structure
**Purpose**: Explore and understand the OneNote notebook structure to find specific content or locations

**Arguments**:
- `search_target` (required): What you're looking for (e.g., 'meeting notes', 'project documentation', 'personal notes')
- `search_scope` (optional): Search scope - 'notebooks' to list all notebooks, 'specific_notebook' to search within a known notebook
- `notebook_id` (optional): Specific notebook ID to search within (only needed if search_scope is 'specific_notebook')

**Use Case**: When you need to understand your OneNote structure or find specific content across notebooks.

#### 2. navigate-to-section
**Purpose**: Navigate to a specific section or section group within a OneNote notebook

**Arguments**:
- `notebook_id` (required): The ID of the notebook to navigate within
- `target_section` (required): Name or description of the section you want to find
- `include_section_groups` (optional): Whether to also search within section groups (true/false)

**Use Case**: When you know which notebook you want to work in but need to find a specific section.

#### 3. create-structured-page
**Purpose**: Create a new OneNote page with structured content and proper HTML formatting

**Arguments**:
- `section_id` (required): The ID of the section where the page will be created
- `page_title` (required): Title for the new page
- `content_type` (required): Type of content to create - 'meeting_notes', 'project_documentation', 'personal_notes', 'task_list', 'custom'
- `custom_content` (optional): Custom HTML content (only needed if content_type is 'custom')
- `include_data_ids` (optional): Whether to include data-id attributes for future targeting (true/false)

**Use Case**: When creating new pages with specific content structures and templates.

#### 4. update-page-content
**Purpose**: Update existing OneNote page content with precise modifications

**Arguments**:
- `page_id` (required): The ID of the page to update
- `update_type` (required): Type of update - 'replace_all', 'append_content', 'insert_before', 'insert_after', 'replace_section'
- `target_element` (optional): Target element (data-id, generated id, or 'body'/'title')
- `new_content` (required): HTML content to add or replace
- `position` (optional): Position for insert operations - 'before' or 'after'

**Use Case**: When you need to modify existing page content with specific targeting.

#### 5. search-onenote-content
**Purpose**: Search for specific content across OneNote notebooks with advanced filtering

**Arguments**:
- `search_query` (required): Search term to look for in page titles
- `notebook_id` (optional): Notebook ID to search within (leave empty to search all notebooks)
- `search_context` (optional): Context for the search - 'meeting_notes', 'project_docs', 'personal', 'all'
- `include_content` (optional): Whether to also retrieve page content for found pages (true/false)

**Use Case**: When you need to find specific content across your OneNote notebooks.

#### 6. organize-onenote-structure
**Purpose**: Help organize OneNote structure by creating sections, section groups, and moving content

**Arguments**:
- `notebook_id` (required): The ID of the notebook to organize
- `organization_type` (required): Type of organization - 'create_sections', 'create_section_groups', 'move_pages', 'copy_content'
- `structure_plan` (required): Description of the desired structure or organization plan
- `source_pages` (optional): Comma-separated list of page IDs to move/copy (only for move_pages/copy_content)

**Use Case**: When reorganizing your OneNote structure for better content management.

#### 7. backup-and-migrate
**Purpose**: Create backups and migrate content between OneNote locations

**Arguments**:
- `operation_type` (required): Type of operation - 'copy_page', 'copy_section', 'move_page', 'backup_content'
- `source_id` (required): ID of the source page or section
- `target_container_id` (required): ID of the target notebook or section group
- `rename_as` (optional): Optional new name for the copied/moved content
- `verify_operation` (optional): Whether to verify the operation completed successfully (true/false)

**Use Case**: When backing up content or moving it between different OneNote locations.

#### 8. extract-media-content
**Purpose**: Extract and manage media content (images, files) from OneNote pages

**Arguments**:
- `page_id` (required): The ID of the page containing media content
- `media_type` (required): Type of media to extract - 'all', 'images', 'files', 'specific_item'
- `page_item_id` (optional): Specific page item ID to extract (only for 'specific_item')
- `custom_filename` (optional): Custom filename for the extracted content
- `include_metadata` (optional): Whether to include metadata with the extracted content (true/false)

**Use Case**: When you need to extract embedded media from OneNote pages.

#### 9. create-content-workflow
**Purpose**: Create automated workflows for OneNote content management

**Arguments**:
- `workflow_type` (required): Type of workflow - 'meeting_template', 'project_setup', 'content_migration', 'regular_backup'
- `target_notebook_id` (required): Notebook ID where the workflow will be executed
- `workflow_parameters` (optional): JSON string with workflow-specific parameters
- `template_content` (optional): Template content to use for the workflow

**Use Case**: When setting up automated processes for OneNote content management.

#### 10. batch-content-operations
**Purpose**: Perform batch operations on multiple OneNote pages or sections

**Arguments**:
- `operation_type` (required): Type of batch operation - 'update_multiple_pages', 'copy_multiple_pages', 'search_and_update'
- `target_ids` (required): Comma-separated list of page/section IDs to operate on
- `operation_commands` (required): JSON array of commands to execute on each target
- `verify_results` (optional): Whether to verify each operation completed successfully (true/false)

**Use Case**: When performing the same operation on multiple pages or sections.

#### 11. validate-onenote-operation
**Purpose**: Validate OneNote operations and handle common errors

**Arguments**:
- `operation_type` (required): Type of operation to validate - 'create_page', 'update_content', 'move_content', 'search'
- `operation_parameters` (required): JSON string with the operation parameters to validate
- `error_context` (optional): Description of any errors encountered or validation concerns
- `suggest_fixes` (optional): Whether to suggest fixes for validation issues (true/false)

**Use Case**: When troubleshooting OneNote operations or validating parameters before execution.

### Using Prompts

Prompts are accessed through the MCP protocol's prompt system. Each prompt provides:
- **Contextual guidance** for the specific operation
- **Best practices** and workflow recommendations
- **Available tools** and their usage patterns
- **Error handling** advice
- **Step-by-step instructions** for complex operations

## Tool Reference

### 1. listNotebooks

Lists all OneNote notebooks for the authenticated user. **Note:** This function automatically handles pagination and will return all notebooks, even if the data is paginated by the Microsoft Graph API.

**Parameters:** None

**Returns:** Array of notebook objects with the following structure:
```json
[
  {
    "id": "notebook-id",
    "displayName": "Notebook Name"
  }
]
```

**Example:**
```bash
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{"name": "listNotebooks", "arguments": {}}'
```

**Response:**
```json
{
  "content": [
    {
      "id": "0-4D24C77F19546939!40109",
      "displayName": "My Personal Notebook"
    },
    {
      "id": "0-896fbac8f72d01b02c5950345e65f588!1-4D24C77F19546939!39705",
      "displayName": "Work Notes"
    }
  ]
}
```

### 2. listSections

Lists all sections within a specified notebook or section group, including sections from all nested section groups. **Note:** This function automatically handles pagination and will return all sections, even if the data is paginated by the Microsoft Graph API. The tool recursively expands section groups to show all sections regardless of nesting level.

**Parameters:**
- `containerId` (string, optional): The ID of the notebook or section group to list sections for. If not provided, uses the default notebook from configuration

**Returns:** Array of section objects with the following structure:
```json
[
  {
    "id": "section-id",
    "name": "Section Name",
    "parentNotebook": {
      "id": "notebook-id",
      "name": "Notebook Name"
    },
    "parentSectionGroup": {
      "id": "section-group-id",
      "name": "Section Group Name"
    }
  }
]
```

**Behavior:**
- When called with a notebook ID: Returns all direct sections plus all sections from all section groups within the notebook (including nested section groups at any depth)
- When called with a section group ID: Returns all sections within that specific section group plus all sections from any nested section groups within it
- Each section includes parent information to show its location in the hierarchy
- Handles multi-level nesting of section groups recursively

**Example:**
```bash
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "listSections",
    "arguments": {
      "containerId": "0-4D24C77F19546939!40109"
    }
  }'
```

**Response:**
```json
{
  "content": [
    {
      "id": "0-4D24C77F19546939!40109!1-4D24C77F19546939!40109",
      "displayName": "Quick Notes"
    },
    {
      "id": "0-4D24C77F19546939!40109!1-4D24C77F19546939!40110",
      "displayName": "Meeting Notes"
    }
  ]
}
```

### 3. listSectionGroups

Lists all section groups within a specified notebook, section, or section group. **Note:** This function automatically handles pagination and will return all section groups, even if the data is paginated by the Microsoft Graph API. Each section group includes parent information to show its location in the hierarchy.

**Parameters:**
- `containerId` (string, optional): The ID of the notebook, section, or section group to list section groups from. If not provided, uses the default notebook from configuration

**Returns:** Array of section group objects with the following structure:
```json
[
  {
    "id": "section-group-id",
    "name": "Section Group Name",
    "parentNotebook": {
      "id": "notebook-id",
      "name": "Notebook Name"
    },
    "parentSectionGroup": {
      "id": "parent-section-group-id",
      "name": "Parent Section Group Name"
    },
    "parentContainer": {
      "id": "container-id",
      "type": "container"
    }
  }
]
```

**Behavior:**
- When called with a notebook ID: Returns all section groups within the notebook
- When called with a section ID: Returns all section groups within that section
- When called with a section group ID: Returns all nested section groups within that section group
- Each section group includes parent information to show its location in the hierarchy
- Parent information includes notebook, section group, or container details as available

**Example:**
```bash
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "listSectionGroups",
    "arguments": {
      "containerId": "0-4D24C77F19546939!40109"
    }
  }'
```

**Response:**
```json
{
  "content": [
    {
      "id": "0-4D24C77F19546939!40109!2-4D24C77F19546939!40113",
      "name": "Project Groups",
      "parentNotebook": {
        "id": "0-4D24C77F19546939!40109",
        "name": "My Notebook"
      }
    },
    {
      "id": "0-4D24C77F19546939!40109!2-4D24C77F19546939!40114",
      "name": "Personal Groups",
      "parentNotebook": {
        "id": "0-4D24C77F19546939!40109",
        "name": "My Notebook"
      }
    }
  ]
}
```

### 4. listPages

Lists all pages within a specified section. **Note:** This function automatically handles pagination and will return all pages, even if the data is paginated by the Microsoft Graph API.

**Parameters:**
- `sectionId` (string, required): The ID of the section to list pages for

**Returns:** Array of page objects with the following structure:
```json
[
  {
    "id": "page-id",
    "title": "Page Title"
  }
]
```

**Example:**
```bash
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "listPages",
    "arguments": {
      "sectionId": "0-4D24C77F19546939!40109!1-4D24C77F19546939!40109"
    }
  }'
```

**Response:**
```json
{
  "content": [
    {
      "id": "0-4D24C77F19546939!40109!1-4D24C77F19546939!40109!1-4D24C77F19546939!40111",
      "title": "Meeting Notes - 2024-01-15"
    },
    {
      "id": "0-4D24C77F19546939!40109!1-4D24C77F19546939!40109!1-4D24C77F19546939!40112",
      "title": "Project Ideas"
    }
  ]
}
```

### 5. createSection

Creates a new section in a specified notebook using the Microsoft Graph SDK.

**Parameters:**
- `containerId` (string, optional): The ID of the notebook or section group to create the section in. If not provided, uses the default notebook from configuration
- `displayName` (string, required): The display name for the new section

**Returns:** Created section object with metadata

**Example:**
```bash
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "createSection",
    "arguments": {
      "notebookId": "0-4D24C77F19546939!40109",
      "displayName": "New Project Section"
    }
  }'
```

**Response:**
```json
{
  "content": {
    "id": "0-4D24C77F19546939!40109!1-4D24C77F19546939!40115",
    "displayName": "New Project Section",
    "createdDateTime": "2024-01-15T10:30:00Z"
  }
}
```

### 6. createPage

Creates a new page in a specified section with HTML content.

**Parameters:**
- `sectionId` (string, required): The ID of the section to create the page in
- `title` (string, required): The title of the new page
- `content` (string, required): HTML content for the page body

**Returns:** Created page object with metadata

**Example:**
```bash
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "createPage",
    "arguments": {
      "sectionId": "0-4D24C77F19546939!40109!1-4D24C77F19546939!40109",
      "title": "New Meeting Notes",
      "content": "<h1>Meeting Notes</h1><p>This is a new page with <strong>formatted</strong> content.</p>"
    }
  }'
```

**Response:**
```json
{
  "content": {
    "id": "0-4D24C77F19546939!40109!1-4D24C77F19546939!40109!1-4D24C77F19546939!40113",
    "title": "New Meeting Notes",
    "createdDateTime": "2024-01-15T10:30:00Z"
  }
}
```

### 7. updatePageContent

Updates the HTML content of an existing page with simple replacement.

**Parameters:**
- `pageId` (string, required): The ID of the page to update
- `content` (string, required): New HTML content for the page

**Returns:** Success confirmation message

**Example:**
```bash
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "updatePageContent",
    "arguments": {
      "pageId": "0-4D24C77F19546939!40109!1-4D24C77F19546939!40109!1-4D24C77F19546939!40111",
      "content": "<h1>Updated Meeting Notes</h1><p>This content has been <em>updated</em>.</p>"
    }
  }'
```

**Response:**
```json
{
  "content": "Page content updated successfully"
}
```

### 7.5. updatePageContentAdvanced

Update specific parts of a OneNote page using advanced commands (append, insert, prepend, replace, delete). This is the preferred way to add, change, or delete sections of a page. Use this tool to target specific elements (by `data-id`, generated id, or special targets like 'body' or 'title') and perform fine-grained updates. Only use a full page update if you intend to replace the entire page content.

**CRITICAL: Table Update Restrictions**
- **Tables must be updated as complete units**: You CANNOT update individual table cells (td), headers (th), or rows (tr)
- **Target the entire table**: Always use the table element's data-id as the target
- **Replace complete table HTML**: Include the full table structure in your replacement content
- **Example**: Target "table:{table-data-id}" and replace with complete "<table>...</table>" HTML
- **Why**: OneNote requires table integrity to prevent layout corruption

**How to get data-id values:**
- Use the `getPageContent` tool with `forUpdate=true` to retrieve HTML with `data-id` attributes. These data-id values can then be used as targets in your update commands.

**How to use the data-tag attribute for built-in note tags:**
- Use the `data-tag` attribute to add and update check boxes, stars, and other built-in note tags on a OneNote page.
- To add or update a built-in note tag, just use the `data-tag` attribute on a supported element.
- You can define a `data-tag` on the following elements: `p`, `ul`, `ol`, `li` (see more about note tags on lists), `img`, `h1`-`h6`, `title`.
- Guidelines for lists:
  - Use `p` elements for to-do lists (no bullet/number, easier to update).
  - To create/update lists with the same note tag for all items, define `data-tag` on the `ul` or `ol`.
  - To create/update lists with unique note tags for some/all items, define `data-tag` on `li` elements and do not nest them in a `ul` or `ol`.
  - To update specific `li` elements, target them individually and define `data-tag` on the `li`.
- Microsoft Graph rules:
  - The `data-tag` on a `ul` or `ol` overrides all child `li` elements.
  - Unique `data-tag` settings are honored for list items only if the `li` elements are not nested in a `ul` or `ol`, or if an `li` is individually addressed in an update.
  - Unnested `li` elements sent in input HTML are returned in a `ul` in the output HTML.
  - In output HTML, all data-tag list settings are defined on `span` elements on the list items.
- **Possible data-tag values:**
  - shape[:status], to-do, to-do:completed, important, question, definition, highlight, contact, address, phone-number, web-site-to-visit, idea, password, critical, project-a, project-b, remember-for-later, movie-to-see, book-to-read, music-to-listen-to, source-for-article, remember-for-blog, discuss-with-person-a, discuss-with-person-a:completed, discuss-with-person-b, discuss-with-person-b:completed, discuss-with-manager, discuss-with-manager:completed, send-in-email, schedule-meeting, schedule-meeting:completed, call-back, call-back:completed, to-do-priority-1, to-do-priority-1:completed, to-do-priority-2, to-do-priority-2:completed, client-request, client-request:completed

**How to position elements absolutely on a OneNote page:**
- The body element must specify `data-absolute-enabled="true"`. If omitted or set to false, all body content is rendered inside a _default absolute positioned div and all position settings are ignored.
- Only `div`, `img`, and `object` elements can be absolute positioned elements.
- Absolute positioned elements must specify `style="position:absolute"`.
- Absolute positioned elements must be direct children of the body element. Any direct children of the body that aren't absolute positioned `div`, `img`, or `object` elements are rendered as static content inside the absolute positioned _default div.
- Absolute positioned elements are positioned at their specified `top` and `left` coordinates, relative to the 0:0 starting position at the top, left corner of the page above the title area.
- If an absolute positioned element omits the `top` or `left` coordinate, the missing coordinate is set to its default value: `top:120px` or `left:48px`.
- Absolute positioned elements cannot be nested or contain positioned elements. The API ignores any position settings specified on nested elements inside an absolute positioned div, renders the nested content inside the absolute positioned parent div, and returns a warning in the api.diagnostics property in the response.

**Parameters:**
- `pageId` (string, required): The ID of the page to update
- `commands` (array, required): JSON array of update commands with target, action, position, and content

**Preferred Usage:** Use this tool to add, change, or delete parts of a page. Only use a full page update if you intend to replace the entire page content.

**Examples:**
- Append content to a section:
  ```json
  [
    {"target": "data-id:section-123", "action": "append", "content": "<p>New content</p>"}
  ]
  ```
- Replace the title:
  ```json
  [
    {"target": "title", "action": "replace", "content": "New Page Title"}
  ]
  ```
- Delete a specific element:
  ```json
  [
    {"target": "data-id:item-456", "action": "delete"}
  ]
  ```
- Insert content before an element:
  ```json
  [
    {"target": "data-id:section-789", "action": "insert", "position": "before", "content": "<div>Inserted above</div>"}
  ]
  ```

**Note:** For `append` actions, do not include the `position` parameter as it will be automatically excluded from the API request.

**Response:**
```json
{
  "content": "Page content updated successfully with advanced commands"
}
```

### 8. deletePage

Deletes a page by its ID.

**Parameters:**
- `pageId` (string, required): The ID of the page to delete

**Returns:** Success confirmation message

**Example:**
```bash
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "deletePage",
    "arguments": {
      "pageId": "0-4D24C77F19546939!40109!1-4D24C77F19546939!40109!1-4D24C77F19546939!40111"
    }
  }'
```

**Response:**
```json
{
  "content": "Page deleted successfully"
}
```

### 9. getPageContent

Retrieves the HTML content of a OneNote page by ID. Optionally, set the `forUpdate` parameter to `true` to include `data-id` attributes in the returned HTML. These `data-id` values are required for advanced page updates and allow you to target specific elements for modification, insertion, or deletion.

**Parameters:**
- `pageId` (string, required): The ID of the page to fetch content for
- `forUpdate` (string, optional): Set to 'true' to include `data-id` attributes for advanced updates

**Returns:** HTML content as a string

**Tip:** Use `forUpdate=true` to extract `data-id` values for use with the advanced update tool (see below).

**Example:**
```bash
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "getPageContent",
    "arguments": {
      "pageId": "0-4D24C77F19546939!40109!1-4D24C77F19546939!40109!1-4D24C77F19546939!40111",
      "forUpdate": "true"
    }
  }'
```

**Response:**
```json
{
  "content": "<html><head><title>Meeting Notes - 2024-01-15</title></head><body><h1>Meeting Notes</h1><p>This is the page content with <strong>formatting</strong>.</p><img src=\"https://graph.microsoft.com/beta/me/onenote/resources/0-896fbac8f72d01b02c5950345e65f588!1-4D24C77F19546939!39705/$value\" data-src-type=\"image/jpeg\" alt=\"Meeting diagram\" /></body></html>"
}
```

### 10. listPageItems

Lists all embedded items (images, files, objects) within a page.

**Parameters:**
- `pageId` (string, required): The ID of the page to list items for

**Returns:** Array of page item objects with metadata

**Example:**
```bash
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "listPageItems",
    "arguments": {
      "pageId": "0-4D24C77F19546939!40109!1-4D24C77F19546939!40109!1-4D24C77F19546939!40111"
    }
  }'
```

**Response:**
```json
{
  "content": [
    {
      "tagName": "img",
      "pageItemId": "0-896fbac8f72d01b02c5950345e65f588!1-4D24C77F19546939!39705",
      "attributes": {
        "src": "https://graph.microsoft.com/beta/me/onenote/resources/0-896fbac8f72d01b02c5950345e65f588!1-4D24C77F19546939!39705/$value",
        "data-src-type": "image/jpeg",
        "alt": "Meeting diagram",
        "width": "800",
        "height": "600"
      },
      "originalUrl": "https://graph.microsoft.com/beta/me/onenote/resources/0-896fbac8f72d01b02c5950345e65f588!1-4D24C77F19546939!39705/$value"
    }
  ]
}
```

### 11. getPageItem

Retrieves complete page item data including binary content and metadata.

**Parameters:**
- `pageId` (string, required): The ID of the page containing the item
- `pageItemId` (string, required): The ID of the page item to retrieve

**Returns:** JSON object with base64-encoded content and metadata

**Example:**
```bash
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "getPageItem",
    "arguments": {
      "pageId": "0-4D24C77F19546939!40109!1-4D24C77F19546939!40109!1-4D24C77F19546939!40111",
      "pageItemId": "0-896fbac8f72d01b02c5950345e65f588!1-4D24C77F19546939!39705"
    }
  }'
```

**Response:**
```json
{
  "content": {
    "contentType": "image/jpeg",
    "filename": "0-896fbac8f72d01b02c5950345e65f588!1-4D24C77F19546939!39705.jpg",
    "size": 24576,
    "content": "data:;base64,/9j/4AAQSkZJRgABAQEAYABgAAD/2wBDAAEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQH/2wBDAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQH/wAARCAABAAEDASIAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAv/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/8QAFQEBAQAAAAAAAAAAAAAAAAAAAAX/xAAUEQEAAAAAAAAAAAAAAAAAAAAA/9oADAMBAAIRAxAAPwA/8A",
    "tagName": "img",
    "attributes": {
      "src": "https://graph.microsoft.com/beta/me/onenote/resources/0-896fbac8f72d01b02c5950345e65f588!1-4D24C77F19546939!39705/$value",
      "data-src-type": "image/jpeg",
      "alt": "Meeting diagram",
      "width": "800",
      "height": "600"
    },
    "originalUrl": "https://graph.microsoft.com/beta/me/onenote/resources/0-896fbac8f72d01b02c5950345e65f588!1-4D24C77F19546939!39705/$value"
  }
}
```

### 12. searchPages

Searches for pages by title within a specific notebook using case-insensitive matching. **Note:** This function recursively searches through all sections and section groups in the notebook, providing comprehensive coverage of the entire notebook structure.

**Parameters:**
- `query` (string, required): The search query to match against page titles
- `notebookId` (string, optional): The ID of the notebook to search within. If not provided, uses the default notebook from configuration

**Returns:** Array of matching page objects with section context information

**Example:**
```bash
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "searchPages",
    "arguments": {
      "query": "meeting",
      "notebookId": "0-4D24C77F19546939!40109"
    }
  }'
```

**Response:**
```json
{
  "content": [
    {
      "id": "0-4D24C77F19546939!40109!1-4D24C77F19546939!40109!1-4D24C77F19546939!40111",
      "title": "Meeting Notes - 2024-01-15",
      "sectionName": "Quick Notes",
      "sectionId": "0-4D24C77F19546939!40109!1-4D24C77F19546939!40109",
      "sectionPath": "/Quick Notes"
    },
    {
      "id": "0-4D24C77F19546939!40109!1-4D24C77F19546939!40110!1-4D24C77F19546939!40114",
      "title": "Team Meeting Agenda",
      "sectionName": "Work Projects",
      "sectionId": "0-4D24C77F19546939!40109!1-4D24C77F19546939!40110",
      "sectionPath": "/Work Projects/Project A"
    }
  ]
}
```

**Note:** This search function:
- Recursively traverses all sections and section groups in the specified notebook
- Uses client-side filtering for reliable title matching
- Provides rich context information including section name, ID, and hierarchy path
- Handles nested section groups automatically
- Continues searching even if individual sections fail
- Returns comprehensive results with full location context

### 13. copyPage

Copies a page from one section to another using the Microsoft Graph API's copyToSection endpoint. This operation is asynchronous and the server automatically polls for completion.

**Parameters:**
- `pageId` (string, required): The ID of the page to copy
- `targetSectionId` (string, required): The ID of the target section to copy the page to

**Returns:** Copied page metadata with operation details

**Example:**
```bash
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "copyPage",
    "arguments": {
      "pageId": "0-4D24C77F19546939!40109!1-4D24C77F19546939!40109!1-4D24C77F19546939!40111",
      "targetSectionId": "0-4D24C77F19546939!40109!1-4D24C77F19546939!40110"
    }
  }'
```

**Response:**
```json
{
  "content": {
    "id": "0-51a71cfa797e422f84f3096603953930!6-4D24C77F19546939!40109",
    "operationId": "copy-operation-id-123",
    "status": "Completed"
  }
}
```

**Note:** This operation uses the Microsoft Graph beta API's copyToSection endpoint. The operation is asynchronous and the server automatically:
- Validates the 202 status code for accepted operations
- Extracts the operation ID from the response
- Polls the operation status up to 30 times with random delays (1-3 seconds)
- Returns the new page ID when the operation completes successfully
- Handles operation failures and timeouts gracefully

### 14. getOnenoteOperation

Gets the status of an asynchronous OneNote operation (e.g., copy page operation).

**Parameters:**
- `operationId` (string, required): The ID of the operation to check status for

**Returns:** Operation status and metadata

**Example:**
```bash
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "getOnenoteOperation",
    "arguments": {
      "operationId": "copy-operation-id-123"
    }
  }'
```

**Response:**
```json
{
  "content": {
    "id": "copy-operation-id-123",
    "status": "Completed",
    "resourceLocation": "https://graph.microsoft.com/beta/users/gabe@landq.org/onenote/pages/0-51a71cfa797e422f84f3096603953930!6-4D24C77F19546939!40109",
    "createdDateTime": "2025-01-18T20:16:25Z",
    "lastActionDateTime": "2025-01-18T20:16:30Z"
  }
}
```

**Note:** This tool is primarily used internally by the copyPage operation, but can be used independently to check the status of any OneNote operation. The `resourceLocation` field contains the URL of the created resource when the operation completes successfully.

### 15. movePage

Moves a page from one section to another by copying it to the target section and then deleting the original.

**Parameters:**
- `pageId` (string, required): The ID of the page to move
- `targetSectionId` (string, required): The ID of the target section to move the page to

**Returns:** Moved page metadata with operation details

**Example:**
```bash
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "movePage",
    "arguments": {
      "pageId": "0-4D24C77F19546939!40109!1-4D24C77F19546939!40109!1-4D24C77F19546939!40111",
      "targetSectionId": "0-4D24C77F19546939!40109!1-4D24C77F19546939!40110"
    }
  }'
```

**Response:**
```json
{
  "content": {
    "id": "0-4D24C77F19546939!40109!1-4D24C77F19546939!40109!1-4D24C77F19546939!40113",
    "title": "Meeting Notes - 2024-01-15",
    "movedDateTime": "2024-01-15T11:30:00Z",
    "targetSectionId": "0-4D24C77F19546939!40109!1-4D24C77F19546939!40110",
    "originalPageId": "0-4D24C77F19546939!40109!1-4D24C77F19546939!40109!1-4D24C77F19546939!40111",
    "operation": "copy_and_delete"
  }
}
```

**Note:** This operation performs a copy-then-delete sequence. If the copy succeeds but the delete fails, the operation will still be considered successful (you'll have a copy in the target section). The response includes operation details to track the success of both steps.

## Error Handling

All tools return structured error responses when operations fail:

**Error Response Format:**
```json
{
  "error": {
    "type": "error_type",
    "message": "Detailed error message"
  }
}
```

**Common Error Types:**
- `authentication_error`: Token expired or invalid
- `permission_error`: Insufficient permissions
- `not_found_error`: Resource not found
- `validation_error`: Invalid input parameters
- `network_error`: Network connectivity issues
- `content_error`: Content processing failures

## Rate Limiting

The server respects Microsoft Graph API rate limits and implements automatic retry logic with exponential backoff.

## Content Types

The server supports various content types for embedded items:
- **Images**: JPEG, PNG, GIF, WebP
- **Documents**: PDF, Word, Excel, PowerPoint
- **Other**: Any binary content type

## Security Considerations

- All OneNote IDs are validated and sanitized
- Input parameters are validated before processing
- Sensitive data is not logged
- HTTPS is required for production deployments

## Performance Notes

- Large images are automatically scaled to reduce bandwidth
- Content is streamed efficiently to minimize memory usage
- Connection pooling is used for optimal performance
- Caching is implemented where appropriate 