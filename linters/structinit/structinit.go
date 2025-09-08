package structinit

import (
	"fmt"
	"go/ast"
	"go/token"
	"reflect"
	"strings"

	"golang.org/x/tools/go/analysis"
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
	ResultType: reflect.TypeOf([]structError{}),
	FactTypes:  []analysis.Fact{new(accumulatedFieldCounts)},
}

type fieldCounts = map[string]int

var analyzerForTests = &analysis.Analyzer{
	Name:       "teststructinit",
	Doc:        "check for struct field initializations",
	Run:        func(p *analysis.Pass) (interface{}, error) { return run(true, p) },
	ResultType: reflect.TypeOf([]structError{}),
	FactTypes:  []analysis.Fact{new(accumulatedFieldCounts)},
}

type structError struct {
	Pos     token.Pos
	Message string
}

type accumulatedFieldCounts struct {
	fieldCounts
}

func (f *accumulatedFieldCounts) AFact() {}

func run(dryRun bool, pass *analysis.Pass) (interface{}, error) {
	var (
		foundErrors []structError
		structs     = countFieldsInPackageAndItsDeps(pass)
	)
	for _, f := range pass.Files {
		ast.Inspect(f, func(node ast.Node) bool {
			// For every composite literal check that number of elements in
			// the literal match the number of struct fields.
			if cl, ok := node.(*ast.CompositeLit); ok {
				stName := pass.TypesInfo.Types[cl].Type.String()
				initializedFields := len(cl.Elts)
				if declaredFields, found := structs[stName]; found && declaredFields != initializedFields {
					foundErrors = append(foundErrors, structError{
						Pos:     cl.Pos(),
						Message: errorMessage(stName, initializedFields, declaredFields),
					})
				}
			}
			return true
		})
	}

	if !dryRun {
		for _, err := range foundErrors {
			pass.Report(analysis.Diagnostic{
				Pos:      err.Pos,
				Message:  err.Message,
				Category: "structinit",
			})
		}
	}

	return foundErrors, nil
}

func countFieldsInPackageAndItsDeps(pass *analysis.Pass) fieldCounts {
	counts := mergeFieldCountsAcrossVisitedPackages(pass)
	for _, f := range pass.Files {
		markedStructs := make(map[position]bool)
		ast.Inspect(f, func(node ast.Node) bool {
			switch n := node.(type) {
			case *ast.Comment:
				if strings.Contains(n.Text, linterTip) {
					commentPos := getNodePosition(pass, node)
					markedStructs[commentPos.nextLine()] = true
				}
			case *ast.TypeSpec:
				if structDecl, ok := n.Type.(*ast.StructType); ok {
					if markedStructs[getNodePosition(pass, node)] {
						counts[pass.Pkg.Path()+"."+n.Name.Name] = countStructFields(structDecl)
					}
				}
			}
			return true
		})
	}
	pass.ExportPackageFact(&accumulatedFieldCounts{counts})
	return counts
}

func mergeFieldCountsAcrossVisitedPackages(pass *analysis.Pass) (merged fieldCounts) {
	merged = make(fieldCounts)
	for _, packageFieldCounts := range pass.AllPackageFacts() {
		for k, v := range packageFieldCounts.Fact.(*accumulatedFieldCounts).fieldCounts {
			merged[k] = v
		}
	}
	return
}

func countStructFields(structDecl *ast.StructType) (fieldCount int) {
	for _, field := range structDecl.Fields.List {
		fieldCount += max(1, len(field.Names))
	}
	return
}

type position struct {
	fileName string
	line     int
}

func (p position) nextLine() position {
	return position{p.fileName, p.line + 1}
}

func getNodePosition(pass *analysis.Pass, node ast.Node) position {
	p := pass.Fset.Position(node.Pos())
	return position{p.Filename, p.Line}
}

func errorMessage(structName string, initializedFields, declaredFields int) string {
	return fmt.Sprintf("struct: %q initialized with: %v of total: %v fields", structName, initializedFields, declaredFields)
}
