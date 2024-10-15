package ollamaclient

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// GetBetweenResponse is given the start of code and end of code and will try to complete what is in between
// This function will ignore oc.TrimSpace and not trim blanks.
func (oc *Config) GetBetweenResponse(prompt, suffix string) (OutputResponse, error) {
	var (
		temperature float64
		cacheKey    string
		seed        = oc.SeedOrNegative
	)
	if prompt == "" {
		return OutputResponse{}, errors.New("the prompt can not be empty")
	}
	if seed < 0 {
		temperature = oc.TemperatureIfNegativeSeed
	} else {
		temperature = 0 // Since temperature is set to 0 when seed >=0
		// The cache is only used for fixed seeds and a temperature of 0
		keyData := struct {
			Prompts     []string
			ModelName   string
			Seed        int
			Temperature float64
		}{
			Prompts:     []string{prompt, suffix},
			ModelName:   oc.ModelName,
			Seed:        seed,
			Temperature: temperature,
		}
		keyDataBytes, err := json.Marshal(keyData)
		if err != nil {
			return OutputResponse{}, err
		}
		hash := sha256.Sum256(keyDataBytes)
		cacheKey = hex.EncodeToString(hash[:])
		if Cache == nil {
			if err := InitCache(); err != nil {
				return OutputResponse{}, err
			}
		}
		if entry, err := Cache.Get(cacheKey); err == nil {
			var res OutputResponse
			err = json.Unmarshal(entry, &res)
			if err != nil {
				return OutputResponse{}, err
			}
			return res, nil
		}
	}
	reqBody := GenerateRequest{
		Model:  oc.ModelName,
		Prompt: prompt,
		Suffix: suffix,
		Options: RequestOptions{
			Seed:        seed,        // set to -1 to make it random
			Temperature: temperature, // set to 0 together with a specific seed to make output reproducible
		},
	}
	if oc.ContextLength != 0 {
		reqBody.Options.ContextLength = oc.ContextLength
	}
	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return OutputResponse{}, err
	}
	if oc.Verbose {
		fmt.Printf("Sending request to %s/api/generate: %s\n", oc.ServerAddr, string(reqBytes))
	}
	HTTPClient := &http.Client{
		Timeout: oc.HTTPTimeout,
	}
	resp, err := HTTPClient.Post(oc.ServerAddr+"/api/generate", mimeJSON, bytes.NewBuffer(reqBytes))
	if err != nil {
		return OutputResponse{}, err
	}
	defer resp.Body.Close()
	response := OutputResponse{
		Role: "assistant",
	}
	var sb strings.Builder
	decoder := json.NewDecoder(resp.Body)
	for {
		var genResp GenerateResponse
		if err := decoder.Decode(&genResp); err != nil {
			break
		}
		sb.WriteString(genResp.Response)
		if genResp.Done {
			response.PromptTokens = genResp.PromptEvalCount
			response.ResponseTokens = genResp.EvalCount
			break
		}
	}
	response.Response = strings.TrimPrefix(sb.String(), "\n")
	if cacheKey != "" {
		data, err := json.Marshal(response)
		if err != nil {
			return OutputResponse{}, err
		}
		Cache.Set(cacheKey, data)
	}
	return response, nil
}

// Complete is a convenience function for completing code between two given strings of code
func (oc *Config) Complete(codeStart, codeEnd string) (string, error) {
	if err := oc.PullIfNeeded(true); err != nil {
		return "", err
	}
	response, err := oc.GetBetweenResponse(codeStart, codeEnd)
	if err != nil {
		return "", err
	}
	return response.Response, nil
}

// StreamBetween sends a request to the Ollama API and returns the generated output via a callback function.
// The callback function is given a string and "true" when the streaming is done (or if an error occurred).
func (oc *Config) StreamBetween(callbackFunction func(string, bool), prompt, suffix string) error {
	defer callbackFunction("", true)
	var (
		temperature float64
		seed        = oc.SeedOrNegative
	)
	if prompt == "" {
		return errors.New("the prompt can not be empty")
	}
	if seed < 0 {
		temperature = oc.TemperatureIfNegativeSeed
	}
	reqBody := GenerateRequest{
		Model:  oc.ModelName,
		System: oc.SystemPrompt,
		Prompt: prompt,
		Suffix: suffix,
		Stream: true,
		Options: RequestOptions{
			Seed:        seed,        // set to -1 to make it random
			Temperature: temperature, // set to 0 together with a specific seed to make output reproducible
		},
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
	resp, err := HTTPClient.Post(oc.ServerAddr+"/api/generate", mimeJSON, bytes.NewBuffer(reqBytes))
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
