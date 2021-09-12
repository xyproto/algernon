package benchmarked

import (
	"sync"
)

// Equal checks if two slices of bytes are equal
var Equal = equal33 // overall best equal function

func examineCenter(start, stop int, a, b *[]byte, wg *sync.WaitGroup, differ *bool) {
	if start == stop {
		wg.Done()
		return
	}
	m := start + (stop-start)/2
	//fmt.Printf("range %d to %d, center %d\n", start, stop, m)
	if (*a)[m] != (*b)[m] {
		*differ = true
		wg.Done()
		return
	}
	wg.Add(2)
	go examineCenter(start, m, a, b, wg, differ)
	go examineCenter(m, stop, a, b, wg, differ)
	wg.Done()
}

func equal1(a, b []byte) bool {
	return string(a) == string(b)
}

func equal2(a, b []byte) bool {
	la := len(a)
	lb := len(b)
	switch la {
	case 0:
		return lb == 0
	case 1:
		return lb == 1 && a[0] == b[0]
	case 2:
		return lb == 2 && a[0] == b[0] && a[1] == b[1]
	case 3:
		return lb == 3 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2]
	case 4:
		return lb == 4 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3]
	case lb:
		break
	default: // la != lb
		return false
	}
	// The length is 5 or above, start at index 4
	for i := 4; i < la; i++ {
		if i >= lb {
			return false
		} else if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equal3(a, b []byte) bool {
	la := len(a)
	lb := len(b)
	switch la {
	case 0:
		return lb == 0
	case 1:
		return lb == 1 && a[0] == b[0]
	case 2:
		return lb == 2 && a[0] == b[0] && a[1] == b[1]
	case 3:
		return lb == 3 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2]
	case 4:
		return lb == 4 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3]
	case 5:
		return lb == 5 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3] && a[4] == b[4]
	case lb:
		break
	default: // la != lb
		return false
	}
	// The length is 6 or above, so start at index 5
	// First check the exponential locations, from 5
	for x := 5; x < la; x *= 2 {
		if x >= lb || a[x] != b[x] {
			return false
		}
	}
	// Index 6 is now the first unchecked position
	for i := 6; i < la; i++ {
		if i >= lb || a[i] != b[i] {
			return false
		}
	}
	return true
}

func equal4(a, b []byte) bool {
	la := len(a)
	lb := len(b)
	switch la {
	case 0:
		return lb == 0
	case 1:
		return lb == 1 && a[0] == b[0]
	case 2:
		return lb == 2 && a[0] == b[0] && a[1] == b[1]
	case 3:
		return lb == 3 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2]
	case 4:
		return lb == 4 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3]
	case 5:
		return lb == 5 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3] && a[4] == b[4]
	case lb:
		break
	default: // la != lb
		return false
	}
	return string(a) == string(b)
}

func equal5(a, b []byte) bool {
	la := len(a)
	lb := len(b)
	switch la {
	case 0:
		return lb == 0
	case 1:
		return lb == 1 && a[0] == b[0]
	case 2:
		return lb == 2 && a[0] == b[0] && a[1] == b[1]
	case 3:
		return lb == 3 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2]
	case 4:
		return lb == 4 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3]
	case 5:
		return lb == 5 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3] && a[4] == b[4]
	case 6:
		return lb == 6 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3] && a[4] == b[4] && a[5] == b[5]
	case lb:
		break
	default: // la != lb
		return false
	}
	return string(a) == string(b)
}

func equal6(a, b []byte) bool {
	la := len(a)
	lb := len(b)
	switch la {
	case 0:
		return lb == 0
	case 1:
		return lb == 1 && a[0] == b[0]
	case 2:
		return lb == 2 && a[0] == b[0] && a[1] == b[1]
	case 3:
		return lb == 3 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2]
	case 4:
		return lb == 4 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3]
	case lb:
		return !(string(a) != string(b))
	default: // la != lb
		return false
	}
}

func equal7(a, b []byte) bool {
	la := len(a)
	lb := len(b)
	switch la {
	case 5:
		return lb == 4 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3] && a[4] == b[4]
	case 4:
		return lb == 4 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3]

	case 3:
		return lb == 3 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2]
	case 2:
		return lb == 2 && a[0] == b[0] && a[1] == b[1]
	case 1:
		return lb == 1 && a[0] == b[0]
	case 0:
		return lb == 0
	case lb:
		return !(string(a) != string(b))
	default: // la != lb
		return false
	}
}

