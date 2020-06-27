// +build !darwin,!dragonfly,!freebsd,!linux,!netbsd,!openbsd,!solaris

package platformdep

// UNIX-like systems uses logo_unix.go instead.

// Banner return the server name, version number and description
func Banner(versionString, description string) string {
	return "\n" + versionString + "\n" + description
}
