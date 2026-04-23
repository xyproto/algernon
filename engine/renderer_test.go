package engine

import (
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
)

// TestDefaultRenderersRegistered checks that every built-in renderer is wired up
func TestDefaultRenderersRegistered(t *testing.T) {
	want := []string{
		".md", ".markdown",
		".gcss", ".scss",
		".jsx",
		".happ", ".hyper", ".hyper.jsx", ".hyper.js",
	}
	for _, ext := range want {
		if _, ok := defaultRenderers.lookup(ext); !ok {
			t.Errorf("no renderer registered for %s", ext)
		}
	}
}

// TestRegistryPreventsDuplicates ensures the registry panics on a duplicate extension
func TestRegistryPreventsDuplicates(t *testing.T) {
	r := newRendererRegistry()
	r.register(markdownRenderer{})
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on duplicate extension")
		}
	}()
	r.register(markdownRenderer{}) // same extensions -> panic
}

// stubRenderer records whether Render was called and for which extension.
type stubRenderer struct {
	exts   []string
	gotExt string
}

func (s *stubRenderer) Extensions() []string { return s.exts }
func (s *stubRenderer) Render(_ *Config, w http.ResponseWriter, _ *http.Request, _, ext string) error {
	s.gotExt = ext
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
	return nil
}

// TestRegistryLookup confirms that extension lookup is exact and case-sensitive.
func TestRegistryLookup(t *testing.T) {
	r := newRendererRegistry()
	stub := &stubRenderer{exts: []string{".foo", ".bar"}}
	r.register(stub)

	gotExts := make([]string, 0)
	for _, ext := range []string{".foo", ".bar"} {
		if _, ok := r.byExt[ext]; ok {
			gotExts = append(gotExts, ext)
		}
	}
	sort.Strings(gotExts)
	if len(gotExts) != 2 || gotExts[0] != ".bar" || gotExts[1] != ".foo" {
		t.Errorf("unexpected extensions: %v", gotExts)
	}

	if _, ok := r.lookup(".FOO"); ok {
		t.Error("lookup should be case-sensitive; .FOO should not match .foo")
	}
	if _, ok := r.lookup(".unknown"); ok {
		t.Error("lookup should return false for unknown extensions")
	}
}

// TestDispatchRendererRoundTrip exercises dispatchRenderer with a registered stub
func TestDispatchRendererRoundTrip(t *testing.T) {
	// Snapshot and restore the package-level registry so the stub does not leak to other tests
	saved := defaultRenderers
	defaultRenderers = newRendererRegistry()
	t.Cleanup(func() { defaultRenderers = saved })

	stub := &stubRenderer{exts: []string{".stub"}}
	defaultRenderers.register(stub)

	ac := &Config{}
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	if !ac.dispatchRenderer(rec, req, "example.stub", ".stub") {
		t.Fatal("dispatchRenderer returned false for a registered extension")
	}
	if stub.gotExt != ".stub" {
		t.Errorf("stub.gotExt = %q, want %q", stub.gotExt, ".stub")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("rec.Code = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "ok" {
		t.Errorf("rec.Body = %q, want %q", rec.Body.String(), "ok")
	}

	// Miss should return false and not touch the response
	rec2 := httptest.NewRecorder()
	if ac.dispatchRenderer(rec2, req, "example.unknown", ".unknown") {
		t.Error("dispatchRenderer returned true for an unregistered extension")
	}
	if rec2.Code != http.StatusOK { // httptest defaults to 200 until written
		// httptest.NewRecorder defaults Code to 200; verify nothing was written
		if rec2.Body.Len() != 0 {
			t.Errorf("unexpected body written on miss: %q", rec2.Body.String())
		}
	}
}
