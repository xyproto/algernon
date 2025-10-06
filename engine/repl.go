package engine

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/chzyer/readline"
	"github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"github.com/xyproto/algernon/lua/codelib"
	"github.com/xyproto/algernon/lua/convert"
	"github.com/xyproto/algernon/lua/datastruct"
	"github.com/xyproto/algernon/lua/jnode"
	"github.com/xyproto/algernon/lua/pure"
	"github.com/xyproto/algernon/platformdep"
	"github.com/xyproto/ask"
	lua "github.com/xyproto/gopher-lua"
	"github.com/xyproto/vt"
)

const exitMessage = "bye"

// Export Lua functions specific to the REPL
func exportREPLSpecific(L *lua.LState) {
	// Attempt to return a more informative text than the memory location.
	// Can take several arguments, just like print().
	L.SetGlobal("pprint", L.NewFunction(func(L *lua.LState) int {
		var buf bytes.Buffer
		top := L.GetTop()
		for i := 1; i <= top; i++ {
			convert.PprintToWriter(&buf, L.Get(i))
			if i != top {
				buf.WriteString("\t")
			}
		}

		// Output the combined text
		fmt.Println(buf.String())

		return 0 // number of results
	}))

	// Get the current directory since this is probably in the REPL
	L.SetGlobal("scriptdir", L.NewFunction(func(L *lua.LState) int {
		scriptpath, err := os.Getwd()
		if err != nil {
			logrus.Error(err)
			L.Push(lua.LString("."))
			return 1 // number of results
		}
		top := L.GetTop()
		if top == 1 {
			// Also include a separator and a filename
			fn := L.ToString(1)
			scriptpath = filepath.Join(scriptpath, fn)
		}
		// Now have the correct absolute scriptpath
		L.Push(lua.LString(scriptpath))
		return 1 // number of results
	}))

	// Given a glob, like "md/*.md", read the files with the scriptdir() as the starting point.
	// Then return all the contents of the files as a table.
	L.SetGlobal("readglob", L.NewFunction(func(L *lua.LState) int {
		var (
			pattern  = L.ToString(1)
			basepath string
			err      error
		)
		if L.GetTop() == 2 {
			basepath = L.ToString(2)
		} else {
			basepath, err = os.Getwd()
			if err != nil {
				logrus.Error(err)
				L.Push(lua.LNil)
				return 1
			}
		}
		matches, err := filepath.Glob(filepath.Join(basepath, pattern))
		if err != nil {
			L.Push(lua.LNil)
			return 1
		}
		results := L.NewTable()
		for _, match := range matches {
			content, err := os.ReadFile(match)
			if err != nil {
				logrus.Error(err)
				L.Push(lua.LNil)
				return 1
			}
			results.Append(lua.LString(content))
		}
		L.Push(results)
		return 1
	}))
}

// Syntax highlight the given line
func highlight(line string) string {
	unprocessed := line
	unprocessed, comment := vt.ColorSplit(unprocessed, "//", 0, vt.DarkGray, vt.DarkGray, false)
	module, unprocessed := vt.ColorSplit(unprocessed, ":", vt.LightGreen, vt.Red, 0, true)
	function := ""
	if unprocessed != "" {
		// Green function names
		if strings.Contains(unprocessed, "(") {
			fields := strings.SplitN(unprocessed, "(", 2)
			function = vt.LightGreen.Get(fields[0])
			unprocessed = "(" + fields[1]
		} else if strings.Contains(unprocessed, "|") {
			unprocessed = "<magenta>" + strings.ReplaceAll(unprocessed, "|", "<white>|</white><magenta>") + "</magenta>"
		}
	}
	unprocessed, typed := vt.ColorSplit(unprocessed, "->", 0, vt.LightBlue, vt.Red, false)
	unprocessed = strings.ReplaceAll(unprocessed, "string", vt.LightBlue.Get("string"))
	unprocessed = strings.ReplaceAll(unprocessed, "number", vt.LightYellow.Get("number"))
	unprocessed = strings.ReplaceAll(unprocessed, "function", vt.LightCyan.Get("function"))
	return module + function + unprocessed + typed + comment
}

// Output syntax highlighted help text, with an additional usage message
func outputHelp(o *vt.TextOutput, helpText string) {
	for _, line := range strings.Split(helpText, "\n") {
		o.Println(highlight(line))
	}
	o.Println(usageMessage)
}

