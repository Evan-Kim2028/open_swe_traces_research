// cgscan: syntactic call-graph scan for one task unit.
//
// Per hidden Test* root, computes the symbols the test reaches transitively
// through the excised package(s). Excised functions appear in the env tree as
// panic stubs, so their edges come from gold.patch's restored bodies instead.
// Also emits the symbols gold.patch defines/uses and the package decl table.
//
// Symbol string forms:
//   pkgpath.Func          plain function or package-level name
//   pkgpath.(T).Method    method on type T
//   bare:Name             receiver-unknown selector (method/field name only)
//
// Usage:
//   cgscan -src <module root> -gold <gold.patch> -pkgs <reldir,...>
//          -hidden <test.go,...> [-nametest <relpath.go:TestName,...>] -out <json>
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type FuncInfo struct {
	QName    string
	Uses     map[string]bool
	VarTypes map[string]string
	Results  []string
}

type PkgInfo struct {
	Path        string
	Imports     map[string]string
	Funcs       map[string]*FuncInfo
	Types       map[string]bool
	Vars        map[string]bool
	MethodNames map[string][]string
}

type TestResult struct {
	Defs []string `json:"defs"`
	Refs []string `json:"refs"`
}

type Out struct {
	Module     string              `json:"module"`
	Tests      map[string]TestResult `json:"tests"`
	GoldDefs   []string            `json:"gold_defs"`
	GoldRefs   []string            `json:"gold_refs"`
	Excised    []string            `json:"excised"`
	PkgSymbols []string            `json:"pkg_symbols"`
	Warnings   []string            `json:"warnings,omitempty"`
}

var (
	srcRoot, goldPath, outPath string
	pkgsFlag, hiddenFlag       string
	nametestFlag               string
	hiddenRoot                 string
	modPath                    string
	pkgs                       = map[string]*PkgInfo{}
	funcIndex                  = map[string]*FuncInfo{}
	methodByName               = map[string][]string{}
	warnings                   []string
	fset                       = token.NewFileSet()
)

func main() {
	flag.StringVar(&srcRoot, "src", "", "module root containing go.mod")
	flag.StringVar(&goldPath, "gold", "", "path to gold.patch")
	flag.StringVar(&pkgsFlag, "pkgs", "", "comma-separated package dirs relative to -src")
	flag.StringVar(&hiddenFlag, "hidden", "", "comma-separated hidden test .go files")
	flag.StringVar(&nametestFlag, "nametest", "", "comma-separated relpath.go:TestName roots")
	flag.StringVar(&hiddenRoot, "hiddenroot", "", "hidden suite root; in-package test files under it map to module pkgs")
	flag.StringVar(&outPath, "out", "", "output json path")
	flag.Parse()

	modPath = modulePath(srcRoot)
	for _, d := range strings.Split(pkgsFlag, ",") {
		if d = strings.TrimSpace(d); d != "" {
			loadPkg(filepath.Join(srcRoot, filepath.FromSlash(d)), filepath.ToSlash(d))
		}
	}
	buildIndexes()
	gold := parseGold(goldPath)

	tests := map[string]TestResult{}
	var hiddenFiles []string
	for _, hf := range strings.Split(hiddenFlag, ",") {
		if hf = strings.TrimSpace(hf); hf != "" {
			hiddenFiles = append(hiddenFiles, hf)
		}
	}
	for name, res := range scanTestGroups(hiddenFiles, hiddenRoot, gold) {
		tests[name] = res
	}
	var namedFiles []string
	wantNamed := map[string]bool{}
	for _, nt := range strings.Split(nametestFlag, ",") {
		if nt = strings.TrimSpace(nt); nt != "" {
			if parts := strings.SplitN(nt, ":", 2); len(parts) == 2 {
				namedFiles = append(namedFiles, filepath.Join(srcRoot, filepath.FromSlash(parts[0])))
				wantNamed[parts[1]] = true
			}
		}
	}
	for name, res := range scanTestGroups(namedFiles, srcRoot, gold) {
		if wantNamed[name] {
			tests["orig:"+name] = res
		}
	}

	out := Out{
		Module:     modPath,
		Tests:      tests,
		GoldDefs:   sortedKeys(gold.defs),
		GoldRefs:   sortedKeys(gold.refs),
		Excised:    sortedKeys(gold.excised),
		PkgSymbols: pkgSymbols(),
		Warnings:   warnings,
	}
	data, _ := json.MarshalIndent(out, "", "  ")
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func modulePath(root string) string {
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		warnings = append(warnings, "no go.mod at "+root)
		return ""
	}
	for _, ln := range strings.Split(string(data), "\n") {
		if ln = strings.TrimSpace(ln); strings.HasPrefix(ln, "module ") {
			return strings.Fields(ln)[1]
		}
	}
	return ""
}

