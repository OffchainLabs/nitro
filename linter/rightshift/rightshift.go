package main

import (
	"go/ast"
	"go/token"
	"reflect"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/singlechecker"
)

var Analyzer = &analysis.Analyzer{
	Name:       "rightshift",
	Doc:        "check for 1 >> x operation",
	Run:        func(p *analysis.Pass) (interface{}, error) { return run(false, p) },
	ResultType: reflect.TypeOf(Result{}),
}

var analyzerForTests = &analysis.Analyzer{
	Name:       "testrightshift",
	Doc:        "check for pointer comparison (for tests)",
	Run:        func(p *analysis.Pass) (interface{}, error) { return run(true, p) },
	ResultType: reflect.TypeOf(Result{}),
}

// rightShiftError indicates the position of pointer comparison.
type rightShiftError struct {
	Pos     token.Position
	Message string
}

// Result is returned from the checkStruct function, and holds all rightshift
// operations.
type Result struct {
	Errors []rightShiftError
}

func run(dryRun bool, pass *analysis.Pass) (interface{}, error) {
	var ret Result
	for _, f := range pass.Files {
		ast.Inspect(f, func(node ast.Node) bool {
			be, ok := node.(*ast.BinaryExpr)
			if !ok {
				return true
			}
			// Check if the expression is '1 >> x'.
			if be.Op == token.SHR && isOne(be.X) {
				err := rightShiftError{
					Pos:     pass.Fset.Position(be.Pos()),
					Message: "found rightshift ('1 >> x') expression, did you mean '1 << x' ?",
				}
				ret.Errors = append(ret.Errors, err)
				if !dryRun {
					pass.Report(analysis.Diagnostic{
						Pos:      pass.Fset.File(f.Pos()).Pos(err.Pos.Offset),
						Message:  err.Message,
						Category: "pointercheck",
					})
				}
			}
			return true
		},
		)
	}
	return ret, nil
}

// isOne checks if the expression is a constant 1.
func isOne(expr ast.Expr) bool {
	bl, ok := expr.(*ast.BasicLit)
	return ok && bl.Kind == token.INT && bl.Value == "1"
}

func main() {
	singlechecker.Main(Analyzer)
}
