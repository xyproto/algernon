package datablock

import (
	"github.com/bmizerany/assert"
	"testing"
)

func TestInterface(t *testing.T) {
	fs := NewFileStat(false, 0)
	if !fs.IsDir(".") {
		t.Error("isDir failed to recognize .")
	}
	if !fs.IsDir("/") {
		t.Error("isDir failed to recognize /")
	}
}

func TestExists(t *testing.T) {
	fs := NewFileStat(false, 0)
	assert.Equal(t, fs.Exists("LICENSE"), true)
	assert.NotEqual(t, fs.Exists("LICENSES-SCHMISENCES"), true)
}