// ---------- package loading ----------

func loadPkg(dir, rel string) {
	rel = strings.Trim(rel, "/")
	pkgPath := modPath
	if rel != "" && rel != "." {
		pkgPath = modPath + "/" + rel
	}
	p := &PkgInfo{
		Path: pkgPath, Imports: map[string]string{},
		Funcs: map[string]*FuncInfo{}, Types: map[string]bool{},
		Vars: map[string]bool{}, MethodNames: map[string][]string{},
	}
	pkgs[pkgPath] = p
	entries, err := os.ReadDir(dir)
	if err != nil {
		warnings = append(warnings, "cannot read pkg dir "+dir)
		return
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") {
			parseFile(filepath.Join(dir, e.Name()), p)
		}
	}
}

func parseFile(fname string, p *PkgInfo) {
	f, err := parser.ParseFile(fset, fname, nil, 0)
	if err != nil {
		warnings = append(warnings, "parse fail "+fname+": "+err.Error())
		return
	}
	for _, im := range f.Imports {
		ip := strings.Trim(im.Path.Value, `"`)
		alias := path.Base(ip)
		if im.Name != nil {
			alias = im.Name.Name
		}
		if alias != "_" && alias != "." {
			p.Imports[alias] = ip
		}
	}
	for _, d := range f.Decls {
		switch decl := d.(type) {
		case *ast.FuncDecl:
			qname := p.Path + "." + decl.Name.Name
			if decl.Recv != nil && len(decl.Recv.List) > 0 {
				rt := recvTypeName(decl.Recv.List[0].Type)
				qname = p.Path + ".(" + rt + ")." + decl.Name.Name
				p.MethodNames[decl.Name.Name] = append(p.MethodNames[decl.Name.Name], qname)
			}
			fi := &FuncInfo{QName: qname, Uses: map[string]bool{}, VarTypes: map[string]string{}}
			if decl.Type.Results != nil {
				for _, r := range decl.Type.Results.List {
					if tn := typeName(r.Type, p); tn != "" {
						fi.Results = append(fi.Results, tn)
					}
				}
			}
			bindParams(decl.Type.Params, p, fi)
			p.Funcs[qname] = fi
			if decl.Body != nil {
				collectUses(decl.Body, p, fi)
			}
		case *ast.GenDecl:
			for _, sp := range decl.Specs {
				switch s := sp.(type) {
				case *ast.TypeSpec:
					p.Types[s.Name.Name] = true
				case *ast.ValueSpec:
					for _, nm := range s.Names {
						p.Vars[nm.Name] = true
					}
				}
			}
		}
	}
}

func bindParams(fl *ast.FieldList, p *PkgInfo, fi *FuncInfo) {
	if fl == nil {
		return
	}
	for _, prm := range fl.List {
		tn := qualType(p, typeName(prm.Type, p))
		for _, nm := range prm.Names {
			if tn != "" {
				fi.VarTypes[nm.Name] = tn
			}
		}
	}
}

