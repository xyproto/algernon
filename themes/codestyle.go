package themes

const (
	// DefaultCustomCodeStyle is the default highlight.js style for custom CSS
	DefaultCustomCodeStyle = "github"
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
