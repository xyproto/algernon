// Package pstore provides a way to keep track of users, login states and permissions.
package pstore

import (
	"net/http"
	"strings"

	"github.com/xyproto/pinterface"
)

// Permissions keeps track of the permissions for various path prefixes
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
	Version = 3.1
)

// New will initialize a Permissions struct with all the default settings.
// This will also connect to the database host at port 3306.
func New() (*Permissions, error) {
	state, err := NewUserStateSimple()
	if err != nil {
		return nil, err
	}
	return NewPermissions(state), nil
}

// NewWithConf will initialize a Permissions struct with a database connection string.
func NewWithConf(connectionString string) (*Permissions, error) {
	state, err := NewUserState(connectionString, true)
	if err != nil {
		return nil, err
	}
	return NewPermissions(state), nil
}

// NewWithDSN will initialize a Permissions struct with a DSN
func NewWithDSN(connectionString string, databaseName string) (*Permissions, error) {
	state, err := NewUserStateWithDSN(connectionString, databaseName, true)
	if err != nil {
		return nil, err
	}
	return NewPermissions(state), nil
}

// NewPermissions will initialize a Permissions struct with the given UserState and
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

// SetDenyFunction can be used to specify a http.HandlerFunc for when the permissions are denied
func (perm *Permissions) SetDenyFunction(f http.HandlerFunc) {
	perm.denied = f
}

// DenyFunction can be used to retrieve the currently configured http.HandlerFunc for when permissions are denied
func (perm *Permissions) DenyFunction() http.HandlerFunc {
	return perm.denied
}

// UserState will return the UserState struct
func (perm *Permissions) UserState() pinterface.IUserState {
	return perm.state
}

// Clear will treat all URLs as public
func (perm *Permissions) Clear() {
	perm.adminPathPrefixes = []string{}
	perm.userPathPrefixes = []string{}
}

// AddAdminPath will add an URL prefix that will enforce that URLs starting
// with that prefix will only be for logged in administrators
func (perm *Permissions) AddAdminPath(prefix string) {
	perm.adminPathPrefixes = append(perm.adminPathPrefixes, prefix)
}

// AddUserPath will add an URL prefix that will enforce that URLs starting
// with that prefix will only be for logged in users
func (perm *Permissions) AddUserPath(prefix string) {
	perm.userPathPrefixes = append(perm.userPathPrefixes, prefix)
}

// AddPublicPath will add an URL prefix that will enforce that URLs starting
// with that prefix will be public. This overrides the other prefixes.
func (perm *Permissions) AddPublicPath(prefix string) {
	perm.publicPathPrefixes = append(perm.publicPathPrefixes, prefix)
}

// SetAdminPath will add URL prefixes that will enforce that URLs starting
// with those prefixes will only be for logged in administrators
func (perm *Permissions) SetAdminPath(pathPrefixes []string) {
	perm.adminPathPrefixes = pathPrefixes
}

// SetUserPath will add URL prefixes that will enforce that URLs starting
// with those prefixes will only be for logged in users
func (perm *Permissions) SetUserPath(pathPrefixes []string) {
	perm.userPathPrefixes = pathPrefixes
}

// SetPublicPath will add URL prefixes that will enforce that URLs starting
// with those prefixes will be public. This overrides the other prefixes.
func (perm *Permissions) SetPublicPath(pathPrefixes []string) {
	perm.publicPathPrefixes = pathPrefixes
}

// PermissionDenied is the default "permission denied" http handler.
func PermissionDenied(w http.ResponseWriter, req *http.Request) {
	http.Error(w, "Permission denied.", http.StatusForbidden)
}

// Rejected will check if a given request should be rejected.
func (perm *Permissions) Rejected(w http.ResponseWriter, req *http.Request) bool {

	path := req.URL.Path // the path of the URL that the user wish to visit

	// If it's not "/" and set to be public regardless of permissions
	if perm.rootIsPublic && path == "/" {
		return false
	}

	// Reject if it is an admin page and user does not have admin permissions
	for _, prefix := range perm.adminPathPrefixes {
		if strings.HasPrefix(path, prefix) {
			if !perm.state.AdminRights(req) {
				return true
			}
		}
	}

	// Reject if it's a user page and the user does not have user rights
	for _, prefix := range perm.userPathPrefixes {
		if strings.HasPrefix(path, prefix) {
			if !perm.state.UserRights(req) {
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

	return !found
}

// ServeHTTP is the middleware handler (compatible with Negroni)
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
