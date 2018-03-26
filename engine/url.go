package engine

import (
	"os/exec"

	log "github.com/sirupsen/logrus"
)

// OpenURL tries to open an URL with the system browser
func (ac *Config) OpenURL(host, colonPort string, httpsPrefix bool) {
	if host == "" {
		host = "localhost"
	}
	protocol := "http://"
	if httpsPrefix {
		protocol = "https://"
	}
	url := protocol + host + colonPort
	log.Info("Running: " + ac.openExecutable + " " + url)
	cmd := exec.Command(ac.openExecutable, url)
	cmd.Run()
}
