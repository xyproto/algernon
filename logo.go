// +build !darwin,!dragonfly,!freebsd,!linux,!nacl,!netbsd,!openbsd,!solaris

package algernon

// UNIX-like systems uses logo_unix.go instead.

// Return the server name, version number and description
func banner() string {
	return "\n" + versionString + "\n" + description
}
