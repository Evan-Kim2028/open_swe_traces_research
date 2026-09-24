// goclosure emits per-file AST facts for a Go source tree: function decls,
// call expressions, local variable types (light inference), type decls and
// imports. stdin: {"root": "/abs/tree"} — stdout: JSON. Used by
// src/openswe_traces/synth/closure_metrics.py to compute closure metrics for
// excised units. go/ast only (no go/types): the trees are obfuscated and have
// no module cache, so full type checking is not available.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type request struct {
	Root string `json:"root"`
}

type funcFact struct {
	Name      string `json:"name"`
	Recv      string `json:"recv"`
	Start     int    `json:"start"`
	End       int    `json:"end"`
	BodyHash  string `json:"body_hash"`
	BodyStart int    `json:"body_start"`
	BodyEnd   int    `json:"body_end"`
}

type callFact struct {
	Name       string `json:"name"`
	Qual       string `json:"qual"`
	Line       int    `json:"line"`
	Caller     string `json:"caller"`
	CallerRecv string `json:"caller_recv"`
}

type varFact struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Line  int    `json:"line"`
	Kind  string `json:"kind"`
	Scope string `json:"scope"`
}

type fieldFact struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type typeFact struct {
	Name   string      `json:"name"`
	Kind   string      `json:"kind"`
	Fields []fieldFact `json:"fields"`
}

type fileFacts struct {
	Pkg     string            `json:"pkg"`
	Imports map[string]string `json:"imports"`
	Funcs   []funcFact        `json:"funcs"`
	Calls   []callFact        `json:"calls"`
	Vars    []varFact         `json:"vars"`
	Types   []typeFact        `json:"types"`
}

type treeFacts struct {
	Files map[string]*fileFacts `json:"files"`
}

// typeName reduces an expression to a plain type name (deref, index, parens).
func typeName(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return typeName(t.X)
	case *ast.IndexExpr:
		return typeName(t.X)
	case *ast.IndexListExpr:
		return typeName(t.X)
	case *ast.ParenExpr:
		return typeName(t.X)
	case *ast.SelectorExpr:
		return typeName(t.X) + "." + t.Sel.Name
	default:
		return ""
	}
}

func recvName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return ""
	}
	return typeName(fn.Recv.List[0].Type)
}

func bodyHash(fset *token.FileSet, path string, fn *ast.FuncDecl) (string, int, int) {
	if fn.Body == nil {
		return "", 0, 0
	}
	start := fset.Position(fn.Body.Pos()).Line
	end := fset.Position(fn.Body.End()).Line
	file := fset.File(fn.Body.Pos())
	if file == nil {
		return "", start, end
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", start, end
	}
	lo := fset.Position(fn.Body.Pos()).Offset
	hi := fset.Position(fn.Body.End()).Offset
	if lo < 0 || hi > len(data) || lo >= hi {
		return "", start, end
	}
	seg := data[lo:hi]
	sum := sha256.Sum256(seg)
	return hex.EncodeToString(sum[:]), start, end
}

// qualText renders a selector base expression: `a.b.N(...)` -> "a.b",
// `pkg.N(...)` -> "pkg", `N(...)` -> "".
func qualText(fset *token.FileSet, e ast.Expr) string {
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return qualText(fset, t.X) + "." + t.Sel.Name
	case *ast.StarExpr:
		return "*" + qualText(fset, t.X)
	case *ast.ParenExpr:
		return "(" + qualText(fset, t.X) + ")"
	case *ast.CallExpr:
		return qualText(fset, t.Fun) + "(...)"
	case *ast.IndexExpr:
		return qualText(fset, t.X)
	case *ast.TypeAssertExpr:
		return qualText(fset, t.X)
	default:
		return fset.Position(e.Pos()).String() + "-" + fset.Position(e.End()).String()
	}
}

