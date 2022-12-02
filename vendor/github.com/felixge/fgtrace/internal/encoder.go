package internal

// Implementation of Chrome's Trace Event Format, see:
// https://docs.google.com/document/d/1CvAClvFfyA5R-PhYUmn5OOQtYMH4h6I0nSsKchNAySU/preview

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/DataDog/gostackparse"
)

func NewEncoder(w io.Writer) (*Encoder, error) {
	e := &Encoder{
		w:     w,
		json:  *json.NewEncoder(w),
		first: true,
	}
	_, err := w.Write([]byte("["))
	return e, err
}

func Unmarshal(data []byte) (*TraceData, error) {
	var tr TraceData
	return &tr, json.Unmarshal(data, &tr.Events)
}

type TraceData struct {
	Events []*Event
}

func (t *TraceData) MetaHz() int {
	for _, e := range t.Events {
		if e.Ph == "M" && e.Name == "hz" {
			hz, _ := e.Args["hz"].(float64)
			return int(hz)
		}
	}
	return 0
}

func (t *TraceData) Filter(fn func(*Event) bool) *TraceData {
	nt := &TraceData{Events: make([]*Event, 0, len(t.Events))}
	for _, e := range t.Events {
		if fn(e) {
			nt.Events = append(nt.Events, e)
		}
	}
	return nt
}

// CallGraph returns a graph of all function calls (stack traces) in the trace.
func (t *TraceData) CallGraph() *Node {
	root := &Node{}
	goroutineTrees := map[int64][]*Node{}
	for _, e := range t.Events {
		goroutineID := e.Pid
		l := len(goroutineTrees[goroutineID])
		switch e.Ph {
		case "B":
			child := &Node{Func: e.Name}
			if l == 0 {
				goroutineTrees[goroutineID] = []*Node{child}
				root.Children = append(root.Children, child)
			} else {
				parent := goroutineTrees[goroutineID][l-1]
				var found bool
				for _, pChild := range parent.Children {
					if pChild.Func == child.Func {
						child = pChild
						found = true
						break
					}
				}
				if !found {
					parent.Children = append(parent.Children, child)
				}
				goroutineTrees[goroutineID] = append(goroutineTrees[goroutineID], child)
			}
		case "E":
			goroutineTrees[goroutineID] = goroutineTrees[goroutineID][0 : l-1]
		}
	}
	return root
}

func (t *TraceData) Len() int {
	return len(t.Events)
}

type Node struct {
	Func     string
	Children []*Node
}

func (n *Node) HasLeaf(fn string) bool {
	if len(n.Children) == 0 {
		return n.Func == fn
	}
	for _, child := range n.Children {
		if child.HasLeaf(fn) {
			return true
		}
	}
	return false
}

func (n *Node) String() string {
	var s string

	var visit func(n *Node, depth int)
	visit = func(n *Node, depth int) {
		if depth > -1 {
			s += fmt.Sprintf("%s%s\n", strings.Repeat("  ", depth), n.Func)
		}
		for _, child := range n.Children {
			visit(child, depth+1)
		}
	}
	visit(n, -1)

	return s
}

type Event struct {
	Name string `json:"name,omitempty"`
	Ph   string `json:"ph,omitempty"`
	// Ts is the tracing clock timestamp of the event. The timestamps are
	// provided at microsecond granularity.
	Ts   float64                `json:"ts"`
	Pid  int64                  `json:"pid,omitempty"`
	Tid  int64                  `json:"tid,omitempty"`
	Args map[string]interface{} `json:"args,omitempty"`
}

// Encoder implements a small subset of the "Trace Event Format" spec needed to
// make fgtrace output data that can be displayed by perfetto.dev.
// https://docs.google.com/document/d/1CvAClvFfyA5R-PhYUmn5OOQtYMH4h6I0nSsKchNAySU/preview
type Encoder struct {
	w     io.Writer
	json  json.Encoder
	first bool
}

func (e *Encoder) CustomMeta(name string, value interface{}) error {
	ev := Event{
		Name: name,
		Ph:   "M",
		Args: map[string]interface{}{name: value},
	}
	return e.encode(&ev)
}

func (e *Encoder) Encode(ts float64, prev, current *gostackparse.Goroutine) error {
	ev := Event{Ts: ts, Tid: 1}
	prevLen := 0
	if prev != nil {
		prevLen = len(prev.Stack)
		ev.Pid = int64(prev.ID)
	}
	currentLen := 0
	if current != nil {
		currentLen = len(current.Stack)
		ev.Pid = int64(current.ID)
	}

	if prev == nil {
		metaEv := ev
		metaEv.Ts = 0
		metaEv.Name = "process_name"
		metaEv.Ph = "M"
		name := fmt.Sprintf("G%d", current.ID)
		if current.CreatedBy != nil {
			name += " " + current.CreatedBy.Func
		}
		metaEv.Args = map[string]interface{}{"name": name}
		if err := e.encode(&metaEv); err != nil {
			return err
		}
	}

	// Determine the number of stack frames that are identical between prev and
	// current going from root frame (e.g. main) to the leaf frame.
	commonDepth := prevLen
	for i := 0; i < prevLen; i++ {
		ci := currentLen - i - 1
		pi := prevLen - i - 1
		if ci < 0 || prev.Stack[pi].Func != current.Stack[ci].Func {
			commonDepth = i
			break
		}
	}

	// Emit end events for prev stack frames that are no longer part of the
	// current stack going from leaf to root frame.
	for pi := 0; pi < prevLen-commonDepth; pi++ {
		ev.Ph = "E"
		ev.Name = prev.Stack[pi].Func
		if err := e.encode(&ev); err != nil {
			return err
		}
	}

	// Emit start events for current stack frames that were not part of the prev
	// stack going from root to leaf frame.
	for i := commonDepth; i < currentLen; i++ {
		ci := currentLen - i - 1
		ev.Ph = "B"
		ev.Name = current.Stack[ci].Func
		if err := e.encode(&ev); err != nil {
			return err
		}
	}
	return nil
}

func (e *Encoder) encode(ev *Event) error {
	if !e.first {
		if _, err := e.w.Write([]byte(",")); err != nil {
			return err
		}
	} else {
		e.first = false
	}
	return e.json.Encode(ev)
}

func (e *Encoder) Finish() error {
	_, err := e.w.Write([]byte("]"))
	return err
}
