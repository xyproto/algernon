// +build !windows

package term

import (
	"time"

	"github.com/pkg/term/termios"
	"golang.org/x/sys/unix"
)

// Term represents an asynchronous communications port.
type Term struct {
	name string
	fd   int
	orig unix.Termios // original state of the terminal, see Open and Restore
}

// SetAttr returns an option function which will apply the provided modifier to
// a unix.Termios before using that unix.Termios to set the state of the
// Term. This allows a developer to manually set attributes for the terminal.
// Here's an example case to set a terminal into raw mode, but then re-enable
// the 'opost' attribute:
//
//     func EnableOutputPostprocess(a *unix.Termios) uintptr {
//         a.Oflag |= unix.OPOST
//         return termios.TCSANOW
//     }
//
//     func init() {
//         t, _ = term.Open("/dev/tty")
//         t.SetRaw()
//         t.SetOption(term.SetAttr(EnableOutputPostprocess))
//     }
func SetAttr(modifier func(*unix.Termios) uintptr) func(*Term) error {
	return func(t *Term) error {
		a, err := termios.Tcgetattr(uintptr(t.fd))
		if err != nil {
			return err
		}
		action := modifier(a)
		return termios.Tcsetattr(uintptr(t.fd), action, a)
	}
}

// SetCbreak sets cbreak mode.
func (t *Term) SetCbreak() error {
	return t.SetOption(CBreakMode)
}

// CBreakMode places the terminal into cbreak mode.
func CBreakMode(t *Term) error {
	a, err := termios.Tcgetattr(uintptr(t.fd))
	if err != nil {
		return err
	}
	termios.Cfmakecbreak(a)
	return termios.Tcsetattr(uintptr(t.fd), termios.TCSANOW, a)
}

// SetRaw sets raw mode.
func (t *Term) SetRaw() error {
	return t.SetOption(RawMode)
}

// RawMode places the terminal into raw mode.
func RawMode(t *Term) error {
	a, err := termios.Tcgetattr(uintptr(t.fd))
	if err != nil {
		return err
	}
	termios.Cfmakeraw(a)
	return termios.Tcsetattr(uintptr(t.fd), termios.TCSANOW, a)
}

// Speed sets the baud rate option for the terminal.
func Speed(baud int) func(*Term) error {
	return func(t *Term) error {
		return t.setSpeed(baud)
	}
}

// SetSpeed sets the receive and transmit baud rates.
func (t *Term) SetSpeed(baud int) error {
	return t.SetOption(Speed(baud))
}

func (t *Term) setSpeed(baud int) error {
	a, err := termios.Tcgetattr(uintptr(t.fd))
	if err != nil {
		return err
	}

	err = (*attr)(a).setSpeed(baud)
	if err != nil {
		return err
	}

	return termios.Tcsetattr(uintptr(t.fd), termios.TCSANOW, a)
}

// GetSpeed gets the current output baud rate.
func (t *Term) GetSpeed() (int, error) {
	a, err := termios.Tcgetattr(uintptr(t.fd))
	if err != nil {
		return 0, err
	}
	return (*attr)(a).getSpeed()
}

func clamp(v, lo, hi int64) int64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// timeoutVals converts d into values suitable for termios VMIN and VTIME ctrl chars
func timeoutVals(d time.Duration) (uint8, uint8) {
	if d > 0 {
		// VTIME is expressed in terms of deciseconds
		vtimeDeci := d.Nanoseconds() / 1e6 / 100
		// ensure valid range
		vtime := uint8(clamp(vtimeDeci, 1, 0xff))
		return 0, vtime
	}
	// block indefinitely until we receive at least 1 byte
	return 1, 0
}

// ReadTimeout sets the read timeout option for the terminal.
func ReadTimeout(d time.Duration) func(*Term) error {
	return func(t *Term) error {
		return t.setReadTimeout(d)
	}
}

// SetReadTimeout sets the read timeout.
// A zero value for d means read operations will not time out.
func (t *Term) SetReadTimeout(d time.Duration) error {
	return t.SetOption(ReadTimeout(d))
}

