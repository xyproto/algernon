package gat

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// IgnoreVendor if using Go 1.5 vendor experiment
var IgnoreVendor = (os.Getenv("GO15VENDOREXPERIMENT") == "1")

type Run struct {
	Tags string
}

func (run Run) RunAll() {
	if IgnoreVendor {
		pkgs := goList()
		run.goTest(pkgs...)
	} else {
		run.goTest("./...")
	}
}

func (run Run) RunOnChange(file string) {
	if isGoFile(file) {
		// TODO: optimization, skip if no test files exist
		packageDir := "./" + filepath.Dir(file) // watchDir = ./
		run.goTest(packageDir)
	}
}

func (run Run) goTest(pkgs ...string) {
	args := []string{"test"}
	if len(run.Tags) > 0 {
		args = append(args, []string{"-tags", run.Tags}...)
	}
	args = append(args, pkgs...)

	command := "go"

	if _, err := os.Stat("Godeps/Godeps.json"); err == nil {
		args = append([]string{"go"}, args...)
		command = "godep"
	}

	cmd := exec.Command(command, args...)
	// cmd.Dir watchDir = ./

	PrintCommand(cmd.Args) // includes "go"

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Println(err)
	}
	PrintCommandOutput(out)

	RedGreen(cmd.ProcessState.Success())
	ShowDuration(cmd.ProcessState.UserTime())
}

func goList() []string {
	cmd := exec.Command("go", "list", "./...")
	out, err := cmd.Output()
	if err != nil {
		log.Println(err)
	}
	allPkgs := strings.Split(string(out), "\n")

	pkgs := []string{}
	// remove packages that contain /vendor/ or are blank (last newline)
	for _, pkg := range allPkgs {
		if len(pkg) != 0 && !strings.Contains(pkg, "/vendor/") {
			pkgs = append(pkgs, pkg)
		}
	}
	return pkgs
}

func isGoFile(file string) bool {
	return filepath.Ext(file) == ".go"
}
