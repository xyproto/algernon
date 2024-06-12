package engine

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	log "github.com/sirupsen/logrus"
)

// OpenURL tries to open an URL with the system browser
func (ac *Config) OpenURL(host, cPort string, httpsPrefix bool) {
	// Build the URL
	var sb strings.Builder
	if httpsPrefix {
		sb.WriteString("https://")
	} else {
		sb.WriteString("http://")
	}
	if host == "" {
		sb.WriteString("localhost")
	} else {
		sb.WriteString(host)
	}
	sb.WriteString(cPort)
	url := sb.String()

	cmd := exec.Command(ac.openExecutable, url)
	if ac.openExecutable == "" {
		switch runtime.GOOS {
		case "linux", "solaris":
			cmd = exec.Command("xdg-open", url)
		case "windows":
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		default: // darwin, bsd etc
			cmd = exec.Command("open", url)
		}
	}

	// Open the URL
	log.Info(fmt.Sprintf("Running: %s", cmd.String()))
	cmd.Run()
}
