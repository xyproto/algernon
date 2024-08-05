package ollamaclient

// Tool represents a tool or function that can be used by the Ollama client
type Tool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Parameters  struct {
			Type       string                  `json:"type"`
			Properties map[string]ToolProperty `json:"properties"`
			Required   []string                `json:"required"`
		} `json:"parameters"`
	} `json:"function"`
}

// ToolProperty represents a property of a tool's parameter
type ToolProperty struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Enum        []string `json:"enum"`
}

// ToolCallFunction represents the function call details within a tool call
type ToolCallFunction struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

// ToolCall represents a call to a tool function
type ToolCall struct {
	Function ToolCallFunction `json:"function"`
}

// OutputChat represents the output from a chat request, including the role, content, tool calls, and any errors
type OutputChat struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls"`
	Error     string     `json:"error"`
}
