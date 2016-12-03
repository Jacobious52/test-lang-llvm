package main

import (
	"fmt"

	"llvm.org/llvm/final/bindings/go/llvm"
)

// BinopPrecedence operator precedence for binary operators
var BinopPrecedence = map[string]int{
	"=": 9,
	"<": 10,
	"+": 20,
	"-": 20,
	"*": 40,
}

func tokenPrecedence(tok Token) int {
	if tok.ty == NUMBERLITERAL {
		return -1
	}

	if prec, ok := BinopPrecedence[tok.name]; ok {
		return prec
	}

	return -1
}

// Parser parsers stuff for lexer
type Parser struct {
	lexer     Lexer
	anonCount int
}

func (p *Parser) next() Token {
	return p.lexer.next()
}

func (p *Parser) have(expected string) bool {
	return expected == p.lexer.current.name
}

func (p *Parser) expect(expected string) bool {
	if expected == p.lexer.current.name {
		p.next()
		return true
	}

	p.error("Expected " + expected)
	return false

}

func (p *Parser) error(str string) ExprAST {
	fmt.Printf("%v: %v. Got token '%v' instead\n", p.lexer.current.pos, str, p.lexer.current.name)
	return nil
}

// ParseNumberExpr parses a number
func (p *Parser) ParseNumberExpr() ExprAST {
	result := NumberExprAST{p.lexer.current.value}
	p.next()
	return result
}

// ParseParenExpr ::= '(' expression ')'
func (p *Parser) ParseParenExpr() ExprAST {
	p.next()
	v := p.ParseExpr()
	if v == nil {
		return nil
	}

	if !p.have(")") {
		return p.error("expected ')'")
	}
	p.next()
	return v
}

// ParseIdentifierExpr ::= identifier |
// ::= identifier '(' expression* ')'
func (p *Parser) ParseIdentifierExpr() ExprAST {
	name := p.lexer.current.name
	p.next()

	if !p.have("(") {
		return VariableExprAST{name}
	}

	p.next()
	var args []ExprAST

	if !p.have(")") {
		for {
			if arg := p.ParseExpr(); arg != nil {
				args = append(args, arg)
			} else {
				return nil
			}

			if p.have(")") {
				break
			}

			if !p.have(",") {
				return p.error("Expected ')' or ',' in argument list")
			}

			p.next()
		}
	}

	p.next()

	return CallExprAST{name, args}
}

// ParsePrimaryExpr ::= identifierexpr |
// ::= numberexpr |
// ::= parenexpr
func (p *Parser) ParsePrimaryExpr() ExprAST {

	if p.lexer.current.ty == IDENTIFIER {
		return p.ParseIdentifierExpr()
	}
	if p.lexer.current.ty == NUMBERLITERAL {
		return p.ParseNumberExpr()
	}
	if p.lexer.current.ty == IF {
		return p.ParseIfExpr()
	}
	if p.have("(") {
		return p.ParseParenExpr()
	}

	return p.error("Unknown token when expecting an expression")
}

// ParseBinOpRHS ::= ('+' primary)*
func (p *Parser) ParseBinOpRHS(exprPrec int, LHS ExprAST) ExprAST {
	for {
		tokPrec := tokenPrecedence(p.lexer.current)
		if tokPrec < exprPrec {
			return LHS
		}

		binOp := p.lexer.current
		p.next()

		RHS := p.ParsePrimaryExpr()
		if RHS == nil {
			return nil
		}

		nextPrec := tokenPrecedence(p.lexer.current)
		if tokPrec < nextPrec {
			RHS = p.ParseBinOpRHS(tokPrec+1, RHS)
			if RHS == nil {
				return nil
			}
		}

		LHS = BinaryExprAST{binOp.name, LHS, RHS}
	}
}

// ParseExpr ::= primary binops
func (p *Parser) ParseExpr() ExprAST {
	LHS := p.ParsePrimaryExpr()
	if LHS == nil {
		return nil
	}

	return p.ParseBinOpRHS(0, LHS)
}

