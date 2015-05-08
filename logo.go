// +build !darwin,!dragonfly,!freebsd,!linux,!nacl,!netbsd,!openbsd,!solaris

package main

// Return the server name, version number and description
func banner() string {
	return "\n" + versionString + "\n" + description
}
