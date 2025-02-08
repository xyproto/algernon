package run3

import (
	"bytes"
	"os/exec"
	"strings"

	"github.com/xyproto/algernon/lua/convert"
	lua "github.com/xyproto/gopher-lua"
)

// Helper executes the given shell command in the specified working directory.
// It converts stdout and stderr into slices of strings (one entry per line) and then
// uses convert.Strings2table to return two Lua tables. The exit code is also pushed.
// If the command is empty, both output tables are empty and exit code 1 is returned.
func Helper(L *lua.LState, command string, workingDir string) int {
	if command == "" { // no command given
		L.Push(L.NewTable())   // stdout: empty table
		L.Push(L.NewTable())   // stderr: empty table
		L.Push(lua.LNumber(1)) // exit code 1
		return 3
	}
	cmd := exec.Command("sh", "-c", command)
	if workingDir != "" {
		cmd.Dir = workingDir
	}
	// Set up buffers for stdout and stderr.
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	// Run command.
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1 // fallback if no exit code is available
		}
	}
	// Normalize Windows CRLF to LF and split output into lines.
	outStr := strings.ReplaceAll(stdout.String(), "\r\n", "\n")
	errStr := strings.ReplaceAll(stderr.String(), "\r\n", "\n")
	outLines := strings.Split(outStr, "\n")
	errLines := strings.Split(errStr, "\n")
	// Remove the final empty element if present (as io.popen:lines would not yield an empty last line).
	if len(outLines) > 0 && outLines[len(outLines)-1] == "" {
		outLines = outLines[:len(outLines)-1]
	}
	if len(errLines) > 0 && errLines[len(errLines)-1] == "" {
		errLines = errLines[:len(errLines)-1]
	}
	// Use the convert.Strings2table helper to convert slices to Lua tables.
	L.Push(convert.Strings2table(L, outLines))
	L.Push(convert.Strings2table(L, errLines))
	L.Push(lua.LNumber(exitCode))
	return 3
}
