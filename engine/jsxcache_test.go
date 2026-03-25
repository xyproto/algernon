package engine

import (
	"testing"
)

func TestBundleCacheMemory(t *testing.T) {
	bc := newBundleCache()

	// Empty cache should use 0 bytes
	if bc.BytesUsed() != 0 {
		t.Errorf("empty cache: expected 0, got %d", bc.BytesUsed())
	}

	// Add an entry
	bc.mu.Lock()
	data := []byte("test bundle data")
	bc.entries["test.js"] = bundleCacheEntry{data: data}
	bc.hits["test.js"] = 0
	bc.mu.Unlock()

	expected := uint64(len(data))
	if bc.BytesUsed() != expected {
		t.Errorf("BytesUsed: expected %d, got %d", expected, bc.BytesUsed())
	}

	// Add another entry
	bc.mu.Lock()
	data2 := []byte("more test data for bundle")
	bc.entries["test2.js"] = bundleCacheEntry{data: data2}
	bc.hits["test2.js"] = 0
	bc.mu.Unlock()

	expected = uint64(len(data) + len(data2))
	if bc.BytesUsed() != expected {
		t.Errorf("BytesUsed with 2 entries: expected %d, got %d", expected, bc.BytesUsed())
	}
}

func TestBundleCacheEviction(t *testing.T) {
	bc := newBundleCache()

	// Add entries with different hit counts
	bc.mu.Lock()
	bc.entries["hot.js"] = bundleCacheEntry{data: []byte("hot content")}
	bc.hits["hot.js"] = 10

	bc.entries["cold.js"] = bundleCacheEntry{data: []byte("this is a bigger cold entry")}
	bc.hits["cold.js"] = 0
	bc.mu.Unlock()

	// Evict - should remove cold.js (0 hits)
	bc.mu.Lock()
	ok := bc.evictLocked()
	bc.mu.Unlock()

	if !ok {
		t.Error("evictLocked should return true")
	}

	bc.mu.RLock()
	_, hasCold := bc.entries["cold.js"]
	_, hasHot := bc.entries["hot.js"]
	bc.mu.RUnlock()

	if hasCold {
		t.Error("cold.js should be evicted")
	}
	if !hasHot {
		t.Error("hot.js should still be in cache")
	}
}

func TestBundleCacheEvictionSize(t *testing.T) {
	bc := newBundleCache()

	// Add entries with same hit count - should evict by size
	bc.mu.Lock()
	bc.entries["small.js"] = bundleCacheEntry{data: []byte("small")}
	bc.hits["small.js"] = 0

	bc.entries["large.js"] = bundleCacheEntry{data: []byte("this is a much larger entry")}
	bc.hits["large.js"] = 0
	bc.mu.Unlock()

	initialUsage := bc.BytesUsed()

	// Evict once - should remove the larger one
	bc.mu.Lock()
	bc.evictLocked()
	bc.mu.Unlock()

	if bc.BytesUsed() >= initialUsage {
		t.Error("eviction should reduce memory usage")
	}

	// small.js should still be there
	bc.mu.RLock()
	_, hasSmall := bc.entries["small.js"]
	_, hasLarge := bc.entries["large.js"]
	bc.mu.RUnlock()

	if hasLarge {
		t.Error("large.js should be evicted")
	}
	if !hasSmall {
		t.Error("small.js should still be in cache")
	}
}

func TestBundleCacheClear(t *testing.T) {
	bc := newBundleCache()

	bc.mu.Lock()
	bc.entries["test.js"] = bundleCacheEntry{data: []byte("test")}
	bc.hits["test.js"] = 5
	bc.mu.Unlock()

	bc.Clear()

	if bc.BytesUsed() != 0 {
		t.Errorf("after Clear: expected 0 bytes, got %d", bc.BytesUsed())
	}

	bc.mu.RLock()
	if len(bc.entries) != 0 || len(bc.hits) != 0 {
		t.Error("Clear should empty both maps")
	}
	bc.mu.RUnlock()
}
