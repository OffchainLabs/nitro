// Copyright 2023-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

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
// Note: comment should be on the line directly above the struct definition
const linterTip = "// lint:require-exhaustive-initialization"

// Analyzer implements struct analyzer for structs that are annotated with
// `linterTip`, it checks that every instantiation initializes all the fields.
//
// For every package the Analyzer is run on, a slice of `structError` will be returned:
// one error per single incorrect struct initialization. Additionally, every Analyzer
// invocation will produce a `Fact` useful for dependent packages. It contains (accumulated)
// information about field count for every encountered struct.
var Analyzer = &analysis.Analyzer{
	Name:       "structinit",
	Doc:        "check for struct field initializations",
	Run:        run,
	ResultType: reflect.TypeOf([]structError{}),
	FactTypes:  []analysis.Fact{new(fieldCounts)},
}

// Mapping from a struct identifier to the number of declared fields.
type fieldCounts struct {
	counts map[string]int
}

// AFact required implementation for `fieldCounts` to be usable as a `Fact`.
func (f *fieldCounts) AFact() {}

// Error describing incorrect struct initialization.
type structError struct {
	Pos     token.Pos
	Message string
}

// Analyzer logic entrypoint.
func run(pass *analysis.Pass) (interface{}, error) {
	// Firstly, gather all field counts from the current package and all its dependencies.
	var markedStructs = countFieldsInPackageAndItsDeps(pass)

	var foundErrors []structError
	// Secondly, do the second traversal over the package and inspect every struct initialization.
	for _, f := range pass.Files {
		ast.Inspect(f, func(node ast.Node) bool {
			if cl, ok := node.(*ast.CompositeLit); ok {
				stName := pass.TypesInfo.Types[cl].Type.String()
				initializedFields := len(cl.Elts)
				if declaredFields, found := markedStructs.counts[stName]; found && declaredFields != initializedFields {
					foundErrors = append(foundErrors, structError{
						Pos:     cl.Pos(),
						Message: errorMessage(stName, initializedFields, declaredFields),
					})
				}
			}
			return true
		})
	}

	for _, err := range foundErrors {
		pass.Report(analysis.Diagnostic{
			Pos:      err.Pos,
			Message:  err.Message,
			Category: "structinit",
		})
	}

	return foundErrors, nil
}

// Find the number of fields for every struct in the current package and all its dependencies (including indirect).
func countFieldsInPackageAndItsDeps(pass *analysis.Pass) fieldCounts {
	accumulator := mergeFieldCountsAcrossVisitedPackages(pass)
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
						accumulator.counts[pass.Pkg.Path()+"."+n.Name.Name] = countStructFields(structDecl)
					}
				}
			}
			return true
		})
	}
	pass.ExportPackageFact(&accumulator)
	return accumulator
}

// Merge facts from all the already visited packages into a single `fieldCounts` object.
func mergeFieldCountsAcrossVisitedPackages(pass *analysis.Pass) fieldCounts {
	merged := make(map[string]int)
	for _, packageFact := range pass.AllPackageFacts() {
		if fieldCounts, ok := packageFact.Fact.(*fieldCounts); ok {
			for k, v := range fieldCounts.counts {
				merged[k] = v
			}
		}
	}
	return fieldCounts{counts: merged}
}

// Given a struct declaration AST node, count all its fields (including unnamed and single-type-multi name).
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
