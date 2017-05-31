package main

import (
	"flag"
	"os"
	"testing"
)

func init() {
	exit = func(_ int) {}
}

func Test_main_flagV(t *testing.T) {
	resetForTesting(nil)

	os.Args = []string{os.Args[0], "-v"}

	main()
}

func Test_main_noArgsReadAllErr(t *testing.T) {
	stdinBak := stdin

	defer func() {
		stdin = stdinBak
	}()

	stdin = nil

	resetForTesting(nil)

	os.Args = []string{os.Args[0]}

	main()
}

func Test_main_noArgsCompileErr(t *testing.T) {
	stdinBak := stdin

	defer func() {
		stdin = stdinBak
	}()

	var err error

	stdin, err = os.Open("test/0002.gcss")

	if err != nil {
		t.Errorf("error occurred [error: %q]", err.Error())
	}

	resetForTesting(nil)

	os.Args = []string{os.Args[0]}

	main()
}

func Test_main_noArgs(t *testing.T) {
	resetForTesting(nil)

	os.Args = []string{os.Args[0]}

	main()
}

func Test_main_argsLGreaterThanValidLen(t *testing.T) {
	resetForTesting(nil)

	os.Args = []string{os.Args[0], "test1", "test2"}

	main()
}

func Test_main_compileErr(t *testing.T) {
	resetForTesting(nil)

	os.Args = []string{os.Args[0], "test/not_exist_file"}

	main()
}

func Test_main(t *testing.T) {
	resetForTesting(nil)

	os.Args = []string{os.Args[0], "test/0001.gcss"}

	main()
}

// resetForTesting clears all flag state and sets the usage function as directed.
// After calling ResetForTesting, parse errors in flag handling will not
// exit the program.
func resetForTesting(usage func()) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flag.Usage = usage
}
