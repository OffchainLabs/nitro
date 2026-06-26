// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package logfmt

import (
	"go/ast"
	"go/token"
	"go/types"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
)

const gethLogPkgSuffix = "go-ethereum/log"

var logFuncNames = map[string]bool{
	"Trace": true, "Debug": true, "Info": true,
	"Warn": true, "Error": true, "Crit": true,
}

// Matches printf format verbs like %v, %s, %d, %w, %+v, %02x, etc.
// Space is a valid printf flag (e.g. "% d") but is deliberately excluded to
// avoid false positives on prose like "100% done".
var formatVerbRe = regexp.MustCompile(`%[+\-#0]*\*?[0-9]*\.?\*?[0-9]*[vTtbcdoOqxXUeEfFgGspw]`)

var Analyzer = &analysis.Analyzer{
	Name:       "logfmt",
	Doc:        "check for printf-style format verbs in structured log message strings",
	Run:        run,
	ResultType: reflect.TypeOf(Result{}),
}

type logfmtError struct {
	Pos     token.Position
	Message string
}

type Result struct {
	Errors []logfmtError
}

func run(pass *analysis.Pass) (interface{}, error) {
	var ret Result
	for _, f := range pass.Files {
		ast.Inspect(f, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || len(call.Args) == 0 {
				return true
			}

			if !isLogCall(pass, call) {
				return true
			}

			lit, ok := call.Args[0].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}

			msg, err := strconv.Unquote(lit.Value)
			if err != nil {
				return true
			}

			if formatVerbRe.MatchString(msg) {
				e := logfmtError{
					Pos:     pass.Fset.Position(lit.Pos()),
					Message: "log message contains printf-style format verb; structured logger does not interpret format verbs",
				}
				ret.Errors = append(ret.Errors, e)
				pass.Report(analysis.Diagnostic{
					Pos:      lit.Pos(),
					Message:  e.Message,
					Category: "logfmt",
				})
			}

			return true
		})
	}
	return ret, nil
}

func isLogCall(pass *analysis.Pass, call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	if !logFuncNames[sel.Sel.Name] {
		return false
	}

	// Package-level call: log.Warn(...)
	if ident, ok := sel.X.(*ast.Ident); ok {
		if obj := pass.TypesInfo.Uses[ident]; obj != nil {
			if pkgName, ok := obj.(*types.PkgName); ok {
				return strings.HasSuffix(pkgName.Imported().Path(), gethLogPkgSuffix)
			}
		}
	}

	// Method call on Logger instance: logger.Warn(...)
	if selection, ok := pass.TypesInfo.Selections[sel]; ok {
		if obj := selection.Obj(); obj != nil && obj.Pkg() != nil {
			return strings.HasSuffix(obj.Pkg().Path(), gethLogPkgSuffix)
		}
	}

	return false
}
