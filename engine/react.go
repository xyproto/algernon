package engine

// React integration: embedded runtime, HMR, and react-refresh support

import (
	"bytes"
	_ "embed"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/sirupsen/logrus"
)

// Embedded React 19 runtime

//go:embed assets/react19/react.development.js
var react19DevJS []byte

//go:embed assets/react19/react.production.min.js
var react19ProdJS []byte

//go:embed assets/react19/react-dom.development.js
var reactDOM19DevJS []byte

//go:embed assets/react19/react-dom.production.min.js
var reactDOM19ProdJS []byte

//go:embed assets/reactrefresh/react-refresh-runtime.js
var reactRefreshRuntimeJS []byte

// defaultReactVersion is the React major version used by index.jsx and index.tsx
const defaultReactVersion = 19

// reactVersionPaths holds the URL paths for a React version's embedded scripts
type reactVersionPaths struct {
	reactDev     string
	reactProd    string
	reactDOMDev  string
	reactDOMProd string
}

// reactPaths maps React major versions to their embedded script URL paths.
// When adding a new React version, embed its assets and add an entry here.
var reactPaths = map[int]reactVersionPaths{
	19: {
		reactDev:     react19Path,
		reactProd:    react19ProdPath,
		reactDOMDev:  reactDOM19Path,
		reactDOMProd: reactDOM19ProdPath,
	},
}

const (
	react19Path        = "/@algernon/react19/react.js"
	react19ProdPath    = "/@algernon/react19/react.min.js"
	reactDOM19Path     = "/@algernon/react19/react-dom.js"
	reactDOM19ProdPath = "/@algernon/react19/react-dom.min.js"

	// hmrRefreshRuntimePath is the URL for the embedded react-refresh runtime
	hmrRefreshRuntimePath = "/@algernon/react-refresh.js"
)

// registerReact19Handlers registers the embedded React 19 endpoints on mux
func registerReact19Handlers(mux *http.ServeMux) {
	serve := func(name string, data []byte) http.HandlerFunc {
		return func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Type", "text/javascript;charset=utf-8")
			if n, err := w.Write(data); err != nil || n == 0 {
				logrus.Errorf("Could not serve %s", name)
			}
		}
	}
	mux.HandleFunc(react19Path, serve("react.development.js", react19DevJS))
	mux.HandleFunc(react19ProdPath, serve("react.production.min.js", react19ProdJS))
	mux.HandleFunc(reactDOM19Path, serve("react-dom.development.js", reactDOM19DevJS))
	mux.HandleFunc(reactDOM19ProdPath, serve("react-dom.production.min.js", reactDOM19ProdJS))
}

// HMRRefreshRuntimeHandler serves the embedded react-refresh runtime
func (ac *Config) HMRRefreshRuntimeHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/javascript;charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(reactRefreshRuntimeJS)
}

// hmrRootCaptureScript is injected before </head> to patch ReactDOM.createRoot
// so that hot-swapped IIFEs reuse existing roots and root.render() is
// suppressed during hot-swaps (letting react-refresh drive the update).
const hmrRootCaptureScript = `(function(){` +
	`if(!window.ReactDOM||typeof ReactDOM.createRoot!=='function'){return;}` +
	`if(window.__algernonHMRRoots){return;}` +
	`var m=new Map();` +
	`window.__algernonHMRRoots=m;` +
	`window.__algernonHMRActive=false;` +
	`window.__algernonHMRBegin=function(){window.__algernonHMRActive=true;};` +
	`window.__algernonHMREnd=function(){window.__algernonHMRActive=false;};` +
	`var orig=ReactDOM.createRoot.bind(ReactDOM);` +
	`ReactDOM.createRoot=function(c,o){` +
	`if(m.has(c)){return m.get(c);}` +
	`var r=orig(c,o);` +
	`var rr=r.render.bind(r);` +
	`r.render=function(el){if(window.__algernonHMRActive){return;}return rr(el);};` +
	`m.set(c,r);return r;` +
	`};` +
	`}());`

