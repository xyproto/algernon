package main

import (
	"errors"
	"fmt"
	"github.com/xyproto/permissionbolt"
	"github.com/xyproto/permissions2"
	"github.com/xyproto/permissionsql"
	"github.com/xyproto/pinterface"
	"github.com/xyproto/simplebolt"
	"github.com/xyproto/simplemaria"
	"github.com/xyproto/simpleredis"
)

// API version number check

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
	fmt.Println("API dependency check")
	New("simplebolt", simplebolt.Version, 3.0).Status()
	New("permissionbolt", permissionbolt.Version, 2.0).Status()
	New("simpleredis", simpleredis.Version, 2.0).Status()
	New("permissions", permissions.Version, 2.2).Status()
	New("simplemaria", simplemaria.Version, 2.0).Status()
	New("permissionsql", permissionsql.Version, 2.0).Status()
	New("pinterface", pinterface.Version, 2.0).Status()
}
