package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"reflect"
	"strings"

	"github.com/fatih/structtag"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func New(conf any) ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{Analyzer}, nil
}

var Analyzer = &analysis.Analyzer{
	Name:       "koanfcheck",
	Doc:        "check for koanf misconfigurations",
	Run:        func(p *analysis.Pass) (interface{}, error) { return run(false, p) },
	ResultType: reflect.TypeOf(Result{}),
}

var analyzerForTests = &analysis.Analyzer{
	Name:       "testkoanfcheck",
	Doc:        "check for koanf misconfigurations (for tests)",
	Run:        func(p *analysis.Pass) (interface{}, error) { return run(true, p) },
	ResultType: reflect.TypeOf(Result{}),
}

// koanfError indicates the position of an error in configuration.
type koanfError struct {
	Pos     token.Position
	Message string
}

// Result is returned from the checkStruct function, and holds all the
// configuration errors.
type Result struct {
	Errors []koanfError
}

func run(dryRun bool, pass *analysis.Pass) (interface{}, error) {
	var ret Result
	for _, f := range pass.Files {
		ast.Inspect(f, func(node ast.Node) bool {
			var res Result
			switch v := node.(type) {
			case *ast.StructType:
				res = checkStruct(pass, v)
			case *ast.FuncDecl:
				res = checkFlagDefs(pass, v)
			default:
			}
			for _, err := range res.Errors {
				ret.Errors = append(ret.Errors, err)
				if !dryRun {
					pass.Report(analysis.Diagnostic{
						Pos:      pass.Fset.File(f.Pos()).Pos(err.Pos.Offset),
						Message:  err.Message,
						Category: "koanf",
					})
				}
			}
			return true
		},
		)
	}
	return ret, nil
}

func containsFlagSet(params []*ast.Field) bool {
	for _, p := range params {
		se, ok := p.Type.(*ast.StarExpr)
		if !ok {
			continue
		}
		sle, ok := se.X.(*ast.SelectorExpr)
		if !ok {
			continue
		}
		if sle.Sel.Name == "FlagSet" {
			return true
		}
	}
	return false
}

// checkFlagDefs checks flag definitions in the function.
// Result contains list of errors where flag name doesn't match field name.
func checkFlagDefs(pass *analysis.Pass, f *ast.FuncDecl) Result {
	// Ignore functions that does not get flagset as parameter.
	if !containsFlagSet(f.Type.Params.List) {
		return Result{}
	}
	var res Result
	for _, s := range f.Body.List {
		es, ok := s.(*ast.ExprStmt)
		if !ok {
			continue
		}
		callE, ok := es.X.(*ast.CallExpr)
		if !ok {
			continue
		}
		if len(callE.Args) != 3 {
			continue
		}
		sl, ok := extractStrLit(callE.Args[0])
		if !ok {
			continue
		}
		s, ok := selector(callE.Args[1])
		if !ok {
			continue
		}
		if normSL := normalize(sl); !strings.EqualFold(normSL, s) {
			res.Errors = append(res.Errors, koanfError{
				Pos:     pass.Fset.Position(f.Pos()),
				Message: fmt.Sprintf("koanf tag name: %q doesn't match the field: %q", sl, s),
			})
		}

	}
	return res
}

func selector(e ast.Expr) (string, bool) {
	n, ok := e.(ast.Node)
	if !ok {
		return "", false
	}
	se, ok := n.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	return se.Sel.Name, true
}

// Extracts literal from expression that is either:
// - string literal or
// - sum of variable and string literal.
// E.g.
// strLitFromSum(`"max-size"`) = "max-size"
// - strLitFromSum(`prefix + ".enable"“) = ".enable".
func extractStrLit(e ast.Expr) (string, bool) {
	if s, ok := strLit(e); ok {
		return s, true
	}
	if be, ok := e.(*ast.BinaryExpr); ok {
		if be.Op == token.ADD {
			if s, ok := strLit(be.Y); ok {
				// Drop the prefix dot.
				return s[1:], true
			}
		}
	}
	return "", false
}

func strLit(e ast.Expr) (string, bool) {
	if s, ok := e.(*ast.BasicLit); ok {
		if s.Kind == token.STRING {
			return strings.Trim(s.Value, "\""), true
		}
	}
	return "", false
}

func checkStruct(pass *analysis.Pass, s *ast.StructType) Result {
	var res Result
	for _, f := range s.Fields.List {
		if f.Tag == nil {
			continue
		}
		tags, err := structtag.Parse(strings.Trim((f.Tag.Value), "`"))
		if err != nil {
			continue
		}
		tag, err := tags.Get("koanf")
		if err != nil {
			continue
		}
		tagName := strings.ReplaceAll(tag.Name, "-", "")
		fieldName := f.Names[0].Name
		if !strings.EqualFold(tagName, fieldName) {
			res.Errors = append(res.Errors, koanfError{
				Pos:     pass.Fset.Position(f.Pos()),
				Message: fmt.Sprintf("field name: %q doesn't match tag name: %q\n", fieldName, tagName),
			})
		}
	}
	return res
}

func main() {
	singlechecker.Main(Analyzer)
}
