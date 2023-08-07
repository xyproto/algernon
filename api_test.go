// API version number check
package main

import (
	"fmt"
	"testing"

	"github.com/bmizerany/assert"
	"github.com/xyproto/algernon/engine"
	"github.com/xyproto/permissionbolt/v2"
	permissions "github.com/xyproto/permissions2/v2"
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
	assert.Equal(t, New("simplebolt", simplebolt.Version, 5.1).Check(), nil)
	assert.Equal(t, New("permissionbolt", permissionbolt.Version, 2.6).Check(), nil)
	assert.Equal(t, New("simpleredis", simpleredis.Version, 2.6).Check(), nil)
	assert.Equal(t, New("permissions2", permissions.Version, 2.6).Check(), nil)
	assert.Equal(t, New("pinterface", pinterface.Version, 5.3).Check(), nil)
	assert.Equal(t, New("engine", engine.Version, 2.0).Check(), nil)

	// These adds many dependencies when testing
	// assert.Equal(t, New("simplemaria", simplemaria.Version, 3.0).Check(), nil)
	// assert.Equal(t, New("permissionsql", permissionsql.Version, 2.0).Check(), nil)
	// assert.Equal(t, New("simplehstore", simplehstore.Version, 2.3).Check(), nil)
	// assert.Equal(t, New("pstore", pstore.Version, 3.1).Check(), nil)
}
