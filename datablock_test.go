package main

import (
	"bytes"
	"github.com/bmizerany/assert"
	"testing"
)

func TestRoundtrip(t *testing.T) {
	data := []byte("some data")

	compressed, _, err := compress(data, false)
	assert.Equal(t, err, nil)

	decompressed, _, err := decompress(compressed)
	assert.Equal(t, err, nil)

	assert.Equal(t, true, bytes.Equal(data, decompressed))
}
