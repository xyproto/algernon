package zerossl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client acts as a ZeroSSL API client. It facilitates ZeroSSL certificate operations.
type Client struct {
	// REQUIRED: Your ZeroSSL account access key.
	AccessKey string `json:"access_key"`

	// Optionally adjust the base URL of the API.
	// Default: https://api.zerossl.com
	BaseURL string `json:"base_url,omitempty"`

	// Optionally configure a custom HTTP client.
	HTTPClient *http.Client `json:"-"`
}

func (c Client) httpGet(ctx context.Context, endpoint string, qs url.Values, target any) error {
	url := c.url(endpoint, qs)
	return c.httpRequest(ctx, http.MethodGet, url, nil, target)
}

func (c Client) httpPost(ctx context.Context, endpoint string, qs url.Values, payload, target any) error {
	var reqBody io.Reader
	if payload != nil {
		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(payloadJSON)
	}
	url := c.url(endpoint, qs)
	return c.httpRequest(ctx, http.MethodPost, url, reqBody, target)
}

func (c Client) httpRequest(ctx context.Context, method, reqURL string, reqBody io.Reader, target any) error {
	r, err := http.NewRequestWithContext(ctx, method, reqURL, reqBody)
	if err != nil {
		return err
	}
	if reqBody != nil {
		r.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient().Do(r)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck

	// because the ZeroSSL API doesn't use HTTP status codes to indicate an error,
	// nor does each response body have a consistent way of detecting success/error,
	// we have to work around this by buffering the entire response body and then
	// checking it for expected value(s) to determine if there's an error
	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024*5))
	if err != nil {
		return fmt.Errorf("failed reading response body: %v", err)
	}

	// assume error first, since the ZeroSSL API does not use status codes for errors
	// (see https://github.com/caddyserver/zerossl/issues/3)
	var apiError APIError
	if err := json.NewDecoder(bytes.NewReader(respBytes)).Decode(&apiError); err != nil {
		return fmt.Errorf("decoding JSON error body failed: %v (raw=%s)", err, respBytes)
	}
	if apiError.Success != nil && !*apiError.Success {
		// remove access_key from URL so it doesn't leak into logs
		u, err := url.Parse(reqURL)
		if err != nil {
			reqURL = fmt.Sprintf("<invalid url: %v>", err)
		}
		if u != nil {
			q, err := url.ParseQuery(u.RawQuery)
			if err == nil {
				q.Set(accessKeyParam, "redacted")
				u.RawQuery = q.Encode()
				reqURL = u.String()
			}
		}
		apiError.URL = reqURL
		return apiError
	}

	// if there was no error, decode into target payload
	if target != nil {
		if err = json.NewDecoder(bytes.NewReader(respBytes)).Decode(target); err != nil {
			return fmt.Errorf("request succeeded, but decoding JSON response body failed: %v (raw=%s)", err, respBytes)
		}
	}

	return nil
}

func (c Client) url(endpoint string, qs url.Values) string {
	baseURL := c.BaseURL
	if baseURL == "" {
		baseURL = BaseURL
	}

	// for consistency, ensure endpoint starts with /
	// and base URL does NOT end with /.
	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	if qs == nil {
		qs = url.Values{}
	}
	qs.Set(accessKeyParam, c.AccessKey)

	return fmt.Sprintf("%s%s?%s", baseURL, endpoint, qs.Encode())
}

func (c Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return httpClient
}

var httpClient = &http.Client{
	Timeout: 2 * time.Minute,
}

// anyBool is a hacky type that accepts true or 1 (or their string variants),
// or "yes" or "y", and any casing variants of the same, as a boolean true when
// unmarshaling JSON. Everything else is boolean false.
//
// This is needed due to type inconsistencies in ZeroSSL's API with "success" values.
type anyBool bool

// UnmarshalJSON satisfies json.Unmarshaler according to
// this type's documentation.
func (ab *anyBool) UnmarshalJSON(b []byte) error {
	if len(b) == 0 {
		return io.EOF
	}
	switch strings.ToLower(string(b)) {
	case `true`, `"true"`, `1`, `"1"`, `"yes"`, `"y"`:
		*ab = true
	}
	return nil
}

// MarshalJSON marshals ab to either true or false.
func (ab *anyBool) MarshalJSON() ([]byte, error) {
	if ab != nil && *ab {
		return []byte("true"), nil
	}
	return []byte("false"), nil
}

const accessKeyParam = "access_key"
