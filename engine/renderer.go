package engine

import (
	"net/http"

	"github.com/sirupsen/logrus"
)

// Renderer handles a small file extension by reading it through the cache and
// writing a response. Extensions that need more request context stay in the FilePage switch.
type Renderer interface {
	// Extensions returns the lowercased file extensions (with the dot) handled by this renderer
	Extensions() []string
	// Render writes the response for the given file. Returned errors are logged by the caller.
	Render(ac *Config, w http.ResponseWriter, req *http.Request, filename, ext string) error
}

// rendererRegistry maps a file extension to its Renderer
type rendererRegistry struct {
	byExt map[string]Renderer
}

func newRendererRegistry() *rendererRegistry {
	return &rendererRegistry{byExt: make(map[string]Renderer)}
}

// register adds a Renderer under each of its declared extensions, and panics on collision
func (r *rendererRegistry) register(rd Renderer) {
	for _, ext := range rd.Extensions() {
		if _, exists := r.byExt[ext]; exists {
			panic("engine: duplicate renderer registered for extension " + ext)
		}
		r.byExt[ext] = rd
	}
}

// lookup returns the Renderer for ext, or (nil, false) if none is registered
func (r *rendererRegistry) lookup(ext string) (Renderer, bool) {
	rd, ok := r.byExt[ext]
	return rd, ok
}

// defaultRenderers is the package-level registry, populated by init() in renderers_builtin.go
var defaultRenderers = newRendererRegistry()

// dispatchRenderer invokes the Renderer registered for ext, if any.
// Returns true if a renderer was invoked, false if the caller should fall through.
func (ac *Config) dispatchRenderer(w http.ResponseWriter, req *http.Request, filename, ext string) bool {
	rd, ok := defaultRenderers.lookup(ext)
	if !ok {
		return false
	}
	if err := rd.Render(ac, w, req, filename, ext); err != nil {
		logrus.Error(err)
	}
	return true
}
