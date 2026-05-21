package engine

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"

	"github.com/xyproto/datablock"
	lua "github.com/xyproto/gopher-lua"
)

// buildTestPlugin compiles the example Go plugin and returns the path to the
// resulting binary. The test is skipped if the build fails.
func buildTestPlugin(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "testplugin")
	pluginSrc, err := filepath.Abs(filepath.Join("..", "plugins", "go"))
	if err != nil {
		t.Skipf("could not resolve plugin source path: %v", err)
	}
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = pluginSrc
	if out, buildErr := cmd.CombinedOutput(); buildErr != nil {
		t.Skipf("could not build test plugin: %v\n%s", buildErr, out)
	}
	return binaryPath
}

// pluginTestConfig returns a minimal Config sufficient for plugin tests.
func pluginTestConfig(t *testing.T, dir string) *Config {
	t.Helper()
	return &Config{
		fs:                  datablock.NewFileStat(false, 0),
		serverDirOrFilename: dir,
	}
}

// TestPlugin_Basic verifies that Plugin() loads Lua code correctly and that
// the resulting Lua function (add3) returns the right value.
func TestPlugin_Basic(t *testing.T) {
	binaryPath := buildTestPlugin(t)
	cfg := pluginTestConfig(t, filepath.Dir(binaryPath))
	defer cfg.closePluginClients()

	L := lua.NewState()
	defer L.Close()
	cfg.LoadPluginFunctions(L, nil)

	if err := L.DoString(fmt.Sprintf(`Plugin(%q)`, binaryPath)); err != nil {
		t.Fatalf("Plugin() failed: %v", err)
	}
	// add3(1, 2) == 1+2+3 == 6
	// add3 calls CallPlugin which returns a JSON string, so coerce via tonumber.
	if err := L.DoString(`result = tonumber(add3(1, 2))`); err != nil {
		t.Fatalf("add3() call failed: %v", err)
	}
	v := L.GetGlobal("result")
	if v.Type() != lua.LTNumber || float64(v.(lua.LNumber)) != 6.0 {
		t.Errorf("add3(1,2): expected 6, got %v (type %v)", v, v.Type())
	}
}

// TestPlugin_KeepRunning verifies that calling Plugin(..., true) multiple times
// for the same path stores exactly one cached client.
func TestPlugin_KeepRunning(t *testing.T) {
	binaryPath := buildTestPlugin(t)
	cfg := pluginTestConfig(t, filepath.Dir(binaryPath))
	defer cfg.closePluginClients()

	L := lua.NewState()
	defer L.Close()
	cfg.LoadPluginFunctions(L, nil)

	for range 3 {
		if err := L.DoString(fmt.Sprintf(`Plugin(%q, true)`, binaryPath)); err != nil {
			t.Fatalf("Plugin() with keepRunning failed: %v", err)
		}
	}

	cfg.pluginClientsMu.Lock()
	n := len(cfg.pluginClients)
	cfg.pluginClientsMu.Unlock()
	if n != 1 {
		t.Errorf("expected exactly 1 cached client after 3 keepRunning calls, got %d", n)
	}
}

// TestPlugin_CallPluginReusesClient verifies that CallPlugin transparently
// reuses a persistent client registered by Plugin(..., true).
func TestPlugin_CallPluginReusesClient(t *testing.T) {
	binaryPath := buildTestPlugin(t)
	cfg := pluginTestConfig(t, filepath.Dir(binaryPath))
	defer cfg.closePluginClients()

	L := lua.NewState()
	defer L.Close()
	cfg.LoadPluginFunctions(L, nil)

	// Register a persistent client via Plugin
	if err := L.DoString(fmt.Sprintf(`Plugin(%q, true)`, binaryPath)); err != nil {
		t.Fatalf("Plugin() failed: %v", err)
	}

	cfg.pluginClientsMu.Lock()
	clientBefore := cfg.pluginClients[binaryPath]
	cfg.pluginClientsMu.Unlock()

	// CallPlugin should reuse the same client (not spawn a new process)
	if err := L.DoString(fmt.Sprintf(`_ = CallPlugin(%q, "Add3", 2, 3)`, binaryPath)); err != nil {
		t.Fatalf("CallPlugin() failed: %v", err)
	}

	cfg.pluginClientsMu.Lock()
	clientAfter := cfg.pluginClients[binaryPath]
	cfg.pluginClientsMu.Unlock()

	if clientBefore != clientAfter {
		t.Error("CallPlugin replaced the cached client instead of reusing it")
	}
}

// TestPlugin_ConcurrentStartPlugin stress-tests startPlugin with many
// goroutines all requesting a keepRunning client for the same path.
// Only one subprocess should be started; all callers should receive the
// same shared client. Run with -race to catch data races.
func TestPlugin_ConcurrentStartPlugin(t *testing.T) {
	binaryPath := buildTestPlugin(t)
	cfg := pluginTestConfig(t, filepath.Dir(binaryPath))
	defer cfg.closePluginClients()

	const numGoroutines = 40
	var wg sync.WaitGroup
	errCh := make(chan error, numGoroutines)

	for range numGoroutines {
		wg.Go(func() {
			_, owned, err := cfg.startPlugin(binaryPath, true, os.Stderr)
			if err != nil {
				errCh <- fmt.Errorf("startPlugin: %w", err)
				return
			}
			if owned {
				errCh <- fmt.Errorf("keepRunning=true must never return owned=true")
			}
		})
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Error(err)
	}

	// Exactly one entry in the map regardless of how many goroutines raced.
	cfg.pluginClientsMu.Lock()
	n := len(cfg.pluginClients)
	cfg.pluginClientsMu.Unlock()
	if n != 1 {
		t.Errorf("expected 1 cached client after %d concurrent starts, got %d", numGoroutines, n)
	}
}

// TestPlugin_ConcurrentCallPlugin runs many goroutines concurrently through
// CallPlugin against a shared keepRunning client, verifying both correctness
// and absence of data races (-race).
func TestPlugin_ConcurrentCallPlugin(t *testing.T) {
	binaryPath := buildTestPlugin(t)
	cfg := pluginTestConfig(t, filepath.Dir(binaryPath))

	// Seed the cache so all goroutines share one client.
	if _, _, err := cfg.startPlugin(binaryPath, true, os.Stderr); err != nil {
		t.Skipf("could not start plugin: %v", err)
	}
	defer cfg.closePluginClients()

	const numGoroutines = 30
	var wg sync.WaitGroup
	errCh := make(chan error, numGoroutines)

	for i := range numGoroutines {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			L := lua.NewState()
			defer L.Close()
			cfg.LoadPluginFunctions(L, nil)

			code := fmt.Sprintf(`result = CallPlugin(%q, "Add3", %d, %d)`, binaryPath, i, i)
			if err := L.DoString(code); err != nil {
				errCh <- fmt.Errorf("goroutine %d: CallPlugin failed: %w", i, err)
				return
			}
			// Parse the JSON reply (a bare integer like "9") via Lua's tonumber.
			if err := L.DoString(`result = tonumber(result)`); err != nil {
				errCh <- fmt.Errorf("goroutine %d: tonumber failed: %w", i, err)
				return
			}
			want := float64(i + i + 3) // Add3(i, i) == i+i+3
			got := L.GetGlobal("result")
			if got.Type() != lua.LTNumber || float64(got.(lua.LNumber)) != want {
				errCh <- fmt.Errorf("goroutine %d: Add3(%d,%d) expected %v, got %v", i, i, i, want, got)
			}
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Error(err)
	}
}
