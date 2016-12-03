package main

import (
	"io"
	"strconv"
	"text/scanner"
)

// Type proves token type
type Type int

// The Type of token
const (
	EOF Type = -(iota + 1)
	DEF
	EXTERN
	IF
	ELSE
	IDENTIFIER
	NUMBERLITERAL
	SYMBOL
)

// Token Represents a token
type Token struct {
	name  string
	ty    Type
	pos   scanner.Position
	value float64
}

// Lexer handles everything to do with tokenising a source file
type Lexer struct {
	reader  io.Reader
	tokens  []Token
	index   int
	current Token
}

// NewLexer creates a new lexer with a reader source
func NewLexer(reader io.Reader) Lexer {
	var tokens []Token
	var token Token
	return Lexer{reader, tokens, 0, token}
}

func (l *Lexer) next() Token {
	l.index++
	l.current = l.tokens[l.index]
	return l.current
}

func whatType(s string) Type {
	switch s {
	case "def":
		return DEF
	case "import":
		return EXTERN
	case "if":
		return IF
	case "else":
		return ELSE
	case "(", ")", ",", "+", "-", "/", "*", ":", ";", "=":
		return SYMBOL
	}

	if _, err := strconv.ParseFloat(s, 64); err == nil {
		return NUMBERLITERAL
	}

	return IDENTIFIER
}

// tokenise flushes the token stream and reads from the reader again.
// resets lexer
func (l *Lexer) tokenise() {
	var s scanner.Scanner
	s.Init(l.reader)
	l.tokens = nil
	var tok rune
	for tok != scanner.EOF {
		tok = s.Scan()

		var value float64
		ty := whatType(s.TokenText())
		if ty == NUMBERLITERAL {
			value, _ = strconv.ParseFloat(s.TokenText(), 64)
		}
		l.tokens = append(l.tokens, Token{s.TokenText(), ty, s.Pos(), value})
	}
	l.tokens = append(l.tokens, Token{"EOF", EOF, s.Pos(), 0})

	l.index = 0
	l.current = l.tokens[l.index]
}
