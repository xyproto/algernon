//go:build windows

package platformdep

// SetupSignals does nothing on Windows, which has no SIGUSR1/SIGUSR2
func SetupSignals(clearCacheFunction func(), printFunction func(format string, args ...interface{})) {
	return
}

// SetupLogRotationSignal does nothing on Windows, which has no SIGHUP
func SetupLogRotationSignal(reopenLogsFunction func(), printFunction func(format string, args ...interface{})) {
	return
}
