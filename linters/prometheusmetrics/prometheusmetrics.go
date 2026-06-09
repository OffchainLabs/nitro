// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package prometheusmetrics

import (
	"fmt"
	"go/ast"
	"go/token"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name:       "prometheusmetrics",
	Doc:        "check that geth metric names produce valid Prometheus names after path separator translation",
	Run:        run,
	ResultType: reflect.TypeOf(Result{}),
}

var validPrometheusName = regexp.MustCompile(`^[a-zA-Z_:][a-zA-Z0-9_:]*$`)

type metricError struct {
	Pos     token.Position
	Message string
}

type Result struct {
	Errors []metricError
}

func run(pass *analysis.Pass) (interface{}, error) {
	var ret Result
	for _, f := range pass.Files {
		ast.Inspect(f, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok || pkg.Name != "metrics" {
				return true
			}
			if !strings.HasPrefix(sel.Sel.Name, "NewRegistered") {
				return true
			}
			if len(call.Args) == 0 {
				return true
			}
			lit, ok := call.Args[0].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			name, err := strconv.Unquote(lit.Value)
			if err != nil {
				return true
			}
			prometheusName := strings.ReplaceAll(name, "/", "_")
			if !validPrometheusName.MatchString(prometheusName) {
				msg := fmt.Sprintf("metric %q translates to invalid Prometheus name %q", name, prometheusName)
				ret.Errors = append(ret.Errors, metricError{
					Pos:     pass.Fset.Position(lit.Pos()),
					Message: msg,
				})
				pass.Report(analysis.Diagnostic{
					Pos:      lit.Pos(),
					Message:  msg,
					Category: "prometheusmetrics",
				})
			}
			return true
		})
	}
	return ret, nil
}
