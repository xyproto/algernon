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

const (
	defaultModel            = "nous-hermes:7b-llama2-q2_K" // tinyllama would also be a good default
	defaultHTTPTimeout      = 10 * time.Minute             // per HTTP request to Ollama
	defaultReproducibleSeed = 1337                         // for when generated output should not be random, but have temperature 0 and a specific seed
)

var (
	// HTTPClient is the HTTP client that will be used to communicate with the Ollama server
	HTTPClient = &http.Client{
		Timeout: defaultHTTPTimeout,
	}
)

// Config represents configuration details for communicating with the Ollama API
type Config struct {
	API              string
	Model            string
	Verbose          bool
	PullTimeout      time.Duration
	ReproducibleSeed int
}

// RequestOptions holds the seed and temperature
type RequestOptions struct {
	Seed        int     `json:"seed"`
	Temperature float64 `json:"temperature"`
}

// GenerateRequest represents the request payload for generating output
type GenerateRequest struct {
	Model   string         `json:"model"`
	Prompt  string         `json:"prompt"`
	Options RequestOptions `json:"options"`
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
		API:              env.Str("OLLAMA_HOST", "http://localhost:11434"),
		Model:            env.Str("OLLAMA_MODEL", defaultModel),
		Verbose:          env.Bool("OLLAMA_VERBOSE"),
		PullTimeout:      defaultPullTimeout,
		ReproducibleSeed: defaultReproducibleSeed,
	}
}

// NewWithModel initializes a new Config using a specified model and environment variables
func NewWithModel(model string) *Config {
	return &Config{
		API:              env.Str("OLLAMA_HOST", "http://localhost:11434"),
		Model:            model,
		Verbose:          env.Bool("OLLAMA_VERBOSE"),
		PullTimeout:      defaultPullTimeout,
		ReproducibleSeed: defaultReproducibleSeed,
	}
}

// NewWithAddr initializes a new Config using a specified address (like http://localhost:11434) and environment variables
func NewWithAddr(addr string) *Config {
	return &Config{
		API:              addr,
		Model:            env.Str("OLLAMA_MODEL", defaultModel),
		Verbose:          env.Bool("OLLAMA_VERBOSE"),
		PullTimeout:      defaultPullTimeout,
		ReproducibleSeed: defaultReproducibleSeed,
	}
}

// NewWithModelAndAddr initializes a new Config using a specified model, address (like http://localhost:11434) and environment variables
func NewWithModelAndAddr(model, addr string) *Config {
	return &Config{
		API:              addr,
		Model:            model,
		Verbose:          env.Bool("OLLAMA_VERBOSE"),
		PullTimeout:      defaultPullTimeout,
		ReproducibleSeed: defaultReproducibleSeed,
	}
}

// NewCustom initializes a new Config using a specified model, address (like http://localhost:11434) and a verbose bool
func NewCustom(model, addr string, verbose bool, pullTimeout time.Duration, reproducibleSeed int) *Config {
	return &Config{
		addr,
		model,
		verbose,
		pullTimeout,
		reproducibleSeed,
	}
}

// SetReproducibleOutput configures the generated output to be reproducible, with temperature 0 and a specific seed
func (oc *Config) SetReproducibleOutput() {
	oc.ReproducibleSeed = defaultReproducibleSeed
}

// SetRandomOutput configures the generated output to not be reproducible
func (oc *Config) SetRandomOutput() {
	oc.ReproducibleSeed = 0
}

// GetOutput sends a request to the Ollama API and returns the generated output
func (oc *Config) GetOutput(prompt string, optionalTrimSpace ...bool) (string, error) {
	reqBody := GenerateRequest{
		Model:  oc.Model,
		Prompt: prompt,
	}

	// Reproducible output
	if oc.ReproducibleSeed > 0 {
		reqBody.Options = RequestOptions{
			Seed:        oc.ReproducibleSeed,
			Temperature: 0,
		}
	} else {
		reqBody.Options = RequestOptions{
			Seed:        -1,
			Temperature: 0.7,
		}
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}
	if oc.Verbose {
		fmt.Printf("Sending request to %s/api/generate: %s\n", oc.API, string(reqBytes))
	}
	resp, err := HTTPClient.Post(oc.API+"/api/generate", "application/json", bytes.NewBuffer(reqBytes))
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
	if len(optionalTrimSpace) > 0 && optionalTrimSpace[0] {
		return strings.TrimSpace(sb.String()), nil
	}
	return strings.TrimPrefix(sb.String(), "\n"), nil
}

// MustOutput returns the output from Ollama, or the error as a string if not
func (oc *Config) MustOutput(prompt string) string {
	output, err := oc.GetOutput(prompt, true)
	if err != nil {
		return err.Error()
	}
	return output
}

// AddEmbedding sends a request to get embeddings for a given prompt
func (oc *Config) AddEmbedding(prompt string) ([]float64, error) {
	reqBody := EmbeddingsRequest{
		Model:  oc.Model,
		Prompt: prompt,
	}
	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return []float64{}, err
	}

	if oc.Verbose {
		fmt.Printf("Sending request to %s/api/embeddings: %s\n", oc.API, string(reqBytes))
	}

	resp, err := HTTPClient.Post(oc.API+"/api/embeddings", "application/json", bytes.NewBuffer(reqBytes))
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

// List collects info about the currently downloaded models
func (oc *Config) List() ([]string, map[string]time.Time, map[string]int64, error) {
	if oc.Verbose {
		fmt.Printf("Sending request to %s/api/tags\n", oc.API)
	}
	resp, err := http.Get(oc.API + "/api/tags")
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
func (oc *Config) SizeOf(model string) (int64, error) {
	model = strings.TrimSpace(model)
	if !strings.Contains(model, ":") {
		model += ":latest"
	}
	names, _, sizeMap, err := oc.List()
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
func (oc *Config) Has(model string) bool {
	model = strings.TrimSpace(model)
	if !strings.Contains(model, ":") {
		model += ":latest"
	}
	if names, _, _, err := oc.List(); err == nil { // success
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
func (oc *Config) HasModel() bool {
	return oc.Has(oc.Model)
}

// PullIfNeeded pulls a model, but only if it's not already there.
// While Pull downloads/updates the model regardless.
func (oc *Config) PullIfNeeded(optionalVerbose ...bool) error {
	if !oc.HasModel() {
		if _, err := oc.Pull(optionalVerbose...); err != nil {
			return err
		}
	}
	return nil
}