func (t *Term) setReadTimeout(d time.Duration) error {
	a, err := termios.Tcgetattr(uintptr(t.fd))
	if err != nil {
		return err
	}
	a.Cc[unix.VMIN], a.Cc[unix.VTIME] = timeoutVals(d)
	return termios.Tcsetattr(uintptr(t.fd), termios.TCSANOW, a)
}

// FlowControl sets the flow control option for the terminal.
func FlowControl(kind int) func(*Term) error {
	return func(t *Term) error {
		return t.setFlowControl(kind)
	}
}

// SetFlowControl sets whether hardware flow control is enabled.
func (t *Term) SetFlowControl(kind int) error {
	return t.SetOption(FlowControl(kind))
}

func (t *Term) setFlowControl(kind int) error {
	a, err := termios.Tcgetattr(uintptr(t.fd))
	if err != nil {
		return err
	}
	switch kind {
	case NONE:
		a.Iflag &^= termios.IXON | termios.IXOFF | termios.IXANY
		a.Cflag &^= termios.CRTSCTS

	case XONXOFF:
		a.Cflag &^= termios.CRTSCTS
		a.Iflag |= termios.IXON | termios.IXOFF | termios.IXANY

	case HARDWARE:
		a.Iflag &^= termios.IXON | termios.IXOFF | termios.IXANY
		a.Cflag |= termios.CRTSCTS
	}
	return termios.Tcsetattr(uintptr(t.fd), termios.TCSANOW, a)
}

// Flush flushes both data received but not read, and data written but not transmitted.
func (t *Term) Flush() error {
	return termios.Tcflush(uintptr(t.fd), termios.TCIOFLUSH)
}

// SendBreak sends a break signal.
func (t *Term) SendBreak() error {
	return termios.Tcsendbreak(uintptr(t.fd), 0)
}

// DCD returns the state of the DCD (data carrier detect) signal.
func (t *Term) DCD() (bool, error) {
	status, err := termios.Tiocmget(uintptr(t.fd))
	return status&unix.TIOCM_CD == unix.TIOCM_CD, err
}

// SetDTR sets the DTR (data terminal ready) signal.
func (t *Term) SetDTR(v bool) error {
	bits := unix.TIOCM_DTR
	if v {
		return termios.Tiocmbis(uintptr(t.fd), bits)
	} else {
		return termios.Tiocmbic(uintptr(t.fd), bits)
	}
}

// DTR returns the state of the DTR (data terminal ready) signal.
func (t *Term) DTR() (bool, error) {
	status, err := termios.Tiocmget(uintptr(t.fd))
	return status&unix.TIOCM_DTR == unix.TIOCM_DTR, err
}

// DSR returns the state of the DSR (data set ready) signal.
func (t *Term) DSR() (bool, error) {
	status, err := termios.Tiocmget(uintptr(t.fd))
	return status&unix.TIOCM_DSR == unix.TIOCM_DSR, err
}

// SetRTS sets the RTS (data terminal ready) signal.
func (t *Term) SetRTS(v bool) error {
	bits := unix.TIOCM_RTS
	if v {
		return termios.Tiocmbis(uintptr(t.fd), bits)
	} else {
		return termios.Tiocmbic(uintptr(t.fd), bits)
	}
}

// RTS returns the state of the RTS (request to send) signal.
func (t *Term) RTS() (bool, error) {
	status, err := termios.Tiocmget(uintptr(t.fd))
	return status&unix.TIOCM_RTS == unix.TIOCM_RTS, err
}

// CTS returns the state of the CTS (clear to send) signal.
func (t *Term) CTS() (bool, error) {
	status, err := termios.Tiocmget(uintptr(t.fd))
	return status&unix.TIOCM_CTS == unix.TIOCM_CTS, err
}

// RI returns the state of the RI (ring indicator) signal.
func (t *Term) RI() (bool, error) {
	status, err := termios.Tiocmget(uintptr(t.fd))
	return status&unix.TIOCM_RI == unix.TIOCM_RI, err
}

// Close closes the device and releases any associated resources.
func (t *Term) Close() error {
	err := unix.Close(t.fd)
	t.fd = -1
	return err
}
