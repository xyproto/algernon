package engine

import (
	"archive/zip"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func writeTestAlg(t *testing.T, path string) {
	t.Helper()
	writeTestAlgPayload(t, path, "-- lua\n")
}

func writeTestAlgPayload(t *testing.T, path, body string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	for _, n := range []string{"app/", "app/index.lua", "app/data.lua"} {
		w, err := zw.Create(n)
		if err != nil {
			t.Fatal(err)
		}
		if !filepath.IsAbs(n) && n[len(n)-1] != '/' {
			if _, err := w.Write([]byte(body)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
}

func newTestConfig(t *testing.T) *Config {
	t.Helper()
	return &Config{
		algCache:      newAlgExtractionCache(),
		serverTempDir: t.TempDir(),
	}
}

func TestExtractAlgConcurrentDedup(t *testing.T) {
	tmp := t.TempDir()
	algPath := filepath.Join(tmp, "myapp.alg")
	writeTestAlg(t, algPath)

	ac := newTestConfig(t)

	const N = 32
	var wg sync.WaitGroup
	dirs := make([]string, N)
	releases := make([]func(), N)
	var errs int32
	for i := range N {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			d, release, err := ac.extractAlg(algPath)
			if err != nil {
				atomic.AddInt32(&errs, 1)
				return
			}
			dirs[i] = d
			releases[i] = release
		}(i)
	}
	wg.Wait()
	if errs != 0 {
		t.Fatalf("%d extraction errors", errs)
	}
	first := dirs[0]
	if first == "" {
		t.Fatal("empty serveDir")
	}
	for i, d := range dirs {
		if d != first {
			t.Fatalf("serveDir[%d]=%q differs from [0]=%q (re-extraction happened)", i, d, first)
		}
	}
	if _, err := os.Stat(filepath.Join(first, "index.lua")); err != nil {
		t.Fatalf("expected index.lua in serveDir: %v", err)
	}

	for _, r := range releases {
		r()
	}
	// After all releases, the cache should still hold the entry (not evicted)
	// and the directory should remain on disk.
	if _, err := os.Stat(first); err != nil {
		t.Fatalf("serveDir removed despite live cache entry: %v", err)
	}
}

func TestExtractAlgReExtractsAfterMtimeChange(t *testing.T) {
	tmp := t.TempDir()
	algPath := filepath.Join(tmp, "myapp.alg")
	writeTestAlg(t, algPath)

	ac := newTestConfig(t)

	dir1, rel1, err := ac.extractAlg(algPath)
	if err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(algPath)
	if err != nil {
		t.Fatal(err)
	}
	later := info.ModTime().Add(2 * time.Second)
	if err := os.Chtimes(algPath, later, later); err != nil {
		t.Fatal(err)
	}

	dir2, rel2, err := ac.extractAlg(algPath)
	if err != nil {
		t.Fatal(err)
	}
	if dir1 == dir2 {
		t.Fatalf("expected fresh extraction after mtime change, got same dir %q", dir1)
	}

	dir3, rel3, err := ac.extractAlg(algPath)
	if err != nil {
		t.Fatal(err)
	}
	if dir2 != dir3 {
		t.Fatalf("expected stable cache after re-extract, got %q then %q", dir2, dir3)
	}

	rel1()
	rel2()
	rel3()
}

func TestExtractAlgReExtractsAfterSizeChange(t *testing.T) {
	tmp := t.TempDir()
	algPath := filepath.Join(tmp, "myapp.alg")
	writeTestAlgPayload(t, algPath, "-- short")

	ac := newTestConfig(t)

	dir1, rel1, err := ac.extractAlg(algPath)
	if err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(algPath)
	if err != nil {
		t.Fatal(err)
	}
	originalMtime := info.ModTime()

	// Rewrite the archive with a different-size payload, then forcibly restore
	// the original mtime so only the size differs.
	writeTestAlgPayload(t, algPath, "-- a much, much longer payload body for size invalidation testing")
	if err := os.Chtimes(algPath, originalMtime, originalMtime); err != nil {
		t.Fatal(err)
	}

	dir2, rel2, err := ac.extractAlg(algPath)
	if err != nil {
		t.Fatal(err)
	}
	if dir1 == dir2 {
		t.Fatalf("expected fresh extraction after size change (mtime preserved), got same dir %q", dir1)
	}

	rel1()
	rel2()
}

func TestExtractAlgEvictedDirRemovedAfterLastRelease(t *testing.T) {
	tmp := t.TempDir()
	algPath := filepath.Join(tmp, "myapp.alg")
	writeTestAlg(t, algPath)

	ac := newTestConfig(t)

	dir1, rel1, err := ac.extractAlg(algPath)
	if err != nil {
		t.Fatal(err)
	}

	// While rel1 still holds a reference, invalidate the cache by changing
	// mtime, then trigger re-extraction.
	info, err := os.Stat(algPath)
	if err != nil {
		t.Fatal(err)
	}
	later := info.ModTime().Add(2 * time.Second)
	if err := os.Chtimes(algPath, later, later); err != nil {
		t.Fatal(err)
	}
	dir2, rel2, err := ac.extractAlg(algPath)
	if err != nil {
		t.Fatal(err)
	}
	if dir1 == dir2 {
		t.Fatal("re-extraction should have produced a fresh directory")
	}

	// Old extraction should still exist because rel1 still holds it.
	if _, err := os.Stat(dir1); err != nil {
		t.Fatalf("old serveDir removed while still referenced: %v", err)
	}

	// Releasing the old reference should trigger cleanup of the old dir.
	rel1()
	// The serveDir may be a subdirectory of the actual extraction target; walk
	// up until we find the algernon-alg-* root and confirm dir1's tree is gone.
	if _, err := os.Stat(dir1); !os.IsNotExist(err) {
		t.Fatalf("expected old serveDir to be removed after final release, stat err=%v", err)
	}

	// New extraction must remain intact and reusable.
	if _, err := os.Stat(filepath.Join(dir2, "index.lua")); err != nil {
		t.Fatalf("new serveDir corrupted: %v", err)
	}
	rel2()
}

func TestExtractAlgFailedExtractionDoesNotPoisonCache(t *testing.T) {
	tmp := t.TempDir()
	algPath := filepath.Join(tmp, "broken.alg")
	if err := os.WriteFile(algPath, []byte("not a zip file"), 0o644); err != nil {
		t.Fatal(err)
	}

	ac := newTestConfig(t)

	if _, _, err := ac.extractAlg(algPath); err == nil {
		t.Fatal("expected error on corrupt archive")
	}
	// Cache must not retain the failed entry.
	ac.algCache.mu.Lock()
	_, present := ac.algCache.entries[mustAbs(t, algPath)]
	ac.algCache.mu.Unlock()
	if present {
		t.Fatal("failed extraction left a poisoned entry in the cache")
	}

	// Now replace with a valid archive and verify it extracts on the next call.
	writeTestAlg(t, algPath)
	dir, release, err := ac.extractAlg(algPath)
	if err != nil {
		t.Fatalf("expected success after replacing broken archive: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "index.lua")); err != nil {
		t.Fatalf("recovered extraction missing expected file: %v", err)
	}
	release()
}

func mustAbs(t *testing.T, p string) string {
	t.Helper()
	a, err := filepath.Abs(p)
	if err != nil {
		t.Fatal(err)
	}
	return a
}
