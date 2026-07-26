package engine

// Middleware that runs before the mux, for all served handlers

import (
	"net/http"
	"strings"

	"github.com/xyproto/algernon/utils"
	"github.com/xyproto/sheepcounter"
)

// canonicalPathMiddleware canonicalizes the request path before the mux, the
// permission system and the filename lookup gets to see it, so that all three
// agree on what the path is.
func canonicalPathMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if strings.ContainsAny(req.URL.Path, "\x00\r\n") {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		if cleaned := utils.CanonicalURLPath(req.URL.Path); cleaned != req.URL.Path {
			req.URL.Path = cleaned
			// Let EscapedPath re-derive the escaped form
			req.URL.RawPath = ""
		}
		next.ServeHTTP(w, req)
	})
}

// permissionMiddleware rejects requests that the permission system does not
// allow. It wraps the entire mux, so that routes registered outside of
// RegisterHandlers are guarded as well.
func (ac *Config) permissionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// The permission system requires a database backend, so perm can be nil
		if ac.perm != nil && ac.perm.Rejected(w, req) {
			// Prepare to count bytes written
			sc := sheepcounter.New(w)
			// Get and call the Permission Denied function
			ac.perm.DenyFunction()(sc, req)
			// Log the response
			ac.LogAccess(req, http.StatusForbidden, sc.Counter())
			// Reject the request by not calling the next handler
			return
		}
		next.ServeHTTP(w, req)
	})
}
