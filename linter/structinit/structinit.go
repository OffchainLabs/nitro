package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"reflect"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/singlechecker"
)

// Tip for linter that struct that has this comment should be included in the
// analysis.
// Note: comment should be directly line above the struct definition.
const linterTip = "// lint:require-exhaustive-initialization"

// Analyzer implements struct analyzer for structs that are annotated with
// `linterTip`, it checks that every instantiation initializes all the fields.
var Analyzer = &analysis.Analyzer{
	Name:       "structinit",
	Doc:        "check for struct field initializations",
	Run:        func(p *analysis.Pass) (interface{}, error) { return run(false, p) },
	ResultType: reflect.TypeOf(Result{}),
}

var analyzerForTests = &analysis.Analyzer{
	Name:       "teststructinit",
	Doc:        "check for struct field initializations",
	Run:        func(p *analysis.Pass) (interface{}, error) { return run(true, p) },
	ResultType: reflect.TypeOf(Result{}),
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
		structs = markedStructs(pass)
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

// markedStructs returns a map of structs that are annotated for linter to check
// that all fields are initialized when the struct is instantiated.
// It maps struct full name (including package path) to number of fields it contains.
func markedStructs(pass *analysis.Pass) map[string]int {
	res := make(map[string]int)
	for _, f := range pass.Files {
		tips := make(map[position]bool)
		ast.Inspect(f, func(node ast.Node) bool {
			switch n := node.(type) {
			case *ast.Comment:
				p := pass.Fset.Position(node.Pos())
				if strings.Contains(n.Text, linterTip) {
					tips[position{p.Filename, p.Line + 1}] = true
				}
			case *ast.TypeSpec:
				if st, ok := n.Type.(*ast.StructType); ok {
					p := pass.Fset.Position(st.Struct)
					if tips[position{p.Filename, p.Line}] {
						fieldsCnt := 0
						for _, field := range st.Fields.List {
							fieldsCnt += len(field.Names)
						}
						res[pass.Pkg.Path()+"."+n.Name.Name] = fieldsCnt
					}
				}
			}
			return true
		})
	}
	return res
}

type position struct {
	fileName string
	line     int
}

func main() {
	singlechecker.Main(Analyzer)
}
