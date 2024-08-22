//go:build android

package env

// userHomeDir returns the same value as os.UserHomeDir for Android
func userHomeDir() (string, error) {
	return "/sdcard", nil
}
