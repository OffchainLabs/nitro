// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package namedfieldsinit

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"reflect"

	"golang.org/x/tools/go/analysis"
)

const fieldThreshold = 5 // Require named fields for structs with more than 5 fields

var Analyzer = &analysis.Analyzer{
	Name:       "namedfieldsinit",
	Doc:        "check that struct literals with many fields use named field initialization",
	Run:        run,
	ResultType: reflect.TypeOf(Result{}),
}

type namedFieldsInitError struct {
	Pos     token.Pos
	Message string
}

type Result struct {
	Errors []namedFieldsInitError
}

func run(pass *analysis.Pass) (interface{}, error) {
	var ret Result
	for _, f := range pass.Files {
		ast.Inspect(f, func(node ast.Node) bool {
			cl, ok := node.(*ast.CompositeLit)
			if !ok {
				return true
			}

			// Get the type of the composite literal
			typ := pass.TypesInfo.Types[cl].Type
			if typ == nil {
				return true
			}

			// Check if it's a struct type
			structType, ok := typ.Underlying().(*types.Struct)
			if !ok {
				// Check for pointer to struct
				if ptr, ok := typ.Underlying().(*types.Pointer); ok {
					structType, ok = ptr.Elem().Underlying().(*types.Struct)
					if !ok {
						return true
					}
				} else {
					return true
				}
			}

			numFields := structType.NumFields()
			if numFields <= fieldThreshold {
				return true
			}

			// Check if any element is using positional (unnamed) initialization
			hasUnnamedFields := false
			for _, elt := range cl.Elts {
				if _, ok := elt.(*ast.KeyValueExpr); !ok {
					hasUnnamedFields = true
					break
				}
			}

			if hasUnnamedFields {
				typeName := pass.TypesInfo.Types[cl].Type.String()
				ret.Errors = append(ret.Errors, namedFieldsInitError{
					Pos: cl.Pos(),
					Message: fmt.Sprintf("struct %q has %d fields and must use named field initialization (threshold: %d)",
						typeName, numFields, fieldThreshold),
				})
			}

			return true
		})
	}

	for _, err := range ret.Errors {
		pass.Report(analysis.Diagnostic{
			Pos:      err.Pos,
			Message:  err.Message,
			Category: "namedfieldsinit",
		})
	}

	return ret, nil
}
