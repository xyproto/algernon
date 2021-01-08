package term

import "golang.org/x/sys/unix"

type attr unix.Termios

const (
	// CBaudMask is the logical of CBAUD and CBAUDEX, except
	// that those values were not exposed via the syscall
	// package.  Many of these values will be redundant, but
	// this long definition ensures we are portable if some
	// architecture defines different values for them (unlikely).
	CBaudMask = unix.B50 |
		unix.B75 |
		unix.B110 |
		unix.B134 |
		unix.B150 |
		unix.B200 |
		unix.B300 |
		unix.B600 |
		unix.B1200 |
		unix.B1800 |
		unix.B2400 |
		unix.B4800 |
		unix.B9600 |
		unix.B19200 |
		unix.B38400 |
		unix.B57600 |
		unix.B115200 |
		unix.B230400 |
		unix.B460800 |
		unix.B500000 |
		unix.B576000 |
		unix.B921600 |
		unix.B1000000 |
		unix.B1152000 |
		unix.B1500000 |
		unix.B2000000 |
		unix.B2500000 |
		unix.B3000000 |
		unix.B3500000 |
		unix.B4000000
)

func (a *attr) getSpeed() (int, error) {
	switch a.Cflag & CBaudMask {
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
	case unix.B500000:
		return 500000, nil
	case unix.B576000:
		return 576000, nil
	case unix.B921600:
		return 921600, nil
	case unix.B1000000:
		return 1000000, nil
	case unix.B1152000:
		return 1152000, nil
	case unix.B1500000:
		return 1500000, nil
	case unix.B2000000:
		return 2000000, nil
	case unix.B2500000:
		return 2500000, nil
	case unix.B3000000:
		return 3000000, nil
	case unix.B3500000:
		return 3500000, nil
	case unix.B4000000:
		return 4000000, nil
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
	case 500000:
		rate = unix.B500000
	case 576000:
		rate = unix.B576000
	case 921600:
		rate = unix.B921600
	case 1000000:
		rate = unix.B1000000
	case 1152000:
		rate = unix.B1152000
	case 1500000:
		rate = unix.B1500000
	case 2000000:
		rate = unix.B2000000
	case 2500000:
		rate = unix.B2500000
	case 3000000:
		rate = unix.B3000000
	case 3500000:
		rate = unix.B3500000
	case 4000000:
		rate = unix.B4000000
	default:
		return unix.EINVAL
	}
	a.Cflag = unix.CS8 | unix.CREAD | unix.CLOCAL | rate
	a.Ispeed = rate
	a.Ospeed = rate
	return nil
}
