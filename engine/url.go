package engine

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/pkg/browser"
	log "github.com/sirupsen/logrus"
)

// OpenURL tries to open an URL with the system browser
func (ac *Config) OpenURL(host, cPort string, httpsPrefix bool) error {
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

	if ac.openExecutable != "" {
		// Custom command for opening URLs
		cmd := exec.Command(ac.openExecutable, url)
		log.Info(fmt.Sprintf("Opening %q with %q", url, cmd.String()))
		return cmd.Run()
	}

	log.Info(fmt.Sprintf("Opening %q in a browser", url))
	return browser.OpenURL(url)
}
