package parser

import "bytes"

var URIs = [][]byte{
	[]byte("http://"),
	[]byte("https://"),
	[]byte("ftp://"),
	[]byte("mailto:"),
}

var Paths = [][]byte{
	[]byte("/"),
	[]byte("./"),
	[]byte("../"),
}

// IsSafeURL returns true if url starts with one of the valid schemes or is a relative path.
func IsSafeURL(url []byte) bool {
	for _, path := range Paths {
		if bytes.HasPrefix(url, path) &&
			(len(url) == len(path) || IsAlnum(url[len(path)])) {
			return true
		}
	}

	for _, prefix := range URIs {
		if len(url) > len(prefix) && hasPrefixCaseInsensitive(url, prefix) && IsAlnum(url[len(prefix)]) {
			return true
		}
	}

	return false
}
