// Package jpath provides a way to search and manipulate JSON documents
package jpath

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"reflect"
	"strconv"
	"strings"
)

// Version contains the version number. The API is stable within the same major version.
const Version = 1.0

type (
	// Node is a JSON document, or a part of a JSON document
	Node struct {
		data interface{}
	}
	// NodeList is a list of nodes
	NodeList []*Node
	// NodeMap is a map of nodes
	NodeMap map[string]*Node
)

// NilNode is an empty node. Used when not finding nodes with Get.
var (
	NilNode        = &Node{nil}
	ErrKeyNotFound = errors.New("Key not found")
)

// New returns a pointer to a new `Node` object
// after unmarshaling `body` bytes
func New(body []byte) (*Node, error) {
	if len(body) == 0 {
		// Use an empty list if no data has been provided
		body = []byte("[]")
	}
	j := new(Node)
	err := j.UnmarshalJSON(body)
	if err != nil {
		return nil, err
	}
	return j, nil
}

// NewNode returns a pointer to a new, empty `Node` object
func NewNode() *Node {
	return &Node{
		data: make(map[string]interface{}),
	}
}

// Interface returns the underlying data
func (j *Node) Interface() interface{} {
	return j.data
}

// JSON returns its marshaled data as `[]byte`
func (j *Node) JSON() ([]byte, error) {
	data, err := j.MarshalJSON()
	if err != nil {
		return []byte{}, err
	}
	return data, nil
}

// MustJSON returns its marshaled data as `[]byte`
func (j *Node) MustJSON() []byte {
	data, err := j.MarshalJSON()
	if err != nil {
		return []byte{}
	}
	return data
}

// PrettyJSON returns its marshaled data as `[]byte` with indentation
func (j *Node) PrettyJSON() ([]byte, error) {
	return json.MarshalIndent(&j.data, "", "  ")
}

// MarshalJSON implements the json.Marshaler interface
func (j *Node) MarshalJSON() ([]byte, error) {
	return json.Marshal(&j.data)
}

// Set modifies `Node` map by `key` and `value`
// Useful for changing single key/value in a `Node` object easily.
func (j *Node) Set(key string, val interface{}) {
	m, ok := j.CheckMap()
	if !ok {
		return
	}
	m[key] = val
}

// SetBranch modifies `Node`, recursively checking/creating map keys for the supplied path,
// and then finally writing in the value.
func (j *Node) SetBranch(branch []string, val interface{}) {
	if len(branch) == 0 {
		j.data = val
		return
	}

	// in order to insert our branch, we need map[string]interface{}
	if _, ok := (j.data).(map[string]interface{}); !ok {
		// have to replace with something suitable
		j.data = make(map[string]interface{})
	}
	curr := j.data.(map[string]interface{})

	for i := 0; i < len(branch)-1; i++ {
		b := branch[i]
		// key exists?
		if _, ok := curr[b]; !ok {
			n := make(map[string]interface{})
			curr[b] = n
			curr = n
			continue
		}

		// make sure the value is the right sort of thing
		if _, ok := curr[b].(map[string]interface{}); !ok {
			// have to replace with something suitable
			n := make(map[string]interface{})
			curr[b] = n
		}

		curr = curr[b].(map[string]interface{})
	}

	// add remaining k/v
	curr[branch[len(branch)-1]] = val
}

// GetKey returns a pointer to a new `Node` object
// for `key` in its `map` representation
// and a bool identifying success or failure
func (j *Node) GetKey(key string) (*Node, bool) {
	m, ok := j.CheckMap()
	if ok {
		if val, ok := m[key]; ok {
			return &Node{val}, true
		}
	}
	return nil, false
}

// GetIndex returns a pointer to a new `Node` object
// for `index` in its slice representation
// and a bool identifying success or failure
func (j *Node) GetIndex(index int) (*Node, bool) {
	a, ok := j.CheckList()
	if ok {
		if len(a) > index {
			return &Node{a[index]}, true
		}
	}
	return nil, false
}

// Get searches for the item as specified by the branch
// within a nested Node and returns a new Node pointer
// the pointer is always a valid Node, allowing for chained operations
//
//   newJs := js.Get("top_level", "entries", 3, "dict")
func (j *Node) Get(branch ...interface{}) *Node {
	jin, ok := j.CheckGet(branch...)
	if !ok {
		return NilNode
	}
	return jin
}

