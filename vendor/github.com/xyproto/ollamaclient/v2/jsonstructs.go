package ollamaclient

import "time"

// RequestOptions holds the seed and temperature
type RequestOptions struct {
	Seed          int     `json:"seed"`
	Temperature   float64 `json:"temperature"`
	ContextLength int64   `json:"num_ctx,omitempty"`
}

// GenerateRequest represents the request payload for generating output
type GenerateRequest struct {
	Model   string         `json:"model"`
	Prompt  string         `json:"prompt,omitempty"`
	Images  []string       `json:"images,omitempty"` // base64 encoded images
	Stream  bool           `json:"stream,omitempty"`
	Options RequestOptions `json:"options,omitempty"`
}

// GenerateResponse represents the response data from the generate API call
type GenerateResponse struct {
	Model              string `json:"model"`
	CreatedAt          string `json:"created_at"`
	Response           string `json:"response"`
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

// EmbeddingsRequest represents the request payload for getting embeddings
type EmbeddingsRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// EmbeddingsResponse represents the response data containing embeddings
type EmbeddingsResponse struct {
	Embeddings []float64 `json:"embedding"`
}

// Model represents a downloaded model
type Model struct {
	Modified time.Time `json:"modified_at"`
	Name     string    `json:"name"`
	Digest   string    `json:"digest"`
	Size     int64     `json:"size"`
}

// ListResponse represents the response data from the tag API call
type ListResponse struct {
	Models []Model `json:"models"`
}

// VersionResponse represents the response data containing the Ollama version
type VersionResponse struct {
	Version string `json:"version"`
}