// componentDeclRE matches top-level React component declarations (function,
// class, const/let/var with arrow or function expression). Only names starting
// with an uppercase letter are matched.
var componentDeclRE = regexp.MustCompile(
	`(?m)^(?:export\s+default\s+|export\s+)?(?:function|class)\s+([A-Z][a-zA-Z0-9_$]*)` +
		`|^(?:export\s+)?(?:const|let|var)\s+([A-Z][a-zA-Z0-9_$]*)\s*=\s*` +
		`(?:function[\s(]|\([^)]*\)\s*=>|React\.memo\b|React\.forwardRef\b)`,
)

// injectRefreshRegistrations appends an IIFE to src that registers every
// detected React component with react-refresh and calls performReactRefresh().
func injectRefreshRegistrations(src, basename string) string {
	seen := make(map[string]bool)
	var regs strings.Builder
	for _, m := range componentDeclRE.FindAllStringSubmatch(src, -1) {
		name := m[1]
		if name == "" {
			name = m[2]
		}
		if name != "" && !seen[name] {
			seen[name] = true
			fmt.Fprintf(&regs,
				"if(typeof %s===\"function\")r(%s,%q);\n",
				name, name, basename+"$$"+name)
		}
	}
	if regs.Len() == 0 {
		return src
	}
	suffix := fmt.Sprintf(
		"\n;(function(){"+
			"if(!window.__algernonHMRRefresh){return;}"+
			"var r=window.__algernonHMRRefresh.register;\n"+
			"%s"+
			"window.__algernonHMRRefresh.performReactRefresh();"+
			"})();\n",
		regs.String(),
	)
	return src + suffix
}

// reactRefreshPlugin returns an esbuild plugin that injects react-refresh
// registrations into every .js/.jsx/.ts/.tsx file (excluding node_modules)
func reactRefreshPlugin() api.Plugin {
	return api.Plugin{
		Name: "algernon-react-refresh",
		Setup: func(build api.PluginBuild) {
			build.OnLoad(api.OnLoadOptions{Filter: `\.(jsx?|tsx?)$`}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
				if strings.Contains(args.Path, "node_modules") {
					return api.OnLoadResult{}, nil
				}
				raw, err := os.ReadFile(args.Path)
				if err != nil {
					return api.OnLoadResult{}, err
				}
				basename := filepath.Base(args.Path)
				wrapped := injectRefreshRegistrations(string(raw), basename)
				loader := loaderForFile(args.Path)
				return api.OnLoadResult{
					Contents: &wrapped,
					Loader:   loader,
				}, nil
			})
		},
	}
}

// reactProdScriptRE matches <script src> attributes that load React production builds,
// including those from /@algernon/react19/
var reactProdScriptRE = regexp.MustCompile(
	`(?i)(src=["'])([^"']*?)(react(?:-dom)?\.production\.min\.js|/@algernon/react19/react(?:-dom)?\.min\.js)`,
)

// swapReactProdToDev replaces React production build <script src> values with
// development equivalents when the page references .jsx or .tsx files.
func (ac *Config) swapReactProdToDev(htmldata []byte) []byte {
	if !bytes.Contains(htmldata, []byte(".jsx")) && !bytes.Contains(htmldata, []byte(".tsx")) {
		return htmldata
	}
	serverRoot, err := filepath.Abs(ac.serverDirOrFilename)
	if err != nil {
		return htmldata
	}
	return reactProdScriptRE.ReplaceAllFunc(htmldata, func(match []byte) []byte {
		subs := reactProdScriptRE.FindSubmatch(match)
		if len(subs) < 4 {
			return match
		}
		dir := string(subs[2])
		prod := string(subs[3])

		// Handle embedded paths for React 19
		if strings.HasPrefix(prod, "/@algernon/react19/") {
			return bytes.Replace(match, []byte(".min.js"), []byte(".js"), 1)
		}

		// Handle local paths
		dev := strings.Replace(prod, ".production.min.js", ".development.js", 1)
		abs := filepath.Join(serverRoot, filepath.FromSlash(strings.TrimPrefix(dir+dev, "/")))
		if _, err := os.Stat(abs); err != nil {
			return match
		}
		return bytes.Replace(match, []byte(prod), []byte(dev), 1)
	})
}
