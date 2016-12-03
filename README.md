# test-lang-llvm
Test (functional) programming language for learning compilation to llvm ir.
Not a serious langauge. For the purposes of learning only

Built in Go programming langauge. Uses llvm Go bindings to generator llvm ir. Bindings must be in go path to compile compiler yourself
Makes use of trivial llvm optimisation like constant folding and tail recusion.

Only one type at this time, doubles
Currently compiler runs in JIT mode. Cntrl^D to insert EOF and compile function or run expression

Example: test.lang

```
def factIter: n, product {
    if n = 0, product
    else factIter(n-1, n*product)
}

def fact: n { factIter(n, 1) }

def fib: n {
    if n < 2, n
    else fib(n-1) + fib(n-2)
}

def rand: {4}

def inc: x { x + 1 }
```