func recvTypeName(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.StarExpr:
		return recvTypeName(t.X)
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr:
		return recvTypeName(t.X)
	case *ast.IndexListExpr:
		return recvTypeName(t.X)
	}
	return ""
}

func typeName(e ast.Expr, p *PkgInfo) string {
	switch t := e.(type) {
	case *ast.StarExpr:
		return typeName(t.X, p)
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		if id, ok := t.X.(*ast.Ident); ok {
			if ip, ok := p.Imports[id.Name]; ok {
				return ip + "." + t.Sel.Name
			}
			return id.Name + "." + t.Sel.Name
		}
	case *ast.IndexExpr:
		return typeName(t.X, p)
	case *ast.IndexListExpr:
		return typeName(t.X, p)
	case *ast.ArrayType:
		return typeName(t.Elt, p)
	case *ast.MapType:
		return typeName(t.Value, p)
	case *ast.ChanType:
		return typeName(t.Value, p)
	case *ast.FuncType:
		return "func"
	}
	return ""
}

// qualType makes a bare same-package type name package-qualified.
func qualType(p *PkgInfo, tn string) string {
	if tn != "" && !strings.Contains(tn, ".") {
		return p.Path + "." + tn
	}
	return tn
}

// methodQName: "pkgpath.Type" + "M" -> "pkgpath.(Type).M".
func methodQName(tn, m string) string {
	i := strings.LastIndex(tn, ".")
	if i < 0 {
		return ""
	}
	return tn[:i] + ".(" + tn[i+1:] + ")." + m
}

func collectUses(body *ast.BlockStmt, p *PkgInfo, fi *FuncInfo) {
	callSels := map[*ast.SelectorExpr]bool{}
	ast.Inspect(body, func(n ast.Node) bool {
		if ce, ok := n.(*ast.CallExpr); ok {
			if se, ok := ast.Unparen(ce.Fun).(*ast.SelectorExpr); ok {
				callSels[se] = true
			}
		}
		return true
	})
	// pass 1: var -> type bindings
	ast.Inspect(body, func(n ast.Node) bool {
		switch s := n.(type) {
		case *ast.AssignStmt:
			if s.Tok.String() != ":=" {
				return true
			}
			for i, lhs := range s.Lhs {
				id, ok := lhs.(*ast.Ident)
				if !ok || i >= len(s.Rhs) {
					continue
				}
				if tn := qualType(p, exprType(s.Rhs[i], p, fi)); tn != "" {
					fi.VarTypes[id.Name] = tn
				}
			}
		case *ast.DeclStmt:
			if gd, ok := s.Decl.(*ast.GenDecl); ok {
				for _, sp := range gd.Specs {
					if vs, ok := sp.(*ast.ValueSpec); ok && vs.Type != nil {
						if tn := qualType(p, typeName(vs.Type, p)); tn != "" {
							for _, nm := range vs.Names {
								fi.VarTypes[nm.Name] = tn
							}
						}
					}
				}
			}
		}
		return true
	})
	// pass 2: uses
	ast.Inspect(body, func(n ast.Node) bool {
		switch e := n.(type) {
		case *ast.CallExpr:
			switch fun := ast.Unparen(e.Fun).(type) {
			case *ast.Ident:
				fi.Uses["call:"+p.Path+"."+fun.Name] = true
			case *ast.SelectorExpr:
				fi.Uses["call:"+selectorName(fun, p, fi)] = true
			case *ast.IndexExpr:
				if id, ok := fun.X.(*ast.Ident); ok {
					fi.Uses["call:"+p.Path+"."+id.Name] = true
				}
			case *ast.IndexListExpr:
				if id, ok := fun.X.(*ast.Ident); ok {
					fi.Uses["call:"+p.Path+"."+id.Name] = true
				}
			}
		case *ast.SelectorExpr:
			if callSels[e] {
				return true
			}
			if id, ok := e.X.(*ast.Ident); ok {
				if ip, ok := p.Imports[id.Name]; ok {
					fi.Uses["ref:"+ip+"."+e.Sel.Name] = true
				} else if tn, ok := fi.VarTypes[id.Name]; ok {
					if mq := methodQName(tn, e.Sel.Name); mq != "" {
						fi.Uses["ref:"+mq] = true
					}
				} else {
					fi.Uses["ref:bare:"+e.Sel.Name] = true
				}
			}
		case *ast.CompositeLit:
			if tn := qualType(p, typeName(e.Type, p)); tn != "" {
				fi.Uses["ref:"+tn] = true
			}
		}
		return true
	})
}

