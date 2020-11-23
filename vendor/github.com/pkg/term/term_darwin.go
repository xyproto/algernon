package term

import "golang.org/x/sys/unix"

type attr unix.Termios

func (a *attr) getSpeed() (int, error) {
	// We generally only care about ospeed, since that's what would
	// be used for padding characters, for example.

	switch a.Ospeed {
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
	default:
		return 0, unix.EINVAL
	}
}

func (a *attr) setSpeed(baud int) error {
	var rate uint64
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
	default:
		return unix.EINVAL
	}
	(*unix.Termios)(a).Cflag = unix.CS8 | unix.CREAD | unix.CLOCAL | rate
	(*unix.Termios)(a).Ispeed = rate
	(*unix.Termios)(a).Ospeed = rate
	return nil
}
