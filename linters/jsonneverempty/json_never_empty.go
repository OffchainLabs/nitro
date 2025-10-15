// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
//
// Based on https://github.com/andydotdev/omitlint

package jsonneverempty

import (
	"fmt"
	"go/ast"
	"go/types"
	"reflect"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name:     "jsonneverempty",
	Doc:      "check if the `omitempty` tag is used for fields that cannot be empty",
	Run:      run,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

func run(pass *analysis.Pass) (interface{}, error) {
	pkgAst, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, fmt.Errorf("`inspect.Analyzer` hasn't been run or didn't return AST for the package `%v`", pass.Pkg.Name())
	}
	pkgAst.Preorder([]ast.Node{(*ast.StructType)(nil)}, func(node ast.Node) {
		structType, isStructType := node.(*ast.StructType)
		if !isStructType {
			panic("node type filtering doesn't work correctly")
		}
		validateStruct(pass, structType)
	})
	return nil, nil
}

func validateStruct(pass *analysis.Pass, structType *ast.StructType) {
	info, infoAvailable := pass.TypesInfo.Types[structType]
	if !infoAvailable {
		fmt.Printf("[WARNING] type info not available for a struct")
		return
	}

	typeInfo, isStructInfo := info.Type.(*types.Struct)
	if !isStructInfo {
		fmt.Printf("[WARNING] type info not a struct")
		return
	}

	for fieldIndex := range typeInfo.NumFields() {
		field := typeInfo.Field(fieldIndex)
		if !field.Exported() {
			continue // ignore unexported fields
		}
		if !taggedWithOmitempty(typeInfo.Tag(fieldIndex)) {
			continue // ignore fields not tagged with "omitempty"
		}
		if !typeCanBeEmpty(field) {
			pass.Report(analysis.Diagnostic{
				Pos:     field.Pos(),
				Message: fmt.Sprintf("field '%v' is marked 'omitempty', but it can never be empty; consider making it a pointer", field.Name()),
			})
		}
	}
}

func taggedWithOmitempty(rawTag string) bool {
	tag := reflect.StructTag(rawTag)
	if jsonTag, isJsonTagged := tag.Lookup("json"); isJsonTagged {
		return strings.Contains(jsonTag, "omitempty")
	}
	return false
}

func typeCanBeEmpty(field *types.Var) bool {
	switch typ := field.Type().Underlying().(type) {
	case *types.Basic,
		*types.Slice,
		*types.Pointer,
		*types.Map,
		*types.Chan,
		*types.Signature,
		*types.Interface:
		return true
	case *types.Array:
		return typ.Len() == 0
	default:
		return false
	}
}
