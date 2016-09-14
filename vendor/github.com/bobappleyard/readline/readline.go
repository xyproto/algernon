/*
This package provides access to basic GNU Readline functions. Currently supported are:

	- getting text from a prompt (via the String() and NewReader() functions).
	- managing the prompt's history (via the AddHistory(), GetHistory(), ClearHistory() and HistorySize() functions).
	- controlling tab completion (via the Completer variable).

Here is a simple example:

	package main

	import (
	    "fmt"
	    "io"
	    "github.com/bobappleyard/readline"
	)

	func main() {
	    for {
	        l, err := readline.String("> ")
	        if err == io.EOF {
	            break
	        }
	        if err != nil {
	            fmt.Println("error: ", err)
	            break
	        }
	        fmt.Println(l)
	        readline.AddHistory(l)
	    }
	}
*/
package readline

/*

#cgo darwin CFLAGS: -I/usr/local/opt/readline/include
#cgo darwin LDFLAGS: -L/usr/local/opt/readline/lib
#cgo LDFLAGS: -lreadline -lhistory

#include <stdio.h>
#include <stdlib.h>
#include <readline/readline.h>
#include <readline/history.h>
#include <readline/keymaps.h>

extern char *_completion_function(char *s, int i);

static char *_completion_function_trans(const char *s, int i) {
	return _completion_function((char *) s, i);
}

static void register_readline() {
	rl_completion_entry_function = _completion_function_trans;
	using_history();
}

*/
import "C"

import (
	"io"
	"os"
	"os/signal"
	"regexp"
	"syscall"
	"unsafe"
)

// The prompt used by Reader(). The prompt can contain ANSI escape
// sequences, they will be escaped as necessary.
var Prompt = "> "

// The continue prompt used by Reader(). The prompt can contain ANSI escape
// sequences, they will be escaped as necessary.
var Continue = ".."

const (
	promptStartIgnore = string(C.RL_PROMPT_START_IGNORE)
	promptEndIgnore   = string(C.RL_PROMPT_END_IGNORE)
)

// If CompletionAppendChar is non-zero, readline will append the
// corresponding character to the prompt after each completion. A
// typical value would be a space.
var CompletionAppendChar = 0

type state byte

const (
	readerStart state = iota
	readerContinue
	readerEnd
)

type reader struct {
	buf   []byte
	state state
}

var shortEscRegex = "\x1b[@-Z\\-_]"
var csiPrefix = "(\x1b[[]|\xC2\x9b)"
var csiParam = "([0-9]+|\"[^\"]*\")"
var csiSuffix = "[@-~]"
var csiRegex = csiPrefix + "(" + csiParam + "(;" + csiParam + ")*)?" + csiSuffix
var escapeSeq = regexp.MustCompile(shortEscRegex + "|" + csiRegex)

// Begin reading lines. If more than one line is required, the continue prompt
// is used for subsequent lines.
func NewReader() io.Reader {
	return new(reader)
}

func (r *reader) getLine() error {
	prompt := Prompt
	if r.state == readerContinue {
		prompt = Continue
	}
	s, err := String(prompt)
	if err != nil {
		return err
	}
	r.buf = []byte(s)
	return nil
}

func (r *reader) Read(buf []byte) (int, error) {
	if r.state == readerEnd {
		return 0, io.EOF
	}
	if len(r.buf) == 0 {
		err := r.getLine()
		if err == io.EOF {
			r.state = readerEnd
		}
		if err != nil {
			return 0, err
		}
		r.state = readerContinue
	}
	copy(buf, r.buf)
	l := len(buf)
	if len(buf) > len(r.buf) {
		l = len(r.buf)
	}
	r.buf = r.buf[l:]
	return l, nil
}

