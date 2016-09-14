package gat_test

import (
	. "launchpad.net/gocheck"
	"testing"
)

func Test(t *testing.T) { TestingT(t) }

type GatSuite struct{}

var _ = Suite(&GatSuite{})
