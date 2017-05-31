package simplehstore

import (
	"bytes"
	"compress/flate"
	"encoding/hex"
	"io/ioutil"
)

// Encode compresses and encodes a given string in order to safely handle *any* UTF-8 characters.
func Encode(value *string) error {
	// Don't encode empty strings
	if *value == "" {
		return nil
	}
	var buf bytes.Buffer
	compressorWriter, err := flate.NewWriter(&buf, 1) // compression level 1 (fastest)
	if err != nil {
		return err
	}
	compressorWriter.Write([]byte(*value))
	compressorWriter.Close()
	*value = hex.EncodeToString(buf.Bytes())
	return nil
}

// Decode decompresses and decodes an encoded string to an UTF-8 string.
func Decode(code *string) error {
	// Don't decode empty strings
	if *code == "" {
		return nil
	}
	unhexedBytes, err := hex.DecodeString(*code)
	if err != nil {
		return err
	}
	buf := bytes.NewBuffer(unhexedBytes)
	decompressorReader := flate.NewReader(buf)
	decompressedBytes, err := ioutil.ReadAll(decompressorReader)
	decompressorReader.Close()
	if err != nil {
		return err
	}
	*code = string(decompressedBytes)
	return nil
}
