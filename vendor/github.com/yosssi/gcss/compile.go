package gcss

import (
	"bytes"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"
)

// extensions
const (
	extCSS  = ".css"
	extGCSS = ".gcss"
)

// cssFilePath converts path's extenstion into a CSS file extension.
var cssFilePath = func(path string) string {
	return convertExt(path, extCSS)
}

// Compile compiles GCSS data which is read from src and
// Writes the result CSS data to the dst.
func Compile(dst io.Writer, src io.Reader) (int, error) {
	data, err := ioutil.ReadAll(src)

	if err != nil {
		return 0, err
	}

	bc, berrc := compileBytes(data)

	bf := new(bytes.Buffer)

BufWriteLoop:
	for {
		select {
		case b, ok := <-bc:
			if !ok {
				break BufWriteLoop
			}

			bf.Write(b)
		case err := <-berrc:
			return 0, err
		}
	}

	return dst.Write(bf.Bytes())
}

// CompileFile parses the GCSS file specified by the path parameter,
// generates a CSS file and returns the path of the generated CSS file
// and an error when it occurs.
func CompileFile(path string) (string, error) {
	data, err := ioutil.ReadFile(path)

	if err != nil {
		return "", err
	}

	cssPath := cssFilePath(path)

	bc, berrc := compileBytes(data)

	done, werrc := write(cssPath, bc, berrc)

	select {
	case <-done:
	case err := <-werrc:
		return "", err
	}

	return cssPath, nil
}

// compileBytes parses the GCSS byte array passed as the s parameter,
// generates a CSS byte array and returns the two channels: the first
// one returns the CSS byte array and the last one returns an error
// when it occurs.
func compileBytes(b []byte) (<-chan []byte, <-chan error) {
	lines := strings.Split(formatLF(string(b)), lf)

	bc := make(chan []byte, len(lines))
	errc := make(chan error)

	go func() {
		ctx := newContext()

		elemc, pErrc := parse(lines)

		for {
			select {
			case elem, ok := <-elemc:
				if !ok {
					close(bc)
					return
				}

				elem.SetContext(ctx)

				switch v := elem.(type) {
				case *mixinDeclaration:
					ctx.mixins[v.name] = v
				case *variable:
					ctx.vars[v.name] = v
				case *atRule, *declaration, *selector:
					bf := new(bytes.Buffer)
					elem.WriteTo(bf)
					bc <- bf.Bytes()
				}
			case err := <-pErrc:
				errc <- err
				return
			}
		}
	}()

	return bc, errc
}

// Path converts path's extenstion into a GCSS file extension.
func Path(path string) string {
	return convertExt(path, extGCSS)
}

// convertExt converts path's extension into ext.
func convertExt(path string, ext string) string {
	return strings.TrimSuffix(path, filepath.Ext(path)) + ext
}
