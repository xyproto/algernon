// +build darwin dragonfly freebsd netbsd openbsd

package platformdep

func DefaultOpenExecutable() string {
	return "open"
}