func equal8(a, b []byte) bool {
	la := len(a)
	lb := len(b)
	switch la {
	case 4:
		return lb == 4 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3]
	case 3:
		return lb == 3 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2]
	case 2:
		return lb == 2 && a[0] == b[0] && a[1] == b[1]
	case 1:
		return lb == 1 && a[0] == b[0]
	case 0:
		return lb == 0
	case lb:
		return !(string(a) != string(b))
	default: // la != lb
		return false
	}
}

func equal9(a, b []byte) bool {
	la := len(a)
	lb := len(b)
	switch la {
	case 3:
		return lb == 3 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2]
	case 2:
		return lb == 2 && a[0] == b[0] && a[1] == b[1]
	case 1:
		return lb == 1 && a[0] == b[0]
	case 0:
		return lb == 0
	case lb:
		return !(string(a) != string(b))
	default: // la != lb
		return false
	}
}

func equal10(a, b []byte) bool {
	la := len(a)
	lb := len(b)
	switch la {
	case 2:
		return lb == 2 && a[0] == b[0] && a[1] == b[1]
	case 1:
		return lb == 1 && a[0] == b[0]
	case 0:
		return lb == 0
	case lb:
		return !(string(a) != string(b))
	default:
		return false
	}
}

func equal11(a, b []byte) bool {
	la := len(a)
	lb := len(b)
	switch la {
	case 2:
		return lb == 2 && a[0] == b[0] && a[1] == b[1]
	case 1:
		return lb == 1 && a[0] == b[0]
	case 0:
		return lb == 0
	case lb:
		return !(string(a[2:]) != string(b[2:]))
	default: // la != lb
		return false
	}
}

func equal12(a, b []byte) bool {
	la := len(a)
	lb := len(b)
	switch la {
	case 1:
		return lb == 1 && a[0] == b[0]
	case 0:
		return lb == 0
	case lb:
		return !(string(a) != string(b))
	default: // la != lb
		return false
	}
}

func equal13(a, b []byte) bool {
	la := len(a)
	lb := len(b)
	switch la {
	case 0:
		return lb == 0
	case lb:
		return !(string(a) != string(b))
	default: // la != lb
		return false
	}
}

func equal14(a, b []byte) bool {
	la := len(a)
	lb := len(b)
	switch la {
	case 0:
		return lb == 0
	case 1:
		return lb == 1 && a[0] == b[0]
	case 2:
		return lb == 2 && a[0] == b[0] && a[1] == b[1]
	case 3:
		return lb == 3 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2]
	case lb:
		return !(string(a) != string(b))
	default: // la != lb
		return false
	}
}

func equal15(a, b []byte) bool {
	la := len(a)
	lb := len(b)
	switch la {
	case 0:
		return lb == 0
	case 1:
		return lb == 1 && a[0] == b[0]
	case 2:
		return lb == 2 && a[0] == b[0] && a[1] == b[1]
	case 3:
		return lb == 3 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2]
	case 4:
		return lb == 3 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3]
	case 5:
		return lb == 3 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3] && a[4] == b[4]
	case lb:
		return !(string(a) != string(b))
	default: // la != lb
		return false
	}
}

func equal16(a, b []byte) bool {
	la := len(a)
	if la < 5 {
		lb := len(b)
		switch la {
		case 0:
			return lb == 0
		case 1:
			return lb == 1 && a[0] == b[0]
		case 2:
			return lb == 2 && a[0] == b[0] && a[1] == b[1]
		case 3:
			return lb == 3 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2]
		case 4:
			return lb == 3 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3]
		}
	}
	return string(a) == string(b)
}

func equal17(a, b []byte) bool {
	la := len(a)
	if la < 9 {
		lb := len(b)
		switch la {
		case 1:
			return lb == 1 && a[0] == b[0]
		case 2:
			return lb == 2 && a[0] == b[0] && a[1] == b[1]
		case 3:
			return lb == 3 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2]
		case 4:
			return lb == 3 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3]
		case 0:
			return lb == 0
		}
	}
	return string(a) == string(b)
}

func equal18(a, b []byte) bool {
	la := len(a)
	if la < 9 {
		lb := len(b)
		switch la {
		case 4:
			return lb == 3 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3]
		case 3:
			return lb == 3 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2]
		case 2:
			return lb == 2 && a[0] == b[0] && a[1] == b[1]
		case 1:
			return lb == 1 && a[0] == b[0]
		case 0:
			return lb == 0
		}
	}
	return string(a) == string(b)
}

