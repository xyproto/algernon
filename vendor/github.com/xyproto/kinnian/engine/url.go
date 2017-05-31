package engine

import (
	"os/exec"

	log "github.com/sirupsen/logrus"
)

// Open an URL with a browser
func (ac *Config) OpenURL(host, colonPort string, https bool) {
	if host == "" {
		host = "localhost"
	}
	prot := "http://"
	if https {
		prot = "https://"
	}
	url := prot + host + colonPort
	log.Info("Running: " + ac.openExecutable + " " + url)
	cmd := exec.Command(ac.openExecutable, url)
	cmd.Run()
}
