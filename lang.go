package main

import "os"

func main() {
	//f, _ := os.Open("test.lang")
	//defer f.Close()

	cg := NewCodeGen("main")

	parser := Parser{NewLexer(os.Stdin), -1}
	parser.REPL(cg)
	cg.Module.Dump()
}
