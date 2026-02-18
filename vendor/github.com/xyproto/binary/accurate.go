package binary

import (
	"bytes"
	"io"
	"os"
	"unicode/utf8"
)

// magicSignature represents a file format signature
type magicSignature struct {
	signature []byte
	name      string
}

// magicSignatures contains magic number signatures for common binary file formats
var magicSignatures = []magicSignature{
	// Images
	{[]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, "PNG"},
	{[]byte{0xFF, 0xD8, 0xFF}, "JPEG"},
	{[]byte{0x47, 0x49, 0x46, 0x38, 0x37, 0x61}, "GIF87a"},
	{[]byte{0x47, 0x49, 0x46, 0x38, 0x39, 0x61}, "GIF89a"},
	{[]byte{0x42, 0x4D}, "BMP"},
	{[]byte{0x00, 0x00, 0x01, 0x00}, "ICO"},
	{[]byte{0x00, 0x00, 0x02, 0x00}, "CUR"},
	{[]byte{0x49, 0x49, 0x2A, 0x00}, "TIFF little-endian"},
	{[]byte{0x4D, 0x4D, 0x00, 0x2A}, "TIFF big-endian"},
	{[]byte{0x52, 0x49, 0x46, 0x46}, "RIFF (WebP, AVI, WAV)"},

	// Executables and libraries
	{[]byte{0x7F, 0x45, 0x4C, 0x46}, "ELF"},
	{[]byte{0x4D, 0x5A}, "DOS/Windows executable"},
	{[]byte{0xCF, 0xFA, 0xED, 0xFE}, "Mach-O 32-bit"},
	{[]byte{0xCE, 0xFA, 0xED, 0xFE}, "Mach-O 32-bit reverse"},
	{[]byte{0xFE, 0xED, 0xFA, 0xCF}, "Mach-O 64-bit"},
	{[]byte{0xFE, 0xED, 0xFA, 0xCE}, "Mach-O 64-bit reverse"},
	{[]byte{0xCA, 0xFE, 0xBA, 0xBE}, "Mach-O fat binary / Java class"},
	{[]byte{0xBE, 0xBA, 0xFE, 0xCA}, "Mach-O fat binary reverse"},

	// Archives and compressed files
	{[]byte{0x50, 0x4B, 0x03, 0x04}, "ZIP/JAR/APK/DOCX"},
	{[]byte{0x50, 0x4B, 0x05, 0x06}, "ZIP empty"},
	{[]byte{0x50, 0x4B, 0x07, 0x08}, "ZIP spanned"},
	{[]byte{0x1F, 0x8B}, "GZIP"},
	{[]byte{0x42, 0x5A, 0x68}, "BZIP2"},
	{[]byte{0xFD, 0x37, 0x7A, 0x58, 0x5A, 0x00}, "XZ"},
	{[]byte{0x5D, 0x00, 0x00}, "LZMA"},
	{[]byte{0x28, 0xB5, 0x2F, 0xFD}, "Zstandard"},
	{[]byte{0x04, 0x22, 0x4D, 0x18}, "LZ4"},
	{[]byte{0x52, 0x61, 0x72, 0x21, 0x1A, 0x07}, "RAR"},
	{[]byte{0x37, 0x7A, 0xBC, 0xAF, 0x27, 0x1C}, "7z"},

	// Documents
	{[]byte{0x25, 0x50, 0x44, 0x46}, "PDF"},
	{[]byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}, "MS Office OLE"},

	// Audio/Video
	{[]byte{0x49, 0x44, 0x33}, "MP3 ID3"},
	{[]byte{0xFF, 0xFB}, "MP3"},
	{[]byte{0xFF, 0xFA}, "MP3"},
	{[]byte{0xFF, 0xF3}, "MP3"},
	{[]byte{0xFF, 0xF2}, "MP3"},
	{[]byte{0x4F, 0x67, 0x67, 0x53}, "OGG"},
	{[]byte{0x66, 0x4C, 0x61, 0x43}, "FLAC"},
	{[]byte{0x00, 0x00, 0x00}, "MP4/MOV partial"},

	// Databases
	{[]byte{0x53, 0x51, 0x4C, 0x69, 0x74, 0x65, 0x20, 0x66, 0x6F, 0x72, 0x6D, 0x61, 0x74, 0x20, 0x33, 0x00}, "SQLite"},

	// Fonts
	{[]byte{0x00, 0x01, 0x00, 0x00}, "TrueType font"},
	{[]byte{0x4F, 0x54, 0x54, 0x4F}, "OpenType font"},
	{[]byte{0x77, 0x4F, 0x46, 0x46}, "WOFF"},
	{[]byte{0x77, 0x4F, 0x46, 0x32}, "WOFF2"},

	// Other
	{[]byte{0x1A, 0x45, 0xDF, 0xA3}, "WebM/MKV"},
	{[]byte{0x00, 0x61, 0x73, 0x6D}, "WebAssembly"},
}

