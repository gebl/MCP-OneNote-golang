package tools

// Tool represents a callable tool/method in the MCP server
// Name: the method name (e.g., "listNotebooks")
// Description: a short description
// Toolset: the toolset/category (e.g., "notebooks")
type Tool struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	InputSchema  map[string]interface{} `json:"inputSchema"`
	OutputSchema map[string]interface{} `json:"outputSchema,omitempty"`
}

// ToolsetRegistry manages enabled toolsets and their tools
type ToolsetRegistry struct {
	Enabled map[string]bool
	tools   map[string]Tool
}

func NewToolsetRegistry(toolsets []string) *ToolsetRegistry {
	enabled := make(map[string]bool)
	for _, t := range toolsets {
		enabled[t] = true
	}
	return &ToolsetRegistry{
		Enabled: enabled,
		tools:   make(map[string]Tool),
	}
}

// RegisterTool adds a tool to the registry if its toolset is enabled
func (r *ToolsetRegistry) RegisterTool(tool Tool) {
	r.tools[tool.Name] = tool
}

// ListTools returns all registered tools
func (r *ToolsetRegistry) ListTools() []Tool {
	tools := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		tools = append(tools, t)
	}
	return tools
}
