all:
	go build -ldflags "-s -w" lang.go lexer.go parser.go ast.go codegen.go

clean:
	rm lang
