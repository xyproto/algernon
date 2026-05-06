package engine

// React integration for Algernon's hot-module replacement (HMR) system

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
)

//go:embed assets/reactrefresh/react-refresh-runtime.js
var reactRefreshRuntimeJS []byte

// hmrRefreshRuntimePath is the URL for the embedded react-refresh runtime
const hmrRefreshRuntimePath = "/@algernon/react-refresh.js"

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
// registrations into every .js/.jsx file (excluding node_modules)
func reactRefreshPlugin() api.Plugin {
	return api.Plugin{
		Name: "algernon-react-refresh",
		Setup: func(build api.PluginBuild) {
			build.OnLoad(api.OnLoadOptions{Filter: `\.(jsx?)$`}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
				if strings.Contains(args.Path, "node_modules") {
					return api.OnLoadResult{}, nil
				}
				raw, err := os.ReadFile(args.Path)
				if err != nil {
					return api.OnLoadResult{}, err
				}
				basename := filepath.Base(args.Path)
				wrapped := injectRefreshRegistrations(string(raw), basename)
				loader := api.LoaderJS
				if strings.HasSuffix(strings.ToLower(args.Path), ".jsx") {
					loader = api.LoaderJSX
				}
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
// development equivalents when the page references .jsx files.
func (ac *Config) swapReactProdToDev(htmldata []byte) []byte {
	if !bytes.Contains(htmldata, []byte(".jsx")) {
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
