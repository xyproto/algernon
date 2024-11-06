// API version number check
package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/xyproto/algernon/engine"
	"github.com/xyproto/permissionbolt/v2"
	"github.com/xyproto/permissions"
	"github.com/xyproto/pinterface"
	"github.com/xyproto/simplebolt"
	"github.com/xyproto/simpleredis/v2"
	//"github.com/xyproto/permissionsql"
	//"github.com/xyproto/pstore"
	//"github.com/xyproto/simplehstore"
	//"github.com/xyproto/simplemaria"
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

func TestAPI(t *testing.T) {
	if err := New("simplebolt", simplebolt.Version, 5.1).Check(); err != nil {
		t.Error(err)
	}
	if err := New("permissionbolt", permissionbolt.Version, 2.6).Check(); err != nil {
		t.Error(err)
	}
	if err := New("simpleredis", simpleredis.Version, 2.6).Check(); err != nil {
		t.Error(err)
	}
	if !strings.HasPrefix(permissions.VersionString, "1.") {
		t.Error(fmt.Errorf("permissions is %q, requires %q", permissions.VersionString, "1.*"))
	}
	if err := New("pinterface", pinterface.Version, 5.3).Check(); err != nil {
		t.Error(err)
	}
	if err := New("engine", engine.Version, 2.0).Check(); err != nil {
		t.Error(err)
	}

	// These adds many dependencies when testing
	// if err := New("simplemaria", simplemaria.Version, 3.0).Check(); err != nil {
	// 	t.Error(err)
	// }
	// if err := New("permissionsql", permissionsql.Version, 2.0).Check(); err != nil {
	// 	t.Error(err)
	// }
	// if err := New("simplehstore", simplehstore.Version, 2.3).Check(); err != nil {
	// 	t.Error(err)
	// }
	// if err := New("pstore", pstore.Version, 3.1).Check(); err != nil {
	// 	t.Error(err)
	// }
}
