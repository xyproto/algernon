package binary

import (
	"encoding/binary"
	"io"
	"os"
	"unicode/utf8"
)

// readUint16 reads a 16-bit value from a byte slice based on endianness
func readUint16(b []byte, bigEndian bool) uint16 {
	if bigEndian {
		return binary.BigEndian.Uint16(b)
	}
	return binary.LittleEndian.Uint16(b)
}

// isValidUTF16 checks if the given byte slice is valid UTF-16.
// It handles both big endian and little endian byte orders.
func isValidUTF16(data []byte) (valid bool) {
	lenData := len(data)
	if lenData < 2 {
		return false
	}
	var bigEndian bool
	switch {
	case data[0] == 0xFE && data[1] == 0xFF:
		bigEndian = true
		data = data[2:] // Remove BOM for big endian
		lenData -= 2
	case data[0] == 0xFF && data[1] == 0xFE:
		bigEndian = false
		data = data[2:] // Remove BOM for little endian
		lenData -= 2
	}
	if lenData%2 != 0 {
		return false // Length of UTF-16 data must be even
	}

	var charValue, nextChar uint16
	for i := 0; i < lenData; i += 2 {
		charValue = readUint16(data[i:], bigEndian)
		switch {
		case charValue >= 0xD800 && charValue <= 0xDBFF:
			// High surrogate
			if i+2 >= lenData {
				return false // High surrogate at end of data is invalid
			}
			nextChar = readUint16(data[i+2:], bigEndian)
			if nextChar < 0xDC00 || nextChar > 0xDFFF {
				return false // Invalid low surrogate following a high surrogate
			}
			i += 2 // Skip the next surrogate pair part
		case charValue >= 0xDC00 && charValue <= 0xDFFF:
			// Low surrogate without preceding high surrogate
			return false
		}
	}
	return true
}

// probablyBinaryDataAndUTF16 uses enhanced heuristics to guess whether data is binary or text.
// Also returns a bool for if it's likely to be UTF-16
func probablyBinaryDataAndUTF16(b []byte) (bool, bool) {
	var zeroCount, controlCharCount, continuousNullCount int
	maxContinuousNull := 0
	distinctBytes := make(map[byte]struct{})

	for _, byteVal := range b {
		distinctBytes[byteVal] = struct{}{}

		if byteVal == 0 {
			zeroCount++
			continuousNullCount++
			if continuousNullCount > maxContinuousNull {
				maxContinuousNull = continuousNullCount
			}
		} else {
			continuousNullCount = 0
		}

		if byteVal < 32 && byteVal != 9 && byteVal != 10 && byteVal != 13 { // excluding tab, LF, CR
			controlCharCount++
		}
	}

	isUtf16 := isValidUTF16(b)

	// High proportion of null bytes or long continuous null byte sequences suggest binary
	if zeroCount > len(b)/2 || maxContinuousNull > 4 {
		return true, isUtf16
	}

	// Valid UTF-8 or UTF-16 data suggests text
	if utf8.Valid(b) || isUtf16 {
		return false, isUtf16
	}

	// High count of control characters suggests binary
	if controlCharCount > len(b)/10 {
		return true, isUtf16
	}

	// Little variety in byte values might suggest binary
	return len(distinctBytes) < len(b)/2, isUtf16
}

// probablyBinaryData uses enhanced heuristics to guess whether data is binary or text.
func probablyBinaryData(b []byte) bool {
	probablyBinary, _ := probablyBinaryDataAndUTF16(b)
	return probablyBinary
}

