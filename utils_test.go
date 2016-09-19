package main

import (
	//	"github.com/bmizerany/assert"
	"testing"
	//	"time"
)

func TestInterface(t *testing.T) {
	fs := NewFileStat(false, 0)
	if !fs.isDir(".") {
		t.Error("isDir failed to recognize .")
	}
	if !fs.isDir("/") {
		t.Error("isDir failed to recognize /")
	}
}

//func TestClearCache(t *testing.T) {
//	fs := NewFileStat(true, 150*time.Millisecond)
//	fs.exists("README.md")
//	assert.Equal(t, 1, len(fs.exCache))
//	fs.Sleep(400 * time.Millisecond) // Sleep cycle time + some time to clear the cache before checking
//	assert.Equal(t, 0, len(fs.exCache))
//	fs.exists("README.md")
//	assert.Equal(t, 1, len(fs.exCache))
//	fs.Sleep(400 * time.Millisecond) // Sleep cycle time + some time to clear the cache before checking
//	assert.Equal(t, 0, len(fs.exCache))
//}
