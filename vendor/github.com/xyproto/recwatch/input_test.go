package main

import (
	. "launchpad.net/gocheck"
)

func (s *LooperSuite) TestMixedCaseCommand(c *C) {
	c.Assert(NormalizeCommand(" Exit"), Equals, EXIT)
}

func (s *LooperSuite) TestUnkownCommand(c *C) {
	c.Assert(NormalizeCommand("sudo"), Equals, UNKNOWN)
}