// hasMagicSignature checks if the data starts with a known binary file signature
func hasMagicSignature(data []byte) bool {
	for _, sig := range magicSignatures {
		if bytes.HasPrefix(data, sig.signature) {
			return true
		}
	}
	return false
}

// isBinaryDataAccurate uses thorough heuristics to determine if data is binary
func isBinaryDataAccurate(data []byte) bool {
	if len(data) == 0 {
		return false
	}

	// Check for known binary file signatures
	if hasMagicSignature(data) {
		return true
	}

	// Count various byte types
	var nullCount, controlCount, highBitCount int
	for _, b := range data {
		if b == 0 {
			nullCount++
		} else if b < 32 && b != 9 && b != 10 && b != 13 { // excluding tab, LF, CR
			controlCount++
		}
		if b > 127 {
			highBitCount++
		}
	}

	// Any null bytes strongly suggest binary (text files rarely have nulls)
	if nullCount > 0 {
		// Allow a small number of nulls in larger files (could be padding)
		if len(data) < 512 || float64(nullCount)/float64(len(data)) > 0.001 {
			return true
		}
	}

	// High proportion of control characters suggests binary
	if controlCount > len(data)/20 { // more than 5% control chars
		return true
	}

	// If there are high-bit bytes, check if they form valid UTF-8
	if highBitCount > 0 {
		if !utf8.Valid(data) && !isValidUTF16(data) {
			return true
		}
	}

	return false
}

// isBinaryDataAccurateAndUTF16 is like isBinaryDataAccurate but also returns
// whether the data appears to be UTF-16 encoded
func isBinaryDataAccurateAndUTF16(data []byte) (bool, bool) {
	if len(data) == 0 {
		return false, false
	}

	isUtf16 := isValidUTF16(data)

	// Check for known binary file signatures
	if hasMagicSignature(data) {
		return true, isUtf16
	}

	// Count various byte types
	var nullCount, controlCount, highBitCount int
	for _, b := range data {
		if b == 0 {
			nullCount++
		} else if b < 32 && b != 9 && b != 10 && b != 13 {
			controlCount++
		}
		if b > 127 {
			highBitCount++
		}
	}

	// Null bytes handling - UTF-16 has many nulls, so be careful
	if nullCount > 0 && !isUtf16 {
		if len(data) < 512 || float64(nullCount)/float64(len(data)) > 0.001 {
			return true, isUtf16
		}
	}

	// High proportion of control characters suggests binary
	if controlCount > len(data)/20 {
		return true, isUtf16
	}

	// If there are high-bit bytes, check if they form valid UTF-8 or UTF-16
	if highBitCount > 0 && !isUtf16 && !utf8.Valid(data) {
		return true, isUtf16
	}

	return false, isUtf16
}

// DataAccurate determines if the given data is binary using thorough analysis.
// Unlike Data, this reads more of the input and uses magic number detection.
func DataAccurate(data []byte) bool {
	l := len(data)
	if l == 0 {
		return false
	}

	// For small data, analyze all of it
	if l <= 512 {
		return isBinaryDataAccurate(data)
	}

	// Check the first 512 bytes (contains magic numbers and headers)
	if isBinaryDataAccurate(data[:512]) {
		return true
	}

	// Check middle
	middle := l / 2
	chunkSize := 256
	if middle+chunkSize <= l {
		if isBinaryDataAccurate(data[middle : middle+chunkSize]) {
			return true
		}
	}

	// Check near end
	if l > chunkSize {
		if isBinaryDataAccurate(data[l-chunkSize:]) {
			return true
		}
	}

	return false
}

