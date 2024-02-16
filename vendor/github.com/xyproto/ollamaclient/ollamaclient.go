// Package ollamaclient can be used for communicating with the Ollama service
package ollamaclient

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/xyproto/env/v2"
)

const defaultModel = "nous-hermes:7b-llama2-q2_K"

// Config represents configuration details for communicating with the Ollama API
type Config struct {
	API     string
	Model   string
	Verbose bool
}

// GenerateRequest represents the request payload for generating output
type GenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// GenerateResponse represents the response data from the generate API call
type GenerateResponse struct {
	Model              string `json:"model"`
	CreatedAt          string `json:"created_at"`
	Response           string `json:"response"`
	Done               bool   `json:"done"`
	Context            []int  `json:"context,omitempty"`
	TotalDuration      int64  `json:"total_duration,omitempty"`
	LoadDuration       int64  `json:"load_duration,omitempty"`
	SampleCount        int    `json:"sample_count,omitempty"`
	SampleDuration     int64  `json:"sample_duration,omitempty"`
	PromptEvalCount    int    `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64  `json:"prompt_eval_duration,omitempty"`
	EvalCount          int    `json:"eval_count,omitempty"`
	EvalDuration       int64  `json:"eval_duration,omitempty"`
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

// PullRequest represents the request payload for pulling a model
type PullRequest struct {
	Name     string `json:"name"`
	Insecure bool   `json:"insecure,omitempty"`
	Stream   bool   `json:"stream,omitempty"`
}

// PullResponse represents the response data from the pull API call
type PullResponse struct {
	Status string `json:"status"`
	Digest string `json:"digest"`
	Total  int64  `json:"total"`
}

// Model represents a downloaded model
type Model struct {
	Name     string    `json:"name"`
	Modified time.Time `json:"modified_at"`
	Size     int64     `json:"size"`
	Digest   string    `json:"digest"`
}

// ListResponse represents the response data from the tag API call
type ListResponse struct {
	Models []Model `json:"models"`
}

// New initializes a new Config using environment variables
func New() *Config {
	return &Config{
		env.Str("OLLAMA_HOST", "http://localhost:11434"),
		env.Str("OLLAMA_MODEL", defaultModel),
		env.Bool("OLLAMA_VERBOSE"),
	}
}

// NewWithModel initializes a new Config using a specified model and environment variables
func NewWithModel(model string) *Config {
	return &Config{
		env.Str("OLLAMA_HOST", "http://localhost:11434"),
		model,
		env.Bool("OLLAMA_VERBOSE"),
	}
}

// GetOutput sends a request to the Ollama API and returns the generated output
func (c *Config) GetOutput(prompt string) (string, error) {
	reqBody := GenerateRequest{
		Model:  c.Model,
		Prompt: prompt,
	}
	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}
	if c.Verbose {
		fmt.Printf("Sending request to /api/generate: %s\n", string(reqBytes))
	}
	resp, err := http.Post(c.API+"/api/generate", "application/json", bytes.NewBuffer(reqBytes))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var sb strings.Builder
	decoder := json.NewDecoder(resp.Body)
	for {
		var genResp GenerateResponse
		if err := decoder.Decode(&genResp); err != nil {
			break
		}
		sb.WriteString(genResp.Response)
		if genResp.Done {
			break
		}
	}
	return sb.String(), nil

}

// AddEmbedding sends a request to get embeddings for a given prompt
func (c *Config) AddEmbedding(prompt string) ([]float64, error) {
	reqBody := EmbeddingsRequest{
		Model:  c.Model,
		Prompt: prompt,
	}
	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return []float64{}, err
	}

	if c.Verbose {
		fmt.Printf("Sending request to /api/embeddings: %s\n", string(reqBytes))
	}

	resp, err := http.Post(c.API+"/api/embeddings", "application/json", bytes.NewBuffer(reqBytes))
	if err != nil {
		return []float64{}, err
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	var embResp EmbeddingsResponse
	if err := decoder.Decode(&embResp); err != nil {
		return []float64{}, err
	}
	return embResp.Embeddings, nil
}

// Pull sends a request to pull a specified model from the Ollama API
func (c *Config) Pull() (string, error) {
	reqBody := PullRequest{
		Name:   c.Model,
		Stream: true,
	}
	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}
	if c.Verbose {
		fmt.Printf("Sending request to /api/pull: %s\n", string(reqBytes))
	}

	resp, err := http.Post(c.API+"/api/pull", "application/json", bytes.NewBuffer(reqBytes))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var sb strings.Builder
	decoder := json.NewDecoder(resp.Body)

	if c.Verbose {
		fmt.Printf("Downloading and/or updating %s...", c.Model)
	}
	for {
		var pullResp PullResponse
		if err := decoder.Decode(&pullResp); err != nil {
			break
		}
		sb.WriteString(pullResp.Status)
		if !strings.HasPrefix(pullResp.Status, "downloading ") && !strings.HasPrefix(pullResp.Status, "pulling ") {
			if strings.HasPrefix(pullResp.Status, "verifying ") { // done downloading
				break
			}
			return "", fmt.Errorf("recevied status when downloading: %s", pullResp.Status)
		}
		if c.Verbose {
			fmt.Print(".")
		}
		// Update the progress status every second
		time.Sleep(1 * time.Second)
	}
	if c.Verbose {
		fmt.Println(" OK")
	}
	return sb.String(), nil
}

// List collects info about the currently downloaded models
func (c *Config) List() ([]string, map[string]time.Time, map[string]int64, error) {
	if c.Verbose {
		fmt.Println("Sending request to /api/tags")
	}
	resp, err := http.Get(c.API + "/api/tags")
	if err != nil {
		return nil, nil, nil, err
	}
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	var listResp ListResponse
	if err := decoder.Decode(&listResp); err != nil {
		return nil, nil, nil, err
	}
	var names []string
	modifiedMap := make(map[string]time.Time)
	sizeMap := make(map[string]int64)
	for _, model := range listResp.Models {
		names = append(names, model.Name)
		modifiedMap[model.Name] = model.Modified
		sizeMap[model.Name] = model.Size
	}
	return names, modifiedMap, sizeMap, nil
}

// SizeOf returns the current size of the given model, or returns (-1, err) if it can't be found
func (c *Config) SizeOf(model string) (int64, error) {
	names, _, sizeMap, err := c.List()
	if err != nil {
		return 0, err
	}
	for _, name := range names {
		if name == model {
			return sizeMap[name], nil
		}
	}
	return -1, errors.New("could not find model: " + model)
}

// Has returns true if the given model exists locally
func (c *Config) Has(model string) bool {
	if names, _, _, err := c.List(); err == nil { // success
		for _, name := range names {
			if name == model {
				return true
			}
		}
	} else {
		fmt.Println("error when calling /api/tags: " + err.Error())
	}
	return false
}

// HasModel returns true if the configured model exists locally
func (c *Config) HasModel() bool {
	return c.Has(c.Model)
}

// PullIfNeeded pulls a model, but only if it's not already there.
// While Pull downloads/updates the model regardless.
func (c *Config) PullIfNeeded() error {
	if !c.HasModel() {
		if _, err := c.Pull(); err != nil {
			return err
		}
	}
	return nil
}
