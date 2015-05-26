package main

import (
	"testing"
)

func TestInterface(t *testing.T) {
	if !isDir(".") {
		t.Error("isDir failed to recognize .")
	}
	if !isDir("/") {
		t.Error("isDir failed to recognize /")
	}
}
