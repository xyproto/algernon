package binary

import (
	"bytes"
	"os"
	"unicode/utf8"
)

// probablyBinaryData is designed to examine a byte slice that is 24 bytes long, or less.
// If there are more than 1/3 null bytes, it's considered to be binary.
// If the bytes can be converted to an utf8 string, it's considered to be text.
func probablyBinaryData(b []byte) bool {
	zeroCount := bytes.Count(b, []byte{0})
	if zeroCount > len(b)/3 {
		return true
	}
	return !utf8.ValidString(string(b))
}

// File tries to determine if the given filename is a binary file by reading the first, last
// and middle 24 bytes, then using the probablyBinaryData function on each of them in turn.
func File(filename string) (bool, error) {
	file, err := os.Open(filename)
	if err != nil {
		return false, err
	}
	defer file.Close()

	// Go to the end of the file, minus 24 bytes
	fileLength, err := file.Seek(-24, os.SEEK_END)
	if err != nil || fileLength < 24 {

		// Go to the start of the file, ignore errors
		_, err = file.Seek(0, os.SEEK_SET)

		// Read up to 24 bytes
		fileBytes := make([]byte, 24)
		n, err := file.Read(fileBytes)
		if err != nil {
			// Could not read the file
			return false, err
		} else if n == 0 {
			// The file is too short, decide it's a text file
			return false, nil
		}
		// Shorten the byte slice in case less than 24 bytes were read
		fileBytes = fileBytes[:n]

		// Check if it's likely to be a binary file, based on the few available bytes
		return probablyBinaryData(fileBytes), nil
	}

	last24 := make([]byte, 24)

	// Read 24 bytes
	last24count, err := file.Read(last24)
	if err != nil {
		return false, err
	}

	if last24count > 0 && probablyBinaryData(last24) {
		return true, nil
	}

	if fileLength-24 >= 24 {
		first24 := make([]byte, 24)
		first24count := 0

		// Go to the start of the file
		if _, err := file.Seek(0, os.SEEK_SET); err != nil {
			// Could not go to the start of the file (!)
			return false, err
		}

		// Read 24 bytes
		first24count, err = file.Read(first24)
		if err != nil {
			return false, err
		}

		if first24count > 0 && probablyBinaryData(first24) {
			return true, nil
		}
	}

	if fileLength-24 >= 48 {

		middle24 := make([]byte, 24)
		middle24count := 0

		middlePos := fileLength / 2

		// Go to the middle of the file, relative to the start. Ignore errors.
		_, _ = file.Seek(middlePos, os.SEEK_SET)

		// Read 24 bytes from where MIGHT be the middle of the file
		middle24count, err = file.Read(middle24)
		if err != nil {
			return false, err
		}

		return middle24count > 0 && probablyBinaryData(middle24), nil
	}

	// If it was a binary file, it should have been catched by one of the returns above
	return false, nil
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
