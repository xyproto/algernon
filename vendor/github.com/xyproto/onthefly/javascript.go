package onthefly

import "errors"

// Link a page up with a JS file
// Takes the url to a JS file as a string
// The given page must have a "head" tag for this to work
// Returns an error if no "head" tag is found, or nil
func (page *Page) LinkToJS(jsURL string) error {
	_, err := page.LinkToJSInHead(jsURL)
	return err
}

// Link to JavaScript in the head
func (page *Page) LinkToJSInHead(jsURL string) (*Tag, error) {
	head, err := page.GetTag("head")
	if err != nil {
		return nil, err
	}
	script := head.AddNewTag("script")
	script.AddAttrib("src", jsURL)
	script.AddAttrib("type", "text/javascript")
	script.AddContent(" ")
	return script, nil
}

// Link to JavaScript, at the end of the body
func (page *Page) LinkToJSInBody(jsURL string) (*Tag, error) {
	body, err := page.GetTag("body")
	if err != nil {
		return nil, err
	}
	script := body.AddNewTag("script")
	script.AddAttrib("src", jsURL)
	script.AddAttrib("type", "text/javascript")
	script.AddContent(" ")
	return script, nil
}

// Add JavaScript code in a script tag in the head tag
// Deprecated, use AddScriptToHead instead.
func AddScriptToHeader(page *Page, js string) error {
	_, err := page.AddScriptToHead(js)
	return err
}

// Add javascript code in a script tag in the head tag
func (page *Page) AddScriptToHead(js string) (*Tag, error) {
	// Check if there's anything to add
	if js == "" {
		return nil, errors.New("No javascript to add")
	}
	// Add a script tag
	head, err := page.GetTag("head")
	if err != nil {
		return nil, err
	}
	script := head.AddNewTag("script")
	script.AddAttrib("type", "text/javascript")
	script.AddContent(js)
	return script, nil
}

// Add javascript code in a script tag at the end of the body tag
func (page *Page) AddScriptToBody(js string) (*Tag, error) {
	// Check if there's anything to add
	if js == "" {
		return nil, errors.New("No javascript to add")
	}
	// Add a script tag
	body, err := page.GetTag("body")
	if err != nil {
		return nil, err
	}
	script := body.AddNewTag("script")
	script.AddAttrib("type", "text/javascript")
	script.AddContent(js)
	return script, nil
}
