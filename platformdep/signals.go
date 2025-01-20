//go:build windows

package platformdep

func SetupSignals(clearCacheFunction func(), printFunction func(format string, args ...interface{})) {
	return
}
