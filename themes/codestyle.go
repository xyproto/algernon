// Package themes contains several different styles that can be applied to ie. rendered Markdown documents
package themes

const (
	// DefaultCustomCodeStyle is the default chroma style for custom CSS
	// See: https://xyproto.github.io/splash/docs/all.html for an overview
	DefaultCustomCodeStyle = "algol"
)

// NewTheme adds a new built-in theme
func NewTheme(theme string, body []byte, codestyle string) {
	builtinThemes[theme] = body
	builtinCodeStyles[theme] = codestyle
}

// ThemeToCodeStyle returns the code highlight style that the given theme implicates
func ThemeToCodeStyle(theme string) string {
	if codeStyle := builtinCodeStyles[string(theme)]; codeStyle != "" {
		return codeStyle
	}
	// Not found, return the default code style
	return DefaultCustomCodeStyle
}
