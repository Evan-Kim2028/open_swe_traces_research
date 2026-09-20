// excise.go — AST-guided byte-splice rewriter for authoring excision patches.
// (adapted from authored_batch2/_tools for the go-github batch3 run)
//
// Modes:
//   stub    <file.go> <Name|Recv.Name>...   replace func body with panic("excised: <spec>")
//   stubx   <file.go> <spec.json>           replace func bodies with custom bodies from JSON map
//                                           {"Name|Recv.Name": "<body text>"}; emits { <body> // excised: <spec> }
//   delfunc <file.go> <Func>...             delete whole funcs (leading doc comment included)
//   unimport <file.go> <pkg-path>...        rewrite imports to "_ path"
//
// Edits are byte splices: everything outside the replaced ranges is preserved
// verbatim so the resulting diff is minimal.
package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"
)

type splice struct {
	start, end int
	repl       string
}

func load(path string) ([]byte, *token.FileSet, *ast.File) {
	fset := token.NewFileSet()
	src, err := os.ReadFile(path)
	if err != nil {
		fatal(err)
	}
	f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		fatal(err)
	}
	return src, fset, f
}

func fatal(v any) { fmt.Fprintln(os.Stderr, "excise:", v); os.Exit(1) }

// matchName reports whether decl matches spec: "Name" matches any func named
// Name; "Recv.Name" matches only methods whose receiver base type is Recv.
func matchName(fn *ast.FuncDecl, spec string) bool {
	if i := strings.IndexByte(spec, '.'); i >= 0 {
		recv, name := spec[:i], spec[i+1:]
		if fn.Name.Name != name || fn.Recv == nil || len(fn.Recv.List) == 0 {
			return false
		}
		t := fn.Recv.List[0].Type
		for {
			switch tt := t.(type) {
			case *ast.StarExpr:
				t = tt.X
			case *ast.IndexExpr:
				t = tt.X
			case *ast.IndexListExpr:
				t = tt.X
			case *ast.Ident:
				return tt.Name == recv
			default:
				return false
			}
		}
	}
	return fn.Name.Name == spec
}

// unimportSpecs rewrites imports whose path is in specs to blank imports.
func unimport(src []byte, fset *token.FileSet, f *ast.File, specs []string) []byte {
	var sp []splice
	matched := map[string]bool{}
	for _, is := range f.Imports {
		p := strings.Trim(is.Path.Value, `"`)
		found := false
		for _, s := range specs {
			if p == s {
				found = true
				matched[s] = true
			}
		}
		if !found {
			continue
		}
		if is.Name != nil {
			sp = append(sp, splice{
				start: fset.Position(is.Name.Pos()).Offset,
				end:   fset.Position(is.Name.End()).Offset,
				repl:  "_",
			})
		} else {
			sp = append(sp, splice{
				start: fset.Position(is.Path.Pos()).Offset,
				end:   fset.Position(is.Path.Pos()).Offset,
				repl:  "_ ",
			})
		}
	}
	for _, s := range specs {
		if !matched[s] {
			fatal("import not found: " + s + " in " + f.Name.Name)
		}
	}
	sort.Slice(sp, func(i, j int) bool { return sp[i].start > sp[j].start })
	out := src
	for _, s := range sp {
		out = append(out[:s.start], append([]byte(s.repl), out[s.end:]...)...)
	}
	return out
}

func apply(src []byte, fset *token.FileSet, f *ast.File, specs []string, bodies map[string]string, mode string) []byte {
	var sp []splice
	matched := map[string]bool{}
	for _, d := range f.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}
		for _, spec := range specs {
			if !matchName(fn, spec) {
				continue
			}
			matched[spec] = true
			switch mode {
			case "stub", "stubx":
				if fn.Body == nil {
					fatal("no body: " + spec)
				}
				body, ok := bodies[spec]
				if mode == "stub" || !ok {
					body = fmt.Sprintf("panic(\"excised: %s\")", spec)
				}
				if !strings.Contains(body, "excised:") {
					body = body + " /* excised: " + spec + " */"
				}
				sp = append(sp, splice{
					start: fset.Position(fn.Body.Lbrace).Offset,
					end:   fset.Position(fn.Body.Rbrace).Offset + 1,
					repl:  "{\n\t" + body + "\n}",
				})
			case "delfunc":
				start := fn.Pos()
				if fn.Doc != nil {
					start = fn.Doc.Pos()
				}
				end := fset.Position(fn.End()).Offset
				if end < len(src) && src[end] == '\n' {
					end++
				}
				sp = append(sp, splice{
					start: fset.Position(start).Offset,
					end:   end,
					repl:  "",
				})
			}
		}
	}
	for _, s := range specs {
		if !matched[s] {
			fatal("not found: " + s + " in " + f.Name.Name)
		}
	}
	sort.Slice(sp, func(i, j int) bool { return sp[i].start > sp[j].start })
	out := src
	for _, s := range sp {
		out = append(out[:s.start], append([]byte(s.repl), out[s.end:]...)...)
	}
	return out
}

func main() {
	if len(os.Args) < 4 {
		fatal("usage: excise <stub|stubx|delfunc|unimport> <file> <name|spec.json>...")
	}
	mode, path := os.Args[1], os.Args[2]
	src, fset, f := load(path)
	var out []byte
	switch mode {
	case "unimport":
		out = unimport(src, fset, f, os.Args[3:])
	case "stub", "delfunc":
		out = apply(src, fset, f, os.Args[3:], nil, mode)
	case "stubx":
		raw, err := os.ReadFile(os.Args[3])
		if err != nil {
			fatal(err)
		}
		var bodies map[string]string
		if err := json.Unmarshal(raw, &bodies); err != nil {
			fatal(err)
		}
		specs := make([]string, 0, len(bodies))
		for k := range bodies {
			specs = append(specs, k)
		}
		out = apply(src, fset, f, specs, bodies, "stubx")
	default:
		fatal("unknown mode: " + mode)
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		fatal(err)
	}
}
