package main

import (
	"fmt"
	"os"

	"llvm.org/llvm/final/bindings/go/llvm"
)

// CodeGenerator llvm thing
type CodeGenerator struct {
	Context     llvm.Context
	Builder     llvm.Builder
	Module      llvm.Module
	Optimizer   llvm.PassManager
	JIT         llvm.ExecutionEngine
	NamedValues map[string]llvm.Value
	Protos      map[string]PrototypeAST
}

// NewCodeGen creates a new code gernetaor
func NewCodeGen(moduleName string) *CodeGenerator {
	module := llvm.NewModule(moduleName)
	module.SetDataLayout(module.DataLayout())

	jit, _ := llvm.NewExecutionEngine(module)
	cg := &CodeGenerator{llvm.NewContext(), llvm.NewBuilder(), module, llvm.NewFunctionPassManagerForModule(module), jit, make(map[string]llvm.Value), make(map[string]PrototypeAST)}

	native := llvm.InitializeNativeTarget()
	if native != nil {
		fmt.Println(native)
		os.Exit(-1)
	}

	cg.JIT.AddModule(cg.Module)

	cg.Optimizer = llvm.NewFunctionPassManagerForModule(cg.Module)

	cg.Optimizer.AddPromoteMemoryToRegisterPass()
	cg.Optimizer.AddInstructionCombiningPass()
	cg.Optimizer.AddReassociatePass()
	cg.Optimizer.AddGVNPass()
	cg.Optimizer.AddCFGSimplificationPass()
	cg.Optimizer.AddTailCallEliminationPass()

	cg.Optimizer.InitializeFunc()

	return cg
}

func (cg CodeGenerator) error(str string) llvm.Value {
	fmt.Println(str)
	return llvm.Value{nil}
}

func (cg *CodeGenerator) getFunction(name string) llvm.Value {
	if f := cg.Module.NamedFunction(name); !f.IsNil() {
		return f
	}

	if ast, ok := cg.Protos[name]; ok {
		return ast.CodeGen(cg)
	}

	return llvm.Value{nil}
}

// CodeGen - outputs code for NumberExprAST
func (ast NumberExprAST) CodeGen(cg *CodeGenerator) llvm.Value {
	return llvm.ConstFloat(llvm.DoubleType(), ast.Value)
}

// CodeGen - outputs code for VariableExprAST
func (ast VariableExprAST) CodeGen(cg *CodeGenerator) llvm.Value {
	if val, ok := cg.NamedValues[ast.Name]; ok {
		return val
	}
	cg.error("Unknown variable name " + ast.Name)
	return llvm.Value{nil}
}

// CodeGen - outputs code for BinaryExprAST
func (ast BinaryExprAST) CodeGen(cg *CodeGenerator) llvm.Value {
	l := ast.LHS.CodeGen(cg)
	r := ast.RHS.CodeGen(cg)

	if l.IsNil() || r.IsNil() {
		return llvm.Value{nil}
	}

	switch ast.Operator {
	case "+":
		return cg.Builder.CreateFAdd(l, r, "addtmp")
	case "-":
		return cg.Builder.CreateFSub(l, r, "subtmp")
	case "*":
		return cg.Builder.CreateFMul(l, r, "multmp")
	case "/":
		return cg.Builder.CreateFDiv(l, r, "divtmp")
	case "<":
		l = cg.Builder.CreateFCmp(llvm.FloatULT, l, r, "cmptmp")
		return cg.Builder.CreateUIToFP(l, llvm.DoubleType(), "booltmp")
	case "=":
		l = cg.Builder.CreateFCmp(llvm.FloatUEQ, l, r, "cmptemp")
		return cg.Builder.CreateUIToFP(l, llvm.DoubleType(), "booltmp")
	default:
		return cg.error("invalid binary operator")
	}
}

