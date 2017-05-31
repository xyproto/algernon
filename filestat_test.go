package main

import (
	"github.com/xyproto/datablock"
	"testing"
)

func TestInterface(t *testing.T) {
	fs := datablock.NewFileStat(false, 0)
	if !fs.IsDir(".") {
		t.Error("isDir failed to recognize .")
	}
	if !fs.IsDir("/") {
		t.Error("isDir failed to recognize /")
	}
}
