package main

import (
	"bytes"
	"github.com/klauspost/pgzip"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
)

var preferSpeed = true // prefer speed over best compression ratio?

// DataBlock represents a block of data that may be compressed
type DataBlock struct {
	data       []byte
	compressed bool
	length     int
}

var (
	// EmptyDataBlock is an empty data block
	EmptyDataBlock = &DataBlock{[]byte{}, false, 0}
)

// Create a new uncompressed data block.
func NewDataBlock(data []byte) *DataBlock {
	return &DataBlock{data, false, len(data)}
}

// Create a new data block where the data may already be compressed.
func newDataBlockSpecified(data []byte, compressed bool) *DataBlock {
	return &DataBlock{data, compressed, len(data)}
}

// Return the original, uncompressed data, length and an error.
// Will decompress if needed.
func (b *DataBlock) UncompressedData() ([]byte, int, error) {
	if b.compressed {
		return decompress(b.data)
	}
	return b.data, b.length, nil
}

// Return the uncompressed data or an empty byte slice
func (b *DataBlock) MustData() []byte {
	if b.compressed {
		data, _, err := decompress(b.data)
		if err != nil {
			log.Fatal(err)
			return []byte{}
		}
		return data
	}
	return b.data
}

// Return the compressed data, length and an error. Will compress if needed.
// If speed is true, speed is preferred over the compression ratio.
func (b *DataBlock) Gzipped() ([]byte, int, error) {
	if !b.compressed {
		return compress(b.data, preferSpeed)
	}
	return b.data, b.length, nil
}

// Compress this data block.
// If speed is set, speed is favored over compression ratio.
func (b *DataBlock) Compress() error {
	if b.compressed {
		return nil
	}
	data, bytesWritten, err := compress(b.data, preferSpeed)
	if err != nil {
		return err
	}
	b.data = data
	b.compressed = true
	b.length = bytesWritten
	return nil
}

// Decompress this data block.
func (b *DataBlock) Decompress() error {
	if !b.compressed {
		return nil
	}
	data, bytesWritten, err := decompress(b.data)
	if err != nil {
		return err
	}
	b.data = data
	b.compressed = false
	b.length = bytesWritten
	return nil
}

// Check if the data block is compressed
func (b *DataBlock) IsCompressed() bool {
	return b.compressed
}

// Return the length of the data, as a string
func (b *DataBlock) StringLength() string {
	return strconv.Itoa(b.length)
}

// Return the size of this data block
func (b *DataBlock) Length() int {
	return b.length
}

// Check if there is data present
func (b *DataBlock) HasData() bool {
	return 0 != b.length
}

// Wrap an error message as a data block.
// Can be used when reporting errors as a web page.
func errorToDataBlock(err error) *DataBlock {
	return NewDataBlock([]byte(err.Error()))
}

// Write the data to the client. Gzip if suitable.
// gzipped must be set if the given data is already compressed.
// "speed" should be set to true if speed is prefered over compression ratio.
func (b *DataBlock) ToClient(w http.ResponseWriter, req *http.Request) {
	canGzip := clientCanGzip(req)

	// Compress or decompress the data as needed. Add headers if compression is used.
	if !canGzip {
		// No compression
		if err := b.Decompress(); err != nil {
			// Unable to decompress gzipped data!
			log.Fatal(err)
		}
	} else if b.compressed { // If the given data is already compressed, serve it as such
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Add("Vary", "Accept-Encoding")
	} else if b.Length() > gzipThreshold { // If the data is over a certain size, compress and serve
		// Set gzip headers
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Add("Vary", "Accept-Encoding")
		// Compress
		if err := b.Compress(); err != nil {
			// Write uncompressed data if gzip should fail
			log.Error(err)
			w.Header().Set("Content-Encoding", "identity")
		}
	}

	// Set the length and write the data to the client
	w.Header().Set("Content-Length", b.StringLength())
	w.Write(b.data)
}

// Compress data using pgzip. Returns the data, bytes written and an error.
func compress(data []byte, speed bool) ([]byte, int, error) {
	if len(data) == 0 {
		return []byte{}, 0, nil
	}
	var buf bytes.Buffer
	_, err := gzipWrite(&buf, data, speed)
	if err != nil {
		return nil, 0, err
	}
	data = buf.Bytes()
	return data, len(data), nil
}

// Decompress data using pgzip. Returns the data, bytes written and an error.
func decompress(data []byte) ([]byte, int, error) {
	if len(data) == 0 {
		return []byte{}, 0, nil
	}
	var buf bytes.Buffer
	_, err := gunzipWrite(&buf, data)
	if err != nil {
		return nil, 0, err
	}
	data = buf.Bytes()
	return data, len(data), nil
}

// Write gzipped data to a Writer. Returns bytes written and an error.
func gzipWrite(w io.Writer, data []byte, speed bool) (int, error) {
	// Write gzipped data to the client
	level := pgzip.BestCompression
	if speed {
		level = pgzip.BestSpeed
	}
	gw, err := pgzip.NewWriterLevel(w, level)
	defer gw.Close()
	bytesWritten, err := gw.Write(data)
	if err != nil {
		return 0, err
	}
	return bytesWritten, nil
}

// Write gunzipped data to a Writer. Returns bytes written and an error.
func gunzipWrite(w io.Writer, data []byte) (int, error) {
	// Write gzipped data to the client
	gr, err := pgzip.NewReader(bytes.NewBuffer(data))
	defer gr.Close()
	data, err = ioutil.ReadAll(gr)
	if err != nil {
		return 0, err
	}
	bytesWritten, err := w.Write(data)
	if err != nil {
		return 0, err
	}
	return bytesWritten, nil
}
