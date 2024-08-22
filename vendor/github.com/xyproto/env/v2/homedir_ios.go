//go:build ios

package env

// userHomeDir returns the same value as os.UserHomeDir for iOS
func userHomeDir() (string, error) {
	return "/", nil
}
