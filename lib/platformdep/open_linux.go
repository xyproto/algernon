// +build linux nacl solaris

package platformdep

func DefaultOpenExecutable() string {
	return "xdg-open"
}
