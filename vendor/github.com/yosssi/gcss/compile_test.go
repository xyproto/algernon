package gcss

import (
	"errors"
	"os"
)
import "strings"
import "testing"

var errErrReader = errors.New("errReader error")

type errReader struct{}

func (r *errReader) Read(p []byte) (int, error) {
	return 0, errErrReader
}

func TestCompile_readAllErr(t *testing.T) {
	if _, err := Compile(os.Stdout, &errReader{}); err != errErrReader {
		t.Errorf("error should be %+v [actual: %+v]", errErrReader, err)
	}
}

func TestCompile_compileBytesErr(t *testing.T) {
	r, err := os.Open("test/0015.gcss")

	if err != nil {
		t.Errorf("error occurred [error: %q]", err.Error())
	}

	_, err = Compile(os.Stdout, r)

	if expected, actual := "indent is invalid [line: 5]", err.Error(); actual != expected {
		t.Errorf("error should be %+q [actual: %+q]", expected, actual)
	}
}

func TestCompile(t *testing.T) {
	r, err := os.Open("test/0016.gcss")

	if err != nil {
		t.Errorf("error occurred [error: %q]", err.Error())
	}

	if _, err := Compile(os.Stdout, r); err != nil {
		t.Errorf("error occurred [error: %q]", err.Error())
	}
}

func TestCompileFile_readFileErr(t *testing.T) {
	_, err := CompileFile("not_exist_file")

	if err == nil {
		t.Error("error should be occurred")
	}

	if expected, actual := "open not_exist_file: ", err.Error(); !strings.HasPrefix(actual, expected) || !os.IsNotExist(err) {
		t.Errorf("err should be %q [actual: %q]", expected, actual)
	}
}

func TestCompileFile_compileStringErr(t *testing.T) {
	_, err := CompileFile("test/0004.gcss")

	if err == nil {
		t.Error("error should be occurred")
	}

	if expected, actual := "indent is invalid [line: 5]", err.Error(); expected != actual {
		t.Errorf("err should be %q [actual: %q]", expected, actual)
	}
}

func TestCompileFile_writeErr(t *testing.T) {
	cssFileBack := cssFilePath

	cssFilePath = func(_ string) string {
		return "not_exist_dir/not_exist_file"
	}

	_, err := CompileFile("test/0003.gcss")

	if err == nil {
		t.Error("error should be occurred")
	}

	if expected, actual := "open not_exist_dir/not_exist_file: ", err.Error(); !strings.HasPrefix(actual, expected) || !os.IsNotExist(err) {
		t.Errorf("err should be %q [actual: %q]", expected, actual)
	}

	cssFilePath = cssFileBack
}

func TestCompileFile(t *testing.T) {
	path, err := CompileFile("test/0003.gcss")

	if err != nil {
		t.Errorf("error occurred [error: %q]", err.Error())
	}

	if expected := "test/0003.css"; expected != path {
		t.Errorf("path should be %q [actual: %q]", expected, path)
	}
}

func TestCompileFile_pattern2(t *testing.T) {
	gcssPath := "test/0007.gcss"

	path, err := CompileFile(gcssPath)

	if err != nil {
		t.Errorf("error occurred [error: %q]", err.Error())
	}

	if expected := cssFilePath(path); expected != path {
		t.Errorf("path should be %q [actual: %q]", expected, path)
	}
}

func Test_complieBytes(t *testing.T) {
	compileBytes([]byte(""))
}

func TestPath(t *testing.T) {
	path := "/test"

	if expected, actual := path+extGCSS, Path(path+extCSS); actual != expected {
		t.Errorf("returned value should be %q [actual: %q]", expected, actual)
	}
}