// ParsePrototype ::= id '(' id* ')'
func (p *Parser) ParsePrototype() ExprAST {

	if p.lexer.current.ty != IDENTIFIER {
		return p.error("Expected function name in prototype")
	}

	fname := p.lexer.current.name
	p.next()

	if !p.expect(":") {
		return p.error("Expected ':' in prototype")
	}

	var args []string
	if !p.have("{") {
		args = append(args, p.lexer.current.name)
		p.next()

		for !p.have("{") {
			p.expect(",")
			args = append(args, p.lexer.current.name)
			p.next()
		}
	}

	if !p.expect("{") {
		return p.error("Expected '{' in prototype")
	}

	return PrototypeAST{fname, args}
}

// ParseDefinition ::= def 'prototype'
func (p *Parser) ParseDefinition() ExprAST {
	p.expect("def")
	proto := p.ParsePrototype()
	if proto == nil {
		return nil
	}

	if e := p.ParseExpr(); e != nil {
		p.expect("}")
		return FunctionAST{proto.(PrototypeAST), e}
	}
	return nil
}

// ParseExtern ::= 'extern' prototype
func (p *Parser) ParseExtern() ExprAST {
	p.expect("import")
	if ast := p.ParsePrototype(); ast != nil {
		p.expect("}")
		return ast
	}
	return nil
}

// ParseTopLevelExpr ::= expression
func (p *Parser) ParseTopLevelExpr() ExprAST {
	if e := p.ParseExpr(); e != nil {
		var args []string
		p.anonCount++
		return FunctionAST{PrototypeAST{fmt.Sprintf("__jit__%v", p.anonCount), args}, e}
	}
	return nil
}

// ParseIfExpr := 'if' expr '{' expr '}' 'else' expr '}'
func (p *Parser) ParseIfExpr() ExprAST {
	p.expect("if")

	cond := p.ParseExpr()
	if cond == nil {
		return nil
	}

	p.expect(",")

	then := p.ParseExpr()
	if then == nil {
		return nil
	}

	p.expect("else")

	el := p.ParseExpr()
	if el == nil {
		return nil
	}

	return IfExprAST{cond, then, el}
}

// HandleTopLevel - performs error recorvery
func (p *Parser) HandleTopLevel(result ExprAST, cg *CodeGenerator) {
	if result == nil {
		fmt.Println("Parse failed!")
	} else {
		if ir := result.CodeGen(cg); !ir.IsNil() {
		}
	}
}

// REPL - Read Evaluate Print Loop
func (p *Parser) REPL(cg *CodeGenerator) {
	for {
		fmt.Print(">>> ")
		p.lexer.tokenise()

		if p.have("") {
			break
		}

		switch p.lexer.current.ty {
		case DEF:
			if expr := p.ParseDefinition(); expr != nil {
				if ir := expr.CodeGen(cg); ir.IsNil() {
					fmt.Println("error compiling function definition")
				}
			}
		case EXTERN:
			if expr := p.ParseExtern(); expr != nil {
				if ir := expr.CodeGen(cg); !ir.IsNil() {
					cg.Protos[expr.(PrototypeAST).Name] = expr.(PrototypeAST)
				} else {
					fmt.Println("error compiling external function definition")
				}
			}
		case EOF:
			return
		default:
			if expr := p.ParseTopLevelExpr(); expr != nil {
				if ir := expr.CodeGen(cg); !ir.IsNil() {
					cg.JIT.AddModule(cg.Module)
					result := cg.JIT.RunFunction(ir, []llvm.GenericValue{})
					fmt.Println("RET:", result.Float(llvm.DoubleType()))
					cg.JIT.RemoveModule(cg.Module)
				} else {
					fmt.Println("error anon function")
				}
			}
		}
	}
}

// File - parse whole file
func (p *Parser) File(cg *CodeGenerator) {
	p.lexer.tokenise()

	for {
		switch p.lexer.current.ty {
		case DEF:
			p.HandleTopLevel(p.ParseDefinition(), cg)
		case EXTERN:
			if expr := p.ParseExtern(); expr != nil {
				if ir := expr.CodeGen(cg); !ir.IsNil() {
					cg.Protos[expr.(PrototypeAST).Name] = expr.(PrototypeAST)
				} else {
					fmt.Println("error compiling external function definition")
				}
			}
		case EOF:
			return
		default:
			p.HandleTopLevel(p.ParseTopLevelExpr(), cg)
			break
		}
	}
}
