package ollamaclient

// GenerateRequest represents the request payload for generating output
type GenerateRequest struct {
	Model   string         `json:"model"`
	System  string         `json:"system,omitempty"`
	Prompt  string         `json:"prompt,omitempty"`
	Suffix  string         `json:"suffix,omitempty"`
	Images  []string       `json:"images,omitempty"` // base64 encoded images
	Options RequestOptions `json:"options,omitempty"`
	Stream  bool           `json:"stream,omitempty"`
}

// GenerateResponse represents the response data from the generate API call
type GenerateResponse struct {
	Model              string `json:"model"`
	CreatedAt          string `json:"created_at"`
	Response           string `json:"response"`
	Error              string `json:"error,omitempty"`
	Context            []int  `json:"context,omitempty"`
	TotalDuration      int64  `json:"total_duration,omitempty"`
	LoadDuration       int64  `json:"load_duration,omitempty"`
	SampleCount        int    `json:"sample_count,omitempty"`
	SampleDuration     int64  `json:"sample_duration,omitempty"`
	PromptEvalCount    int    `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64  `json:"prompt_eval_duration,omitempty"`
	EvalCount          int    `json:"eval_count,omitempty"`
	EvalDuration       int64  `json:"eval_duration,omitempty"`
	Done               bool   `json:"done"`
}

// GenerateChatRequest represents the request payload for generating chat output
type GenerateChatRequest struct {
	Model    string         `json:"model"`
	Suffix   string         `json:"suffix,omitempty"`
	Messages []Message      `json:"messages,omitempty"`
	Images   []string       `json:"images,omitempty"` // base64 encoded images
	Tools    []Tool         `json:"tools,omitempty"`
	Options  RequestOptions `json:"options,omitempty"`
	Stream   bool           `json:"stream"`
}

// GenerateChatResponse represents the response data from the generate chat API call
type GenerateChatResponse struct {
	Model              string          `json:"model"`
	CreatedAt          string          `json:"created_at"`
	DoneReason         string          `json:"done_reason"`
	Error              string          `json:"error,omitempty"`
	Message            MessageResponse `json:"message"`
	TotalDuration      int64           `json:"total_duration,omitempty"`
	LoadDuration       int64           `json:"load_duration,omitempty"`
	PromptEvalCount    int             `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64           `json:"prompt_eval_duration,omitempty"`
	EvalCount          int             `json:"eval_count,omitempty"`
	EvalDuration       int64           `json:"eval_duration,omitempty"`
	Done               bool            `json:"done"`
}

// OutputResponse represents the output from Ollama
type OutputResponse struct {
	Role           string     `json:"role"`
	Response       string     `json:"response"`
	Error          string     `json:"error"`
	ToolCalls      []ToolCall `json:"tool_calls"`
	PromptTokens   int        `json:"prompt_tokens"`
	ResponseTokens int        `json:"response_tokens"`
}
