//go:build !windows

package platformdep

import (
	"os"
	"os/signal"
	"syscall"
)

// SetupSignals installs SIGUSR1/SIGUSR2 handlers that invoke clearCacheFunction in
// a goroutine. printFunction is used to log the received signal.
func SetupSignals(clearCacheFunction func(), printFunction func(format string, args ...any)) {
	// Listen for SIGUSR1 and SIGUSR2 to clear the cache
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGUSR1, syscall.SIGUSR2)
	go func() {
		for {
			// Wait for a signal of the type given to signal.Notify
			sig := <-signals
			printFunction("Received %v", sig)
			// Launch a goroutine for clearing the cache
			go clearCacheFunction()
		}
	}()
}