// CheckGet is like Get, except it also returns a bool
// indicating whenever the branch was found or not
// the Node pointer may be nil
//
//   newJs, ok := js.Get("top_level", "entries", 3, "dict")
func (j *Node) CheckGet(branch ...interface{}) (*Node, bool) {
	jin := j
	var ok bool
	for _, p := range branch {
		switch p.(type) {
		case string:
			jin, ok = jin.GetKey(p.(string))
		case int:
			jin, ok = jin.GetIndex(p.(int))
		default:
			ok = false
		}
		if !ok {
			return nil, false
		}
	}
	return jin, true
}

// CheckNodeMap returns a copy of a Node map, but with values as Nodes
func (j *Node) CheckNodeMap() (NodeMap, bool) {
	m, ok := j.CheckMap()
	if !ok {
		return nil, false
	}
	jm := make(NodeMap)
	for key, val := range m {
		jm[key] = &Node{val}
	}
	return jm, true
}

// CheckNodeList returns a copy of a slice, but with each value as a Node
func (j *Node) CheckNodeList() ([]*Node, bool) {
	a, ok := j.CheckList()
	if !ok {
		return nil, false
	}
	ja := make([]*Node, len(a))
	for key, val := range a {
		ja[key] = &Node{val}
	}
	return ja, true
}

// CheckMap type asserts to `map`
func (j *Node) CheckMap() (map[string]interface{}, bool) {
	if m, ok := (j.data).(map[string]interface{}); ok {
		return m, true
	}
	return nil, false
}

// CheckList type asserts to a slice
func (j *Node) CheckList() ([]interface{}, bool) {
	if a, ok := (j.data).([]interface{}); ok {
		return a, true
	}
	return nil, false
}

// CheckBool type asserts to `bool`
func (j *Node) CheckBool() (bool, bool) {
	if s, ok := (j.data).(bool); ok {
		return s, true
	}
	return false, false
}

// CheckString type asserts to `string`
func (j *Node) CheckString() (string, bool) {
	if s, ok := (j.data).(string); ok {
		return s, true
	}
	return "", false
}

// NodeList guarantees the return of a `[]*Node` (with optional default)
func (j *Node) NodeList(args ...NodeList) NodeList {
	var def NodeList

	switch len(args) {
	case 0:
	case 1:
		def = args[0]
	default:
		log.Panicf("NodeList() received too many arguments %d", len(args))
	}

	if a, ok := j.CheckNodeList(); ok {
		return a
	}

	return def
}

// NodeMap guarantees the return of a `map[string]*Node` (with optional default)
func (j *Node) NodeMap(args ...NodeMap) NodeMap {
	var def NodeMap

	switch len(args) {
	case 0:
	case 1:
		def = args[0]
	default:
		log.Panicf("NodeMap() received too many arguments %d", len(args))
	}

	if a, ok := j.CheckNodeMap(); ok {
		return a
	}

	return def
}

// List guarantees the return of a `[]interface{}` (with optional default)
//
// useful when you want to interate over array values in a succinct manner:
//		for i, v := range js.Get("results").List() {
//			fmt.Println(i, v)
//		}
func (j *Node) List(args ...[]interface{}) []interface{} {
	var def []interface{}

	switch len(args) {
	case 0:
	case 1:
		def = args[0]
	default:
		log.Panicf("List() received too many arguments %d", len(args))
	}

	if a, ok := j.CheckList(); ok {
		return a
	}

	return def
}

// Map guarantees the return of a `map[string]interface{}` (with optional default)
//
// useful when you want to interate over map values in a succinct manner:
//		for k, v := range js.Get("dictionary").Map() {
//			fmt.Println(k, v)
//		}
func (j *Node) Map(args ...map[string]interface{}) map[string]interface{} {
	var def map[string]interface{}

	switch len(args) {
	case 0:
	case 1:
		def = args[0]
	default:
		log.Panicf("Map() received too many arguments %d", len(args))
	}

	a, ok := j.CheckMap()
	if ok {
		return a
	}

	return def
}

// String guarantees the return of a `string` (with optional default)
//
// useful when you explicitly want a `string` in a single value return context:
//     myFunc(js.Get("param1").String(), js.Get("optional_param").String("my_default"))
func (j *Node) String(args ...string) string {
	var def string

	switch len(args) {
	case 0:
	case 1:
		def = args[0]
	default:
		log.Panicf("String() received too many arguments %d", len(args))
	}

	s, ok := j.CheckString()
	if ok {
		return s
	}

	return def
}

// Int guarantees the return of an `int` (with optional default)
//
// useful when you explicitly want an `int` in a single value return context:
//     myFunc(js.Get("param1").Int(), js.Get("optional_param").Int(5150))
func (j *Node) Int(args ...int) int {
	var def int

	switch len(args) {
	case 0:
	case 1:
		def = args[0]
	default:
		log.Panicf("Int() received too many arguments %d", len(args))
	}

	i, ok := j.CheckInt()
	if ok {
		return i
	}

	return def
}