// Read a line with the given prompt. The prompt can contain ANSI
// escape sequences, they will be escaped as necessary.
func String(prompt string) (string, error) {
	prompt = "\x1b[0m" + prompt // Prepend a 'reset' ANSI escape sequence
	prompt = escapeSeq.ReplaceAllString(prompt, promptStartIgnore+"$0"+promptEndIgnore)
	p := C.CString(prompt)
	rp := C.readline(p)
	s := C.GoString(rp)
	C.free(unsafe.Pointer(p))
	if rp != nil {
		C.free(unsafe.Pointer(rp))
		return s, nil
	}
	return s, io.EOF
}

// This function provides entries for the tab completer.
var Completer = func(query, ctx string) []string {
	return nil
}

var entries []*C.char

// This function can be assigned to the Completer variable to use
// readline's default filename completion, or it can be called by a
// custom completer function to get a list of files and filter it.
func FilenameCompleter(query, ctx string) []string {
	var compls []string
	var c *C.char
	q := C.CString(query)

	for i := 0; ; i++ {
		if c = C.rl_filename_completion_function(q, C.int(i)); c == nil {
			break
		}
		compls = append(compls, C.GoString(c))
		C.free(unsafe.Pointer(c))
	}

	C.free(unsafe.Pointer(q))

	return compls
}

//export _completion_function
func _completion_function(p *C.char, _i C.int) *C.char {
	C.rl_completion_append_character = C.int(CompletionAppendChar)
	i := int(_i)
	if i == 0 {
		es := Completer(C.GoString(p), C.GoString(C.rl_line_buffer))
		entries = make([]*C.char, len(es))
		for i, x := range es {
			entries[i] = C.CString(x)
		}
	}
	if i >= len(entries) {
		return nil
	}
	return entries[i]
}

func SetWordBreaks(cs string) {
	C.rl_completer_word_break_characters = C.CString(cs)
}

// Add an item to the history.
func AddHistory(s string) {
	n := HistorySize()
	if n == 0 || s != GetHistory(n-1) {
		C.add_history(C.CString(s))
	}
}

// Retrieve a line from the history.
func GetHistory(i int) string {
	e := C.history_get(C.int(i + 1))
	if e == nil {
		return ""
	}
	return C.GoString(e.line)
}

// Clear the screen
func ClearScreen() {
	var x, y C.int = 0, 0
	C.rl_clear_screen(x, y)
}

// rl_forced_update_display / redraw
func ForceUpdateDisplay() {
	C.rl_forced_update_display()
}

// Replace current line
func ReplaceLine(text string, clearUndo int) {
	C.rl_replace_line(C.CString(text), C.int(clearUndo))
}

// Redraw current line
func RefreshLine() {
	var x, y C.int = 0, 0
	C.rl_refresh_line(x, y)
}

// Deletes all the items in the history.
func ClearHistory() {
	C.clear_history()
}

// Returns the number of items in the history.
func HistorySize() int {
	return int(C.history_length)
}

// Load the history from a file.
func LoadHistory(path string) error {
	p := C.CString(path)
	e := C.read_history(p)
	C.free(unsafe.Pointer(p))

	if e == 0 {
		return nil
	}
	return syscall.Errno(e)
}

// Save the history to a file.
func SaveHistory(path string) error {
	p := C.CString(path)
	e := C.write_history(p)
	C.free(unsafe.Pointer(p))

	if e == 0 {
		return nil
	}
	return syscall.Errno(e)
}

// Cleanup() frees internal memory and restores terminal
// attributes. This function should be called when program execution
// stops before the return of a String() call, so as not to leave the
// terminal in a corrupted state.
func Cleanup() {
	C.rl_free_line_state()
	C.rl_cleanup_after_signal()
}

func handleSignals() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGWINCH)

	for s := range signals {
		switch s {
		case syscall.SIGWINCH:
			C.rl_resize_terminal()
		}
	}
}

func init() {
	C.rl_catch_signals = 0
	C.rl_catch_sigwinch = 0
	C.register_readline()

	go handleSignals()
}
