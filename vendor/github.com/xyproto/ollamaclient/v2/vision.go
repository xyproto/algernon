package ollamaclient

type VisionRequest struct {
	Prompt string   `json:"prompt"`
	Images []string `json:"images"`
}
