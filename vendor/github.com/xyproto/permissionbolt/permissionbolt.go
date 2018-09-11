// Package permissionbolt provides middleware for keeping track of users, login states and permissions.
package permissionbolt

import (
	"github.com/xyproto/pinterface"
	"net/http"
	"strings"
)

// The Permissions structure keeps track of the permissions for various path prefixes
type Permissions struct {
	state              *UserState
	adminPathPrefixes  []string
	userPathPrefixes   []string
	publicPathPrefixes []string
	rootIsPublic       bool
	denied             http.HandlerFunc
}

const (
	// Version number. Stable API within major version numbers.
	Version = 2.1
)

// New initializes a Permissions struct with all the default settings.
func New() (*Permissions, error) {
	state, err := NewUserStateSimple()
	if err != nil {
		return nil, err
	}
	return NewPermissions(state), nil
}

// NewWithConf initializes a Permissions struct with a database filename
func NewWithConf(filename string) (*Permissions, error) {
	state, err := NewUserState(filename, true)
	if err != nil {
		return nil, err
	}
	return NewPermissions(state), nil

}

// NewPermissions initializes a Permissions struct with the given UserState and
// a few default paths for admin/user/public path prefixes.
func NewPermissions(state *UserState) *Permissions {
	// default permissions
	return &Permissions{state,
		[]string{"/admin"},         // admin path prefixes
		[]string{"/repo", "/data"}, // user path prefixes
		[]string{"/", "/login", "/register", "/favicon.ico", "/style", "/img", "/js",
			"/favicon.ico", "/robots.txt", "/sitemap_index.xml"}, // public
		true,
		PermissionDenied}
}

// SetDenyFunction specifies a http.HandlerFunc for when the permissions are denied
func (perm *Permissions) SetDenyFunction(f http.HandlerFunc) {
	perm.denied = f
}

// DenyFunction returns the currently configured http.HandlerFunc for when permissions are denied
func (perm *Permissions) DenyFunction() http.HandlerFunc {
	return perm.denied
}

// UserState retrieves the UserState struct
func (perm *Permissions) UserState() pinterface.IUserState {
	return perm.state
}

// Clear sets every permission to public
func (perm *Permissions) Clear() {
	perm.adminPathPrefixes = []string{}
	perm.userPathPrefixes = []string{}
}

// AddAdminPath adds an URL path prefix for pages that are only accessible for logged in administrators
func (perm *Permissions) AddAdminPath(prefix string) {
	perm.adminPathPrefixes = append(perm.adminPathPrefixes, prefix)
}

// AddUserPath adds an URL path prefix for pages that are only accessible for logged in users
func (perm *Permissions) AddUserPath(prefix string) {
	perm.userPathPrefixes = append(perm.userPathPrefixes, prefix)
}

// AddPublicPath adds an URL path prefix for pages that are public
func (perm *Permissions) AddPublicPath(prefix string) {
	perm.publicPathPrefixes = append(perm.publicPathPrefixes, prefix)
}

// SetAdminPath sets all URL path prefixes for pages that are only accessible for logged in administrators
func (perm *Permissions) SetAdminPath(pathPrefixes []string) {
	perm.adminPathPrefixes = pathPrefixes
}

// SetUserPath sets all URL path prefixes for pages that are only accessible for logged in users
func (perm *Permissions) SetUserPath(pathPrefixes []string) {
	perm.userPathPrefixes = pathPrefixes
}

// SetPublicPath sets all URL path prefixes for pages that are public
func (perm *Permissions) SetPublicPath(pathPrefixes []string) {
	perm.publicPathPrefixes = pathPrefixes
}

// PermissionDenied is the default "permission denied" handler function
func PermissionDenied(w http.ResponseWriter, req *http.Request) {
	http.Error(w, "Permission denied.", http.StatusForbidden)
}

// Rejected checks if a given http request should be rejected
func (perm *Permissions) Rejected(w http.ResponseWriter, req *http.Request) bool {
	path := req.URL.Path // the path of the URL that the user wish to visit

	// If it's not "/" and set to be public regardless of permissions
	if !(perm.rootIsPublic && path == "/") {

		// Reject if it is an admin page and user does not have admin permissions
		for _, prefix := range perm.adminPathPrefixes {
			if strings.HasPrefix(path, prefix) {
				if !perm.state.AdminRights(req) {
					// Reject
					return true
				}
			}
		}

		// Reject if it's a user page and the user does not have user rights
		for _, prefix := range perm.userPathPrefixes {
			if strings.HasPrefix(path, prefix) {
				if !perm.state.UserRights(req) {
					// Reject
					return true
				}
			}
		}

		// Reject if it's not a public page
		found := false
		for _, prefix := range perm.publicPathPrefixes {
			if strings.HasPrefix(path, prefix) {
				found = true
				break
			}
		}
		if !found {
			// Reject
			return true
		}

	}

	// Not rejected
	return false
}

// Middleware handler (compatible with Negroni)
func (perm *Permissions) ServeHTTP(w http.ResponseWriter, req *http.Request, next http.HandlerFunc) {
	// Check if the user has the right admin/user rights
	if perm.Rejected(w, req) {
		// Get and call the Permission Denied function
		perm.DenyFunction()(w, req)
		// Reject the request by not calling the next handler below
		return
	}

	// Call the next middleware handler
	next(w, req)
}
