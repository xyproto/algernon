package engine

import (
	"os"
	"sync"

	"github.com/sirupsen/logrus"
)

// defaultLogPermissions is the file mode for the server log and the access
// logs. It matches the "create" line in the logrotate configuration.
const defaultLogPermissions = 0o640

// logWriter is an append-only log file that keeps its file handle open instead
// of re-opening it for every line, and that can be re-opened after log rotation.
type logWriter struct {
	f        *os.File
	filename string
	perm     os.FileMode
	mu       sync.Mutex
}

// openLogWriter opens the given filename for appending, creating it if missing
func openLogWriter(filename string, perm os.FileMode) (*logWriter, error) {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, perm)
	if err != nil {
		return nil, err
	}
	return &logWriter{f: f, filename: filename, perm: perm}, nil
}

// Write makes logWriter an io.Writer, so that it can be given to logrus
func (lw *logWriter) Write(p []byte) (int, error) {
	lw.mu.Lock()
	defer lw.mu.Unlock()
	if lw.f == nil {
		return len(p), nil
	}
	return lw.f.Write(p)
}

// WriteLine writes one line, adding the trailing newline
func (lw *logWriter) WriteLine(line string) {
	lw.mu.Lock()
	defer lw.mu.Unlock()
	if lw.f == nil {
		return
	}
	if _, err := lw.f.WriteString(line + "\n"); err != nil {
		logrus.Warnf("Can not write to %s: %s", lw.filename, err)
	}
}

// Reopen closes and re-opens the log file, for use after log rotation
func (lw *logWriter) Reopen() error {
	lw.mu.Lock()
	defer lw.mu.Unlock()
	if lw.f != nil {
		lw.f.Close()
		lw.f = nil
	}
	f, err := os.OpenFile(lw.filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, lw.perm)
	if err != nil {
		return err
	}
	lw.f = f
	return nil
}

// Close closes the log file, if it is open
func (lw *logWriter) Close() error {
	lw.mu.Lock()
	defer lw.mu.Unlock()
	if lw.f == nil {
		return nil
	}
	err := lw.f.Close()
	lw.f = nil
	return err
}

// logWriters returns the log files that are in use
func (ac *Config) logWriters() []*logWriter {
	var lws []*logWriter
	for _, lw := range []*logWriter{ac.commonAccessLog, ac.combinedAccessLog, ac.serverLog} {
		if lw != nil {
			lws = append(lws, lw)
		}
	}
	return lws
}

// ReopenLogs re-opens the access logs and the server log, after log rotation
func (ac *Config) ReopenLogs() {
	for _, lw := range ac.logWriters() {
		if err := lw.Reopen(); err != nil {
			logrus.Errorf("Could not re-open %s: %s", lw.filename, err)
		}
	}
}

// closeLogs closes any open log files
func (ac *Config) closeLogs() {
	for _, lw := range ac.logWriters() {
		lw.Close()
	}
}
