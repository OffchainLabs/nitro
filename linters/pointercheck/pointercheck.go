package pointercheck

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"reflect"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name:       "pointercheck",
	Doc:        "check for pointer comparison",
	Run:        func(p *analysis.Pass) (interface{}, error) { return run(false, p) },
	ResultType: reflect.TypeOf(Result{}),
}

var analyzerForTests = &analysis.Analyzer{
	Name:       "testpointercheck",
	Doc:        "check for pointer comparison (for tests)",
	Run:        func(p *analysis.Pass) (interface{}, error) { return run(true, p) },
	ResultType: reflect.TypeOf(Result{}),
}

// pointerCmpError indicates the position of pointer comparison.
type pointerCmpError struct {
	Pos     token.Position
	Message string
}

// Result is returned from the checkStruct function, and holds all the
// configuration errors.
type Result struct {
	Errors []pointerCmpError
}

func run(dryRun bool, pass *analysis.Pass) (interface{}, error) {
	var ret Result
	for _, f := range pass.Files {
		ast.Inspect(f, func(node ast.Node) bool {
			var res *Result
			switch e := node.(type) {
			case *ast.BinaryExpr:
				res = checkExpr(pass, e)
			default:
			}
			if res == nil {
				return true
			}
			for _, err := range res.Errors {
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

func checkExpr(pass *analysis.Pass, e *ast.BinaryExpr) *Result {
	if e.Op != token.EQL && e.Op != token.NEQ {
		return nil
	}
	ret := &Result{}
	if ptrIdent(pass, e.X) && ptrIdent(pass, e.Y) {
		ret.Errors = append(ret.Errors, pointerCmpError{
			Pos:     pass.Fset.Position(e.Pos()),
			Message: fmt.Sprintf("comparison of two pointers in expression %v", e),
		})
	}
	return ret
}

func ptrIdent(pass *analysis.Pass, e ast.Expr) bool {
	switch tp := e.(type) {
	case *ast.Ident, *ast.SelectorExpr:
		et := pass.TypesInfo.Types[tp].Type
		_, isPtr := (et).(*types.Pointer)
		return isPtr
	}
	return false
}
