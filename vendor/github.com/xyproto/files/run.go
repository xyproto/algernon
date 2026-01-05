package files

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
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
