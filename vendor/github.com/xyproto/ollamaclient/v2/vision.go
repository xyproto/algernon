package ollamaclient

// VisionRequest represents a vision prompt with one or more base64 encoded images
type VisionRequest struct {
	Prompt string   `json:"prompt"`
	Images []string `json:"images"`
}
