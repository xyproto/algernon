package main

import (
	. "launchpad.net/gocheck"
	"testing"
)

func Test(t *testing.T) { TestingT(t) }

type LooperSuite struct{}

var _ = Suite(&LooperSuite{})