func exprType(e ast.Expr, p *PkgInfo, fi *FuncInfo) string {
	switch r := ast.Unparen(e).(type) {
	case *ast.CompositeLit:
		return typeName(r.Type, p)
	case *ast.UnaryExpr:
		if cl, ok := r.X.(*ast.CompositeLit); ok {
			return typeName(cl.Type, p)
		}
	case *ast.CallExpr:
		return callResultType(r, p, fi)
	case *ast.Ident:
		return fi.VarTypes[r.Name]
	}
	return ""
}

func selectorName(se *ast.SelectorExpr, p *PkgInfo, fi *FuncInfo) string {
	if id, ok := se.X.(*ast.Ident); ok {
		if ip, ok := p.Imports[id.Name]; ok {
			return ip + "." + se.Sel.Name
		}
		if tn, ok := fi.VarTypes[id.Name]; ok {
			if mq := methodQName(tn, se.Sel.Name); mq != "" {
				return mq
			}
		}
		return "bare:" + se.Sel.Name
	}
	return "bare:" + se.Sel.Name
}

func callResultType(ce *ast.CallExpr, p *PkgInfo, fi *FuncInfo) string {
	switch fun := ast.Unparen(ce.Fun).(type) {
	case *ast.Ident:
		if _, ok := p.Types[fun.Name]; ok {
			return fun.Name // conversion T(x)
		}
		if f, ok := p.Funcs[p.Path+"."+fun.Name]; ok && len(f.Results) > 0 {
			return f.Results[0]
		}
	case *ast.SelectorExpr:
		if id, ok := fun.X.(*ast.Ident); ok {
			if ip, ok := p.Imports[id.Name]; ok {
				nm := fun.Sel.Name
				if strings.HasPrefix(nm, "New") && len(nm) > 3 {
					return ip + "." + strings.TrimPrefix(nm, "New")
				}
				// constructor-style: pkg.MakeT() not recognized; leave unbound
				return ""
			}
		}
	}
	return ""
}

func buildIndexes() {
	for _, p := range pkgs {
		for qn, fi := range p.Funcs {
			funcIndex[qn] = fi
		}
		for mn, qns := range p.MethodNames {
			methodByName[mn] = append(methodByName[mn], qns...)
		}
	}
}

func pkgSymbols() []string {
	var out []string
	for _, p := range pkgs {
		for qn := range p.Funcs {
			out = append(out, qn)
		}
		for t := range p.Types {
			out = append(out, p.Path+"."+t)
		}
		for v := range p.Vars {
			out = append(out, p.Path+"."+v)
		}
	}
	sort.Strings(out)
	return out
}

// ---------- gold patch ----------

type GoldInfo struct {
	defs    map[string]bool
	refs    map[string]bool
	excised map[string]bool
}

var (
	excisedRe = regexp.MustCompile(`panic\("excised:\s*([^"]+)"\)`)
	funcHdrRe = regexp.MustCompile(`^func\s+(?:\(\s*\w+\s+\*?[\w.]+(?:\[[\w,\s]*\])?\s*\)\s+)?([A-Za-z_]\w*)\s*\(`)
	recvOfRe  = regexp.MustCompile(`^func\s+\(\s*\w+\s+\*?([\w.]+)`)
	goldUses  = map[string]map[string]bool{}
)

