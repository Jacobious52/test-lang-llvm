package main

import "llvm.org/llvm/final/bindings/go/llvm"

// ExprAST - represents a expression interface
type ExprAST interface {
	CodeGen(cg *CodeGenerator) llvm.Value
}

// NumberExprAST - represents a constant number
type NumberExprAST struct {
	Value float64
}

// VariableExprAST - represents a variable name
type VariableExprAST struct {
	Name string
}

// BinaryExprAST - represents a binary expression
type BinaryExprAST struct {
	Operator string
	LHS, RHS ExprAST
}

// CallExprAST - represents a function call expression
type CallExprAST struct {
	Callee string
	Args   []ExprAST
}

// PrototypeAST - represents a function definition itself.
type PrototypeAST struct {
	Name string
	Args []string
}

// FunctionAST - represents a function definition itself.
type FunctionAST struct {
	Proto PrototypeAST
	Body  ExprAST
}

// IfExprAST - represents an if/else statement
type IfExprAST struct {
	cond, then, el ExprAST
}
