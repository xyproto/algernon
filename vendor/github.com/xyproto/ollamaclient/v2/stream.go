package ollamaclient

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// GenerateChatRequest represents the request payload for generating chat output
type GenerateChatRequest struct {
	Model    string            `json:"model"`
	Messages []Message         `json:"messages,omitempty"`
	Images   []string          `json:"images,omitempty"` // base64 encoded images
	Stream   bool              `json:"stream"`
	Tools    []json.RawMessage `json:"tools,omitempty"`
	Options  RequestOptions    `json:"options,omitempty"`
}

// GenerateChatResponse represents the response data from the generate chat API call
type GenerateChatResponse struct {
	Model              string          `json:"model"`
	CreatedAt          string          `json:"created_at"`
	Message            MessageResponse `json:"message"`
	DoneReason         string          `json:"done_reason"`
	Done               bool            `json:"done"`
	TotalDuration      int64           `json:"total_duration,omitempty"`
	LoadDuration       int64           `json:"load_duration,omitempty"`
	PromptEvalCount    int             `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64           `json:"prompt_eval_duration,omitempty"`
	EvalCount          int             `json:"eval_count,omitempty"`
	EvalDuration       int64           `json:"eval_duration,omitempty"`
}

// Message is a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// MessageResponse represents the response data from the generate API call
type MessageResponse struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls"`
}

// StreamOutput sends a request to the Ollama API and returns the generated output via a callback function.
// The callback function is given a string and "true" when the streaming is done (or if an error occurred).
func (oc *Config) StreamOutput(callbackFunction func(string, bool), promptAndOptionalImages ...string) error {
	defer callbackFunction("", true)
	var (
		temperature float64
		seed        = oc.SeedOrNegative
	)
	if len(promptAndOptionalImages) == 0 {
		return errors.New("at least one prompt must be given (and then optionally, base64 encoded JPG or PNG image strings)")
	}
	prompt := promptAndOptionalImages[0]
	var images []string
	if len(promptAndOptionalImages) > 1 {
		images = promptAndOptionalImages[1:]
	}
	if seed < 0 {
		temperature = oc.TemperatureIfNegativeSeed
	}
	var reqBody GenerateRequest
	if len(images) > 0 {
		reqBody = GenerateRequest{
			Model:  oc.ModelName,
			Prompt: prompt,
			Images: images,
			Stream: true,
			Options: RequestOptions{
				Seed:        seed,        // set to -1 to make it random
				Temperature: temperature, // set to 0 together with a specific seed to make output reproducible
			},
		}
	} else {
		reqBody = GenerateRequest{
			Model:  oc.ModelName,
			Prompt: prompt,
			Stream: true,
			Options: RequestOptions{
				Seed:        seed,        // set to -1 to make it random
				Temperature: temperature, // set to 0 together with a specific seed to make output reproducible
			},
		}
	}
	if oc.ContextLength != 0 {
		reqBody.Options.ContextLength = oc.ContextLength
	}
	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	if oc.Verbose {
		fmt.Printf("Sending request to %s/api/generate: %s\n", oc.ServerAddr, string(reqBytes))
	}
	HTTPClient := &http.Client{
		Timeout: oc.HTTPTimeout,
	}
	resp, err := HTTPClient.Post(oc.ServerAddr+"/api/generate", "application/json", bytes.NewBuffer(reqBytes))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var (
		decoder = json.NewDecoder(resp.Body)
		first   = true
		answer  string
	)
	for {
		var genResp GenerateResponse
		if err := decoder.Decode(&genResp); err != nil {
			break
		}
		answer = genResp.Response
		if first {
			if len(answer) > 0 && answer[0] == ' ' {
				answer = answer[1:]
			}
			first = false
		}
		callbackFunction(answer, false)
		if genResp.Done {
			break
		}
	}
	return nil
}
