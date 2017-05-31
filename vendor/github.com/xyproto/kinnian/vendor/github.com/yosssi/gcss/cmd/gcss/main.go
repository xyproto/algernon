package main

import (
	"flag"
	"io"
	"os"

	"github.com/yosssi/gcss"
)

const validArgsLen = 1

var exit = os.Exit
var stdin = os.Stdin

func main() {
	v := flag.Bool("v", false, "Print the version and exit.")

	flag.Parse()

	if *v {
		writeTo(os.Stdout, gcss.Version)
		exit(0)
		return
	}

	args := flag.Args()
	argsL := len(args)

	if argsL > validArgsLen {
		writeTo(os.Stderr, "The number of the command line args should be 1.")
		exit(1)
		return
	}

	if argsL == 0 {
		if _, err := gcss.Compile(os.Stdout, stdin); err != nil {
			writeTo(os.Stderr, err.Error())
			exit(1)
			return
		}
	} else {
		path, err := gcss.CompileFile(args[0])

		if err != nil {
			writeTo(os.Stderr, err.Error())
			exit(1)
			return
		}

		writeTo(os.Stdout, "compiled "+path)
	}
}

// writeTo writes s to w.
func writeTo(w io.Writer, s string) {
	w.Write([]byte(s + "\n"))
}
