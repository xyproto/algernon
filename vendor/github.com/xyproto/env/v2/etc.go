package env

import (
	"bytes"
	"fmt"
	"os"
	"strings"
)

// isDir checks if the given path is a directory (could also be a symlink)
func isDir(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}

// EtcEnvironment tries to find the value of the given variable name in /etc/environment.
// The variable name could be ie. "JAVA_HOME".
func EtcEnvironment(envVarName string) (string, error) {
	// Find the definition of ie. JAVA_HOME within /etc/environment
	data, err := os.ReadFile("/etc/environment")
	if err != nil {
		return "", err
	}
	lines := bytes.Split(data, []byte{'\n'})
	for _, line := range lines {
		if bytes.Contains(line, []byte(envVarName)) && bytes.Count(line, []byte("=")) == 1 {
			fields := bytes.SplitN(line, []byte("="), 2)
			javaPath := strings.TrimSpace(string(fields[1]))
			if !isDir(javaPath) {
				continue
			}
			return javaPath, nil
		}
	}
	return "", fmt.Errorf("could not find the value of %s in /etc/environment", envVarName)
}
