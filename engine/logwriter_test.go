package engine

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// After the file has been moved aside, Reopen must create a new one and write there.
func TestLogWriterReopenAfterRotation(t *testing.T) {
	dir := t.TempDir()
	name := filepath.Join(dir, "access.log")

	lw, err := openLogWriter(name, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer lw.Close()

	lw.WriteLine("before")

	rotated := name + ".1"
	if err := os.Rename(name, rotated); err != nil {
		t.Fatal(err)
	}
	if err := lw.Reopen(); err != nil {
		t.Fatal(err)
	}
	lw.WriteLine("after")

	oldData, err := os.ReadFile(rotated)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(oldData)); got != "before" {
		t.Errorf("rotated file = %q, want %q", got, "before")
	}

	newData, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("a new log file should have been created: %v", err)
	}
	if got := strings.TrimSpace(string(newData)); got != "after" {
		t.Errorf("new file = %q, want %q", got, "after")
	}
}

// Concurrent writers must not lose or interleave lines.
func TestLogWriterConcurrentWrites(t *testing.T) {
	dir := t.TempDir()
	name := filepath.Join(dir, "access.log")

	lw, err := openLogWriter(name, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer lw.Close()

	const writers = 50
	var wg sync.WaitGroup
	wg.Add(writers)
	for i := range writers {
		go func() {
			defer wg.Done()
			lw.WriteLine(strings.Repeat("x", 100) + string(rune('a'+i%26)))
		}()
	}
	wg.Wait()

	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != writers {
		t.Fatalf("got %d lines, want %d", len(lines), writers)
	}
	for i, line := range lines {
		if len(line) != 101 {
			t.Fatalf("line %d has length %d, want 101 (torn write)", i, len(line))
		}
	}
}

// Writing to a closed logWriter must not panic.
func TestLogWriterWriteAfterClose(t *testing.T) {
	dir := t.TempDir()
	lw, err := openLogWriter(filepath.Join(dir, "access.log"), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	lw.Close()
	lw.WriteLine("ignored")
	if _, err := lw.Write([]byte("ignored")); err != nil {
		t.Errorf("Write after Close returned %v, want nil", err)
	}
}

// The log files must not be readable by everyone, no matter the umask
func TestLogWriterPermissions(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "algernon.log")
	lw, err := openLogWriter(filename, defaultLogPermissions)
	if err != nil {
		t.Fatal(err)
	}
	defer lw.Close()
	lw.WriteLine("hello")

	info, err := os.Stat(filename)
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode&0o007 != 0 {
		t.Errorf("%s has mode %o, want no permissions for others", filename, mode)
	}
}