// Float64 guarantees the return of a `float64` (with optional default)
//
// useful when you explicitly want a `float64` in a single value return context:
//     myFunc(js.Get("param1").Float64(), js.Get("optional_param").Float64(5.150))
func (j *Node) Float64(args ...float64) float64 {
	var def float64

	switch len(args) {
	case 0:
	case 1:
		def = args[0]
	default:
		log.Panicf("Float64() received too many arguments %d", len(args))
	}

	f, ok := j.CheckFloat64()
	if ok {
		return f
	}

	return def
}

// Bool guarantees the return of a `bool` (with optional default)
//
// useful when you explicitly want a `bool` in a single value return context:
//     myFunc(js.Get("param1").Bool(), js.Get("optional_param").Bool(true))
func (j *Node) Bool(args ...bool) bool {
	var def bool

	switch len(args) {
	case 0:
	case 1:
		def = args[0]
	default:
		log.Panicf("Bool() received too many arguments %d", len(args))
	}

	b, ok := j.CheckBool()
	if ok {
		return b
	}

	return def
}

// Int64 guarantees the return of an `int64` (with optional default)
//
// useful when you explicitly want an `int64` in a single value return context:
//     myFunc(js.Get("param1").Int64(), js.Get("optional_param").Int64(5150))
func (j *Node) Int64(args ...int64) int64 {
	var def int64

	switch len(args) {
	case 0:
	case 1:
		def = args[0]
	default:
		log.Panicf("Int64() received too many arguments %d", len(args))
	}

	i, ok := j.CheckInt64()
	if ok {
		return i
	}

	return def
}

// Uint64 guarantees the return of an `uint64` (with optional default)
//
// useful when you explicitly want an `uint64` in a single value return context:
//     myFunc(js.Get("param1").Uint64(), js.Get("optional_param").Uint64(5150))
func (j *Node) Uint64(args ...uint64) uint64 {
	var def uint64

	switch len(args) {
	case 0:
	case 1:
		def = args[0]
	default:
		log.Panicf("Uint64() received too many arguments %d", len(args))
	}

	i, ok := j.CheckUint64()
	if ok {
		return i
	}

	return def
}