func equal19(a, b []byte) bool {
	la := len(a)
	if la < 5 {
		lb := len(b)
		switch la {
		case 4:
			return lb == 3 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3]
		case 3:
			return lb == 3 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2]
		case 2:
			return lb == 2 && a[0] == b[0] && a[1] == b[1]
		case 1:
			return lb == 1 && a[0] == b[0]
		case 0:
			return lb == 0
		}
	}
	return string(a) == string(b)
}

func equal20(a, b []byte) bool {
	switch len(a) {
	case 0:
		return len(b) == 0
	case 1:
		return len(b) == 1 && a[0] == b[0]
	case 2:
		return len(b) == 2 && a[0] == b[0] && a[1] == b[1]
	}
	return string(a) == string(b)
}

func equal21(a, b []byte) bool {
	switch len(a) {
	case 0:
		return len(b) == 0
	case 1:
		return len(b) == 1 && a[0] == b[0]
	case 2:
		return len(b) == 2 && a[0] == b[0] && a[1] == b[1]
	case 3:
		return len(b) == 2 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2]
	}
	return string(a) == string(b)
}

func equal22(a, b []byte) bool {
	switch len(a) {
	case 0:
		return len(b) == 0
	case 1:
		return len(b) == 1 && a[0] == b[0]
	case 2:
		return len(b) == 2 && a[0] == b[0] && a[1] == b[1]
	case 3:
		return len(b) == 2 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2]
	case 4:
		return len(b) == 2 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3]
	}
	return string(a) == string(b)
}

func equal23(a, b []byte) bool {
	switch len(a) {
	case 0:
		return len(b) == 0
	case 1:
		return len(b) == 1 && a[0] == b[0]
	case 2:
		return len(b) == 2 && a[0] == b[0] && a[1] == b[1]
	case 3:
		return len(b) == 3 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2]
	case 4:
		return len(b) == 4 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3]
	case 5:
		return len(b) == 5 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3] && a[4] == b[4]
	}
	return string(a) == string(b)
}

func equal24(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	switch len(a) {
	case 0:
		return true
	case 1:
		return a[0] == b[0]
	case 2:
		return a[0] == b[0] && a[1] == b[1]
	case 3:
		return a[0] == b[0] && a[1] == b[1] && a[2] == b[2]
	case 4:
		return a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3]
	case 5:
		return a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3] && a[4] == b[4]
	}
	return string(a) == string(b)
}

func equal25(a, b []byte) bool {
	l := len(a)
	if l != len(b) {
		return false
	}
	switch l {
	case 0:
		return true
	case 1:
		return a[0] == b[0]
	case 2:
		return a[0] == b[0] && a[1] == b[1]
	case 3:
		return a[0] == b[0] && a[1] == b[1] && a[2] == b[2]
	case 4:
		return a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3]
	case 5:
		return a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3] && a[4] == b[4]
	case 6:
		return a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3] && a[4] == b[4] && a[5] == b[5]
	}
	return string(a) == string(b)
}

func equal26(a, b []byte) bool {
	la := len(a)
	lb := len(b)
	switch lb {
	case 0:
		return la == 0
	case 1:
		return la == 1 && a[0] == b[0]
	case 2:
		return la == 2 && a[1] == b[1] && a[0] == b[0]
	case 3:
		return la == 3 && a[2] == b[2] && a[1] == b[1] && a[0] == b[0]
	case 4:
		return la == 4 && a[3] == b[3] && a[2] == b[2] && a[1] == b[1] && a[0] == b[0]
	case 5:
		return la == 5 && a[4] == b[4] && a[3] == b[3] && a[2] == b[2] && a[1] == b[1] && a[0] == b[0]
	case 6:
		return la == 6 && a[5] == b[5] && a[4] == b[4] && a[3] == b[3] && a[2] == b[2] && a[1] == b[1] && a[0] == b[0]
	case 7:
		return la == 7 && a[6] == b[6] && a[5] == b[5] && a[4] == b[4] && a[3] == b[3] && a[2] == b[2] && a[1] == b[1] && a[0] == b[0]
	case 8:
		return la == 8 && a[7] == b[7] && a[6] == b[6] && a[5] == b[5] && a[4] == b[4] && a[3] == b[3] && a[2] == b[2] && a[1] == b[1] && a[0] == b[0]
	case lb:
		break
	default: // la != lb
		return false
	}
	return string(b) == string(a)
}

