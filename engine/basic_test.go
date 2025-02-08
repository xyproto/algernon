package engine

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	lua "github.com/xyproto/gopher-lua"
)

// tableToString converts a Lua table (assumed to be an array of strings)
// into a single string with each element separated by a newline.
func tableToString(L *lua.LState, tbl *lua.LTable) string {
	var lines []string
	n := tbl.Len()
	for i := 1; i <= n; i++ {
		lines = append(lines, tbl.RawGetInt(i).String())
	}
	return strings.Join(lines, "\n")
}

// testRun3Command is a helper function that calls the Lua function "run3"
// with the provided command string and then passes the returned stdout, stderr,
// and exit code to the provided assertion function.
func testRun3Command(t *testing.T, L *lua.LState, command string, assert func(stdout, stderr string, exitCode int)) {
	fn := L.GetGlobal("run3")
	if err := L.CallByParam(lua.P{
		Fn:      fn,
		NRet:    3,
		Protect: true,
	}, lua.LString(command)); err != nil {
		t.Fatalf("Error calling run3 with command %q: %v", command, err)
	}

	// The run3 function now pushes two Lua tables and an exit code.
	stdoutTable, ok := L.Get(-3).(*lua.LTable)
	if !ok {
		t.Fatalf("Expected stdout to be a table, got %v", L.Get(-3).Type())
	}
	stderrTable, ok := L.Get(-2).(*lua.LTable)
	if !ok {
		t.Fatalf("Expected stderr to be a table, got %v", L.Get(-2).Type())
	}
	exitCode := int(lua.LVAsNumber(L.Get(-1)))
	L.Pop(3)

	stdout := tableToString(L, stdoutTable)
	stderr := tableToString(L, stderrTable)
	assert(stdout, stderr, exitCode)
}

// TestRun3System tests the run3 function loaded by LoadBasicSystemFunctions.
func TestRun3System(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	// Create a minimal Config instance.
	cfg := &Config{versionString: "1.0"}

	// Load basic system functions (which register run3).
	cfg.LoadBasicSystemFunctions(L)

	// Test a working command.
	testRun3Command(t, L, "echo hello", func(stdout, stderr string, exitCode int) {
		// echo should output "hello" (without any extra empty lines).
		if strings.TrimSpace(stdout) != "hello" {
			t.Errorf("Expected stdout 'hello', got %q", stdout)
		}
		if stderr != "" {
			t.Errorf("Expected empty stderr, got %q", stderr)
		}
		if exitCode != 0 {
			t.Errorf("Expected exit code 0, got %d", exitCode)
		}
	})

	// Test an empty command. This should return exit code 1.
	testRun3Command(t, L, "", func(stdout, stderr string, exitCode int) {
		if stdout != "" || stderr != "" || exitCode != 1 {
			t.Errorf("For empty command, expected empty stdout/stderr and exit code 1, got stdout=%q, stderr=%q, exitCode=%d", stdout, stderr, exitCode)
		}
	})

	// Test a failing command. This should return exit code 42.
	testRun3Command(t, L, "exit 42", func(stdout, stderr string, exitCode int) {
		if exitCode != 42 {
			t.Errorf("Expected exit code 42 for command 'exit 42', got %d", exitCode)
		}
	})
}

// TestRun3Web tests the run3 function loaded by LoadBasicWeb.
func TestRun3Web(t *testing.T) {
	// Create a temporary directory to act as the script's directory.
	tmpDir, err := os.MkdirTemp("", "test-run3-web")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a dummy script file within tmpDir.
	scriptPath := filepath.Join(tmpDir, "script.lua")
	if err := os.WriteFile(scriptPath, []byte("-- dummy script"), 0644); err != nil {
		t.Fatal(err)
	}

	// Set up dummy HTTP response and request objects.
	rec := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "http://zombo.com", nil)
	if err != nil {
		t.Fatal(err)
	}

	L := lua.NewState()
	defer L.Close()

	// Create a minimal Config instance.
	cfg := &Config{}

	// Provide a no-op flush function.
	flushFunc := func() {}
	httpStatus := &FutureStatus{}

	// Load basic web functions. Note that run3 here sets its working directory to the
	// directory of the given filename (i.e. the directory containing script.lua).
	cfg.LoadBasicWeb(rec, req, L, scriptPath, flushFunc, httpStatus)

	// Test a working command.
	testRun3Command(t, L, "echo hello", func(stdout, stderr string, exitCode int) {
		if strings.TrimSpace(stdout) != "hello" {
			t.Errorf("Expected stdout 'hello', got %q", stdout)
		}
		if stderr != "" {
			t.Errorf("Expected empty stderr, got %q", stderr)
		}
		if exitCode != 0 {
			t.Errorf("Expected exit code 0, got %d", exitCode)
		}
	})

	// Test that the working directory is correctly set.
	// Running "pwd" should output the directory that contains script.lua.
	expectedDir := filepath.Dir(scriptPath)
	testRun3Command(t, L, "pwd", func(stdout, stderr string, exitCode int) {
		if strings.TrimSpace(stdout) != expectedDir {
			t.Errorf("Expected stdout to be %q (the script directory), got %q", expectedDir, stdout)
		}
		if stderr != "" {
			t.Errorf("Expected empty stderr from 'pwd', got %q", stderr)
		}
		if exitCode != 0 {
			t.Errorf("Expected exit code 0 from 'pwd', got %d", exitCode)
		}
	})

	// Test an empty command.
	testRun3Command(t, L, "", func(stdout, stderr string, exitCode int) {
		if stdout != "" || stderr != "" || exitCode != 1 {
			t.Errorf("For empty command in web context, expected empty stdout/stderr and exit code 1, got stdout=%q, stderr=%q, exitCode=%d", stdout, stderr, exitCode)
		}
	})

	// Test a failing command.
	testRun3Command(t, L, "exit 42", func(stdout, stderr string, exitCode int) {
		if exitCode != 42 {
			t.Errorf("Expected exit code 42 for command 'exit 42' in web context, got %d", exitCode)
		}
	})
}
