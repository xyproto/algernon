package main

import (
	"bufio"
	"fmt"
	".."
)

func main(){
	stdOut := bufio.NewWriter(colorable.NewColorableStdout())

	fmt.Fprint(stdOut,"\x1B[3GMove to 3rd Column\n")
	fmt.Fprint(stdOut,"\x1B[1;2HMove to 2nd Column on 1st Line\n")
	stdOut.Flush()
}
