// +build linux nacl solaris

package platformdep

// DefaultOpenExecutable returns the default application for opening URLs
func DefaultOpenExecutable() string {
	return "xdg-open"
}
