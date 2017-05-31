package permissions

import (
	"testing"

	"github.com/xyproto/pinterface"
)

func TestInterface(t *testing.T) {
	// Check that the value qualifies for the interface
	var _ pinterface.IPermissions = New()
}