func pkgOf(qname string) string {
	if i := strings.LastIndex(qname, ".("); i >= 0 {
		return qname[:i]
	}
	return qname[:strings.LastIndex(qname, ".")]
}

func qualifyMarker(pkgPath, marker string) string {
	if i := strings.Index(marker, "."); i > 0 {
		return pkgPath + ".(" + marker[:i] + ")." + marker[i+1:]
	}
	return pkgPath + "." + marker
}

// funcQName builds "pkgpath.(T).F" or "pkgpath.F" from a func signature line.
func funcQName(pkgPath, sigline string) string {
	sigline = strings.TrimSpace(sigline)
	name := ""
	recv := ""
	if m := funcHdrRe.FindStringSubmatch(sigline); m != nil {
		name = m[1]
	}
	if name == "" {
		return ""
	}
	if strings.HasPrefix(sigline, "func (") || strings.HasPrefix(sigline, "func\t(") {
		if m := recvOfRe.FindStringSubmatch(sigline); m != nil {
			recv = m[1]
		}
	}
	if recv != "" {
		if i := strings.Index(recv, "["); i >= 0 {
			recv = recv[:i]
		}
		return pkgPath + ".(" + recv + ")." + name
	}
	return pkgPath + "." + name
}

func parseGold(gf string) *GoldInfo {
	g := &GoldInfo{
		defs: map[string]bool{}, refs: map[string]bool{},
		excised: map[string]bool{},
	}
	data, err := os.ReadFile(gf)
	if err != nil {
		warnings = append(warnings, "no gold.patch at "+gf)
		return g
	}
	var curPkgPath, curFunc, curSig, curBody string
	var curImports map[string]string
	bodyOpen := false

	flush := func() {
		if curFunc == "" {
			return
		}
		sig := strings.TrimSpace(curSig)
		var src string
		if sig == "" {
			sig = "func _x()"
		}
		if strings.HasSuffix(sig, "}") {
			// one-line func: signature line already carries the whole body
			src = "package fake\n" + sig + "\n"
		} else {
			if !strings.HasSuffix(sig, "{") {
				sig += " {"
			}
			src = "package fake\n" + sig + "\n" + curBody + "\n}\n"
		}
		p := &PkgInfo{Path: curPkgPath, Imports: curImports}
		f, err := parser.ParseFile(fset, "gold_body.go", src, 0)
		if err != nil {
			// hunk-header sigs can be truncated mid-params and mid-function
			// hunks add stray '}' lines; retry with a generic signature and
			// balanced braces, then fall back to regex call extraction
			retryBody := strings.TrimRight(curBody, " \t\n")
			for strings.HasSuffix(retryBody, "}") {
				retryBody = strings.TrimRight(retryBody[:len(retryBody)-1], " \t\n")
			}
			f2, err2 := parser.ParseFile(fset, "gold_body.go",
				"package fake\nfunc _x() {\n"+retryBody+"\n}\n", 0)
			if err2 == nil {
				f = f2
			} else {
				uses := regexCallUses(curBody, p)
				goldUses[curFunc] = uses
				for u := range uses {
					if i := strings.Index(u, ":"); i >= 0 {
						g.refs[u[i+1:]] = true
					}
				}
				warnings = append(warnings, "gold body parse fail "+curFunc+" (regex fallback)")
				curFunc, curSig, curBody, bodyOpen = "", "", "", false
				return
			}
		}
		for _, d := range f.Decls {
			if fd, ok := d.(*ast.FuncDecl); ok {
				fi := &FuncInfo{QName: curFunc, Uses: map[string]bool{}, VarTypes: map[string]string{}}
				bindParams(fd.Type.Params, p, fi)
				collectUses(fd.Body, p, fi)
				goldUses[curFunc] = fi.Uses
				for u := range fi.Uses {
					if i := strings.Index(u, ":"); i >= 0 {
						g.refs[u[i+1:]] = true
					}
				}
			}
		}
		curFunc, curSig, curBody, bodyOpen = "", "", "", false
	}

	for _, ln := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(ln, "+++ a/") || strings.HasPrefix(ln, "+++ b/") {
			flush()
			rel := ln[6:]
			dir := path.Dir(rel)
			curPkgPath = modPath
			if dir != "." && dir != "" {
				curPkgPath = modPath + "/" + dir
			}
			if p, ok := pkgs[curPkgPath]; ok {
				curImports = p.Imports
			} else {
				curImports = map[string]string{}
			}
			continue
		}
		if strings.HasPrefix(ln, "@@") {
			if i := strings.LastIndex(ln, "func "); i >= 0 {
				sig := strings.TrimSpace(ln[i:])
				if qn := funcQName(curPkgPath, sig); qn != "" {
					flush()
					curFunc, curSig, bodyOpen = qn, sig, true
					g.defs[qn] = true
				}
			}
			continue
		}
		if len(ln) == 0 {
			continue
		}
		c, text := ln[0], ""
		if len(ln) > 1 {
			text = ln[1:]
		}
		trim := strings.TrimSpace(text)
		switch c {
		case '-':
			if m := excisedRe.FindStringSubmatch(text); m != nil {
				// the signature for this excised func was context above
				qn := qualifyMarker(curPkgPath, m[1])
				if curFunc != qn {
					// hunk header may not have carried the sig; switch
					g.defs[qn] = true
					g.excised[qn] = true
					prevFunc, prevSig := curFunc, curSig
					curFunc, curBody = qn, ""
					if prevFunc == "" || prevSig == "" {
						curSig = ""
					} else {
						curSig = prevSig
					}
				} else {
					g.defs[qn] = true
					g.excised[qn] = true
				}
				bodyOpen = true
			}
		case '+', ' ':
			if qn := funcQName(curPkgPath, trim); qn != "" && strings.HasPrefix(trim, "func") {
				// a real signature line (context or added) — not a literal
				flush()
				curFunc, curSig, bodyOpen = qn, trim, true
				if c == '+' {
					g.defs[qn] = true
				}
				continue
			}
			if bodyOpen && c == '+' {
				curBody += text + "\n"
			}
		}
	}
	flush()
	return g
}

