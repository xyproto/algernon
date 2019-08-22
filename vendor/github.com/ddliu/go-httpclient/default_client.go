// Copyright 2014-2019 Liu Dong <ddliuhb@gmail.com>.
// Licensed under the MIT license.

// Powerful and easy to use http client
package httpclient

import "sync"

// The default client for convenience
var defaultClient = &HttpClient{
	reuseTransport: true,
	reuseJar:       true,
	lock:           new(sync.Mutex),
}

var Defaults = defaultClient.Defaults
var Begin = defaultClient.Begin
var Do = defaultClient.Do
var Get = defaultClient.Get
var Delete = defaultClient.Delete
var Head = defaultClient.Head
var Post = defaultClient.Post
var PostJson = defaultClient.PostJson
var PostMultipart = defaultClient.PostMultipart
var Put = defaultClient.Put
var PutJson = defaultClient.PutJson
var PatchJson = defaultClient.PatchJson
var Options = defaultClient.Options
var Connect = defaultClient.Connect
var Trace = defaultClient.Trace
var Patch = defaultClient.Patch
var WithOption = defaultClient.WithOption
var WithOptions = defaultClient.WithOptions
var WithHeader = defaultClient.WithHeader
var WithHeaders = defaultClient.WithHeaders
var WithCookie = defaultClient.WithCookie
var Cookies = defaultClient.Cookies
var CookieValues = defaultClient.CookieValues
var CookieValue = defaultClient.CookieValue
