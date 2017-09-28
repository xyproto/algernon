// +build darwin dragonfly freebsd netbsd openbsd

package platformdep

// DefaultOpenExecutable returns the default application for opening URLs
func DefaultOpenExecutable() string {
	return "open"
}
