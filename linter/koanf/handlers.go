package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/fatih/structtag"
	"golang.org/x/tools/go/analysis"
)

// handleComposite tracks use of fields in composite literals.
// E.g. `Config{A: 1, B: 2, C: 3}` will increase counters of fields A,B and C.
func handleComposite(pass *analysis.Pass, cl *ast.CompositeLit, cnt map[string]int) {
	id, ok := cl.Type.(*ast.Ident)
	if !ok {
		return
	}
	for _, e := range cl.Elts {
		if kv, ok := e.(*ast.KeyValueExpr); ok {
			if ki, ok := kv.Key.(*ast.Ident); ok {
				fi := pass.TypesInfo.Types[id].Type.String() + "." + ki.Name
				cnt[normalizeID(pass, fi)]++
			}
		}
	}
}

// handleSelector handles selector expression recursively, that is an expression:
// a.B.C.D will update counter for fields: a.B.C.D, a.B.C and a.B.
// It updates counters map in place, increasing corresponding identifiers by
// increaseBy amount.
func handleSelector(pass *analysis.Pass, se *ast.SelectorExpr, increaseBy int, cnt map[string]int) string {
	if e, ok := se.X.(*ast.SelectorExpr); ok {
		// Full field identifier, including package name.
		fi := pass.TypesInfo.Types[e].Type.String() + "." + se.Sel.Name
		cnt[normalizeID(pass, fi)] += increaseBy
		prefix := handleSelector(pass, e, increaseBy, cnt)
		fi = prefix + "." + se.Sel.Name
		cnt[normalizeID(pass, fi)] += increaseBy
		return fi
	}
	// Handle selectors on function calls, e.g. `config().Enabled`.
	if _, ok := se.X.(*ast.CallExpr); ok {
		fi := pass.TypesInfo.Types[se.X].Type.String() + "." + se.Sel.Name
		cnt[normalizeID(pass, fi)] += increaseBy
		return fi
	}
	if ident, ok := se.X.(*ast.Ident); ok {
		if pass.TypesInfo.Types[ident].Type != nil {
			fi := pass.TypesInfo.Types[ident].Type.String() + "." + se.Sel.Name
			cnt[normalizeID(pass, fi)] += increaseBy
			return fi
		}
	}
	return ""
}

// koanfFields returns a map of fields that have koanf tag.
func koanfFields(pass *analysis.Pass) map[string]token.Pos {
	res := make(map[string]token.Pos)
	for _, f := range pass.Files {
		pkgName := f.Name.Name
		ast.Inspect(f, func(node ast.Node) bool {
			if ts, ok := node.(*ast.TypeSpec); ok {
				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					return true
				}
				for _, f := range st.Fields.List {
					if tag := tagFromField(f); tag != "" {
						t := strings.Join([]string{pkgName, ts.Name.Name, f.Names[0].Name}, ".")
						res[t] = f.Pos()
					}
				}
			}
			return true
		})
	}
	return res
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
func checkFlagDefs(pass *analysis.Pass, f *ast.FuncDecl, cnt map[string]int) Result {
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
		s, ok := selectorName(callE.Args[1])
		if !ok {
			continue
		}
		handleSelector(pass, callE.Args[1].(*ast.SelectorExpr), -1, cnt)
		if normSL := normalizeTag(sl); !strings.EqualFold(normSL, s) {
			res.Errors = append(res.Errors, koanfError{
				Pos:     f.Pos(),
				Message: fmt.Sprintf("koanf tag name: %q doesn't match the field: %q", sl, s),
				err:     errIncorrectFlag,
			})
		}

	}
	return res
}

func selectorName(e ast.Expr) (string, bool) {
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

// tagFromField extracts koanf tag from struct field.
func tagFromField(f *ast.Field) string {
	if f.Tag == nil {
		return ""
	}
	tags, err := structtag.Parse(strings.Trim((f.Tag.Value), "`"))
	if err != nil {
		return ""
	}
	tag, err := tags.Get("koanf")
	if err != nil {
		return ""
	}
	return normalizeTag(tag.Name)
}

// checkStruct returns violations where koanf tag name doesn't match field names.
func checkStruct(pass *analysis.Pass, s *ast.StructType) Result {
	var res Result
	for _, f := range s.Fields.List {
		tag := tagFromField(f)
		if tag == "" {
			continue
		}
		fieldName := f.Names[0].Name
		if !strings.EqualFold(tag, fieldName) {
			res.Errors = append(res.Errors, koanfError{
				Pos:     f.Pos(),
				Message: fmt.Sprintf("field name: %q doesn't match tag name: %q\n", fieldName, tag),
				err:     errMismatch,
			})
		}
	}
	return res
}

func normalizeTag(s string) string {
	return strings.ReplaceAll(s, "-", "")
}

func normalizeID(pass *analysis.Pass, id string) string {
	id = strings.TrimPrefix(id, "*")
	return pass.Pkg.Name() + strings.TrimPrefix(id, pass.Pkg.Path())
}
