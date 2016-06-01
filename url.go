package main

import (
	log "github.com/sirupsen/logrus"
	"os/exec"
)

// Open an URL with a browser
func openURL(host, colonPort string, https bool) {
	if host == "" {
		host = "localhost"
	}
	prot := "http://"
	if https {
		prot = "https://"
	}
	url := prot + host + colonPort
	log.Info("Running: " + openExecutable + " " + url)
	cmd := exec.Command(openExecutable, url)
	cmd.Run()
}
