package jpath

import (
	"github.com/bmizerany/assert"
	"io/ioutil"
	"os"
	"testing"
)

func TestAddFile(t *testing.T) {
	var err error
	someJSON := []byte(`{"x":"2", "y":"3"}`)
	documentJSON := []byte(`[{"x":"7", "y":"15"}]`)
	finalJSON := []byte(`[{"x":"7","y":"15"},{"x":"2","y":"3"}]`)
	tmpfile := "/tmp/___jpath.json"
	err = ioutil.WriteFile(tmpfile, documentJSON, 0666)
	assert.Equal(t, nil, err)
	defer os.Remove(tmpfile)

	err = AddJSON(tmpfile, "x", someJSON, false)
	assert.Equal(t, nil, err)

	fileData, err := ioutil.ReadFile(tmpfile)
	assert.Equal(t, nil, err)

	assert.Equal(t, string(fileData), string(finalJSON))

	// Adding data when the path is not found should result in an error
	err = AddJSON(tmpfile, "ost", someJSON, true)
	assert.NotEqual(t, nil, err)
}

func TestGetFile(t *testing.T) {
	var (
		found string
		err   error
	)

	documentJSON := []byte(`[{"x":"7","y":"15"},{"x":"2","y":"3"}]`)
	tmpfile := "/tmp/___jpath.json"
	err = ioutil.WriteFile(tmpfile, documentJSON, 0666)
	assert.Equal(t, nil, err)
	defer os.Remove(tmpfile)

	js, err := New(documentJSON)
	assert.Equal(t, nil, err)
	assert.NotEqual(t, nil, js)

	node := js.GetNode("[1]")
	assert.Equal(t, NilNode, node)

	found, err = GetString(tmpfile, "[1].x")
	assert.Equal(t, nil, err)
	assert.Equal(t, found, "2")

	found, err = GetString(tmpfile, "x.[1].x")
	assert.Equal(t, nil, err)
	assert.Equal(t, found, "2")

	found, err = GetString(tmpfile, "x[1].x")
	assert.Equal(t, nil, err)
	assert.Equal(t, found, "2")
}

// TODO: add the cli tests that fail here
