package structinit

import (
	"fmt"
	"go/ast"
	"go/token"
	"reflect"

	"golang.org/x/tools/go/analysis"
)

// MainAnalyzer implements struct analyzer for structs that are annotated with
// `linterTip`, it checks that every instantiation initializes all the fields.
var MainAnalyzer = &analysis.Analyzer{
	Name:       "structinit",
	Doc:        "check for struct field initializations",
	Run:        func(p *analysis.Pass) (interface{}, error) { return run(false, p) },
	ResultType: reflect.TypeOf(Result{}),
	Requires:   []*analysis.Analyzer{FieldCountAnalyzer},
}

var analyzerForTests = &analysis.Analyzer{
	Name:       "teststructinit",
	Doc:        "check for struct field initializations",
	Run:        func(p *analysis.Pass) (interface{}, error) { return run(true, p) },
	ResultType: reflect.TypeOf(Result{}),
	Requires:   []*analysis.Analyzer{FieldCountAnalyzer},
}

type structError struct {
	Pos     token.Pos
	Message string
}

type Result struct {
	Errors []structError
}

func run(dryRun bool, pass *analysis.Pass) (interface{}, error) {
	var (
		ret     Result
		structs = pass.ResultOf[FieldCountAnalyzer].(fieldCounts)
	)
	for _, f := range pass.Files {
		ast.Inspect(f, func(node ast.Node) bool {
			// For every composite literal check that number of elements in
			// the literal match the number of struct fields.
			if cl, ok := node.(*ast.CompositeLit); ok {
				stName := pass.TypesInfo.Types[cl].Type.String()
				if cnt, found := structs[stName]; found && cnt != len(cl.Elts) {
					ret.Errors = append(ret.Errors, structError{
						Pos:     cl.Pos(),
						Message: fmt.Sprintf("struct: %q initialized with: %v of total: %v fields", stName, len(cl.Elts), cnt),
					})

				}

			}
			return true
		})
	}
	for _, err := range ret.Errors {
		if !dryRun {
			pass.Report(analysis.Diagnostic{
				Pos:      err.Pos,
				Message:  err.Message,
				Category: "structinit",
			})
		}
	}
	return ret, nil
}
