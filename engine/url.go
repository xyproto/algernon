package engine

import (
	"os/exec"
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

	// Open the URL
	log.Info("Running: " + ac.openExecutable + " " + url)
	cmd := exec.Command(ac.openExecutable, url)
	cmd.Run()
}