var (
	rxSelCall = regexp.MustCompile(`\b([A-Za-z_]\w*)\.([A-Za-z_]\w*)\s*\(`)
	rxIdCall  = regexp.MustCompile(`\b([A-Za-z_]\w*)\s*\(`)
	rxSelRef  = regexp.MustCompile(`\b([A-Za-z_]\w*)\.([A-Za-z_]\w*)\b`)
	rxLitType = regexp.MustCompile(`\b([A-Za-z_]\w*)\.([A-Za-z_]\w*)\s*\{`)
	rxBareLit = regexp.MustCompile(`[^.\w]([A-Z]\w*)\s*\{`)
)

var goldSkipIdents = map[string]bool{
	"if": true, "for": true, "switch": true, "func": true, "return": true,
	"go": true, "defer": true, "select": true, "case": true, "range": true,
	"append": true, "cap": true, "close": true, "copy": true, "delete": true,
	"imag": true, "len": true, "make": true, "new": true, "panic": true,
	"print": true, "println": true, "real": true, "recover": true, "clear": true,
	"min": true, "max": true, "string": true, "int": true, "int8": true,
	"int16": true, "int32": true, "int64": true, "uint": true, "uint8": true,
	"uint16": true, "uint32": true, "uint64": true, "uintptr": true,
	"byte": true, "rune": true, "bool": true, "error": true, "float32": true,
	"float64": true, "complex64": true, "complex128": true, "any": true,
	"comparable": true, "stringer": true, "t": true, "err": true, "ok": true,
}

