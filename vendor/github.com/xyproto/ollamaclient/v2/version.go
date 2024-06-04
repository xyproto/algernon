package ollamaclient

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Version sends a request to get the current Ollama version
func (oc *Config) Version() (string, error) {
	if oc.Verbose {
		fmt.Printf("Sending a request to %s/api/version\n", oc.ServerAddr)
	}
	HTTPClient := &http.Client{
		Timeout: oc.HTTPTimeout,
	}
	resp, err := HTTPClient.Get(oc.ServerAddr + "/api/version")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	var versionResponse VersionResponse
	if err := decoder.Decode(&versionResponse); err != nil {
		return "", err
	}
	return versionResponse.Version, nil
}