// Output syntax highlighted help about a specific topic or function
func outputHelpAbout(o *vt.TextOutput, helpText, topic string) {
	switch topic {
	case "help":
		o.Println(vt.DarkGray.Get("Output general help or help about a specific topic."))
		return
	case "webhelp":
		o.Println(vt.DarkGray.Get("Output help about web-related functions."))
		return
	case "confighelp":
		o.Println(vt.DarkGray.Get("Output help about configuration-related functions."))
		return
	case "quit", "exit", "shutdown", "halt":
		o.Println(vt.DarkGray.Get("Quit Algernon."))
		return
	}
	comment := ""
	for _, line := range strings.Split(helpText, "\n") {
		if strings.HasPrefix(line, topic) {
			// Output help text, with some surrounding blank lines
			o.Println("\n" + highlight(line))
			o.Println("\n" + vt.DarkGray.Get(strings.TrimSpace(comment)) + "\n")
			return
		}
		// Gather comments until a non-comment is encountered
		if strings.HasPrefix(line, "//") {
			comment += strings.TrimSpace(line[2:]) + "\n"
		} else {
			comment = ""
		}
	}
	o.Println(vt.DarkGray.Get("Found no help for: ") + vt.White.Get(topic))
}

// Take all functions mentioned in the given help text string and add them to the readline completer
func addFunctionsFromHelptextToCompleter(helpText string, completer *readline.PrefixCompleter) {
	for _, line := range strings.Split(helpText, "\n") {
		if !strings.HasPrefix(line, "//") && strings.Contains(line, "(") {
			parts := strings.Split(line, "(")
			if strings.Contains(line, "()") {
				completer.Children = append(completer.Children, &readline.PrefixCompleter{Name: []rune(parts[0] + "()")})
			} else {
				completer.Children = append(completer.Children, &readline.PrefixCompleter{Name: []rune(parts[0] + "(")})
			}
		}
	}
}

// LoadLuaFunctionsForREPL exports the various Lua functions that might be needed in the REPL
func (ac *Config) LoadLuaFunctionsForREPL(L *lua.LState, o *vt.TextOutput) {
	// Server configuration functions
	ac.LoadServerConfigFunctions(L, "")

	// Other basic system functions, like log()
	ac.LoadBasicSystemFunctions(L)

	// If there is a database backend
	if ac.perm != nil {

		// Retrieve the creator struct
		creator := ac.perm.UserState().Creator()

		// Simpleredis data structures
		datastruct.LoadList(L, creator)
		datastruct.LoadSet(L, creator)
		datastruct.LoadHash(L, creator)
		datastruct.LoadKeyValue(L, creator)

		// For saving and loading Lua functions
		codelib.Load(L, creator)
	}

	// For handling JSON data
	jnode.LoadJSONFunctions(L)
	ac.LoadJFile(L, ac.serverDirOrFilename)
	jnode.Load(L)

	// Extras
	pure.Load(L)

	// Export pprint and scriptdir
	exportREPLSpecific(L)

	// Plugin functionality
	ac.LoadPluginFunctions(L, o)

	// Cache
	ac.LoadCacheFunctions(L)
}

