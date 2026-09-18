// Lists exported defs and same-package callees in the given files. stdin JSON, stdout JSON.
package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
)

type fileSpec struct {
	Path      string `json:"path"`
	HunkLines []int  `json:"hunk_lines"`
}

type request struct {
	Root  string     `json:"root"`
	Files []fileSpec `json:"files"`
}

type identOut struct {
	Name     string `json:"name"`
	Recv     string `json:"recv"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Col      int    `json:"col"`
	Exported bool   `json:"exported"`
	Kind     string `json:"kind"`
	Hunk     bool   `json:"hunk"`
	Role     string `json:"role"`
}

func isExported(name string) bool {
	r, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(r)
}

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

func skipFuncName(name string) bool {
	if name == "init" || name == "main" {
		return true
	}
	for _, p := range []string{"Test", "Benchmark", "Example", "Fuzz"} {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

func callName(call *ast.CallExpr) string {
	switch f := call.Fun.(type) {
	case *ast.Ident:
		return f.Name
	case *ast.SelectorExpr:
		return f.Sel.Name
	default:
		return ""
	}
}

func main() {
	var req request
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		fmt.Fprintf(os.Stderr, "decode: %v\n", err)
		os.Exit(1)
	}
	fset := token.NewFileSet()
	wantBase := map[string]fileSpec{}
	goldFiles := map[string]struct{}{}
	byDir := map[string][]fileSpec{}
	for _, f := range req.Files {
		slash := filepath.ToSlash(f.Path)
		wantBase[filepath.Base(slash)] = f
		goldFiles[slash] = struct{}{}
		dir := filepath.Join(req.Root, filepath.Dir(slash))
		byDir[dir] = append(byDir[dir], f)
	}

	var out []identOut
	seen := map[string]struct{}{}
	add := func(id identOut) {
		key := id.Recv + "|" + id.Name + "|" + id.File + "|" + id.Role
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, id)
	}

	for dir := range byDir {
		pkgs, err := parser.ParseDir(fset, dir, func(info os.FileInfo) bool {
			return strings.HasSuffix(info.Name(), ".go")
		}, 0)
		if err != nil || pkgs == nil {
			continue
		}
		for _, pkg := range pkgs {
			type fnKey struct{ recv, name string }
			pkgFns := map[fnKey]*ast.FuncDecl{}
			pkgFnFile := map[fnKey]string{}
			for fname, file := range pkg.Files {
				rel, err := filepath.Rel(req.Root, fname)
				if err != nil {
					continue
				}
				rel = filepath.ToSlash(rel)
				for _, d := range file.Decls {
					fn, ok := d.(*ast.FuncDecl)
					if !ok || fn.Name == nil {
						continue
					}
					k := fnKey{recvName(fn), fn.Name.Name}
					pkgFns[k] = fn
					pkgFnFile[k] = rel
				}
			}
			for fname, file := range pkg.Files {
				rel, err := filepath.Rel(req.Root, fname)
				if err != nil {
					continue
				}
				rel = filepath.ToSlash(rel)
				spec, ok := wantBase[filepath.Base(fname)]
				if !ok {
					continue
				}
				if _, gold := goldFiles[rel]; !gold {
					// basename matched a different directory
					if filepath.ToSlash(spec.Path) != rel {
						continue
					}
				}
				hunkSet := map[int]bool{}
				for _, ln := range spec.HunkLines {
					hunkSet[ln] = true
				}
				inHunk := func(n ast.Node) bool {
					if n == nil {
						return false
					}
					start := fset.Position(n.Pos()).Line
					end := fset.Position(n.End()).Line
					for ln := range hunkSet {
						if ln >= start && ln <= end {
							return true
						}
					}
					return false
				}
				emitFunc := func(fn *ast.FuncDecl, role string, hunk bool) {
					if fn == nil || fn.Name == nil || skipFuncName(fn.Name.Name) {
						return
					}
					pos := fset.Position(fn.Name.Pos())
					relFile := rel
					k := fnKey{recvName(fn), fn.Name.Name}
					if f, ok := pkgFnFile[k]; ok {
						relFile = f
					}
					add(identOut{
						Name:     fn.Name.Name,
						Recv:     recvName(fn),
						File:     relFile,
						Line:     pos.Line,
						Col:      pos.Column,
						Exported: isExported(fn.Name.Name),
						Kind:     "func",
						Hunk:     hunk,
						Role:     role,
					})
				}
				for _, d := range file.Decls {
					switch n := d.(type) {
					case *ast.FuncDecl:
						if n.Name == nil || skipFuncName(n.Name.Name) {
							continue
						}
						h := inHunk(n)
						exp := isExported(n.Name.Name)
						if !exp && !h {
							continue
						}
						emitFunc(n, "def", h)
						if n.Body == nil {
							continue
						}
						ast.Inspect(n.Body, func(nn ast.Node) bool {
							call, ok := nn.(*ast.CallExpr)
							if !ok {
								return true
							}
							cname := callName(call)
							if cname == "" || skipFuncName(cname) {
								return true
							}
							var matched *ast.FuncDecl
							if fn, ok := pkgFns[fnKey{recvName(n), cname}]; ok {
								matched = fn
							} else if fn, ok := pkgFns[fnKey{"", cname}]; ok {
								matched = fn
							} else {
								for k, fn := range pkgFns {
									if k.name == cname {
										relFile := pkgFnFile[k]
										if _, gold := goldFiles[relFile]; gold {
											matched = fn
											break
										}
									}
								}
							}
							if matched == nil {
								return true
							}
							relFile := pkgFnFile[fnKey{recvName(matched), matched.Name.Name}]
							if _, gold := goldFiles[relFile]; !gold {
								return true
							}
							emitFunc(matched, "callee", false)
							return true
						})
					case *ast.GenDecl:
						if n.Tok != token.TYPE {
							continue
						}
						for _, sp := range n.Specs {
							ts, ok := sp.(*ast.TypeSpec)
							if !ok || ts.Name == nil {
								continue
							}
							if isExported(ts.Name.Name) {
								pos := fset.Position(ts.Name.Pos())
								add(identOut{
									Name:     ts.Name.Name,
									Recv:     "",
									File:     rel,
									Line:     pos.Line,
									Col:      pos.Column,
									Exported: true,
									Kind:     "type",
									Hunk:     inHunk(ts) || inHunk(n),
									Role:     "def",
								})
							}
							iface, ok := ts.Type.(*ast.InterfaceType)
							if !ok || iface.Methods == nil {
								continue
							}
							for _, m := range iface.Methods.List {
								if len(m.Names) == 0 {
									continue
								}
								for _, nm := range m.Names {
									if skipFuncName(nm.Name) {
										continue
									}
									if !isExported(nm.Name) && !inHunk(m) {
										continue
									}
									pos := fset.Position(nm.Pos())
									add(identOut{
										Name:     nm.Name,
										Recv:     ts.Name.Name,
										File:     rel,
										Line:     pos.Line,
										Col:      pos.Column,
										Exported: isExported(nm.Name),
										Kind:     "iface",
										Hunk:     inHunk(m),
										Role:     "def",
									})
								}
							}
						}
					}
				}
			}
		}
	}
	if err := json.NewEncoder(os.Stdout).Encode(out); err != nil {
		fmt.Fprintf(os.Stderr, "encode: %v\n", err)
		os.Exit(1)
	}
}