func main() {
	var req request
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		fmt.Fprintf(os.Stderr, "decode: %v\n", err)
		os.Exit(1)
	}
	root := req.Root
	fset := token.NewFileSet()
	out := treeFacts{Files: map[string]*fileFacts{}}

	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if name == "vendor" || name == "node_modules" || name == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".go") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			return nil
		}
		ff := &fileFacts{
			Pkg:     file.Name.Name,
			Imports: map[string]string{},
			Funcs:   []funcFact{},
			Calls:   []callFact{},
			Vars:    []varFact{},
			Types:   []typeFact{},
		}
		for _, imp := range file.Imports {
			if imp.Path == nil {
				continue
			}
			p := strings.Trim(imp.Path.Value, `"`)
			alias := p
			if i := strings.LastIndex(p, "/"); i >= 0 {
				alias = p[i+1:]
			}
			if imp.Name != nil {
				alias = imp.Name.Name
			}
			ff.Imports[alias] = p
		}
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Name == nil {
					continue
				}
				recv := recvName(d)
				hash, bs, be := bodyHash(fset, path, d)
				ff.Funcs = append(ff.Funcs, funcFact{
					Name:      d.Name.Name,
					Recv:      recv,
					Start:     fset.Position(d.Pos()).Line,
					End:       fset.Position(d.End()).Line,
					BodyHash:  hash,
					BodyStart: bs,
					BodyEnd:   be,
				})
				// params and receiver become known-typed vars
				if d.Recv != nil {
					for _, f := range d.Recv.List {
						if len(f.Names) == 0 {
							continue
						}
						t := typeName(f.Type)
						for _, n := range f.Names {
							ff.Vars = append(ff.Vars, varFact{Name: n.Name, Type: t, Line: 0, Kind: "recv", Scope: d.Name.Name})
						}
					}
				}
				if d.Type != nil && d.Type.Params != nil {
					for _, f := range d.Type.Params.List {
						t := typeName(f.Type)
						if t == "" {
							continue
						}
						for _, n := range f.Names {
							ff.Vars = append(ff.Vars, varFact{Name: n.Name, Type: t, Line: 0, Kind: "param", Scope: d.Name.Name})
						}
					}
				}
				if d.Body == nil {
					continue
				}
				collectVars(fset, ff, d.Body, d.Name.Name)
				ast.Inspect(d.Body, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}
					name := ""
					qual := ""
					switch fn := call.Fun.(type) {
					case *ast.Ident:
						name = fn.Name
					case *ast.SelectorExpr:
						name = fn.Sel.Name
						qual = qualText(fset, fn.X)
					default:
						return true
					}
					ff.Calls = append(ff.Calls, callFact{
						Name:       name,
						Qual:       qual,
						Line:       fset.Position(call.Pos()).Line,
						Caller:     d.Name.Name,
						CallerRecv: recv,
					})
					return true
				})
			case *ast.GenDecl:
				if d.Tok != token.TYPE {
					continue
				}
				for _, spec := range d.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok || ts.Name == nil {
						continue
					}
					tf := typeFact{Name: ts.Name.Name, Kind: "other", Fields: []fieldFact{}}
					switch t := ts.Type.(type) {
					case *ast.StructType:
						tf.Kind = "struct"
						if t.Fields != nil {
							for _, f := range t.Fields.List {
								ft := typeName(f.Type)
								if len(f.Names) == 0 {
									if ft != "" {
										tf.Fields = append(tf.Fields, fieldFact{Name: "", Type: ft})
									}
									continue
								}
								for _, n := range f.Names {
									tf.Fields = append(tf.Fields, fieldFact{Name: n.Name, Type: ft})
								}
							}
						}
					case *ast.InterfaceType:
						tf.Kind = "interface"
					}
					ff.Types = append(ff.Types, tf)
				}
			}
		}
		out.Files[rel] = ff
		return nil
	})

	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(out); err != nil {
		fmt.Fprintf(os.Stderr, "encode: %v\n", err)
		os.Exit(1)
	}
}

// collectVars walks a function body and records lightweight type facts:
// `x := &T{...}`, `x := T{...}`, `x := new(T)`, `x := y.(*T)`, `var x T`.
func collectVars(fset *token.FileSet, ff *fileFacts, body *ast.BlockStmt, scope string) {
	ast.Inspect(body, func(n ast.Node) bool {
		switch st := n.(type) {
		case *ast.AssignStmt:
			if st.Tok != token.DEFINE {
				return true
			}
			if len(st.Lhs) != 1 || len(st.Rhs) != 1 {
				return true
			}
			id, ok := st.Lhs[0].(*ast.Ident)
			if !ok {
				return true
			}
			t := rhsType(st.Rhs[0])
			if t == "" {
				return true
			}
			ff.Vars = append(ff.Vars, varFact{Name: id.Name, Type: t, Line: fset.Position(st.Pos()).Line, Kind: "assign", Scope: scope})
		case *ast.DeclStmt:
			gd, ok := st.Decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.VAR {
				return true
			}
			for _, spec := range gd.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				t := typeName(vs.Type)
				if t == "" {
					continue
				}
				for _, n := range vs.Names {
					ff.Vars = append(ff.Vars, varFact{Name: n.Name, Type: t, Line: fset.Position(vs.Pos()).Line, Kind: "var", Scope: scope})
				}
			}
		}
		return true
	})
}

func rhsType(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.UnaryExpr:
		if t.Op == token.AND {
			if lit, ok := t.X.(*ast.CompositeLit); ok {
				return typeName(lit.Type)
			}
		}
		return ""
	case *ast.CompositeLit:
		return typeName(t.Type)
	case *ast.CallExpr:
		if id, ok := t.Fun.(*ast.Ident); ok && id.Name == "new" {
			if len(t.Args) == 1 {
				return typeName(t.Args[0])
			}
		}
		return ""
	case *ast.TypeAssertExpr:
		return typeName(t.Type)
	default:
		return ""
	}
}
