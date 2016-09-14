package generator

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestGenerator(t *testing.T) {

	testFiles, err := ioutil.ReadDir("test")
	assert.NoError(t, err)

	for _, testFile := range testFiles {
		if strings.HasSuffix(testFile.Name(), "_output.js") ||
			strings.HasSuffix(testFile.Name(), "~") {
			continue
		}

		testName := testFile.Name()[:len(testFile.Name())-3]
		outName := "./test/" + testName + "_output.js"

		inFd, err := os.Open("./test/" + testFile.Name())
		assert.NoError(t, err)
		expectedFd, err := os.Open(outName)
		assert.NoError(t, err)

		in := &bytes.Buffer{}
		expected := &bytes.Buffer{}

		generated, err := ParseAndGenerate(inFd)
		assert.NoError(t, err, testName)
		in.ReadFrom(generated)
		expected.ReadFrom(expectedFd)

		assert.Equal(t, strings.Trim(expected.String(), " \n"), strings.Trim(in.String(), " \n"), testName)
	}

}
