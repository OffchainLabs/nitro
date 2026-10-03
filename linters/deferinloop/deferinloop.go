// Copyright 2024-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package deferinloop

import (
	"go/ast"
	"go/token"
	"reflect"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name:       "deferinloop",
	Doc:        "check for defer statements directly inside loop bodies",
	Run:        run,
	ResultType: reflect.TypeOf(Result{}),
}

type deferInLoopError struct {
	Pos     token.Position
	Message string
}

type Result struct {
	Errors []deferInLoopError
}

func run(pass *analysis.Pass) (interface{}, error) {
	var ret Result
	for _, f := range pass.Files {
		ast.Inspect(f, func(node ast.Node) bool {
			switch node.(type) {
			case *ast.ForStmt, *ast.RangeStmt:
				findDirectDefers(pass, f, node, &ret)
				return false // we handle children ourselves
			}
			return true
		})
	}
	return ret, nil
}

// findDirectDefers walks the body of a loop looking for defer statements
// that are not nested inside a function literal.
func findDirectDefers(pass *analysis.Pass, file *ast.File, loop ast.Node, ret *Result) {
	var body *ast.BlockStmt
	switch l := loop.(type) {
	case *ast.ForStmt:
		body = l.Body
	case *ast.RangeStmt:
		body = l.Body
	}
	if body == nil {
		return
	}
	ast.Inspect(body, func(node ast.Node) bool {
		switch node.(type) {
		case *ast.FuncLit:
			// Don't descend into anonymous functions — defer inside
			// func literals is fine since the function scope ends each iteration.
			return false
		case *ast.DeferStmt:
			err := deferInLoopError{
				Pos:     pass.Fset.Position(node.Pos()),
				Message: "defer called directly in loop body, consider wrapping in an immediately-invoked function literal",
			}
			ret.Errors = append(ret.Errors, err)
			pass.Report(analysis.Diagnostic{
				Pos:      pass.Fset.File(file.Pos()).Pos(err.Pos.Offset),
				Message:  err.Message,
				Category: "deferinloop",
			})
		}
		return true
	})
}