// regexCallUses extracts call/ref uses from an unparseable gold body fragment.
// It cannot type-check, so it emits the same use strings collectUses would
// for the syntactically obvious cases and lets resolveCall do the lookup.
func regexCallUses(body string, p *PkgInfo) map[string]bool {
	uses := map[string]bool{}
	for _, m := range rxSelCall.FindAllStringSubmatch(body, -1) {
		if ip, ok := p.Imports[m[1]]; ok {
			uses["call:"+ip+"."+m[2]] = true
		} else {
			uses["call:bare:"+m[2]] = true
		}
	}
	for _, ix := range rxIdCall.FindAllStringSubmatchIndex(body, -1) {
		name := body[ix[2]:ix[3]]
		if goldSkipIdents[name] || (ix[0] > 0 && body[ix[0]-1] == '.') {
			continue
		}
		uses["call:"+p.Path+"."+name] = true
	}
	for _, m := range rxLitType.FindAllStringSubmatch(body, -1) {
		if ip, ok := p.Imports[m[1]]; ok {
			uses["ref:"+ip+"."+m[2]] = true
		}
	}
	for _, m := range rxBareLit.FindAllStringSubmatch(body, -1) {
		uses["ref:"+p.Path+"."+m[1]] = true
	}
	for _, m := range rxSelRef.FindAllStringSubmatch(body, -1) {
		if ip, ok := p.Imports[m[1]]; ok {
			uses["ref:"+ip+"."+m[2]] = true
		}
	}
	return uses
}

// ---------- test roots & reachability ----------

func fileImports(f *ast.File) map[string]string {
	imports := map[string]string{}
	for _, im := range f.Imports {
		ip := strings.Trim(im.Path.Value, `"`)
		alias := path.Base(ip)
		if im.Name != nil {
			alias = im.Name.Name
		}
		if alias != "_" && alias != "." {
			imports[alias] = ip
		}
	}
	return imports
}

// scanGoFile parses one .go file, returns FuncInfo per declared func
// (keyed by declared name), resolving refs through its own imports.
func scanGoFile(fname, pkgPath string) map[string]*FuncInfo {
	out := map[string]*FuncInfo{}
	f, err := parser.ParseFile(fset, fname, nil, 0)
	if err != nil {
		warnings = append(warnings, "test parse fail "+fname+": "+err.Error())
		return out
	}
	p := &PkgInfo{Path: pkgPath, Imports: fileImports(f)}
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Body == nil {
			continue
		}
		fi := &FuncInfo{QName: pkgPath + "." + fd.Name.Name, Uses: map[string]bool{}, VarTypes: map[string]string{}}
		bindParams(fd.Type.Params, p, fi)
		collectUses(fd.Body, p, fi)
		out[fd.Name.Name] = fi
	}
	return out
}

// testPkgPath resolves the package path a test file's unqualified calls bind
// to: external `foo_test` packages get the synthetic "test" path (locals only);
// in-package files bind to the package mirroring their dir under root.
func testPkgPath(fname, root, pkgName string) string {
	if strings.HasSuffix(pkgName, "_test") {
		return "test"
	}
	rel, err := filepath.Rel(root, filepath.Dir(fname))
	if err != nil || rel == "." || rel == "" {
		return modPath
	}
	return modPath + "/" + filepath.ToSlash(rel)
}

