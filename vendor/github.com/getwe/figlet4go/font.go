package figlet4go

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type font struct {
	hardblank string
	height    int
	fontSlice []string
}

type fontManager struct {
	// font library
	fontLib map[string]*font

	// font name to path
	fontList map[string]string
}

func newFontManager() *fontManager {
	this := &fontManager{}

	this.fontLib = make(map[string]*font)
	this.fontList = make(map[string]string)
	this.loadBuildInFont()

	return this
}

// walk through the path, load all the *.flf font file
func (this *fontManager) loadFont(fontPath string) error {

	return filepath.Walk(fontPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(info.Name(), ".flf") {
			return nil
		}

		fontName := strings.TrimSuffix(info.Name(), ".flf")
		this.fontList[fontName] = path
		return nil
	})
}

func (this *fontManager) loadBuildInFont() error {
	font, err := this.parseFontContent(builtInFont)
	if err != nil {
		return err
	}
	this.fontLib["default"] = font
	return nil
}

func (this *fontManager) loadDiskFont(fontName string) error {

	fontFilePath, ok := this.fontList[fontName]
	if !ok {
		return errors.New("FontName Not Found.")
	}

	// read full file content
	fileBuf, err := ioutil.ReadFile(fontFilePath)
	if err != nil {
		return err
	}

	font, err := this.parseFontContent(string(fileBuf))
	if err != nil {
		return err
	}

	this.fontLib[fontName] = font
	return nil
}

func (this *fontManager) parseFontContent(cont string) (*font, error) {
	lines := strings.Split(cont, "\n")
	if len(lines) < 1 {
		return nil, errors.New("font content error")
	}

	// flf2a$ 7 5 16 -1 12
	// Fender by Scooter 8/94 (jkratten@law.georgetown.edu)
	//
	// Explanation of first line:
	// flf2 - "magic number" for file identification
	// a    - should always be `a', for now
	// $    - the "hardblank" -- prints as a blank, but can't be smushed
	// 7    - height of a character
	// 5    - height of a character, not including descenders
	// 10   - max line length (excluding comment lines) + a fudge factor
	// -1   - default smushmode for this font (like "-m 15" on command line)
	// 12   - number of comment lines

	header := strings.Split(lines[0], " ")

	font := &font{}
	font.hardblank = header[0][len(header[0])-1:]
	font.height, _ = strconv.Atoi(header[1])

	commentEndLine, _ := strconv.Atoi(header[5])
	font.fontSlice = lines[commentEndLine+1:]

	return font, nil
}

func (this *fontManager) getFont(fontName string) (*font, error) {
	font, ok := this.fontLib[fontName]
	if !ok {
		err := this.loadDiskFont(fontName)
		if err != nil {
			font, _ := this.fontLib["default"]
			return font, nil
		}
	}
	font, _ = this.fontLib[fontName]
	return font, nil
}
