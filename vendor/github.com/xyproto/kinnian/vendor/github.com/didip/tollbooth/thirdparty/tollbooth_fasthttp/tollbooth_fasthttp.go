package tollbooth_fasthttp

import (
	"encoding/base64"
	"strings"

	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/config"
	"github.com/didip/tollbooth/errors"
	"github.com/valyala/fasthttp"
)

func LimitHandler(handler fasthttp.RequestHandler, limiter *config.Limiter) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		httpError := LimitByRequest(limiter, ctx)

		if httpError != nil {
			ctx.Response.Header.Set("Content-Type", limiter.MessageContentType)
			ctx.SetStatusCode(httpError.StatusCode)
			ctx.SetBody([]byte(httpError.Message))
			return
		}

		handler(ctx)
	}
}

func LimitByRequest(limiter *config.Limiter, ctx *fasthttp.RequestCtx) *errors.HTTPError {
	sliceKeys := BuildKeys(limiter, ctx)

	//Loop sliceKeys and check if one of them has an error.
	for _, keys := range sliceKeys {
		httpError := tollbooth.LimitByKeys(limiter, keys)

		if httpError != nil {
			return httpError
		}
	}

	return nil
}

// StringInSlice finds needle in a slice of strings.
func StringInSlice(sliceString []string, needle string) bool {
	for _, b := range sliceString {
		if b == needle {
			return true
		}
	}
	return false
}

func ipAddrFromRemoteAddr(s string) string {
	idx := strings.LastIndex(s, ":")
	if idx == -1 {
		return s
	}
	return s[:idx]
}

// RemoteIP finds IP Address given http.Request struct.
func RemoteIP(ipLookups []string, ctx *fasthttp.RequestCtx) string {
	realIP := string(ctx.Request.Header.Peek("X-Real-IP"))
	forwardedFor := string(ctx.Request.Header.Peek("X-Forwarded-For"))

	for _, lookup := range ipLookups {
		if lookup == "RemoteAddr" {
			return ipAddrFromRemoteAddr(ctx.RemoteAddr().String())
		}
		if lookup == "X-Forwarded-For" && forwardedFor != "" {
			// X-Forwarded-For is potentially a list of addresses separated with ","
			parts := strings.Split(forwardedFor, ",")
			for i, p := range parts {
				parts[i] = strings.TrimSpace(p)
			}
			return parts[0]
		}
		if lookup == "X-Real-IP" && realIP != "" {
			return realIP
		}
	}

	return ""
}

// BuildKeys generates a slice of keys to rate-limit by given config and request structs.
func BuildKeys(limiter *config.Limiter, ctx *fasthttp.RequestCtx) [][]string {
	remoteIP := RemoteIP(limiter.IPLookups, ctx)
	path := string(ctx.Path())
	reqMethod := string(ctx.Method())
	sliceKeys := make([][]string, 0)

	// Don't BuildKeys if remoteIP is blank.
	if remoteIP == "" {
		return sliceKeys
	}

	if limiter.Methods != nil && limiter.Headers != nil && limiter.BasicAuthUsers != nil {
		// Limit by HTTP methods and HTTP headers+values and Basic Auth credentials.
		if StringInSlice(limiter.Methods, reqMethod) {
			for headerKey, headerValues := range limiter.Headers {
				headerLen := len(ctx.Request.Header.Peek(headerKey))
				if (headerValues == nil || len(headerValues) <= 0) && headerLen != 0 {
					// If header values are empty, rate-limit all request with headerKey.
					username, _, ok := parseBasicAuth(string(ctx.Request.Header.Peek("Authorization")))
					if ok && StringInSlice(limiter.BasicAuthUsers, username) {
						sliceKeys = append(sliceKeys, []string{remoteIP, path, reqMethod, headerKey, username})
					}

				} else if len(headerValues) > 0 && headerLen != 0 {
					// If header values are not empty, rate-limit all request with headerKey and headerValues.
					for _, headerValue := range headerValues {
						username, _, ok := parseBasicAuth(string(ctx.Request.Header.Peek("Authorization")))
						if ok && StringInSlice(limiter.BasicAuthUsers, username) {
							sliceKeys = append(sliceKeys, []string{remoteIP, path, reqMethod, headerKey, headerValue, username})
						}
					}
				}
			}
		}

	} else if limiter.Methods != nil && limiter.Headers != nil {
		// Limit by HTTP methods and HTTP headers+values.
		if StringInSlice(limiter.Methods, reqMethod) {
			for headerKey, headerValues := range limiter.Headers {
				headerLen := len(ctx.Request.Header.Peek(headerKey))
				if (headerValues == nil || len(headerValues) <= 0) && headerLen != 0 {
					// If header values are empty, rate-limit all request with headerKey.
					sliceKeys = append(sliceKeys, []string{remoteIP, path, reqMethod, headerKey})

				} else if len(headerValues) > 0 && headerLen != 0 {
					// If header values are not empty, rate-limit all request with headerKey and headerValues.
					for _, headerValue := range headerValues {
						sliceKeys = append(sliceKeys, []string{remoteIP, path, reqMethod, headerKey, headerValue})
					}
				}
			}
		}

	} else if limiter.Methods != nil && limiter.BasicAuthUsers != nil {
		// Limit by HTTP methods and Basic Auth credentials.
		if StringInSlice(limiter.Methods, reqMethod) {
			username, _, ok := parseBasicAuth(string(ctx.Request.Header.Peek("Authorization")))
			if ok && StringInSlice(limiter.BasicAuthUsers, username) {
				sliceKeys = append(sliceKeys, []string{remoteIP, path, reqMethod, username})
			}
		}

	} else if limiter.Methods != nil {
		// Limit by HTTP methods.
		if StringInSlice(limiter.Methods, reqMethod) {
			sliceKeys = append(sliceKeys, []string{remoteIP, path, reqMethod})
		}

	} else if limiter.Headers != nil {
		// Limit by HTTP headers+values.
		for headerKey, headerValues := range limiter.Headers {
			headerLen := len(ctx.Request.Header.Peek(headerKey))
			if (headerValues == nil || len(headerValues) <= 0) && headerLen != 0 {
				// If header values are empty, rate-limit all request with headerKey.
				sliceKeys = append(sliceKeys, []string{remoteIP, path, headerKey})

			} else if len(headerValues) > 0 && headerLen != 0 {
				// If header values are not empty, rate-limit all request with headerKey and headerValues.
				for _, headerValue := range headerValues {
					sliceKeys = append(sliceKeys, []string{remoteIP, path, headerKey, headerValue})
				}
			}
		}

	} else if limiter.BasicAuthUsers != nil {
		// Limit by Basic Auth credentials.
		username, _, ok := parseBasicAuth(string(ctx.Request.Header.Peek("Authorization")))
		if ok && StringInSlice(limiter.BasicAuthUsers, username) {
			sliceKeys = append(sliceKeys, []string{remoteIP, path, username})
		}
	} else {
		// Default: Limit by remoteIP and path.
		sliceKeys = append(sliceKeys, []string{remoteIP, path})
	}

	return sliceKeys
}

func parseBasicAuth(auth string) (string, string, bool) {
	const prefix = "Basic "
	if !strings.HasPrefix(auth, prefix) {
		return "", "", false
	}

	c, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
	if err != nil {
		return "", "", false
	}

	cs := string(c)
	s := strings.IndexByte(cs, ':')
	if s < 0 {
		return "", "", false
	}

	return cs[:s], cs[s+1:], true
}
