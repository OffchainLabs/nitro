package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"reflect"
	"strings"
	"unicode"

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
		tagName := normalize(tag.Name)
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

func normalize(s string) string {
	ans := s[:1]
	for i := 1; i < len(s); i++ {
		c := rune(s[i])
		if !isAlphanumeric(c) {
			continue
		}
		if !isAlphanumeric(rune(s[i-1])) && unicode.IsLower(c) {
			c = unicode.ToUpper(c)
		}
		ans += string(c)
	}
	return ans
}

func isAlphanumeric(c rune) bool {
	return unicode.IsLetter(c) || unicode.IsDigit(c)
}

func main() {
	singlechecker.Main(Analyzer)
}
