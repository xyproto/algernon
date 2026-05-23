package engine

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/xyproto/datablock"
)

// newTSXTestConfig builds a minimal *Config suitable for exercising the TSX
// bundling and rendering paths from tests, without going through the normal
// flag/init machinery (which registers global flags and would panic on reuse).
func newTSXTestConfig() *Config {
	return &Config{
		bundleCache: newBundleCache(),
		jsxOptions: api.TransformOptions{
			MinifyWhitespace:  true,
			MinifyIdentifiers: true,
			MinifySyntax:      true,
			Charset:           api.CharsetUTF8,
		},
		cache: datablock.NewFileCache(1024*1024, true, 64*1024, true, 7*1024*1024),
		fs:    datablock.NewFileStat(true, time.Minute),
	}
}

func TestTSXCompilationAndCaching(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "algernon-tsx-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// A simple index.tsx file using JSX + TS syntax. React and ReactDOM are
	// assumed to be available globally, so we don't import them. The component
	// is assigned to window.App to prevent esbuild from tree-shaking it away.
	tsxFile := filepath.Join(tmpDir, "index.tsx")
	tsxContent := `
const App = () => {
	return <div>Hello TypeScript React!</div>;
};
(window as any).App = App;
`
	if err := os.WriteFile(tsxFile, []byte(tsxContent), 0644); err != nil {
		t.Fatal(err)
	}

	ac := newTSXTestConfig()

	bundled, err := ac.bundleFile(tsxFile, []byte(tsxContent), false)
	if err != nil {
		t.Fatalf("Failed to compile/bundle TSX: %v", err)
	}
	if len(bundled) == 0 {
		t.Fatal("Bundled output is empty")
	}
	if !strings.Contains(string(bundled), "Hello TypeScript React!") {
		t.Errorf("Bundled output missing expected string, got: %s", string(bundled))
	}

	// Cache hit: bundling the same file again should hit the in-memory bundle cache.
	ac.bundleCache.Clear()
	if _, err := ac.bundleFile(tsxFile, []byte(tsxContent), false); err != nil {
		t.Fatal(err)
	}

	ac.bundleCache.mu.RLock()
	hitsBefore := ac.bundleCache.hits[tsxFile]
	ac.bundleCache.mu.RUnlock()

	if _, err := ac.bundleFile(tsxFile, []byte(tsxContent), false); err != nil {
		t.Fatal(err)
	}

	ac.bundleCache.mu.RLock()
	hitsAfter := ac.bundleCache.hits[tsxFile]
	ac.bundleCache.mu.RUnlock()

	if hitsAfter != hitsBefore+1 {
		t.Errorf("Expected cache hit, hits count before: %d, after: %d", hitsBefore, hitsAfter)
	}

	// Cache miss: changing the file's mtime should invalidate the cached entry.
	futureTime := time.Now().Add(10 * time.Second)
	if err := os.Chtimes(tsxFile, futureTime, futureTime); err != nil {
		t.Fatal(err)
	}

	if _, err := ac.bundleFile(tsxFile, []byte(tsxContent), false); err != nil {
		t.Fatal(err)
	}

	ac.bundleCache.mu.RLock()
	hitsAfterMiss := ac.bundleCache.hits[tsxFile]
	ac.bundleCache.mu.RUnlock()

	if hitsAfterMiss != 0 {
		t.Errorf("Expected cache hits to be reset to 0 after cache miss, got %d", hitsAfterMiss)
	}
}

func TestTSXRenderer(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "algernon-tsx-render-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tsxFile := filepath.Join(tmpDir, "index.tsx")
	tsxContent := `
const App = () => {
	return <h1>Test Render TSX</h1>;
};
(window as any).App = App;
`
	if err := os.WriteFile(tsxFile, []byte(tsxContent), 0644); err != nil {
		t.Fatal(err)
	}

	ac := newTSXTestConfig()

	if _, ok := defaultRenderers.lookup(".tsx"); !ok {
		t.Fatal("No renderer registered for .tsx")
	}

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/index.tsx", nil)
	rec := httptest.NewRecorder()

	if !ac.dispatchRenderer(rec, req, tsxFile, ".tsx") {
		t.Fatal("dispatchRenderer failed for .tsx")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `<div id="root">`) {
		t.Error("Rendered output should be a full React HTML page containing mount point")
	}
	if !strings.Contains(body, "Test Render TSX") {
		t.Error("Rendered output missing compiled component content")
	}
}

func TestComplexTSXCompilation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "algernon-tsx-complex-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	jsDir := filepath.Join(tmpDir, "js")
	if err := os.MkdirAll(jsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Mock UMD library to exercise relative-import resolution.
	markedFile := filepath.Join(jsDir, "marked.umd.js")
	markedContent := `window.marked = { parse: function(text) { return "parsed: " + text; } };`
	if err := os.WriteFile(markedFile, []byte(markedContent), 0644); err != nil {
		t.Fatal(err)
	}

	tsxFile := filepath.Join(tmpDir, "index.tsx")
	tsxContent := `
import './js/marked.umd.js';

declare const marked: {
	parse: (text: string) => string;
};

interface CommentProps {
	author: string;
	children: any;
}

const Comment = ({ author, children }: CommentProps) => {
	const rawMarkup = marked.parse(children ? children.toString() : '');
	return (
		<div className="comment">
			<h2 className="commentAuthor">{author}</h2>
			<span dangerouslySetInnerHTML={{ __html: rawMarkup }} />
		</div>
	);
};

(window as any).Comment = Comment;
`
	if err := os.WriteFile(tsxFile, []byte(tsxContent), 0644); err != nil {
		t.Fatal(err)
	}

	ac := newTSXTestConfig()

	bundled, err := ac.bundleFile(tsxFile, []byte(tsxContent), false)
	if err != nil {
		t.Fatalf("Failed to compile/bundle complex TSX: %v", err)
	}
	if len(bundled) == 0 {
		t.Fatal("Bundled output is empty")
	}
	if !strings.Contains(string(bundled), "commentAuthor") {
		t.Errorf("Bundled output missing expected HTML class 'commentAuthor'")
	}
}