// FileAndUTF16 tries to determine if the given filename is a binary file by reading the first, last
// and middle 24 bytes, then using the probablyBinaryData function on each of them in turn.
// Also return true if it is probably UTF-16.
func FileAndUTF16(filename string) (bool, bool, error) {
	isUtf16 := false
	file, err := os.Open(filename)
	if err != nil {
		return false, isUtf16, err
	}
	defer file.Close()

	// Go to the end of the file, minus 24 bytes
	fileLength, err := file.Seek(-24, io.SeekEnd)
	if err != nil || fileLength < 24 {

		// Go to the start of the file, ignore errors
		file.Seek(0, io.SeekStart)

		// Read up to 24 bytes
		fileBytes := make([]byte, 24)
		n, err := file.Read(fileBytes)
		if err != nil {
			// Could not read the file
			return false, isUtf16, err
		} else if n == 0 {
			// The file is too short, decide it's a text file
			return false, isUtf16, nil
		}
		// Shorten the byte slice in case less than 24 bytes were read
		fileBytes = fileBytes[:n]

		// Check if it's likely to be a binary file, based on the few available bytes
		var probably bool
		probably, isUtf16 = probablyBinaryDataAndUTF16(fileBytes)
		return probably, isUtf16, nil
	}

	last24 := make([]byte, 24)

	// Read 24 bytes
	last24count, err := file.Read(last24)
	if err != nil {
		return false, isUtf16, err
	}

	if last24count > 0 {
		var probably bool
		probably, isUtf16 = probablyBinaryDataAndUTF16(last24)
		if probably {
			return true, isUtf16, nil
		}
	}

	if fileLength-24 >= 24 {
		first24 := make([]byte, 24)
		first24count := 0

		// Go to the start of the file
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			// Could not go to the start of the file (!)
			return false, isUtf16, err
		}

		// Read 24 bytes
		first24count, err = file.Read(first24)
		if err != nil {
			return false, isUtf16, err
		}

		if first24count > 0 {
			var probably bool
			probably, isUtf16 = probablyBinaryDataAndUTF16(first24)
			if probably {
				return true, isUtf16, nil
			}
		}
	}

	if fileLength-24 >= 48 {

		middle24 := make([]byte, 24)
		middle24count := 0

		middlePos := fileLength / 2

		// Go to the middle of the file, relative to the start. Ignore errors.
		file.Seek(middlePos, io.SeekStart)

		// Read 24 bytes from where MIGHT be the middle of the file
		middle24count, err = file.Read(middle24)
		if err != nil {
			return false, isUtf16, err
		}

		if middle24count > 0 {
			var probably bool
			probably, isUtf16 = probablyBinaryDataAndUTF16(middle24)
			if probably {
				return true, isUtf16, nil
			}
		}
	}

	// If it was a binary file, it should have been catched by one of the returns above
	return false, isUtf16, nil
}

// File tries to determine if the given filename is a binary file by reading the first, last
// and middle 24 bytes, then using the probablyBinaryData function on each of them in turn.
func File(filename string) (bool, error) {
	isBinary, _, err := FileAndUTF16(filename)
	return isBinary, err
}

// DataAndUTF16 tries to determine if the given data is binary by examining the first, last
// and middle 24 bytes, then using the probablyBinaryData function on each of them in turn.
// Also returns a bool if the data appears to be UTF-16.
func DataAndUTF16(data []byte) (bool, bool) {
	l := len(data)
	switch {
	case l == 0:
		return false, false
	case l <= 24:
		return probablyBinaryDataAndUTF16(data)
	case l <= 48:
		isBinary1, isUtf16_1 := probablyBinaryDataAndUTF16(data[:24])
		isBinary2, isUtf16_2 := probablyBinaryDataAndUTF16(data[24:])
		return isBinary1 || isBinary2, isUtf16_1 || isUtf16_2
	case l <= 72:
		isBinary1, isUtf16_1 := probablyBinaryDataAndUTF16(data[:24])
		isBinary2, isUtf16_2 := probablyBinaryDataAndUTF16(data[24:48])
		isBinary3, isUtf16_3 := probablyBinaryDataAndUTF16(data[48:])
		return isBinary1 || isBinary2 || isBinary3, isUtf16_1 || isUtf16_2 || isUtf16_3
	default: // l > 72
		middle := l / 2
		isBinary1, isUtf16_1 := probablyBinaryDataAndUTF16(data[:24])
		isBinary2, isUtf16_2 := probablyBinaryDataAndUTF16(data[middle : middle+24])
		isBinary3, isUtf16_3 := probablyBinaryDataAndUTF16(data[l-24:])
		return isBinary1 || isBinary2 || isBinary3, isUtf16_1 || isUtf16_2 || isUtf16_3
	}
}

// Data tries to determine if the given data is binary by examining the first, last
// and middle 24 bytes, then using the probablyBinaryData function on each of them in turn.
func Data(data []byte) bool {
	l := len(data)
	switch {
	case l == 0:
		return false
	case l <= 24:
		return probablyBinaryData(data)
	case l <= 48:
		return probablyBinaryData(data[:24]) || probablyBinaryData(data[24:])
	case l <= 72:
		return probablyBinaryData(data[:24]) || probablyBinaryData(data[24:48]) || probablyBinaryData(data[48:])
	default: // l > 72
		middle := l / 2
		return probablyBinaryData(data[:24]) || probablyBinaryData(data[middle:middle+24]) || probablyBinaryData(data[l-24:])
	}
}
