package jpath

import (
	"errors"
	"io/ioutil"
	"sync"
)

var (
	// ErrSpecificNode is for when retrieving a node does not return a specific key/value, but perhaps a map
	ErrSpecificNode = errors.New("Could not find a specific node that matched the given path")
)

// JFile represents a JSON file and contains the filename and root node
type JFile struct {
	filename string
	rootnode *Node
	rw       *sync.RWMutex
	pretty   bool // Indent JSON output prettily
}

// NewFile will read the given filename and return a JFile struct
func NewFile(filename string) (*JFile, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	js, err := New(data)
	if err != nil {
		return nil, err
	}
	rw := &sync.RWMutex{}
	return &JFile{filename, js, rw, true}, nil
}

// GetFilename returns the current filename
func (jf *JFile) GetFilename() string {
	return jf.filename
}

// SetPretty can be used for setting the "pretty" flag to true, for indenting
// all JSON output. Set to true by default.
func (jf *JFile) SetPretty(pretty bool) {
	jf.pretty = pretty
}

// SetRW allows a different mutex to be used when writing the JSON documents to file
func (jf *JFile) SetRW(rw *sync.RWMutex) {
	jf.rw = rw
}

// GetNode tries to find the JSON node that corresponds to the given JSON path
func (jf *JFile) GetNode(JSONpath string) (*Node, error) {
	node, _, err := jf.rootnode.GetNodes(JSONpath)
	if node == NilNode {
		return NilNode, errors.New("nil node")
	}
	return node, err
}

// GetString tries to find the string that corresponds to the given JSON path
func (jf *JFile) GetString(JSONpath string) (string, error) {
	node, err := jf.GetNode(JSONpath)
	if err != nil {
		return "", err
	}
	return node.String(), nil
}

// SetString will change the value of the key that the given JSON path points to
func (jf *JFile) SetString(JSONpath, value string) error {
	_, parentNode, err := jf.rootnode.GetNodes(JSONpath)
	if err != nil {
		return err
	}
	m, ok := parentNode.CheckMap()
	if !ok {
		return errors.New("Parent is not a map: " + JSONpath)
	}

	// Set the string
	m[lastpart(JSONpath)] = value

	newdata, err := jf.rootnode.PrettyJSON()
	if err != nil {
		return err
	}

	return jf.Write(newdata)
}

// Write writes the current JSON data to the file
func (jf *JFile) Write(data []byte) error {
	jf.rw.Lock()
	defer jf.rw.Unlock()
	return ioutil.WriteFile(jf.filename, data, 0666)
}

// AddJSON adds JSON data at the given JSON path. If pretty is true, the JSON is indented.
func (jf *JFile) AddJSON(JSONpath string, JSONdata []byte) error {
	if err := jf.rootnode.AddJSON(JSONpath, JSONdata); err != nil {
		return err
	}
	// Use the correct JSON function, depending on the pretty parameter
	JSON := jf.rootnode.JSON
	if jf.pretty {
		JSON = jf.rootnode.PrettyJSON
	}
	data, err := JSON()
	if err != nil {
		return err
	}
	return jf.Write(data)
}

// DelKey removes a key from the map that the JSON path leads to.
// Returns ErrKeyNotFound if the key is not found.
func (jf *JFile) DelKey(JSONpath string) error {
	err := jf.rootnode.DelKey(JSONpath)
	if err != nil {
		return err
	}
	// Use the correct JSON function, depending on the pretty parameter
	JSON := jf.rootnode.JSON
	if jf.pretty {
		JSON = jf.rootnode.PrettyJSON
	}
	data, err := JSON()
	if err != nil {
		return err
	}
	return jf.Write(data)
}

// JSON returns the current JSON data, as prettily formatted JSON
func (jf *JFile) JSON() ([]byte, error) {
	return jf.rootnode.PrettyJSON()
}

// SetString sets a value to the given JSON file at the given JSON path
func SetString(filename, JSONpath, value string) error {
	jf, err := NewFile(filename)
	if err != nil {
		return err
	}
	return jf.SetString(JSONpath, value)
}

// AddJSON adds JSON data to the given JSON file at the given JSON path
func AddJSON(filename, JSONpath string, JSONdata []byte, pretty bool) error {
	jf, err := NewFile(filename)
	if err != nil {
		return err
	}
	jf.SetPretty(pretty)
	return jf.AddJSON(JSONpath, JSONdata)
}

// GetString will find the string that corresponds to the given JSON path,
// given a filename and a simple JSON path expression.
func GetString(filename, JSONpath string) (string, error) {
	jf, err := NewFile(filename)
	if err != nil {
		return "", err
	}
	return jf.GetString(JSONpath)
}

// DelKey removes a key from a map in a JSON file, given a JSON path,
// where the last element of the path is the key to be removed.
func DelKey(filename, JSONpath string) error {
	jf, err := NewFile(filename)
	if err != nil {
		return err
	}
	return jf.DelKey(JSONpath)
}
