package main

import (
	. "github.com/xyproto/term"
)

func main() {
	Init()
	Clear()
	Write(10, 10, "hi", White, Green|Bold)
	SetFg(Red)
	DrawRaw(20, 12, ",,..----..,,\n|          |\n\\__________/")
	Flush()
	WaitForKey()
	Close()
}