// REPL provides a "Read Eval Print" loop for interacting with Lua.
// A variety of functions are exposed to the Lua state.
func (ac *Config) REPL(ready, done chan bool) error {
	var (
		historyFilename string
		err             error
	)

	historydir, err := homedir.Dir()
	if err != nil {
		logrus.Error("Could not find a user directory to store the REPL history.")
		historydir = "."
	}

	// Retrieve a Lua state
	L := ac.luapool.Get()
	// Don't re-use the Lua state
	defer L.Close()

	// Colors and input
	o := vt.NewTextOutput(platformdep.EnableColors, true)

	// Command history file
	historyFilename = filepath.Join(historydir, platformdep.HistoryFilename)

	// Export a selection of functions to the Lua state
	ac.LoadLuaFunctionsForREPL(L, o)

	<-ready // Wait for the server to be ready

	// Tell the user that the server is ready
	o.Println(vt.LightGreen.Get("Ready"))

	// Start the read, eval, print loop
	var (
		line     string
		prompt   = vt.LightCyan.Get("lua> ")
		EOF      bool
		EOFcount int
	)

	var initialPrefixCompleters []readline.PrefixCompleterInterface
	for _, word := range []string{"bye", "confighelp", "cwd", "dir", "exit", "help", "pwd", "quit", "serverdir", "serverfile", "webhelp", "zalgo"} {
		initialPrefixCompleters = append(initialPrefixCompleters, &readline.PrefixCompleter{Name: []rune(word)})
	}

	prefixCompleter := readline.NewPrefixCompleter(initialPrefixCompleters...)

	addFunctionsFromHelptextToCompleter(generalHelpText, prefixCompleter)

	l, err := readline.NewEx(&readline.Config{
		Prompt:            prompt,
		HistoryFile:       historyFilename,
		AutoComplete:      prefixCompleter,
		InterruptPrompt:   "^C",
		EOFPrompt:         "exit",
		HistorySearchFold: true,
	})
	if err != nil {
		logrus.Error("Could not initiate github.com/chzyer/readline: " + err.Error())
	}

	// To be run at server shutdown
	AtShutdown(func() {
		// Verbose mode has different log output at shutdown
		if !ac.verboseMode {
			o.Println(vt.LightBlue.Get(exitMessage))
		}
	})
	for {
		// Retrieve user input
		EOF = false
		if platformdep.Mingw {
			// No support for EOF
			line = ask.Ask(prompt)
		} else {
			if line, err = l.Readline(); err != nil {
				switch {
				case err == io.EOF:
					if ac.debugMode {
						o.Println(vt.LightMagenta.Get(err.Error()))
					}
					EOF = true
				case err == readline.ErrInterrupt:
					logrus.Warn("Interrupted")
					done <- true
					return nil
				default:
					logrus.Error("Error reading line(" + err.Error() + ").")
					continue
				}
			}
		}
		if EOF {
			if ac.ctrldTwice {
				switch EOFcount {
				case 0:
					o.Err("Press ctrl-d again to exit.")
					EOFcount++
					continue
				default:
					done <- true
					return nil
				}
			} else {
				done <- true
				return nil
			}
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		switch line {
		case "help":
			outputHelp(o, generalHelpText)
			continue
		case "webhelp":
			outputHelp(o, webHelpText)
			continue
		case "confighelp":
			outputHelp(o, configHelpText)
			continue
		case "dir":
			// Be more helpful than listing the Lua bytecode contents of the dir function. Call "dir()".
			line = "dir()"
		case "cwd", "pwd":
			if cwd, err := os.Getwd(); err != nil {
				// Might work if Getwd should fail. Should work on Windows, Linux and macOS
				line = "os.getenv'CD' or os.getenv'PWD'"
			} else {
				fmt.Println(cwd)
				continue
			}
		case "serverfile", "serverdir":
			if absdir, err := filepath.Abs(ac.serverDirOrFilename); err != nil {
				fmt.Println(ac.serverDirOrFilename)
			} else {
				fmt.Println(absdir)
			}
			continue
		case "quit", "exit", "shutdown", "halt":
			done <- true
			return nil
		case "zalgo":
			// Easter egg
			o.ErrExit("Ḫ̷̲̫̰̯̭̀̂̑~ͅĚ̥̖̩̘̱͔͈͈ͬ̚ ̦̦͖̲̀ͦ͂C̜͓̲̹͐̔ͭ̏Oͭ͛͂̋ͭͬͬ͆͏̺͓̰͚͠ͅM̢͉̼̖͍̊̕Ḛ̭̭͗̉̀̆ͬ̐ͪ̒S͉̪͂͌̄")
		default:
			topic := ""
			if len(line) > 5 && (strings.HasPrefix(line, "help(") || strings.HasPrefix(line, "help ")) {
				topic = line[5:]
			} else if len(line) > 8 && (strings.HasPrefix(line, "webhelp(") || strings.HasPrefix(line, "webhelp ")) {
				topic = line[8:]
			}
			if len(topic) > 0 {
				topic = strings.TrimSuffix(topic, ")")
				outputHelpAbout(o, generalHelpText+webHelpText+configHelpText, topic)
				continue
			}

		}

		// If the line starts with print, don't touch it
		if strings.HasPrefix(line, "print(") {
			if err = L.DoString(line); err != nil {
				// Output the error message
				o.Err(err.Error())
			}
		} else {
			// Wrap the line in "pprint"
			if err = L.DoString("pprint(" + line + ")"); err != nil {
				// If there was a syntax error, try again without pprint
				if strings.Contains(err.Error(), "syntax error") {
					if err = L.DoString(line); err != nil {
						// Output the error message
						o.Err(err.Error())
					}
					// For other kinds of errors, output the error
				} else {
					// Output the error message
					o.Err(err.Error())
				}
			}
		}
	}
}
