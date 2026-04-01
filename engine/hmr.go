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
const hmrUpdatePrefix = "/__algernon_hmr__/"

// HMRUpdateHandler serves a freshly compiled (never cached) version of a
// JSX or JS file so the browser can hot-swap it without reloading the page.
func (ac *Config) HMRUpdateHandler(w http.ResponseWriter, req *http.Request) {
	relPath := strings.TrimPrefix(req.URL.Path, hmrUpdatePrefix)
	if relPath == "" {
		http.Error(w, "missing path", http.StatusBadRequest)
		return
	}

	if strings.Contains(relPath, "..") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	serverRoot, err := filepath.Abs(ac.serverDirOrFilename)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	absPath := filepath.Join(serverRoot, filepath.FromSlash(relPath))
	if !strings.HasPrefix(absPath+string(filepath.Separator), serverRoot+string(filepath.Separator)) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
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
		result := api.Transform(srcStr, ac.jsxOptions)
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

	loader := api.LoaderJS
	if strings.HasSuffix(strings.ToLower(filename), ".jsx") {
		loader = api.LoaderJSX
	}

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
