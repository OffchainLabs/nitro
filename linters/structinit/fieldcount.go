package structinit

import (
	"go/ast"
	"reflect"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// Tip for linter that struct that has this comment should be included in the
// analysis.
// Note: comment should be directly line above the struct definition.
const linterTip = "// lint:require-exhaustive-initialization"

var FieldCountAnalyzer = &analysis.Analyzer{
	Name:       "fieldcount",
	Doc:        "counts fields for every declared struct",
	Run:        countFields,
	ResultType: reflect.TypeOf(fieldCounts{}),
	FactTypes:  []analysis.Fact{new(packageFact)},
}

type fieldCounts = map[string]int

type packageFact struct {
	fieldCounts
}

func (f *packageFact) AFact() {}

func countFields(pass *analysis.Pass) (interface{}, error) {
	counts := mergeFieldCountsAcrossPackages(pass)
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
	pass.ExportPackageFact(&packageFact{counts})
	return counts, nil
}

func mergeFieldCountsAcrossPackages(pass *analysis.Pass) (merged fieldCounts) {
	merged = make(fieldCounts)
	for _, packageFieldCounts := range pass.AllPackageFacts() {
		for k, v := range packageFieldCounts.Fact.(*packageFact).fieldCounts {
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
