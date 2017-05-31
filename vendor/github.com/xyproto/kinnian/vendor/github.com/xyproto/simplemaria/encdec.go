package simplemaria

import (
	"bytes"
	"compress/flate"
	"encoding/hex"
	"io/ioutil"
)

// MariaDB/MySQL does not handle some characters well.
// Compressing and hex encoding the value is one of many possible ways
// to avoid this. Using BLOB fields and different datatypes is another.
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

// Dehex and decompress the given string
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
