package files

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/xyproto/env/v2"
)

// Run tries to run the given command, without using a shell.
// No output is returned, only an error, if something went wrong.
func Run(commandString string) error {
	parts := strings.Fields(commandString)
	if len(parts) == 0 {
		return errors.New("empty command")
	}
	if WhichCached(parts[0]) == "" {
		return fmt.Errorf("could not find %s in path", parts[0])
	}
	cmd := exec.Command(parts[0], parts[1:]...)
	return cmd.Run()
}

// Shell runs the given command with /bin/sh and returns the combined output
func Shell(shellCommand string) (string, error) {
	const shellPath = "/bin/sh"
	if !IsExecutable(shellPath) {
		if !Exists(shellPath) {
			return "", fmt.Errorf("%s does not exist", shellPath)
		}
		return "", fmt.Errorf("%s is not executable", shellPath)
	}
	cmd := exec.Command(shellPath, "-c", shellCommand)
	cmd.Env = env.Environ()
	data, err := cmd.CombinedOutput()
	return string(bytes.TrimSpace(data)), err
}

// Bash runs the given command with /bin/bash, or "bash" in the $PATH,
// and returns the combined output.
func Bash(bashCommand string) (string, error) {
	const binBash = "/bin/bash"
	var bashPath = binBash
	if !IsExecutable(binBash) {
		bashPath = WhichCached("bash")
		if !IsExecutable(bashPath) {
			if !Exists(binBash) {
				return "", fmt.Errorf("%s does not exist", binBash)
			}
			if !Exists(bashPath) {
				return "", fmt.Errorf("%s does not exist", bashPath)
			}
			return "", fmt.Errorf("%s is not executable", binBash)
		}
	}
	cmd := exec.Command(bashPath, "-c", bashCommand)
	cmd.Env = env.Environ()
	data, err := cmd.CombinedOutput()
	return string(bytes.TrimSpace(data)), err
}

// Fish runs the given command with /usr/bin/fish, or "fish" in the $PATH,
// and returns the combined output.
func Fish(fishCommand string) (string, error) {
	const usrBinFish = "/usr/bin/fish"
	var fishPath = usrBinFish
	if !IsExecutable(usrBinFish) {
		fishPath = WhichCached("fish")
		if !IsExecutable(fishPath) {
			if !Exists(usrBinFish) {
				return "", fmt.Errorf("%s does not exist", usrBinFish)
			}
			if !Exists(fishPath) {
				return "", fmt.Errorf("%s does not exist", fishPath)
			}
			return "", fmt.Errorf("%s is not executable", usrBinFish)
		}
	}
	cmd := exec.Command(fishPath, "-c", fishCommand)
	cmd.Env = env.Environ()
	data, err := cmd.CombinedOutput()
	return string(bytes.TrimSpace(data)), err
}
