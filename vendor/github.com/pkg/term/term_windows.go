package term

import (
	"errors"
)

type Term struct {
}

var errNotSupported = errors.New("not supported")

// Open opens an asynchronous communications port.
func Open(name string, options ...func(*Term) error) (*Term, error) {
	return nil, errNotSupported
}

// SetOption takes one or more option function and applies them in order to Term.
func (t *Term) SetOption(options ...func(*Term) error) error {
	return errNotSupported
}

// Read reads up to len(b) bytes from the terminal. It returns the number of
// bytes read and an error, if any. EOF is signaled by a zero count with
// err set to io.EOF.
func (t *Term) Read(b []byte) (int, error) {
	return 0, errNotSupported
}

// Write writes len(b) bytes to the terminal. It returns the number of bytes
// written and an error, if any. Write returns a non-nil error when n !=
// len(b).
func (t *Term) Write(b []byte) (int, error) {
	return 0, errNotSupported
}

// Close closes the device and releases any associated resources.
func (t *Term) Close() error {
	return errNotSupported
}

// SetCbreak sets cbreak mode.
func (t *Term) SetCbreak() error {
	return errNotSupported
}

// CBreakMode places the terminal into cbreak mode.
func CBreakMode(t *Term) error {
	return errNotSupported
}

// SetRaw sets raw mode.
func (t *Term) SetRaw() error {
	return errNotSupported
}

// RawMode places the terminal into raw mode.
func RawMode(t *Term) error {
	return errNotSupported
}

// Speed sets the baud rate option for the terminal.
func Speed(baud int) func(*Term) error {
	return func(*Term) error { return errNotSupported }
}

// SetSpeed sets the receive and transmit baud rates.
func (t *Term) SetSpeed(baud int) error {
	return errNotSupported
}

// GetSpeed gets the transmit baud rate.
func (t *Term) GetSpeed() (int, error) {
	return 0, errNotSupported
}

// Flush flushes both data received but not read, and data written but not transmitted.
func (t *Term) Flush() error {
	return errNotSupported
}

// SendBreak sends a break signal.
func (t *Term) SendBreak() error {
	return errNotSupported
}

// SetDTR sets the DTR (data terminal ready) signal.
func (t *Term) SetDTR(v bool) error {
	return errNotSupported
}

// DTR returns the state of the DTR (data terminal ready) signal.
func (t *Term) DTR() (bool, error) {
	return false, errNotSupported
}

// SetRTS sets the RTS (data terminal ready) signal.
func (t *Term) SetRTS(v bool) error {
	return errNotSupported
}

// RTS returns the state of the RTS (data terminal ready) signal.
func (t *Term) RTS() (bool, error) {
	return false, errNotSupported
}

// Restore restores the state of the terminal captured at the point that
// the terminal was originally opened.
func (t *Term) Restore() error {
	return errNotSupported
}
