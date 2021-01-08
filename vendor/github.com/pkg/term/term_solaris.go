package term

import (
	"os"
	"syscall"
	"unsafe"

	"github.com/pkg/term/termios"
	"golang.org/x/sys/unix"
)

type attr unix.Termios

func (a *attr) getSpeed() (int, error) {
	// We generally only care about ospeed, since that's what would
	// be used for padding characters, for example.

	rate := termios.Cfgetospeed((*unix.Termios)(a))

	switch rate {
	case unix.B50:
		return 50, nil
	case unix.B75:
		return 75, nil
	case unix.B110:
		return 110, nil
	case unix.B134:
		return 134, nil
	case unix.B150:
		return 150, nil
	case unix.B200:
		return 200, nil
	case unix.B300:
		return 300, nil
	case unix.B600:
		return 600, nil
	case unix.B1200:
		return 1200, nil
	case unix.B1800:
		return 1800, nil
	case unix.B2400:
		return 2400, nil
	case unix.B4800:
		return 4800, nil
	case unix.B9600:
		return 9600, nil
	case unix.B19200:
		return 19200, nil
	case unix.B38400:
		return 38400, nil
	case unix.B57600:
		return 57600, nil
	case unix.B115200:
		return 115200, nil
	case unix.B230400:
		return 230400, nil
	case unix.B460800:
		return 460800, nil
	case unix.B921600:
		return 921600, nil
	default:
		return 0, unix.EINVAL
	}
}

func (a *attr) setSpeed(baud int) error {
	var rate uint32
	switch baud {
	case 50:
		rate = unix.B50
	case 75:
		rate = unix.B75
	case 110:
		rate = unix.B110
	case 134:
		rate = unix.B134
	case 150:
		rate = unix.B150
	case 200:
		rate = unix.B200
	case 300:
		rate = unix.B300
	case 600:
		rate = unix.B600
	case 1200:
		rate = unix.B1200
	case 1800:
		rate = unix.B1800
	case 2400:
		rate = unix.B2400
	case 4800:
		rate = unix.B4800
	case 9600:
		rate = unix.B9600
	case 19200:
		rate = unix.B19200
	case 38400:
		rate = unix.B38400
	case 57600:
		rate = unix.B57600
	case 115200:
		rate = unix.B115200
	case 230400:
		rate = unix.B230400
	case 460800:
		rate = unix.B460800
	case 921600:
		rate = unix.B921600
	default:
		return unix.EINVAL
	}

	err := termios.Cfsetispeed((*unix.Termios)(a), uintptr(rate))
	if err != nil {
		return err
	}

	err = termios.Cfsetospeed((*unix.Termios)(a), uintptr(rate))
	if err != nil {
		return err
	}

	return nil
}

// Open opens an asynchronous communications port.
func Open(name string, options ...func(*Term) error) (*Term, error) {

	// copied from https://github.com/kofemann/opensolaris/blob/master/usr/src/uts/common/sys/stropts.h#L229
	// to avoid cgo dependency.
	const (
		STR    = ('S' << 8)
		I_PUSH = (STR | 02)
	)

	fd, e := unix.Open(name, unix.O_NOCTTY|unix.O_CLOEXEC|unix.O_NDELAY|unix.O_RDWR, 0666)
	if e != nil {
		return nil, &os.PathError{"open", name, e}
	}

	modules := [2]string{"ptem", "ldterm"}
	for _, mod := range modules {
		err := unix.IoctlSetInt(fd, I_PUSH, int(uintptr(unsafe.Pointer(syscall.StringBytePtr(mod)))))
		if err != nil {
			return nil, err
		}
	}

	orig, err := termios.Tcgetattr(uintptr(t.fd))
	if err != nil {
		return nil, err
	}
	t := Term{name: name, fd: fd, *orig}

	if err := t.SetOption(options...); err != nil {
		return nil, err
	}

	return &t, unix.SetNonblock(t.fd, false)
}

// Restore restores the state of the terminal captured at the point that
// the terminal was originally opened.
func (t *Term) Restore() error {
	return termios.Tcsetattr(uintptr(t.fd), termios.TCSANOW, &t.orig)
}
