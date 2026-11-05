package vt

import (
	"strconv"
	"strings"
)

// MouseButton is the button that a MouseEvent is about
type MouseButton int

// Mouse buttons. MouseNone is used for releases reported without a button and
// for motion without a button held down.
const (
	MouseNone MouseButton = iota
	MouseLeft
	MouseMiddle
	MouseRight
	MouseWheelUp
	MouseWheelDown
	MouseWheelLeft
	MouseWheelRight
)

// MouseEvent is a mouse button press, release, drag or scroll wheel movement.
// X and Y are 0-based cell coordinates, as for SetXY.
type MouseEvent struct {
	X, Y    uint
	Button  MouseButton
	Release bool
	Motion  bool
	Shift   bool
	Alt     bool
	Ctrl    bool
}

// ParseMouseEvent parses a key string that ReadKey returned after EnableMouse
// was called. It supports both SGR (ESC [ < b ; x ; y M/m) and the older
// X10 / normal (ESC [ M b x y) mouse reports.
func ParseMouseEvent(key string) (MouseEvent, bool) {
	if rest, ok := strings.CutPrefix(key, "\x1b[<"); ok && len(rest) > 0 {
		final := rest[len(rest)-1]
		if final != 'M' && final != 'm' {
			return MouseEvent{}, false
		}
		fields := strings.Split(rest[:len(rest)-1], ";")
		if len(fields) != 3 {
			return MouseEvent{}, false
		}
		var numbers [3]int
		for i, field := range fields {
			n, err := strconv.Atoi(field)
			if err != nil || n < 0 {
				return MouseEvent{}, false
			}
			numbers[i] = n
		}
		if numbers[1] < 1 || numbers[2] < 1 {
			return MouseEvent{}, false
		}
		event := newMouseEvent(numbers[0], uint(numbers[1]-1), uint(numbers[2]-1))
		if final == 'm' {
			event.Release = true
		}
		return event, true
	}
	if rest, ok := strings.CutPrefix(key, "\x1b[M"); ok && len(rest) == 3 {
		if rest[0] < 32 || rest[1] < 33 || rest[2] < 33 {
			return MouseEvent{}, false
		}
		event := newMouseEvent(int(rest[0]-32), uint(rest[1]-33), uint(rest[2]-33))
		if event.Button == MouseNone && !event.Motion {
			event.Release = true
		}
		return event, true
	}
	return MouseEvent{}, false
}

// newMouseEvent decodes the button and modifier bits of a mouse report
func newMouseEvent(b int, x, y uint) MouseEvent {
	event := MouseEvent{
		X:      x,
		Y:      y,
		Motion: b&32 != 0,
		Shift:  b&4 != 0,
		Alt:    b&8 != 0,
		Ctrl:   b&16 != 0,
	}
	low := b & 3
	switch {
	case b&64 != 0:
		event.Button = MouseWheelUp + MouseButton(low)
	case low == 3:
		event.Button = MouseNone
	default:
		event.Button = MouseLeft + MouseButton(low)
	}
	return event
}
