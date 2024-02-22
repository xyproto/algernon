// Package ollamaclient can be used for communicating with the Ollama service
package ollamaclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/allegro/bigcache/v3"
	"github.com/xyproto/env/v2"
)

const (
	defaultModel       = "gemma"          // tinyllama would also be a good default
	defaultHTTPTimeout = 10 * time.Minute // per HTTP request to Ollama
	defaultFixedSeed   = 256              // for when generated output should not be random, but have temperature 0 and a specific seed
	defaultPullTimeout = 48 * time.Hour   // pretty generous, in case someone has a poor connection
)

// Cache is used for caching reproducible results from Ollama (seed -1, temperature 0)
var Cache *bigcache.BigCache = nil

// InitCache initializes the BigCache cache
func InitCache() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	config := bigcache.DefaultConfig(24 * time.Hour)
	config.HardMaxCacheSize = 256 // MB
	config.StatsEnabled = false
	config.Verbose = false
	c, err := bigcache.New(ctx, config)
	if err != nil {
		return err
	}
	Cache = c
	return nil
}

// Config represents configuration details for communicating with the Ollama API
type Config struct {
	ServerAddr                string
	ModelName                 string
	SeedOrNegative            int
	TemperatureIfNegativeSeed float64
	PullTimeout               time.Duration
	HTTPTimeout               time.Duration
	TrimSpace                 bool
	Verbose                   bool
}

// New initializes a new Config using environment variables
func New() *Config {
	return &Config{
		ServerAddr:                env.Str("OLLAMA_HOST", "http://localhost:11434"),
		ModelName:                 env.Str("OLLAMA_MODEL", defaultModel),
		SeedOrNegative:            defaultFixedSeed,
		TemperatureIfNegativeSeed: 0.8,
		PullTimeout:               defaultPullTimeout,
		HTTPTimeout:               defaultHTTPTimeout,
		TrimSpace:                 true,
		Verbose:                   env.Bool("OLLAMA_VERBOSE"),
	}
}

// NewConfig initializes a new Config using a specified model, address (like http://localhost:11434) and a verbose bool
func NewConfig(serverAddr, modelName string, seedOrNegative int, temperatureIfNegativeSeed float64, pTimeout, hTimeout time.Duration, trimSpace, verbose bool) *Config {
	return &Config{
		ServerAddr:                serverAddr,
		ModelName:                 modelName,
		SeedOrNegative:            seedOrNegative,
		TemperatureIfNegativeSeed: temperatureIfNegativeSeed,
		PullTimeout:               pTimeout,
		HTTPTimeout:               hTimeout,
		TrimSpace:                 trimSpace,
		Verbose:                   verbose,
	}
}

// SetReproducible configures the generated output to be reproducible, with temperature 0 and a specific seed.
// It takes an optional random seed.
func (oc *Config) SetReproducible(optionalSeed ...int) {
	if len(optionalSeed) > 0 {
		oc.SeedOrNegative = optionalSeed[0]
		return
	}
	oc.SeedOrNegative = defaultFixedSeed
}

// SetRandom configures the generated output to not be reproducible
func (oc *Config) SetRandom() {
	oc.SeedOrNegative = -1
}

// GetOutput sends a request to the Ollama API and returns the generated output.
func (oc *Config) GetOutput(prompt string) (string, error) {
	var (
		temperature float64
		cacheKey    string
		seed        = oc.SeedOrNegative
	)
	if seed < 0 {
		temperature = oc.TemperatureIfNegativeSeed
	} else {
		// The cache is only used for fixed seeds and a temperature of 0
		cacheKey = prompt + "-" + oc.ModelName
		if Cache == nil {
			if err := InitCache(); err != nil {
				return "", err
			}
		}
		if entry, err := Cache.Get(cacheKey); err == nil {
			return string(entry), nil
		}
	}
	reqBody := GenerateRequest{
		Model:  oc.ModelName,
		Prompt: prompt,
		Options: RequestOptions{
			Seed:        seed,        // set to -1 to make it random
			Temperature: temperature, // set to 0 together with a specific seed to make output reproducible
		},
	}
	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}
	if oc.Verbose {
		fmt.Printf("Sending request to %s/api/generate: %s\n", oc.ServerAddr, string(reqBytes))
	}
	HTTPClient := &http.Client{
		Timeout: oc.HTTPTimeout,
	}
	resp, err := HTTPClient.Post(oc.ServerAddr+"/api/generate", "application/json", bytes.NewBuffer(reqBytes))
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
	outputString := strings.TrimPrefix(sb.String(), "\n")
	if oc.TrimSpace {
		outputString = strings.TrimSpace(outputString)
	}
	if cacheKey != "" {
		Cache.Set(cacheKey, []byte(outputString))
	}
	return outputString, nil
}

// MustOutput returns the output from Ollama, or the error as a string if not
func (oc *Config) MustOutput(prompt string) string {
	output, err := oc.GetOutput(prompt)
	if err != nil {
		return err.Error()
	}
	return output
}

// List collects info about the currently downloaded models
func (oc *Config) List() ([]string, map[string]time.Time, map[string]int64, error) {
	if oc.Verbose {
		fmt.Printf("Sending request to %s/api/tags\n", oc.ServerAddr)
	}
	resp, err := http.Get(oc.ServerAddr + "/api/tags")
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
	return -1, fmt.Errorf("could not find model: %s", model)
}

// Has returns true if the given model exists
func (oc *Config) Has(model string) (bool, error) {
	model = strings.TrimSpace(model)
	if !strings.Contains(model, ":") {
		model += ":latest"
	}
	if names, _, _, err := oc.List(); err == nil { // success
		for _, name := range names {
			if name == model {
				return true, nil
			}
		}
	} else {
		return false, err
	}
	return false, nil // could list models, but could not find the given model name
}

// HasModel returns true if the configured model exists
func (oc *Config) HasModel() (bool, error) {
	return oc.Has(oc.ModelName)
}

// PullIfNeeded pulls a model, but only if it's not already there.
// While Pull downloads/updates the model regardless.
// Also takes an optional bool for if progress bars should be used when models are being downloaded.
func (oc *Config) PullIfNeeded(optionalVerbose ...bool) error {
	if found, err := oc.HasModel(); err != nil {
		return err
	} else if !found {
		if _, err := oc.Pull(optionalVerbose...); err != nil {
			return err
		}
	}
	return nil
}

// CloseCache signals the shutdown of the cache
func CloseCache() {
	if Cache != nil {
		Cache.Close()
	}
}

// ClearCache removes the current cache entries
func ClearCache() {
	if Cache != nil {
		Cache.Reset()
	}
}