// DataAccurateAndUTF16 determines if the given data is binary using thorough analysis.
// Also returns whether the data appears to be UTF-16 encoded.
func DataAccurateAndUTF16(data []byte) (bool, bool) {
	l := len(data)
	if l == 0 {
		return false, false
	}

	// For small data, analyze all of it
	if l <= 512 {
		return isBinaryDataAccurateAndUTF16(data)
	}

	// Check the first 512 bytes
	isBinary, isUtf16 := isBinaryDataAccurateAndUTF16(data[:512])
	if isBinary {
		return true, isUtf16
	}

	// Check middle
	middle := l / 2
	chunkSize := 256
	if middle+chunkSize <= l {
		isBinary, isUtf16Middle := isBinaryDataAccurateAndUTF16(data[middle : middle+chunkSize])
		isUtf16 = isUtf16 || isUtf16Middle
		if isBinary {
			return true, isUtf16
		}
	}

	// Check near end
	if l > chunkSize {
		isBinary, isUtf16End := isBinaryDataAccurateAndUTF16(data[l-chunkSize:])
		isUtf16 = isUtf16 || isUtf16End
		if isBinary {
			return true, isUtf16
		}
	}

	return false, isUtf16
}

// FileAccurate determines if the given file is binary using thorough analysis.
// Unlike File, this reads more of the file and uses magic number detection.
func FileAccurate(filename string) (bool, error) {
	isBinary, _, err := FileAccurateAndUTF16(filename)
	return isBinary, err
}

// FileAccurateAndUTF16 determines if the given file is binary using thorough analysis.
// Also returns whether the file appears to be UTF-16 encoded.
func FileAccurateAndUTF16(filename string) (bool, bool, error) {
	isUtf16 := false
	file, err := os.Open(filename)
	if err != nil {
		return false, isUtf16, err
	}
	defer file.Close()

	// Get file size
	stat, err := file.Stat()
	if err != nil {
		return false, isUtf16, err
	}
	fileSize := stat.Size()

	// Empty files are considered text
	if fileSize == 0 {
		return false, isUtf16, nil
	}

	// Read the first 512 bytes for magic number detection
	headerSize := int64(512)
	if fileSize < headerSize {
		headerSize = fileSize
	}

	header := make([]byte, headerSize)
	n, err := file.Read(header)
	if err != nil && err != io.EOF {
		return false, isUtf16, err
	}
	header = header[:n]

	var isBinary bool
	isBinary, isUtf16 = isBinaryDataAccurateAndUTF16(header)
	if isBinary {
		return true, isUtf16, nil
	}

	// For small files, we've already read everything
	if fileSize <= 512 {
		return false, isUtf16, nil
	}

	// Read a chunk from the middle
	middlePos := fileSize / 2
	chunkSize := int64(256)
	if middlePos+chunkSize > fileSize {
		chunkSize = fileSize - middlePos
	}

	if _, err := file.Seek(middlePos, io.SeekStart); err == nil {
		middleChunk := make([]byte, chunkSize)
		if n, err := file.Read(middleChunk); err == nil && n > 0 {
			middleChunk = middleChunk[:n]
			isBinaryMiddle, isUtf16Middle := isBinaryDataAccurateAndUTF16(middleChunk)
			isUtf16 = isUtf16 || isUtf16Middle
			if isBinaryMiddle {
				return true, isUtf16, nil
			}
		}
	}

	// Read the last 256 bytes
	endPos := fileSize - 256
	if endPos < 0 {
		endPos = 0
	}
	if endPos > 512 { // Don't re-read what we already checked
		if _, err := file.Seek(endPos, io.SeekStart); err == nil {
			endChunk := make([]byte, 256)
			if n, err := file.Read(endChunk); err == nil && n > 0 {
				endChunk = endChunk[:n]
				isBinaryEnd, isUtf16End := isBinaryDataAccurateAndUTF16(endChunk)
				isUtf16 = isUtf16 || isUtf16End
				if isBinaryEnd {
					return true, isUtf16, nil
				}
			}
		}
	}

	return false, isUtf16, nil
}
