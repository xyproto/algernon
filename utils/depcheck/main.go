package main

import (
	"errors"
	"fmt"
	"github.com/xyproto/permissions2"
	"github.com/xyproto/simplebolt"
	"github.com/xyproto/simpleredis"
)

// Over engineered version number check!

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
		return errors.New(fmt.Sprintf("is %.1f, needs version %.1f",
			v.current, v.target))
	}
	return nil
}

func (v *VersionInfo) Status() {
	fmt.Print("\t" + v.name + "...")
	if err := v.Check(); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("ok")
	}
}

func main() {
	fmt.Println("Dependency check")
	New("permissions2", permissions.Version, 2.2).Status()
	New("simpleredis", simpleredis.Version, 1.2).Status()
	New("simplebolt", simplebolt.Version, 1.0).Status()
}
