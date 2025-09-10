// Copyright 2023-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package koanf

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"reflect"

	"golang.org/x/tools/go/analysis"
)

var (
	errUnused   = errors.New("unused")
	errMismatch = errors.New("mismmatched field name and tag in a struct")
	// e.g. f.Int("max-sz", DefaultBatchPosterConfig.MaxSize, "maximum batch size")
	errIncorrectFlag = errors.New("mismatching flag initialization")
)

var Analyzer = &analysis.Analyzer{
	Name:       "koanfcheck",
	Doc:        "check for koanf misconfigurations",
	Run:        run,
	ResultType: reflect.TypeOf(Result{}),
}

// koanfError indicates the position of an error in configuration.
type koanfError struct {
	Pos     token.Pos
	Message string
	err     error
}

// Result is returned from the checkStruct function, and holds all the
// configuration errors.
type Result struct {
	Errors []koanfError
}

func run(pass *analysis.Pass) (interface{}, error) {
	var (
		ret Result
		cnt = make(map[string]int)
		// koanfFields map contains all the struct koanfFields that have koanf tag.
		// It identifies field as "{pkgName}.{structName}.{field_Name}".
		// e.g. "a.BatchPosterConfig.Enable", "a.BatchPosterConfig.MaxSize"
		koanfFields = koanfFields(pass)
	)
	for _, f := range pass.Files {
		ast.Inspect(f, func(node ast.Node) bool {
			var res Result
			switch v := node.(type) {
			case *ast.StructType:
				res = checkStruct(pass, v)
			case *ast.FuncDecl:
				res = checkFlagDefs(pass, v, cnt)
			case *ast.SelectorExpr:
				handleSelector(pass, v, 1, cnt)
			case *ast.IfStmt:
				if se, ok := v.Cond.(*ast.SelectorExpr); ok {
					handleSelector(pass, se, 1, cnt)
				}
			case *ast.CompositeLit:
				handleComposite(pass, v, cnt)
			default:
			}
			ret.Errors = append(ret.Errors, res.Errors...)
			return true
		})
	}
	for k := range koanfFields {
		if cnt[k] == 0 {
			ret.Errors = append(ret.Errors,
				koanfError{
					Pos:     koanfFields[k],
					Message: fmt.Sprintf("field %v not used", k),
					err:     errUnused,
				})
		}
	}
	for _, err := range ret.Errors {
		pass.Report(analysis.Diagnostic{
			Pos:      err.Pos,
			Message:  err.Message,
			Category: "koanf",
		})
	}
	return ret, nil
}
