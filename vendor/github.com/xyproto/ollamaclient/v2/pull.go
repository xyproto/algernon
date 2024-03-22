package ollamaclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/xyproto/env/v2"
)

// PullRequest represents the request payload for pulling a model
type PullRequest struct {
	Name     string `json:"name"`
	Insecure bool   `json:"insecure,omitempty"`
	Stream   bool   `json:"stream,omitempty"`
}

// PullResponse represents the response data from the pull API call
type PullResponse struct {
	Status    string `json:"status"`
	Digest    string `json:"digest"`
	Total     int64  `json:"total"`
	Completed int64  `json:"completed"`
}

var (
	spinner = []string{"-", "\\", "|", "/"}
	colors  = map[string]string{
		"blue":    "\033[94m",
		"cyan":    "\033[96m",
		"gray":    "\033[37m",
		"magenta": "\033[95m",
		"red":     "\033[91m",
		"white":   "\033[97m",
		"reset":   "\033[0m",
	}
)

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func generateColorizedProgressBar(progress float64, width int) string {
	progressInt := int(progress / 100 * float64(width))
	bar := colors["blue"] + strings.Repeat("=", progressInt)
	if progressInt > width/3 {
		bar += colors["magenta"] + strings.Repeat("=", max(0, progressInt-width/3))
	}
	if progressInt > 2*width/3 {
		bar += colors["cyan"] + strings.Repeat("=", max(0, progressInt-2*width/3))
	}
	bar += colors["white"] + strings.Repeat(" ", width-max(progressInt, width)) + colors["reset"]
	return bar
}

// Pull takes an optional verbose bool and tries to pull the current oc.Model
func (oc *Config) Pull(optionalVerbose ...bool) (string, error) {
	if env.Bool("NO_COLOR") {
		// Skip colors
		for k := range colors {
			colors[k] = ""
		}
	}
	verbose := oc.Verbose
	if len(optionalVerbose) > 0 && optionalVerbose[0] {
		verbose = true
	}

	reqBody := PullRequest{
		Name:   oc.ModelName,
		Stream: true,
	}
	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}
	if verbose {
		fmt.Printf("Sending request to %s/api/pull: %s\n", oc.ServerAddr, string(reqBytes))
	}

	resp, err := http.Post(oc.ServerAddr+"/api/pull", "application/json", bytes.NewBuffer(reqBytes))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var sb strings.Builder
	decoder := json.NewDecoder(resp.Body)

	downloadStarted := time.Now()
	spinnerPosition := 0
	var lastDigest string // Track the last hash

OUT:
	for {
		var resp PullResponse
		if err := decoder.Decode(&resp); err != nil {
			return sb.String(), err
		}

		shortDigest := strings.TrimPrefix(resp.Digest, "sha256:")
		if len(shortDigest) > 8 {
			shortDigest = shortDigest[:8]
		}

		// Check if the hash has changed (indicating a new part of the download)
		if lastDigest != "" && lastDigest != resp.Digest {
			if verbose {
				fmt.Println() // Insert a newline for a new part
			}
		}
		lastDigest = resp.Digest // Update the lastDigest for the next loop

		if resp.Total == 0 {
			if verbose {
				fmt.Printf("\r%sPulling manifest... %s%s", colors["white"], spinner[spinnerPosition%len(spinner)], colors["reset"])
				spinnerPosition++
			}
		} else {
			progress := float64(resp.Completed) / float64(resp.Total) * 100
			progressBar := generateColorizedProgressBar(progress, 30)
			displaySizeCompleted := humanize.Bytes(uint64(resp.Completed))
			displaySizeTotal := humanize.Bytes(uint64(resp.Total))

			// Strip the unit from the number right before "/"
			if spacePos := strings.Index(displaySizeCompleted, " "); spacePos != -1 {
				displaySizeCompleted = displaySizeCompleted[:spacePos]
			}

			if verbose {
				fmt.Printf("\r%s%s - %s [%s] %.2f%% - %s/%s %s", colors["white"], oc.ServerAddr, shortDigest, progressBar, progress, displaySizeCompleted, displaySizeTotal, colors["reset"])
			}
		}

		if resp.Status == "success" {
			if verbose {
				fmt.Printf("\r%s - Download complete!\033[K\n", oc.ModelName)
			}
			break OUT
		}

		if time.Since(downloadStarted) > defaultPullTimeout {
			return sb.String(), fmt.Errorf("downloading %s timed out after %v", oc.ModelName, defaultPullTimeout)
		}
	}

	return sb.String(), nil
}
