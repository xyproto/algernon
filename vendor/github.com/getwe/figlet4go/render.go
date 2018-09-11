package figlet4go

import (
	"errors"
	"fmt"
	"github.com/fatih/color"
	"strings"
)

type RenderOptions struct {
	FontName string // font name

	FontColor []color.Attribute // every ascii byte's color
}

func NewRenderOptions() *RenderOptions {
	opt := &RenderOptions{}
	opt.FontName = "default"
	return opt
}

type AsciiRender struct {
	fontMgr *fontManager
}

func NewAsciiRender() *AsciiRender {
	this := &AsciiRender{}

	this.fontMgr = newFontManager()
	return this
}

// walk through the path, load all the *.flf font file
func (this *AsciiRender) LoadFont(fontPath string) error {
	return this.fontMgr.loadFont(fontPath)
}

// render with default options
func (this *AsciiRender) Render(asciiStr string) (string, error) {
	return this.render(asciiStr, NewRenderOptions())
}

// render with options
func (this *AsciiRender) RenderOpts(asciiStr string, opts *RenderOptions) (string, error) {
	return this.render(asciiStr, opts)
}

func (this *AsciiRender) convertChar(font *font, char rune) ([]string, error) {

	if char < 0 || char > 127 {
		return nil, errors.New("Not Ascii")
	}

	height := font.height
	begintRow := (int(char) - 32) * height

	word := make([]string, height, height)

	for i := 0; i < height; i++ {
		row := font.fontSlice[begintRow+i]
		row = strings.Replace(row, "@", "", -1)
		row = strings.Replace(row, font.hardblank, " ", -1)
		word[i] = row
	}

	return word, nil
}

func (this *AsciiRender) render(asciiStr string, opt *RenderOptions) (string, error) {

	font, _ := this.fontMgr.getFont(opt.FontName)

	wordlist := make([][]string, 0)
	for _, char := range asciiStr {
		word, err := this.convertChar(font, char)
		if err != nil {
			return "", err
		}
		wordlist = append(wordlist, word)
	}

	result := ""

	wordColorFunc := make([]func(a ...interface{}) string, len(wordlist))
	for i, _ := range wordColorFunc {
		if i < len(opt.FontColor) {
			wordColorFunc[i] = color.New(opt.FontColor[i]).SprintFunc()
		} else {
			wordColorFunc[i] = fmt.Sprint
		}
	}

	for i := 0; i < font.height; i++ {
		for j := 0; j < len(wordlist); j++ {
			result += wordColorFunc[j]((wordlist[j][i]))
		}
		result += "\n"
	}
	return result, nil
}
