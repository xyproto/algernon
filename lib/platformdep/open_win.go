// +build !linux,!nacl,!solaris,!darwin,!dragonfly,!freebsd,!netbsd,!openbsd

package platformdep

// Not Linux, not BSDs

func DefaultOpenExecutable() string {
	return "explorer"
}
