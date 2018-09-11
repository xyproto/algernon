package onthefly

// Various JavaScript and JQuery functions

// fn returns JavaScript code wrapped in an anonymous function
func fn(source string) string {
	return "function() { " + source + " }"
}

// quote quotes the given string in a simple way, by wrapping it in double quotes
func quote(src string) string {
	return "\"" + src + "\""
}

// event returns JavaScript code that runs the given JavaScript code at the
// given event on the given tag. For example the "click" event.
func event(tagname, event, source string) string {
	return "$(" + quote(tagname) + ")." + event + "(" + fn(source) + ");"
}

// method returns JavaScript code that calls a given method on a given
// tag, with the given string value as the argument.
// The value is used raw and not quoted.
func method(tagname, methodname, value string) string {
	return "$(" + quote(tagname) + ")." + methodname + "(" + value + ");"
}

// methodString returns JavaScript code that calls a given method on a given
// tag, with the given string value as the argument
// The value is quoted.
func methodString(tagname, methodname, value string) string {
	return method(tagname, methodname, quote(value))
}

// OnDocumentReady returns JavaScript code the runs the given JavaScript code when the HTML document is ready in the browser.
func OnDocumentReady(source string) string {
	return "$(document).ready(" + fn(source) + ");"
}

// Alert returns JavaScript code that displays a pretty intruding message box. The "msg" will be quoted.
func Alert(msg string) string {
	return "alert(" + quote(msg) + ");"
}

// OnClick returns JavaScript code that runs the given JavaScript code when the given tag name is clicked on
func OnClick(tagname, source string) string {
	return event(tagname, "click", source)
}

// SetText returns JavaScript code that sets the text of the given tag name
func SetText(tagname, text string) string {
	return methodString(tagname, "text", text)
}

// SetHTML returns JavaScript code that sets the HTML of the given tag name
func SetHTML(tagname, html string) string {
	return method(tagname, "html", html)
}

// SetValue returns JavaScript code that quotes and then sets the contents of the given tag name
func SetValue(tagname, val string) string {
	return methodString(tagname, "val", val)
}

// SetRawValue returns JavaScript code that sets the contents of a given tag name, without quoting
func SetRawValue(tagname, val string) string {
	return method(tagname, "val", val)
}

// Hide returns JavaScript code that hides the given tag name
func Hide(tagname string) string {
	return "$(" + quote(tagname) + ").hide();"
}

// HideAnimated returns JavaScript code that hides the given tag name in an animated way
func HideAnimated(tagname string) string {
	return "$(" + quote(tagname) + ").hide('normal');" // 'fast', 'normal', 'slow' or milliseconds
}

// Show returns JavaScript code that shows the given tag name
func Show(tagname string) string {
	return "$(" + quote(tagname) + ").show();"
}

// Focus returns JavaScript code that sets focus on the given tag name
func Focus(tagname string) string {
	return "$(" + quote(tagname) + ").focus();"
}

// ShowAnimated returns JavaScript code that displays the given tag name in an animated way
func ShowAnimated(tagname string) string {
	return "$(" + quote(tagname) + ").show('normal');" // 'fast', 'normal', 'slow' or milliseconds
}

// ShowInline returns JavaScript code that styles the given tag with "display:inline"
func ShowInline(tagname string) string {
	return "$(" + quote(tagname) + ").css('display', 'inline');"
}

// ShowInlineAnimated returns JavaScript code that show the given tag with "display:inline", then hides it and then shows it in an animated way
func ShowInlineAnimated(tagname string) string {
	return ShowInline(tagname) + Hide(tagname) + ShowAnimated(tagname)
}

// ShowInlineAnimatedIf returns JavaScript that shows the given tag in an animated way if the value from the given URL is "1"
func ShowInlineAnimatedIf(booleanURL, tagname string) string {
	return "$.get(" + quote(booleanURL) + ", function(data) { if (data == \"1\") {" + ShowInlineAnimated(tagname) + "}; });"
}

// Return JavaScript that loads the contents of the given URL into the given tag name
func Load(tagname, url string) string {
	return methodString(tagname, "load", url)
}

// HidIfNot returns JavaScript that will hide a tag if booleanURL doesn't return "1"
func HideIfNot(booleanURL, tagname string) string {
	return "$.get(" + quote(booleanURL) + ", function(data) { if (data != \"1\") {" + Hide(tagname) + "}; });"
}

// ShowAnimatedIf returns JavaScript that will show a tag if booleanURL returns "1"
func ShowAnimatedIf(booleanURL, tagname string) string {
	return "$.get(" + quote(booleanURL) + ", function(data) { if (data == \"1\") {" + ShowAnimated(tagname) + "}; });"
}

// ScrollDownAnimated returns JavaScript code that will slowly scroll the page down
func ScrollDownAnimated() string {
	return "$('html, body').animate({scrollTop:$(document).height()}, 'slow');"
}

// JS wraps JavaScript code in a <script> tag
func JS(source string) string {
	if source != "" {
		return "<script type=\"text/javascript\">" + source + "</script>"
	}
	return ""
}

// Returns HTML that will run the given JavaScript code once the document is ready.
// Returns an empty string if there is no JavaScript code to run.
func DocumentReadyJS(source string) string {
	if source != "" {
		return JS(OnDocumentReady(source))
	}
	return ""
}

// Redirect returns JavaScript code that redirects to the given URL
func Redirect(URL string) string {
	return "window.location.href = \"" + URL + "\";"
}
