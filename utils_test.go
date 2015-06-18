package main

import (
	"bytes"
	"github.com/bmizerany/assert"
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

func TestRoundtrip(t *testing.T) {
	data := []byte("some data")

	compressed, err := compress(data)
	assert.Equal(t, err, nil)

	decompressed, err := decompress(compressed)
	assert.Equal(t, err, nil)

	assert.Equal(t, true, bytes.Equal(data, decompressed))
}
