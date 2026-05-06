package engine

import (
	_ "embed"
	"net/http"

	"github.com/sirupsen/logrus"
)

// React 19 integration

//go:embed assets/react19/react.development.js
var react19DevJS []byte

//go:embed assets/react19/react.production.min.js
var react19ProdJS []byte

//go:embed assets/react19/react-dom.development.js
var reactDOM19DevJS []byte

//go:embed assets/react19/react-dom.production.min.js
var reactDOM19ProdJS []byte

const (
	// react19Path is the URL path for the embedded React 19 development JS
	react19Path = "/@algernon/react19/react.js"

	// react19ProdPath is the URL path for the embedded React 19 production JS
	react19ProdPath = "/@algernon/react19/react.min.js"

	// reactDOM19Path is the URL path for the embedded React DOM 19 development JS
	reactDOM19Path = "/@algernon/react19/react-dom.js"

	// reactDOM19ProdPath is the URL path for the embedded React DOM 19 production JS
	reactDOM19ProdPath = "/@algernon/react19/react-dom.min.js"
)

// react19Handler returns an http.HandlerFunc that serves the given embedded
// React 19 JavaScript bytes with a JavaScript Content-Type
func react19Handler(name string, data []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/javascript;charset=utf-8")
		if n, err := w.Write(data); err != nil || n == 0 {
			logrus.Errorf("Could not serve %s", name)
		}
	}
}

// registerReact19Handlers registers the embedded React 19 endpoints on mux
func registerReact19Handlers(mux *http.ServeMux) {
	mux.HandleFunc(react19Path, react19Handler("react.development.js", react19DevJS))
	mux.HandleFunc(react19ProdPath, react19Handler("react.production.min.js", react19ProdJS))
	mux.HandleFunc(reactDOM19Path, react19Handler("react-dom.development.js", reactDOM19DevJS))
	mux.HandleFunc(reactDOM19ProdPath, react19Handler("react-dom.production.min.js", reactDOM19ProdJS))
}
