package resources

import (
	"fmt"
)

// toolDescriptions contains all tool descriptions as Go string constants
var toolDescriptions = map[string]string{
	"getAuthStatus": "Check authentication status including token expiry and refresh availability. Shows if you're logged in to OneNote.",

	"refreshToken": "Refresh the current authentication token to extend session without full re-authentication.",

	"initiateAuth": "Start OAuth authentication flow. Returns a URL to visit in your browser to authenticate with Microsoft. Server waits for callback on localhost:8080.\n\nInitial Setup workflow: 1. Start server 2. Check status with getAuthStatus 3. Authenticate with initiateAuth 4. Visit provided URL in browser 5. Server automatically receives callback and updates tokens. Always explain next steps to user.",

	"clearAuth": "Logout by clearing all stored authentication tokens. Requires re-authentication for future OneNote operations.",

	"listNotebooks": "List all OneNote notebooks accessible to the authenticated user. Returns notebook ID, name, and default status flags as a JSON array.\n\nAlways start here when users ask \"Show me my notebooks\" or \"What notebooks do I have?\". Use for ID discovery since users think in names but API requires IDs - this tool translates names to IDs. Pattern: listNotebooks() → find notebook by name matching → use ID for subsequent operations. Always tell user what was found: \"I found your 'Work' notebook (ID: xxx). It contains 5 sections...\"\n\nRESPONSE FORMAT: Returns JSON array with objects containing:\n- id: Notebook ID for API operations\n- name: Display name of the notebook\n- isAPIDefault: Boolean indicating if this is OneNote's default notebook according to Microsoft Graph API\n- isConfigDefault: Boolean indicating if this matches your server's configured default notebook name (ONENOTE_DEFAULT_NOTEBOOK_NAME)\n\nCRITICAL: This is the ONLY way to translate notebook names to IDs. All other tools require actual IDs, NOT names. Use this tool first to get the ID for any notebook name.",

	"createSection": "Create a new section in a notebook or section group.\n\nUser says \"Create a new section called X\" → Step 1: Use getNotebookSections first to see the structure → Step 2: Get container ID (use the notebook ID or section group ID as containerID) → Step 3: createSection(containerID, cleanName). Always validate names for illegal characters and suggest alternatives.\n\n**HIERARCHY:** Sections can only be created inside notebooks or section groups, NOT inside other sections.\n**NAME RESTRICTIONS:** Cannot contain: ? * \\ / : < > | & # ' ' % ~\nUse alternatives: & → \"and\", / → \"-\", : → \"-\", etc.",

	"createSectionGroup": "Create a new section group in a notebook or another section group.\n\nSection groups can contain sections and other section groups, providing hierarchical organization. User says \"Create a folder/group called X\" → Step 1: Use getNotebookSections first to see the structure → Step 2: Get container ID (use the notebook ID or section group ID as containerID) → Step 3: createSectionGroup(containerID, cleanName).\n\n**HIERARCHY:** Section groups can be created inside notebooks or other section groups, NOT inside sections.\n**NAME RESTRICTIONS:** Cannot contain: ? * \\ / : < > | & # ' ' % ~\nUse alternatives: & → \"and\", / → \"-\", : → \"-\", etc.",

	"getSelectedNotebook": "Get the currently selected notebook's metadata from the server's memory cache.\n\nReturns the full notebook object with all metadata including ID, displayName, sections, etc. This is the \"active\" notebook that other tools will operate on by default.\n\nIf no notebook is selected, returns an error message instructing the user to use selectNotebook first.\n\nThe notebook is initially set to the configured default notebook on server startup, or the first available notebook if no default is configured.",

	"selectNotebook": "Select a notebook by name or ID to set as the active notebook in the server's memory cache.\n\nThe selected notebook becomes the \"active\" notebook that other tools operate on by default. This notebook choice persists for the entire server session.\n\nParameters:\n- identifier (required): Either the notebook name (e.g., \"My Work Notebook\") or notebook ID\n\nAfter selection, the notebook's full metadata is cached in memory for fast retrieval by getSelectedNotebook and other tools.\n\nUse listNotebooks first to see available notebooks if you're unsure of the exact name or need the ID.",

	"getNotebookSections": "Shows your notebook's organizational structure - sections and section groups in a tree format. Use this first to understand your notebook layout and get IDs for other operations.\n\nDisplays sections and section groups nested like folders containing subfolders and files. For example, if you have a 'Tasks' section group with sub-sections for 'Work', 'Personal', and 'Projects', this tool will show that hierarchy.\n\n**How to use:** First use this tool to see your notebook's structure, then use listPages on a specific section to view the pages within that section.\n\nFeatures:\n- Returns nested tree structure with all sections and section groups\n- Handles pagination automatically to retrieve all results\n- Caches results in memory for fast subsequent access (5-minute cache)\n- Provides IDs needed for other operations like listPages and createSection\n\nResponse includes notebook root object with displayName, id, and children array containing sections and section groups with full metadata.\n\nIf no notebook is currently selected, returns an error instructing to use selectNotebook first.",

	"clearCache": "Clear all cached data including notebook sections and pages cache.\n\nThis tool clears all cached data to force fresh retrieval from the Microsoft Graph API on the next request. This is useful when:\n- You suspect the cache contains stale data\n- You've made changes outside of this MCP server that aren't reflected in cached data\n- You want to ensure you're getting the most up-to-date information\n- Troubleshooting cache-related issues\n\nThe cache includes:\n- Sections tree structure from the selected notebook\n- Pages metadata for all sections that have been accessed\n- Does NOT clear the currently selected notebook (use selectNotebook to change that)\n\nAfter clearing the cache, subsequent calls to getNotebookSections and listPages will fetch fresh data from the API and rebuild the cache.",

	"listPages": "⚠️ CRITICAL: sectionID must be an actual ID (e.g. '0-abc123...'), never a section name. Use getNotebookSections to find section IDs.\n\nLists all pages in a specific section. You must use the exact Section ID (e.g., `0-...`) from tools like `getNotebookSections` or `selectNotebook`, not the section name. This is because section names are not unique and can change.\n\nReturns page ID, title, creation date, and metadata for each page.\n\nUser says \"Show me pages in X section\" → Step 1: Find section ID (search workflow) → Step 2: listPages(sectionID). For \"Get me the latest page from X section\" → listPages(sectionID) → sort by lastModifiedDateTime → getPageContent(mostRecentPageId). Always use discovery pattern - never assume IDs are known.",

	"getPageContent": "Get the full HTML content of a OneNote page by its ID. Returns complete page content including text, formatting, images, and structure.\n\nUser says \"Show me the content of page X\" → getPageContent(pageID). Set forUpdate='true' when you plan to use updatePageContentAdvanced - this adds data-id attributes to HTML elements that are required for targeting specific elements in updates.\n\n**CONTENT FORMAT:** Returns OneNote's native HTML format with special data-id attributes for element targeting in updates. Images are embedded as data URLs for immediate display.\n\n**UPDATE PREPARATION:** Use forUpdate=true when you plan to update the page content - this includes data-id attributes required by updatePageContentAdvanced tool.",

	"createPage": "Create a new page in a section with content that can be HTML, Markdown, or plain text. **AUTOMATIC FORMAT DETECTION:** The tool intelligently detects your content format and converts it to proper HTML for OneNote.\n\nUser says \"Create a page called X with content Y\" → createPage(sectionID, title, content). Always validate title for illegal characters and suggest alternatives.\n\n**CONTENT FORMAT SUPPORT:**\n- **HTML:** Direct HTML input is preserved as-is\n- **Markdown:** Automatically converted to HTML (headers, lists, code blocks, tables, etc.)\n- **Plain Text:** Automatically wrapped in proper HTML paragraph tags with line break handling\n\n**FORMAT DETECTION:** Automatically detects:\n- HTML: Content with HTML tags (e.g., `<p>`, `<h1>`, `<div>`)\n- Markdown: Content with Markdown syntax (e.g., `# Headers`, `- Lists`, `**Bold**`)\n- ASCII: Plain text content (automatically wrapped in `<p>` tags)\n\n**TITLE RESTRICTIONS:** Cannot contain: ? * \\ / : < > | & # ' ' % ~\nUse alternatives: & → \"and\", / → \"-\", : → \"-\", etc.\n\n**EXAMPLES:**\n- Plain text: \"This is my note\" → Converts to `<p>This is my note</p>`\n- Markdown: \"# Header\\n- Item 1\\n- Item 2\" → Converts to proper HTML\n- HTML: \"<h1>Header</h1><p>Content</p>\" → Used as-is\n\n**RESPONSE:** Returns success status, page ID, detected format, and conversion details.",

	"updatePageContentAdvanced": "Update specific elements in a OneNote page using command-based targeting with automatic format detection. **AUTOMATIC FORMAT DETECTION:** Each command's content can be HTML, Markdown, or plain text - automatically detected and converted to proper HTML.\n\n**WORKFLOW:** Step 1: getPageContent(pageID, forUpdate=true) to get current content with data-id attributes → Step 2: Identify target elements and their data-id values → Step 3: updatePageContentAdvanced with commands array.\n\n**CONTENT FORMAT SUPPORT (per command):**\n- **HTML:** Direct HTML input is preserved as-is\n- **Markdown:** Automatically converted to HTML (headers, lists, code blocks, tables, etc.)\n- **Plain Text:** Automatically wrapped in proper HTML paragraph tags with line break handling\n\n**TARGETING OPTIONS:**\n- data-id: \"data-id:element-123\" (most precise, get from getPageContent with forUpdate=true)\n- title: \"title\" (targets page title)\n- element selectors: \"h1\", \"p:first\", \"table\" (CSS-like selectors)\n\n**COMMANDS:**\n- append: Add content after target element\n- insert: Add content at specific position (after, before, inside)\n- replace: Replace target element completely\n- delete: Remove target element\n\n**POSITION OPTIONS (for insert):**\n- after: Insert after the target element\n- before: Insert before the target element  \n- inside: Insert inside the target element (at the end)\n\n**TABLE RESTRICTIONS:** Tables must be updated as complete units. You cannot update individual cells - you must replace the entire table element.\n\n**COMMAND FORMAT:**\n```json\n{\n  \"target\": \"data-id:element-123\",\n  \"action\": \"replace\",\n  \"content\": \"# Markdown Header\\n- List item\",\n  \"position\": \"after\"\n}\n```\n\n**FORMAT DETECTION EXAMPLES:**\n- Replace with Markdown: {\"target\": \"title\", \"action\": \"replace\", \"content\": \"# New Header\"}\n- Append plain text: {\"target\": \"data-id:element-123\", \"action\": \"append\", \"content\": \"Simple text paragraph\"}\n- Insert HTML: {\"target\": \"data-id:element-456\", \"action\": \"insert\", \"content\": \"<strong>Bold HTML</strong>\", \"position\": \"before\"}\n- Markdown list: {\"target\": \"body\", \"action\": \"append\", \"content\": \"- Item 1\\n- Item 2\\n- Item 3\"}\n\n**RESPONSE:** Returns success status, commands processed count, and detailed format detection results for each command including detected format and conversion metrics.",

	"deletePage": "Delete a OneNote page permanently. This action cannot be undone.\n\nUser says \"Delete page X\" → confirm with user → deletePage(pageID). Always confirm destructive operations with the user before proceeding.\n\n**WARNING:** This is a permanent operation. Once deleted, the page and all its content cannot be recovered. Always verify the correct page ID and consider asking for user confirmation.",

	"getPageItemContent": "Get a page item (image, file) by ID. Returns binary data with proper MIME type and automatically scales images unless fullSize is specified.",

	"listPageItems": "List all items (images, files) attached to a specific page. Returns JSON array with pageItemID, tagName, type, and data-attachment (if present).",

	"copyPage": "Copy a page from one section to another. Creates a duplicate page in the target section.",

	"movePage": "Move a page from one section to another by copying then deleting the original.",

	"updatePageContent": "Replace the entire content of a page with content that can be HTML, Markdown, or plain text. **AUTOMATIC FORMAT DETECTION:** The tool intelligently detects your content format and converts it to proper HTML for OneNote.\n\n**CONTENT FORMAT SUPPORT:**\n- **HTML:** Direct HTML input is preserved as-is\n- **Markdown:** Automatically converted to HTML (headers, lists, code blocks, tables, etc.)\n- **Plain Text:** Automatically wrapped in proper HTML paragraph tags with line break handling\n\n**FORMAT DETECTION:** Automatically detects:\n- HTML: Content with HTML tags (e.g., `<p>`, `<h1>`, `<div>`)\n- Markdown: Content with Markdown syntax (e.g., `# Headers`, `- Lists`, `**Bold**`)\n- ASCII: Plain text content (automatically wrapped in `<p>` tags)\n\n**USE CASE:** Use for complete page content replacement. For partial updates to specific elements, use updatePageContentAdvanced instead.\n\n**EXAMPLES:**\n- Plain text: \"Updated content\" → Converts to `<p>Updated content</p>`\n- Markdown: \"# New Header\\n- Updated item\" → Converts to proper HTML\n- HTML: \"<h1>New Header</h1><p>Updated content</p>\" → Used as-is\n\n**RESPONSE:** Returns success status, detected format, and conversion details.",
}

// GetToolDescription returns the description for a specific tool
func GetToolDescription(toolName string) (string, error) {
	desc, exists := toolDescriptions[toolName]
	if !exists {
		return "", fmt.Errorf("description not found for tool: %s", toolName)
	}
	return desc, nil
}

// MustGetToolDescription returns the description for a tool or panics if not found
// This should only be used during server initialization where we want to fail fast
func MustGetToolDescription(toolName string) string {
	desc, exists := toolDescriptions[toolName]
	if !exists {
		panic(fmt.Sprintf("Tool description not found: %s", toolName))
	}
	return desc
}

// GetAllDescriptions returns all available tool descriptions
func GetAllDescriptions() map[string]string {
	// Return a copy to prevent modification of the original map
	result := make(map[string]string, len(toolDescriptions))
	for k, v := range toolDescriptions {
		result[k] = v
	}
	return result
}
