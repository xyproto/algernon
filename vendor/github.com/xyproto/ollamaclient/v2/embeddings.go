package ollamaclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// Embeddings sends a request to get embeddings for a given prompt
func (oc *Config) Embeddings(prompt string) ([]float64, error) {
	reqBody := EmbeddingsRequest{
		Model:  oc.ModelName,
		Prompt: prompt,
	}
	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return []float64{}, err
	}
	if oc.Verbose {
		fmt.Printf("Sending request to %s/api/embeddings: %s\n", oc.ServerAddr, string(reqBytes))
	}
	HTTPClient := &http.Client{
		Timeout: oc.HTTPTimeout,
	}
	resp, err := HTTPClient.Post(oc.ServerAddr+"/api/embeddings", "application/json", bytes.NewBuffer(reqBytes))
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