// CodeGen - outputs code for CallExprAST
func (ast CallExprAST) CodeGen(cg *CodeGenerator) llvm.Value {
	function := cg.getFunction(ast.Callee)
	if function.IsNil() {
		return cg.error("Unknown function referenced: " + ast.Callee)
	}

	if function.ParamsCount() != len(ast.Args) {
		return cg.error(fmt.Sprintln("Incorrect number arguments passed. Expected", function.ParamsCount(), " got", len(ast.Args)))
	}

	var args []llvm.Value
	for _, arg := range ast.Args {
		argV := arg.CodeGen(cg)
		if argV.IsNil() {
			return llvm.Value{nil}
		}
		args = append(args, arg.CodeGen(cg))
	}
	return cg.Builder.CreateCall(function, args, "calltmp")
}

// CodeGen - outputs code for PrototypeAST
func (ast PrototypeAST) CodeGen(cg *CodeGenerator) llvm.Value {
	doubles := make([]llvm.Type, len(ast.Args))
	for i := 0; i < len(ast.Args); i++ {
		doubles[i] = llvm.DoubleType()
	}

	ft := llvm.FunctionType(llvm.DoubleType(), doubles, false)
	llvm.AddFunction(cg.Module, ast.Name, ft)
	function := cg.Module.NamedFunction(ast.Name)

	if function.Name() != ast.Name {
		function.EraseFromParentAsFunction()
		function = cg.getFunction(ast.Name)
	}

	if function.BasicBlocksCount() != 0 {
		return cg.error("Redefinition of function")
	}

	if function.ParamsCount() != len(ast.Args) {
		return cg.error("Redefinition of function with different number of args")
	}

	for i, arg := range function.Params() {
		arg.SetName(ast.Args[i])
	}

	return function
}

// CodeGen - outputs code for FunctionAST
func (ast FunctionAST) CodeGen(cg *CodeGenerator) llvm.Value {

	proto := ast.Proto

	cg.Protos[proto.Name] = ast.Proto
	function := cg.getFunction(proto.Name)

	if function.IsNil() {
		function.EraseFromParentAsFunction()
		return llvm.Value{nil}
	}

	bb := llvm.AddBasicBlock(function, "entry")
	cg.Builder.SetInsertPoint(bb, bb.FirstInstruction())

	for k := range cg.NamedValues {
		delete(cg.NamedValues, k)
	}

	for i, arg := range function.Params() {
		cg.NamedValues[arg.Name()] = function.Param(i)
	}

	if retVal := ast.Body.CodeGen(cg); !retVal.IsNil() {
		cg.Builder.CreateRet(retVal)

		llvm.VerifyFunction(function, llvm.PrintMessageAction)
		cg.Optimizer.RunFunc(function)

		return function
	}

	function.EraseFromParentAsFunction()
	return llvm.Value{nil}
}

// CodeGen for if
func (ast IfExprAST) CodeGen(cg *CodeGenerator) llvm.Value {
	cond := ast.cond.CodeGen(cg)
	if cond.IsNil() {
		return llvm.Value{nil}
	}

	cond = cg.Builder.CreateFCmp(llvm.FloatONE, cond, llvm.ConstFloat(llvm.DoubleType(), 0), "ifcond")

	function := cg.Builder.GetInsertBlock().Parent()

	thenBlock := llvm.AddBasicBlock(function, "then")
	elseBlock := llvm.AddBasicBlock(function, "else")
	mergeBlock := llvm.AddBasicBlock(function, "merge")

	cg.Builder.CreateCondBr(cond, thenBlock, elseBlock)

	cg.Builder.SetInsertPointAtEnd(thenBlock)
	then := ast.then.CodeGen(cg)
	if then.IsNil() {
		return llvm.Value{nil}
	}

	cg.Builder.CreateBr(mergeBlock)
	thenBlock = cg.Builder.GetInsertBlock()

	cg.Builder.SetInsertPointAtEnd(elseBlock)
	el := ast.el.CodeGen(cg)
	if el.IsNil() {
		return llvm.Value{nil}
	}
	cg.Builder.CreateBr(mergeBlock)
	elseBlock = cg.Builder.GetInsertBlock()

	cg.Builder.SetInsertPointAtEnd(mergeBlock)
	phiNode := cg.Builder.CreatePHI(llvm.DoubleType(), "iftmp")
	phiNode.AddIncoming([]llvm.Value{then}, []llvm.BasicBlock{thenBlock})
	phiNode.AddIncoming([]llvm.Value{el}, []llvm.BasicBlock{elseBlock})
	return phiNode
}