// NewFromReader returns a *Node by decoding from an io.Reader
func NewFromReader(r io.Reader) (*Node, error) {
	j := new(Node)
	dec := json.NewDecoder(r)
	err := dec.Decode(&j.data)
	return j, err
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (j *Node) UnmarshalJSON(p []byte) error {
	return json.Unmarshal(p, &j.data)
}

// CheckFloat64 coerces into a float64
func (j *Node) CheckFloat64() (float64, bool) {
	switch j.data.(type) {
	case float32, float64:
		return reflect.ValueOf(j.data).Float(), true
	case int, int8, int16, int32, int64:
		return float64(reflect.ValueOf(j.data).Int()), true
	case uint, uint8, uint16, uint32, uint64:
		return float64(reflect.ValueOf(j.data).Uint()), true
	}
	return 0, false
}

// CheckInt coerces into an int
func (j *Node) CheckInt() (int, bool) {
	switch j.data.(type) {
	case float32, float64:
		return int(reflect.ValueOf(j.data).Float()), true
	case int, int8, int16, int32, int64:
		return int(reflect.ValueOf(j.data).Int()), true
	case uint, uint8, uint16, uint32, uint64:
		return int(reflect.ValueOf(j.data).Uint()), true
	}
	return 0, false
}

// CheckInt64 coerces into an int64
func (j *Node) CheckInt64() (int64, bool) {
	switch j.data.(type) {
	case float32, float64:
		return int64(reflect.ValueOf(j.data).Float()), true
	case int, int8, int16, int32, int64:
		return reflect.ValueOf(j.data).Int(), true
	case uint, uint8, uint16, uint32, uint64:
		return int64(reflect.ValueOf(j.data).Uint()), true
	}
	return 0, false
}

// CheckUint64 coerces into an uint64
func (j *Node) CheckUint64() (uint64, bool) {
	switch j.data.(type) {
	case float32, float64:
		return uint64(reflect.ValueOf(j.data).Float()), true
	case int, int8, int16, int32, int64:
		return uint64(reflect.ValueOf(j.data).Int()), true
	case uint, uint8, uint16, uint32, uint64:
		return reflect.ValueOf(j.data).Uint(), true
	}
	return 0, false
}

// GetNodes will find the JSON node (and parent node) that corresponds to the given JSON path
func (j *Node) GetNodes(JSONpath string) (*Node, *Node, error) {
	parent := j
	if JSONpath == "x" || JSONpath == "" {
		// If the root node is a map or list with one element or less, use that as the node
		if m, ok := j.CheckNodeMap(); ok && len(m) <= 1 {
			return parent, NilNode, nil
		} else if l, ok := j.CheckNodeList(); ok && len(l) <= 1 {
			return parent, NilNode, nil
		}
		// We may have encountered a list with more than one item, for example
		return parent, NilNode, nil
	}
	// JSON path starting with x[ is a special case.
	if strings.HasPrefix(JSONpath, "x[") {
		// Add a "." between "x" and "[".
		JSONpath = "x." + JSONpath[1:]
	}
	// The "current node" starts out with being the root node
	n := j
	if strings.Contains(JSONpath, ".") {
		for i, part := range strings.Split(JSONpath, ".") {
			if i == 0 && (part == "" || part == "x") {
				// If the current node is a map or list with one element or less, use that as the next node
				if m, ok := n.CheckNodeMap(); ok && len(m) <= 1 {
					n = parent
				} else if l, ok := n.CheckNodeList(); ok && len(l) <= 1 {
					n = parent
				}
			} else if strings.Contains(part, "[") {
				fields := strings.SplitN(part, "[", 2)
				name := fields[0]
				secondpart := fields[1]
				fields = strings.SplitN(secondpart, "]", 2)
				stringIndex := fields[0]
				index, err := strconv.Atoi(stringIndex)
				if err != nil {
					return NilNode, NilNode, errors.New("Invalid index: " + stringIndex)
				}
				parent = n
				if name == "" {
					n = n.Get(index)
				} else {
					parent = n.Get(name)
					n = parent.Get(index)
				}
			} else {
				parent = n
				n = n.Get(part)
			}
		}
	} else {
		parent = n
		part := JSONpath
		n = n.Get(part)
	}
	return n, parent, nil
}

// GetNode will find the JSON node that corresponds to the given JSON path, or nil.
func (j *Node) GetNode(JSONpath string) *Node {
	node, _, err := j.GetNodes(JSONpath)
	if err != nil {
		return NilNode
	}
	return node
}

// AddJSON adds JSON data to a list. The JSON path must refer to a list.
func (j *Node) AddJSON(JSONpath string, JSONdata []byte) error {
	node := j.GetNode(JSONpath)
	l, ok := node.CheckList()
	if !ok {
		return errors.New("Can only add JSON data to a list. Not a list: " + node.Info())
	}
	newNode, err := New(JSONdata)
	if err != nil {
		return err
	}
	node.data = append(l, newNode)
	return nil
}

// DelKey removes a key in a map, given a JSON path to a map.
// Returns ErrKeyNotFound if the key is not found.
func (j *Node) DelKey(JSONpath string) error {
	_, mapnode, err := j.GetNodes(JSONpath)
	if err != nil {
		return err
	}
	m, ok := mapnode.CheckMap()
	if !ok {
		return errors.New("Can only remove a key from a map. Not a map: " + mapnode.Info())
	}
	keyToRemove := lastpart(JSONpath)
	foundKey := false
	for k := range m {
		if k == keyToRemove {
			foundKey = true
			break
		}
	}
	if !foundKey {
		return ErrKeyNotFound
	}
	delete(m, lastpart(JSONpath))
	return nil
}

// Info returns a description of the node
func (j *Node) Info() string {
	var buf bytes.Buffer
	if j == NilNode {
		buf.WriteString("Nil Node")
	} else if m, ok := j.CheckMap(); ok {
		buf.WriteString(fmt.Sprintf("Map with %d elements.", len(m)))
	} else if l, ok := j.CheckList(); ok {
		buf.WriteString(fmt.Sprintf("List with %d elements.", len(l)))
	} else if s, ok := j.CheckString(); ok {
		buf.WriteString(fmt.Sprintf("String: %s", s))
	} else if s, ok := j.CheckInt(); ok {
		buf.WriteString(fmt.Sprintf("Int: %d", s))
	} else if b, ok := j.CheckBool(); ok {
		buf.WriteString(fmt.Sprintf("Bool: %v", b))
	} else if i, ok := j.CheckInt64(); ok {
		buf.WriteString(fmt.Sprintf("Int64: %v", i))
	} else if u, ok := j.CheckUint64(); ok {
		buf.WriteString(fmt.Sprintf("Uint64: %v", u))
	} else if f, ok := j.CheckFloat64(); ok {
		buf.WriteString(fmt.Sprintf("Float64: %v", f))
	} else {
		buf.WriteString("Unknown node type")
	}
	return buf.String()
}