func equal27(a, b []byte) bool {
	la := len(a)
	lb := len(b)
	switch la {
	case 0:
		return lb == 0
	case 1:
		return lb == 1 && a[0] == b[0]
	case 2:
		return lb == 2 && a[0] == b[0] && a[1] == b[1]
	case 3:
		return lb == 3 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2]
	case 4:
		return lb == 4 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3]
	case lb:
		break
	default: // la != lb
		return false
	}
	// The length is 5 or above, start at index 4
	for i := 4; i < la; i++ {
		if i >= lb {
			return false
		} else if a[i] != b[i] {
			return false
		} else if i >= 16 {
			return string(b) == string(a)
		}
	}
	return true
}

func equal28(a, b []byte) bool {
	la := len(a)
	lb := len(b)
	switch la {
	case 0:
		return lb == 0
	case 1:
		return lb == 1 && a[0] == b[0]
	case 2:
		return lb == 2 && a[0] == b[0] && a[1] == b[1]
	case 3:
		return lb == 3 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2]
	case 4:
		return lb == 4 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3]
	case lb:
		break
	default: // la != lb
		return false
	}
	// The length is 5 or above, start at index 4
	return string(b[4:]) == string(a[4:])
}

func equal29(a, b []byte) bool {
	la := len(a)
	lb := len(b)
	switch la {
	case 0:
		return lb == 0
	case 1:
		return lb == 1 && a[0] == b[0]
	case 2:
		return lb == 2 && a[0] == b[0] && a[1] == b[1]
	case 3:
		return lb == 3 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2]
	case 4:
		return lb == 4 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3]
	case 5:
		return lb == 5 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3] && a[4] == b[4]
	case 6:
		return lb == 6 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3] && a[4] == b[4] && a[5] == b[5]
	case 7:
		return lb == 7 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3] && a[4] == b[4] && a[5] == b[5] && a[6] == b[6]
	case lb:
		break
	default: // la != lb
		return false
	}
	// The length is 8 or above, start at index 7
	return string(b[7:]) == string(a[7:])
}

func equal30(a, b []byte) bool {
	la := len(a)
	lb := len(b)
	if la != lb {
		return false
	}
	if la == 0 { // && lb == 0
		return true
	}
	if la <= 4 {
		switch la {
		case 1:
			return lb == 1 && a[0] == b[0]
		case 2:
			return lb == 2 && a[0] == b[0] && a[1] == b[1]
		case 3:
			return lb == 3 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2]
		case 4:
			return lb == 3 && a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3]
		}
	}
	return string(a[4:]) == string(b[4:])
}

func equal31(a, b []byte) bool {
	l := len(a)
	if l != len(b) {
		return false
	}
	switch l {
	case 0:
		return true
	case 1:
		return a[0] == b[0]
	case 2:
		return a[0] == b[0] && a[1] == b[1]
	case 3:
		return a[0] == b[0] && a[1] == b[1] && a[2] == b[2]
	case 4:
		return a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3]
	case 5:
		return a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3] && a[4] == b[4]
	case 6:
		return a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3] && a[4] == b[4] && a[5] == b[5]
	}
	return string(a[6:]) == string(b[6:])
}

func equal32(a, b []byte) bool {
	l := len(a)
	if l != len(b) {
		return false
	}
	switch l {
	case 0:
		return true
	case 1:
		return a[0] == b[0]
	case 2:
		return a[0] == b[0] && a[1] == b[1]
	case 3:
		return a[0] == b[0] && a[1] == b[1] && a[2] == b[2]
	case 4:
		return a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3]
	case 5:
		return a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3] && a[4] == b[4]
	case 6:
		return a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3] && a[4] == b[4] && a[5] == b[5]
	case 7:
		return a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3] && a[4] == b[4] && a[5] == b[5] && a[6] == b[6]
	}
	return string(a[7:]) == string(b[7:])
}

func equal33(a, b []byte) bool {
	switch len(a) {
	case 0:
		return len(b) == 0
	case len(b):
		return !(string(a) != string(b))
	default:
		return false
	}
}

func equal34(a, b []byte) bool {
	switch len(a) {
	case 0:
		return len(b) == 0
	case len(b):
		return !(string(a[1:]) != string(b[1:]))
	default:
		return false
	}
}