// scanTestGroups groups test files by (dir, declared package) so calls to
// helpers defined in a sibling file of the same test package still resolve,
// then computes the reach of every Test* function.
func scanTestGroups(files []string, root string, gold *GoldInfo) map[string]TestResult {
	out := map[string]TestResult{}
	type group struct {
		path   string
		locals map[string]*FuncInfo
	}
	groups := map[string]*group{}
	order := []string{}
	for _, fname := range files {
		f, err := parser.ParseFile(fset, fname, nil, 0)
		if err != nil {
			warnings = append(warnings, "test parse fail "+fname+": "+err.Error())
			continue
		}
		pkgName := f.Name.Name
		path := testPkgPath(fname, root, pkgName)
		key := filepath.Dir(fname) + "|" + pkgName
		g, ok := groups[key]
		if !ok {
			g = &group{path: path, locals: map[string]*FuncInfo{}}
			groups[key] = g
			order = append(order, key)
		}
		p := &PkgInfo{Path: path, Imports: fileImports(f)}
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			qn := path + "." + fd.Name.Name
			if fd.Recv != nil && len(fd.Recv.List) > 0 {
				qn = path + ".(" + recvTypeName(fd.Recv.List[0].Type) + ")." + fd.Name.Name
			}
			fi := &FuncInfo{QName: qn, Uses: map[string]bool{}, VarTypes: map[string]string{}}
			bindParams(fd.Type.Params, p, fi)
			collectUses(fd.Body, p, fi)
			g.locals[fd.Name.Name] = fi
		}
	}
	for _, key := range order {
		g := groups[key]
		for name, fi := range g.locals {
			if !strings.HasPrefix(name, "Test") || fi.QName != g.path+"."+name {
				continue // Test-prefixed methods and non-func decls are not roots
			}
			defs, refs := reach(fi, gold, g.locals)
			out[name] = TestResult{Defs: defs, Refs: refs}
		}
	}
	return out
}

func resolveCall(name string, refs map[string]bool, locals map[string]*FuncInfo) (edges []*FuncInfo) {
	if strings.HasPrefix(name, "test.") {
		if fi, ok := locals[strings.TrimPrefix(name, "test.")]; ok {
			return []*FuncInfo{fi}
		}
	}
	if strings.HasPrefix(name, "bare:") {
		for _, qn := range methodByName[strings.TrimPrefix(name, "bare:")] {
			if fi, ok := funcIndex[qn]; ok {
				edges = append(edges, fi)
			}
		}
		return edges
	}
	if fi, ok := funcIndex[name]; ok {
		return []*FuncInfo{fi}
	}
	// basename fallback: a test helper from the same group that was resolved
	// under its package path rather than the "test." convention
	if i := strings.LastIndex(name, "."); i >= 0 {
		if fi, ok := locals[name[i+1:]]; ok {
			return []*FuncInfo{fi}
		}
	}
	refs[name] = true
	return nil
}

func absorb(uses map[string]bool, defs, refs map[string]bool, seen map[string]bool,
	work *[]*FuncInfo, locals map[string]*FuncInfo) {
	for u := range uses {
		switch {
		case strings.HasPrefix(u, "call:"):
			for _, fi := range resolveCall(strings.TrimPrefix(u, "call:"), refs, locals) {
				if !seen[fi.QName] {
					seen[fi.QName] = true
					*work = append(*work, fi)
				}
			}
		case strings.HasPrefix(u, "ref:"):
			v := strings.TrimPrefix(u, "ref:")
			if strings.HasPrefix(v, "bare:") {
				for _, qn := range methodByName[strings.TrimPrefix(v, "bare:")] {
					if fi, ok := funcIndex[qn]; ok && !seen[qn] {
						seen[qn] = true
						*work = append(*work, fi)
					}
				}
				refs[v] = true
			} else {
				refs[v] = true
			}
		}
	}
}

func reach(root *FuncInfo, gold *GoldInfo, locals map[string]*FuncInfo) ([]string, []string) {
	defs := map[string]bool{}
	refs := map[string]bool{}
	seen := map[string]bool{root.QName: true}
	var work []*FuncInfo
	absorb(root.Uses, defs, refs, seen, &work, locals)
	for len(work) > 0 {
		fi := work[len(work)-1]
		work = work[:len(work)-1]
		defs[fi.QName] = true
		uses := fi.Uses
		if gold != nil && gold.excised[fi.QName] {
			uses = goldUses[fi.QName]
		}
		absorb(uses, defs, refs, seen, &work, locals)
	}
	return sortedKeys(defs), sortedKeys(refs)
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
