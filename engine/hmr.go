package engine

// Hot Module Replacement (HMR) update endpoint.
// React-specific logic lives in react.go.

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/sirupsen/logrus"
)

// hmrUpdatePrefix is the URL prefix for the HMR update endpoint
const hmrUpdatePrefix = "/@algernon/hmr/"

// pathContains reports whether p is the same as or a descendant of root.
func pathContains(root, p string) bool {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// HMRUpdateHandler serves a freshly compiled (never cached) version of a
// JSX or JS file so the browser can hot-swap it without reloading the page.
func (ac *Config) HMRUpdateHandler(w http.ResponseWriter, req *http.Request) {
	relPath := strings.TrimPrefix(req.URL.Path, hmrUpdatePrefix)
	if relPath == "" {
		http.Error(w, "missing path", http.StatusBadRequest)
		return
	}
	relPath = filepath.FromSlash(relPath)
	if filepath.IsAbs(relPath) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// Canonicalize the server root.
	serverRoot, err := filepath.Abs(ac.serverDirOrFilename)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if resolved, err := filepath.EvalSymlinks(serverRoot); err == nil {
		serverRoot = resolved
	}

	// Containment check via filepath.Rel after normalization.
	absPath := filepath.Clean(filepath.Join(serverRoot, relPath))
	if !pathContains(serverRoot, absPath) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	// Re-check after resolving symlinks inside the root.
	if resolved, err := filepath.EvalSymlinks(absPath); err == nil {
		if !pathContains(serverRoot, resolved) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		absPath = resolved
	}

	src, err := os.ReadFile(absPath)
	if err != nil {
		logrus.Errorf("hmr: cannot read %s: %s", absPath, err)
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	var data []byte
	if needsBundling(src) {
		data, err = bundleUncached(absPath, src, ac.jsxOptions, ac.autoRefresh)
		if err != nil {
			logrus.Errorf("hmr: compile error for %s: %s", absPath, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		srcStr := string(src)
		if ac.autoRefresh {
			srcStr = injectRefreshRegistrations(srcStr, filepath.Base(absPath))
		}
		opts := ac.jsxOptions
		opts.Loader = loaderForFile(absPath)
		result := api.Transform(srcStr, opts)
		if len(result.Errors) > 0 {
			msgs := make([]string, len(result.Errors))
			for i, e := range result.Errors {
				msgs[i] = e.Text
			}
			http.Error(w, strings.Join(msgs, "\n"), http.StatusInternalServerError)
			return
		}
		data = result.Code
	}

	w.Header().Set("Content-Type", "text/javascript;charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(data)
}

// bundleUncached runs esbuild on the given source without consulting the cache
func bundleUncached(filename string, srcData []byte, jsxOpts api.TransformOptions, withRefresh bool) ([]byte, error) {
	dir := filepath.Dir(filename)

	loader := loaderForFile(filename)

	contents := string(srcData)
	if withRefresh {
		contents = injectRefreshRegistrations(contents, filepath.Base(filename))
	}

	opts := api.BuildOptions{
		Bundle:            true,
		Platform:          api.PlatformBrowser,
		Format:            api.FormatIIFE,
		MinifyWhitespace:  jsxOpts.MinifyWhitespace,
		MinifyIdentifiers: jsxOpts.MinifyIdentifiers,
		MinifySyntax:      jsxOpts.MinifySyntax,
		Charset:           jsxOpts.Charset,
		Write:             false,
		AbsWorkingDir:     dir,
		LogLevel:          api.LogLevelSilent,
		Stdin: &api.StdinOptions{
			Contents:   contents,
			ResolveDir: dir,
			Sourcefile: filename,
			Loader:     loader,
		},
	}
	if withRefresh {
		opts.Plugins = []api.Plugin{reactRefreshPlugin()}
	}

	result := api.Build(opts)
	if len(result.Errors) > 0 {
		msgs := make([]string, len(result.Errors))
		for i, e := range result.Errors {
			msgs[i] = e.Text
		}
		return nil, fmt.Errorf("bundle %s: %s", filepath.Base(filename), strings.Join(msgs, "; "))
	}
	if len(result.OutputFiles) == 0 {
		return nil, fmt.Errorf("bundle %s: no output produced", filepath.Base(filename))
	}
	return result.OutputFiles[0].Contents, nil
}
