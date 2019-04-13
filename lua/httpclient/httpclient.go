// Package httpclient provides Lua functions for a HTTP client
package httpclient

import (
	"github.com/ddliu/go-httpclient"
	log "github.com/sirupsen/logrus"
	"github.com/xyproto/algernon/lua/convert"
	"github.com/xyproto/gopher-lua"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type HTTPClient struct {
	timeout   int
	userAgent string
	language  string
	client    *httpclient.HttpClient
	cookieMap map[string]string
	invalid   bool
}

func NewHTTPClient() *HTTPClient {
	var hc HTTPClient
	hc.client = httpclient.NewHttpClient()
	hc.timeout = 10
	return &hc
}

// Begin applies the settings in the HTTPClient struct and returns a httpclient.HttpClient
func (hc *HTTPClient) Begin() *httpclient.HttpClient {
	hclient := hc.client.Begin()
	if hc.userAgent != "" {
		hclient = hclient.WithOption(httpclient.OPT_USERAGENT, hc.userAgent)
	}
	if hc.timeout != 0 {
		hclient = hclient.WithOption(httpclient.OPT_TIMEOUT, hc.timeout)
	}
	if hc.language != "" {
		hclient = hclient.WithHeader("Accept-Language", hc.language)
	}
	if len(hc.cookieMap) != 0 {
		for k, v := range hc.cookieMap {
			hclient = hclient.WithCookie(&http.Cookie{
				Name:  k,
				Value: v,
			})
		}
	}
	return hclient.WithOption(httpclient.OPT_UNSAFE_TLS, hc.invalid)
}

const (
	// HTTPClientClass is an identifier for the HTTPClient class in Lua
	HTTPClientClass = "HTTPClient"
)

// Get the first argument, "self", and cast it from userdata to a library (which is really a hash map).
func checkHTTPClientClass(L *lua.LState) *HTTPClient {
	ud := L.CheckUserData(1)
	if hc, ok := ud.Value.(*HTTPClient); ok {
		return hc
	}
	L.ArgError(1, "HTTPClient expected")
	return nil
}

// Create a new httpclient.HttpClient. The Lua function takes no arguments.
func constructHTTPClient(L *lua.LState, userAgent string) (*lua.LUserData, error) {
	// Create a new HTTP Client
	hc := NewHTTPClient()

	// Default user agent is the same as the server name
	hc.userAgent = userAgent

	// Create a new userdata struct
	ud := L.NewUserData()
	ud.Value = hc
	L.SetMetatable(ud, L.GetTypeMetatable(HTTPClientClass))
	return ud, nil
}

// hcGet is a Lua function for running the GET method on a given URL.
// The first argument is the URL.
// It can also take the following optional arguments:
// * A table with URL arguments
// * A table with HTTP headers
// The response body is returned as a string.
func hcGet(L *lua.LState) int {
	hc := checkHTTPClientClass(L) // arg 1
	URL := L.ToString(2)          // arg 2
	if URL == "" {
		L.ArgError(2, "URL expected")
		return 0 // no results
	}

	// URL VALUES
	uv := make(url.Values)
	argTable := L.ToTable(3) // arg 3 (optiona)
	if argTable != nil {
		argMap := convert.Table2interfaceMap(argTable)
		for k, interfaceValue := range argMap {
			switch v := interfaceValue.(type) {
			case int:
				uv.Add(k, strconv.Itoa(v))
			case string:
				uv.Add(k, v)
			default:
				// TODO: Also support floats?
				log.Warn("Unrecognized value in table:", v)
			}
		}
	}
	encodedValues := uv.Encode()
	if encodedValues != "" {
		URL += "?" + encodedValues
	}

	// HTTP HEADERS
	headers := make(map[string]string)
	headerTable := L.ToTable(4) // arg 4 (optional)
	if headerTable != nil {
		headerMap := convert.Table2interfaceMap(headerTable)
		for k, interfaceValue := range headerMap {
			switch v := interfaceValue.(type) {
			case int:
				headers[k] = strconv.Itoa(v)
			case string:
				headers[k] = v
			default:
				log.Warn("Unrecognized value in table:", v)
			}
		}
	}

	//log.Info("GET " + URL)

	// GET the given URL with the given HTTP headers
	resp, err := hc.Begin().Do("GET", URL, headers, nil)
	if err != nil {
		log.Error(err)
		return 0 // no results
	}

	// Read the returned body
	bodyString, err := resp.ToString()
	if err != nil {
		log.Error(err)
		return 0 // no results
	}

	// Return a string
	L.Push(lua.LString(bodyString))
	return 1 // number of results
}

// hcPost is a Lua function for running the POST method on a given URL.
// The first argument is the URL.
// It can also take the following optional arguments:
// * A table with URL arguments
// * A table with HTTP headers
// * A string that is the POST body
// The response body is returned as a string.
func hcPost(L *lua.LState) int {
	hc := checkHTTPClientClass(L) // arg 1
	URL := L.ToString(2)          // arg 2
	if URL == "" {
		L.ArgError(2, "URL expected")
		return 0 // no results
	}

	// URL VALUES
	uv := make(url.Values)
	argTable := L.ToTable(3) // arg 3 (optiona)
	if argTable != nil {
		argMap := convert.Table2interfaceMap(argTable)
		for k, interfaceValue := range argMap {
			switch v := interfaceValue.(type) {
			case int:
				uv.Add(k, strconv.Itoa(v))
			case string:
				uv.Add(k, v)
			default:
				// TODO: Also support floats?
				log.Warn("Unrecognized value in table:", v)
			}
		}
	}
	encodedValues := uv.Encode()
	if encodedValues != "" {
		URL += "?" + encodedValues
	}

	// HTTP HEADERS
	headers := make(map[string]string)
	headerTable := L.ToTable(4) // arg 4 (optional)
	if headerTable != nil {
		headerMap := convert.Table2interfaceMap(headerTable)
		for k, interfaceValue := range headerMap {
			switch v := interfaceValue.(type) {
			case int:
				headers[k] = strconv.Itoa(v)
			case string:
				headers[k] = v
			default:
				log.Warn("Unrecognized value in table:", v)
			}
		}
	}

	// Body
	bodyReader := strings.NewReader(L.ToString(5)) // arg 5 (optional)

	//log.Info("POST " + URL)

	// POST the given URL with the given HTTP headers
	resp, err := hc.Begin().Do("POST", URL, headers, bodyReader)
	if err != nil {
		log.Error(err)
		return 0 // no results
	}

	// Read the returned body
	bodyString, err := resp.ToString()
	if err != nil {
		log.Error(err)
		return 0 // no results
	}

	// Return a string
	L.Push(lua.LString(bodyString))
	return 1 // number of results
}

// hcDo is a Lua function for running a custom HTTP method on a given URL.
// The first argument is the method, like PUT or DELETE.
// The second argument is the URL.
// It can also take the following optional arguments:
// * A table with URL arguments
// * A table with HTTP headers
// The response body is returned as a string.
func hcDo(L *lua.LState) int {
	hc := checkHTTPClientClass(L) // arg 1

	method := L.ToString(2) // arg 2
	if method == "" {
		L.ArgError(2, "Method expected (ie. PUT)")
	}
	URL := L.ToString(3) // arg 3
	if URL == "" {
		L.ArgError(3, "URL expected")
		return 0 // no results
	}

	// URL VALUES
	uv := make(url.Values)
	argTable := L.ToTable(4) // arg 4 (optiona)
	if argTable != nil {
		argMap := convert.Table2interfaceMap(argTable)
		for k, interfaceValue := range argMap {
			switch v := interfaceValue.(type) {
			case int:
				uv.Add(k, strconv.Itoa(v))
			case string:
				uv.Add(k, v)
			default:
				// TODO: Also support floats?
				log.Warn("Unrecognized value in table:", v)
			}
		}
	}
	encodedValues := uv.Encode()
	if encodedValues != "" {
		URL += "?" + encodedValues
	}

	// HTTP HEADERS
	headers := make(map[string]string)
	headerTable := L.ToTable(5) // arg 5 (optional)
	if headerTable != nil {
		headerMap := convert.Table2interfaceMap(headerTable)
		for k, interfaceValue := range headerMap {
			switch v := interfaceValue.(type) {
			case int:
				headers[k] = strconv.Itoa(v)
			case string:
				headers[k] = v
			default:
				log.Warn("Unrecognized value in table:", v)
			}
		}
	}

	// log.Info(method + " " + URL)

	// Connect to the given URL with the given method and the given HTTP headers
	resp, err := hc.Begin().Do(method, URL, headers, nil)
	if err != nil {
		log.Error(err)
		return 0 // no results
	}

	// Read the returned body
	bodyString, err := resp.ToString()
	if err != nil {
		log.Error(err)
		return 0 // no results
	}

	// Return a string
	L.Push(lua.LString(bodyString))
	return 1 // number of results
}

// hcString is a Lua function that returns a descriptive string
func hcString(L *lua.LState) int {
	L.Push(lua.LString("HTTP client based on github.com/ddliu/go-httpclient"))
	return 1 // number of results
}

// hcSetUserAgent is a Lua function for setting the user agent string
func hcSetUserAgent(L *lua.LState) int {
	hc := checkHTTPClientClass(L) // arg 1
	userAgent := L.ToString(2)    // arg 2
	if userAgent == "" {
		L.ArgError(2, "User agent string expected")
		return 0 // no results
	}

	hc.userAgent = userAgent

	return 0 // no results
}

// hcSetInvalid is a Lua function for setting if invalid TLS certificates are OK or not
func hcSetInvalid(L *lua.LState) int {
	hc := checkHTTPClientClass(L) // arg 1
	invalid := L.CheckBool(2)     // arg 2
	hc.invalid = invalid
	return 0 // no results
}

// hcSetCookie sets a cookie name/value on this HTTP client object
func hcSetCookie(L *lua.LState) int {
	hc := checkHTTPClientClass(L) // arg 1
	key := L.ToString(2)          // arg 2
	if key == "" {
		L.ArgError(2, "Expected a cookie key string")
		return 0 // no results
	}
	value := L.ToString(3) // arg 3
	if value == "" {
		L.ArgError(3, "Expected a cookie value string")
		return 0 // no results
	}

	hc.cookieMap[key] = value

	return 0 // no results
}

// hcSetTimeout is a Lua function for setting the timeout
func hcSetTimeout(L *lua.LState) int {
	hc := checkHTTPClientClass(L) // arg 1
	timeout := L.ToInt(2)         // arg 2
	if timeout == 0 {
		L.ArgError(2, "Expected a timeout (in seconds)")
		return 0 // no results
	}

	hc.timeout = timeout

	return 0 // no results
}

// hcSetLanguage is a Lua function for setting the desired language
// for HTTP request.
func hcSetLanguage(L *lua.LState) int {
	hc := checkHTTPClientClass(L) // arg 1
	language := L.ToString(2)     // arg 2
	if language == "" {
		L.ArgError(2, "Accept-Language string expected (ie. \"en-us\")")
		return 0 // no results
	}

	hc.language = language

	return 0 // no results
}

// The hash map methods that are to be registered
var hcMethods = map[string]lua.LGFunction{
	"__tostring":   hcString,
	"SetLanguage":  hcSetLanguage,
	"SetTimeout":   hcSetTimeout,
	"SetCookie":    hcSetCookie,
	"SetUserAgent": hcSetUserAgent,
	"SetInvalid":   hcSetInvalid,
	"GET":          hcGet,
	"POST":         hcPost,
	"DO":           hcDo,

	// TODO: Consider also implementing support for cookies
}

// Load makes functions related to httpclient available to the given Lua state
func Load(L *lua.LState, userAgent string) {

	// Register the HTTPClient class and the methods that belongs with it.
	metaTableHC := L.NewTypeMetatable(HTTPClientClass)
	metaTableHC.RawSetH(lua.LString("__index"), metaTableHC)
	L.SetFuncs(metaTableHC, hcMethods)

	// The constructor for HTTPClient
	L.SetGlobal("HTTPClient", L.NewFunction(func(L *lua.LState) int {
		// Construct a new HTTPClient
		userdata, err := constructHTTPClient(L, userAgent)
		if err != nil {
			log.Error(err)
			return 0 // Number of returned values
		}

		// Return the Lua Page object
		L.Push(userdata)
		return 1 // number of results
	}))

	// Make a HTTP GET request to the given URL
	L.SetGlobal("GET", L.NewFunction(func(L *lua.LState) int {
		// Construct a new HTTPClient
		userdata, err := constructHTTPClient(L, userAgent)
		if err != nil {
			log.Error(err)
			return 0 // Number of returned values
		}
		L.Insert(userdata, 0)
		return hcGet(L) // Return the number of returned values
	}))

	// Make a HTTP POST request to the given URL
	L.SetGlobal("POST", L.NewFunction(func(L *lua.LState) int {
		// Construct a new HTTPClient
		userdata, err := constructHTTPClient(L, userAgent)
		if err != nil {
			log.Error(err)
			return 0 // Number of returned values
		}
		L.Insert(userdata, 0)
		return hcPost(L) // Return the number of returned values
	}))

	// Make a custom HTTP request to a given URL, like "PUT"
	L.SetGlobal("DO", L.NewFunction(func(L *lua.LState) int {
		// Construct a new HTTPClient
		userdata, err := constructHTTPClient(L, userAgent)
		if err != nil {
			log.Error(err)
			return 0 // Number of returned values
		}
		L.Insert(userdata, 0)
		return hcDo(L) // Return the number of returned values
	}))

}
