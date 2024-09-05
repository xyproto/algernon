package engine

import (
	"os/exec"
	"strings"

	"github.com/pkg/browser"
	"github.com/sirupsen/logrus"
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
		logrus.Infof("Opening %q with %q", url, cmd.String())
		return cmd.Run()
	}

	logrus.Infof("Opening %q in a browser", url)
	return browser.OpenURL(url)
}
