package ollamaclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// ShowRequest represents the structure of the request payload for the "show" API.
type ShowRequest struct {
	Model string `json:"model"`
}

// ShowResponse represents the structure of the response payload from the "show" API.
type ShowResponse struct {
	License    string `json:"license"`
	Modelfile  string `json:"modelfile"`
	Parameters string `json:"parameters"`
	Template   string `json:"template"`
	Details    struct {
		ParentModel       string   `json:"parent_model"`
		Format            string   `json:"format"`
		Family            string   `json:"family"`
		Families          []string `json:"families"`
		ParameterSize     string   `json:"parameter_size"`
		QuantizationLevel string   `json:"quantization_level"`
	} `json:"details"`
	ModelInfo struct {
		GeneralArchitecture               string   `json:"general.architecture"`
		GeneralBasename                   string   `json:"general.basename"`
		GeneralFileType                   int      `json:"general.file_type"`
		GeneralFinetune                   string   `json:"general.finetune"`
		GeneralLanguages                  []string `json:"general.languages"`
		GeneralLicense                    string   `json:"general.license"`
		GeneralParameterCount             int64    `json:"general.parameter_count"`
		GeneralQuantizationVersion        int      `json:"general.quantization_version"`
		GeneralSizeLabel                  string   `json:"general.size_label"`
		GeneralTags                       []string `json:"general.tags"`
		GeneralType                       string   `json:"general.type"`
		LlamaAttentionHeadCount           int      `json:"llama.attention.head_count"`
		LlamaAttentionHeadCountKv         int      `json:"llama.attention.head_count_kv"`
		LlamaAttentionLayerNormRmsEpsilon float64  `json:"llama.attention.layer_norm_rms_epsilon"`
		LlamaBlockCount                   int      `json:"llama.block_count"`
		LlamaContextLength                int      `json:"llama.context_length"`
		LlamaEmbeddingLength              int      `json:"llama.embedding_length"`
		LlamaFeedForwardLength            int      `json:"llama.feed_forward_length"`
		LlamaRopeDimensionCount           int      `json:"llama.rope.dimension_count"`
		LlamaRopeFreqBase                 int      `json:"llama.rope.freq_base"`
		LlamaVocabSize                    int      `json:"llama.vocab_size"`
		TokenizerGgmlBosTokenID           int      `json:"tokenizer.ggml.bos_token_id"`
		TokenizerGgmlEosTokenID           int      `json:"tokenizer.ggml.eos_token_id"`
		TokenizerGgmlMerges               any      `json:"tokenizer.ggml.merges"`
		TokenizerGgmlModel                string   `json:"tokenizer.ggml.model"`
		TokenizerGgmlPre                  string   `json:"tokenizer.ggml.pre"`
		TokenizerGgmlTokenType            any      `json:"tokenizer.ggml.token_type"`
		TokenizerGgmlTokens               any      `json:"tokenizer.ggml.tokens"`
	} `json:"model_info"`
	ModifiedAt string `json:"modified_at"`
}

// GetShowInfo sends a request to the "show" API and returns the response with model information.
func (oc *Config) GetShowInfo() (ShowResponse, error) {
	reqBody := ShowRequest{
		Model: oc.ModelName,
	}
	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return ShowResponse{}, err
	}
	if oc.Verbose {
		fmt.Printf("Sending request to %s/api/show: %s\n", oc.ServerAddr, string(reqBytes))
	}
	HTTPClient := &http.Client{
		Timeout: oc.HTTPTimeout,
	}
	resp, err := HTTPClient.Post(oc.ServerAddr+"/api/show", mimeJSON, bytes.NewBuffer(reqBytes))
	if err != nil {
		return ShowResponse{}, err
	}
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)

	var genResp ShowResponse
	if err := decoder.Decode(&genResp); err != nil {
		return ShowResponse{}, err
	}

	return genResp, nil
}
