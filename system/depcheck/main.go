package main

import (
	"errors"
	"fmt"
	"github.com/xyproto/permissions2"
	"github.com/xyproto/simpleredis"
)

type VersionInfo struct {
	name    string
	current float64
	target  float64
}

func New(name string, current, target float64) *VersionInfo {
	return &VersionInfo{name, current, target}
}

func (v *VersionInfo) Check() error {
	if v.current != v.target {
		return errors.New(fmt.Sprintf("needs version %.1f", v.target))
	}
	return nil
}

func (v *VersionInfo) Status() {
	fmt.Print(v.name + "...")
	if err := v.Check(); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("ok")
	}
}

func main() {
	New("permissions2", permissions.Version, 2.1).Status()
	New("simpleredis", simpleredis.Version, 1.1).Status()
}
