package engine

import (
	"os/exec"
	"strings"

	"github.com/pkg/browser"
	"github.com/sirupsen/logrus"
	"github.com/xyproto/algernon/utils"
)

// OpenURL tries to open an URL with the system browser
func (ac *Config) OpenURL(host, cPort string, httpsPrefix bool) error {
	// Build the URL using IPv6-safe formatting
	var sb strings.Builder
	if httpsPrefix {
		sb.WriteString("https://")
	} else {
		sb.WriteString("http://")
	}
	// Combine host and port, then format for URL (handles IPv6 brackets)
	sb.WriteString(utils.HostPortToURL(utils.JoinHostPort(host, cPort)))
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
