// API version number check
package main

import (
	"fmt"
	"github.com/xyproto/permissionbolt"
	"github.com/xyproto/permissions2"
	"github.com/xyproto/permissionsql"
	"github.com/xyproto/pinterface"
	"github.com/xyproto/simplebolt"
	"github.com/xyproto/simplemaria"
	"github.com/xyproto/simpleredis"
)

// VersionInfo helps to keep track of package names and versions
type VersionInfo struct {
	name    string
	current float64
	target  float64
}

// New takes the name of the go package, the current and the desired version
func New(name string, current, target float64) *VersionInfo {
	return &VersionInfo{name, current, target}
}

// Check compares the current and target version
func (v *VersionInfo) Check() error {
	if v.current != v.target {
		return fmt.Errorf("is %.1f, needs version %.1f", v.current, v.target)
	}
	return nil
}

// Status reports if the current version is satisfactory
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
	New("pinterface", pinterface.Version, 3.0).Status()
}
