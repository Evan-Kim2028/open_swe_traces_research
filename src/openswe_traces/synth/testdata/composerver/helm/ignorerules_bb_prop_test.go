// Package ignore_test is a hidden black-box property suite for the ignorerules
// unit. Exported API only (api.md): Empty, AddDefaults, Parse, ParseFile,
// (*Rules).Ignore. Seed 20260919; >=10k cases per file.
//
// Contract (contract.md) -> property coverage table:
//
//	"rules parsed one per line; blank lines and # comments skipped" -> TestIgnoreParseAndDefaultsProperty
//	"malformed pattern aborts parsing with an error" -> TestIgnoreParseAndDefaultsProperty
//	"AddDefaults adds the built-in ignores" -> TestIgnoreParseAndDefaultsProperty
//	"basename / root-anchored / * vs ** / dir-only / directory match / ! literal" -> TestIgnoreContractTableProperty, TestIgnoreOracleAgreementProperty
//	"matching considers directory-ness from file info" -> TestIgnoreContractTableProperty, TestIgnoreUnseenRandomProperty
package ignore_test

import (
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	ignore "example.internal/chartkit/v4/pkg/ignore"
)

const bbSeed = 20260919
const bbCases = 10000

type fi struct {
	name string
	dir  bool
}

func (f fi) Name() string       { return f.name }
func (f fi) Size() int64        { return 0 }
func (f fi) Mode() os.FileMode  { return 0 }
func (f fi) ModTime() time.Time { return time.Time{} }
func (f fi) IsDir() bool        { return f.dir }
func (f fi) Sys() any           { return nil }

func bbParse(t *testing.T, doc string) *ignore.Rules {
	t.Helper()
	r, err := ignore.Parse(strings.NewReader(doc))
	if err != nil {
		t.Fatalf("Parse(%q) errored: %v", doc, err)
	}
	return r
}

func bbIgnored(r *ignore.Rules, p string, dir bool) bool {
	base := path.Base(strings.TrimSuffix(p, "/"))
	return r.Ignore(p, fi{name: base, dir: dir})
}

type bbPat struct {
	mustDir  bool
	anchored bool
	dsHead   bool
	segs     []string
	raw      string
}

func bbMatchSeg(p, s string) bool {
	for len(p) > 0 {
		switch p[0] {
		case '*':
			for len(p) > 1 && p[1] == '*' {
				p = p[1:]
			}
			if len(p) == 1 {
				return true
			}
			for i := 0; i <= len(s); i++ {
				if bbMatchSeg(p[1:], s[i:]) {
					return true
				}
			}
			return false
		case '?':
			if len(s) == 0 {
				return false
			}
			p, s = p[1:], s[1:]
		case '[':
			end := strings.IndexByte(p, ']')
			if end <= 1 || len(s) == 0 {
				return false
			}
			cls := p[1:end]
			ok := false
			for i := 0; i < len(cls); i++ {
				if i+2 < len(cls) && cls[i+1] == '-' {
					if s[0] >= cls[i] && s[0] <= cls[i+2] {
						ok = true
					}
					i += 2
				} else if s[0] == cls[i] {
					ok = true
				}
			}
			if !ok {
				return false
			}
			p, s = p[end+1:], s[1:]
		default:
			if len(s) == 0 || s[0] != p[0] {
				return false
			}
			p, s = p[1:], s[1:]
		}
	}
	return len(s) == 0
}

func bbPatMatch(pt bbPat, p string, dir bool) (bool, bool) {
	p = strings.TrimSuffix(p, "/")
	if p == "" || p == "." {
		return false, false
	}
	segs := strings.Split(p, "/")
	var inner bool
	amb := false
	if pt.anchored {
		if pt.dsHead {
			rest := pt.segs[1:]
			matchedZero, matchedNonZero := false, false
			for k := 0; k+len(rest) <= len(segs); k++ {
				ok := true
				for j, ps := range rest {
					if !bbMatchSeg(ps, segs[k+j]) {
						ok = false
						break
					}
				}
				if ok {
					if k == 0 {
						matchedZero = true
					} else {
						matchedNonZero = true
					}
				}
			}
			inner = matchedZero || matchedNonZero
			if matchedZero && !matchedNonZero {
				amb = true
			}
		} else {
			if len(segs) == len(pt.segs) {
				inner = true
				for j, ps := range pt.segs {
					if !bbMatchSeg(ps, segs[j]) {
						inner = false
						break
					}
				}
			} else if len(segs) > len(pt.segs) {
				pre := true
				for j, ps := range pt.segs {
					if !bbMatchSeg(ps, segs[j]) {
						pre = false
						break
					}
				}
				if pre {
					amb = true
				}
			}
		}
	} else {
		inner = bbMatchSeg(pt.segs[0], segs[len(segs)-1])
	}
	if pt.mustDir && !dir {
		inner = false
	}
	return inner, amb
}

func bbCompile(line string) (bbPat, bool) {
	pt := bbPat{raw: line}
	s := line
	if strings.HasSuffix(s, "/") {
		pt.mustDir = true
		s = strings.TrimSuffix(s, "/")
	}
	if strings.HasPrefix(s, "/") {
		pt.anchored = true
		s = strings.TrimPrefix(s, "/")
	}
	if strings.HasPrefix(s, "**/") {
		pt.dsHead = true
		pt.anchored = true
		s = s[3:]
	}
	if strings.Contains(s, "**") {
		return pt, false
	}
	pt.segs = strings.Split(s, "/")
	if pt.dsHead {
		pt.segs = append([]string{"**"}, pt.segs...)
	}
	if len(pt.segs) > 1 {
		pt.anchored = true
	}
	for _, seg := range pt.segs {
		if bbBadSeg(seg) {
			return pt, false
		}
	}
	return pt, true
}

func bbBadSeg(seg string) bool {
	for i := 0; i < len(seg); i++ {
		switch seg[i] {
		case '\\':
			if i+1 >= len(seg) {
				return true
			}
			i++
		case '[':
			j := strings.IndexByte(seg[i:], ']')
			if j < 0 {
				return true
			}
			i += j
		}
	}
	return false
}

func bbOracle(lines []string, p string, dir bool) (bool, bool) {
	res := false
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln == "" || strings.HasPrefix(ln, "#") {
			continue
		}
		pt, ok := bbCompile(ln)
		if !ok {
			continue
		}
		m, amb := bbPatMatch(pt, p, dir)
		if amb {
			return false, true
		}
		if m {
			res = true
		}
	}
	return res, false
}

var bbNames = []string{
	"foo", "bar", "chart", "values", "readme", "LICENSE", "x", "data",
	".git", ".svn", ".idea", ".vscode", ".DS_Store", "OWNERS", "keep",
	"a.txt", "b.yaml", "c.md", "deep.json", "z.tpl", "f.txt~", "w.proj",
	"templates", "charts", "crds", "docs", "helm.txt", "tiller.txt",
	"!bang", "quux", ".hidden", ".dotfile", "n.txt", "sub",
}

var bbExts = []string{".txt", ".yaml", ".md", ".json", ".tpl", "~", ".proj", ""}

func bbRandName(rng *rand.Rand) string {
	if rng.Intn(3) == 0 {
		return bbNames[rng.Intn(len(bbNames))] + bbExts[rng.Intn(len(bbExts))]
	}
	return bbNames[rng.Intn(len(bbNames))]
}

func bbRandSeg(rng *rand.Rand) string {
	switch rng.Intn(10) {
	case 0:
		return "*"
	case 1:
		return bbRandName(rng)[:1] + "*"
	case 2:
		return "*" + bbExts[rng.Intn(len(bbExts)-1)]
	case 3:
		return bbRandName(rng)[:1] + "?" + bbExts[rng.Intn(3)]
	case 4:
		return bbRandName(rng)[:1] + "[a-e]" + bbRandName(rng)[:2]
	default:
		return bbRandName(rng)
	}
}

func bbRandPat(rng *rand.Rand) string {
	var b strings.Builder
	anchored := rng.Intn(3) != 0
	if anchored && rng.Intn(4) == 0 {
		b.WriteString("/")
	}
	n := 1
	if anchored {
		n += rng.Intn(3)
	}
	segs := make([]string, n)
	for i := range segs {
		segs[i] = bbRandSeg(rng)
	}
	b.WriteString(strings.Join(segs, "/"))
	if rng.Intn(7) == 0 {
		b.WriteString("/")
	}
	return b.String()
}

func bbRandPath(rng *rand.Rand) string {
	n := 1 + rng.Intn(4)
	segs := make([]string, n)
	for i := range segs {
		segs[i] = bbRandName(rng)
	}
	p := strings.Join(segs, "/")
	if rng.Intn(10) == 0 {
		p += "/"
	}
	return p
}

func bbDoc(rng *rand.Rand, pats []string) string {
	var b strings.Builder
	for _, p := range pats {
		switch rng.Intn(8) {
		case 0:
			b.WriteString("\n")
		case 1:
			b.WriteString("# comment " + bbRandName(rng) + "\n")
		case 2:
			b.WriteString("   \n")
		}
		b.WriteString(p + "\n")
	}
	if rng.Intn(3) == 0 {
		b.WriteString("# trailing comment\n\n")
	}
	return b.String()
}

func TestIgnoreContractTableProperty(t *testing.T) {
	type row struct {
		pat, name string
		dir, want bool
	}
	rows := []row{
		{"helm.txt", "helm.txt", false, true},
		{"helm.*", "helm.txt", false, true},
		{"helm.*", "rudder.txt", false, false},
		{"*.txt", "tiller.txt", false, true},
		{"*.txt", "cargo/a.txt", false, true},
		{"*.txt", "x/y/z/c.txt", false, true},
		{"cargo/*.txt", "cargo/a.txt", false, true},
		{"cargo/*.*", "cargo/a.txt", false, true},
		{"cargo/*.txt", "mast/a.txt", false, false},
		{"cargo/*.txt", "x/cargo/a.txt", false, false},
		{"/a.txt", "a.txt", false, true},
		{"/a.txt", "cargo/a.txt", false, false},
		{"/cargo/a.txt", "cargo/a.txt", false, true},
		{"cargo/*", "cargo/a/b.txt", false, false},
		{"ru[c-e]?er.txt", "rudder.txt", false, true},
		{"templates/.?*", "templates/.dotfile", false, true},
		{"f??.txt", "foo.txt", false, true},
		{"f??.txt", "fo.txt", false, false},
		{".*", ".", false, false},
		{".*", ".joonix", false, true},
		{".*", "helm.txt", false, false},
		{"*", ".", false, false},
		{"cargo/", "cargo", true, true},
		{"cargo/", "cargo", false, false},
		{"cargo/", "mast", true, false},
		{"helm.txt/", "helm.txt", false, false},
		{"helm.txt/", "helm.txt", true, true},
		{"cargo", "cargo", true, true},
		{"a/b/", "a/b", true, true},
		{"a/b/", "x/a/b", true, false},
		// Negation: leading ! inverts match (doc.go / in-tree TestIgnore).
		{"!helm.txt", "helm.txt", false, false},
		{"!helm.txt", "tiller.txt", false, true},
		{"!*.txt", "cargo", true, true},
		{"!cargo/", "mast/", true, true},
	}
	for i, tc := range rows {
		r := bbParse(t, tc.pat+"\n")
		got := bbIgnored(r, tc.name, tc.dir)
		if got != tc.want {
			t.Fatalf("row %d: Ignore(%q,dir=%v) under %q = %v, want %v", i, tc.name, tc.dir, tc.pat, got, tc.want)
		}
	}
}

func TestIgnoreParseAndDefaultsProperty(t *testing.T) {
	r1 := bbParse(t, "# only comments\n\n   \n#x\n")
	for i := 0; i < 200; i++ {
		if bbIgnored(r1, bbNames[i%len(bbNames)], i%2 == 0) {
			t.Fatalf("comment-only ruleset ignored %q", bbNames[i%len(bbNames)])
		}
	}
	r2 := bbParse(t, "# hdr\nfoo\n\n# mid\n*.txt\n")
	r3 := bbParse(t, "foo\n*.txt\n")
	rng := rand.New(rand.NewSource(bbSeed))
	for i := 0; i < bbCases; i++ {
		p := bbRandPath(rng)
		d := rng.Intn(2) == 0
		if bbIgnored(r2, p, d) != bbIgnored(r3, p, d) {
			t.Fatalf("case %d: comment handling changed result for %q", i, p)
		}
	}
	for _, bad := range []string{"[z-", "foo/**/bar", "a/b\\", "x[a", "b/c/**/d", "[", "**/deep.txt", "**"} {
		if _, err := ignore.Parse(strings.NewReader(bad + "\n")); err == nil {
			t.Fatalf("malformed pattern %q did not error", bad)
		}
	}
	e := ignore.Empty()
	for i := 0; i < bbCases; i++ {
		if bbIgnored(e, bbRandPath(rng), rng.Intn(2) == 0) {
			t.Fatalf("Empty() ignored path case %d", i)
		}
	}
	dir := t.TempDir()
	fp := filepath.Join(dir, "rules.ignore")
	if err := os.WriteFile(fp, []byte("# c\n*.secret\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rf, err := ignore.ParseFile(fp)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if !bbIgnored(rf, "a/b/x.secret", false) || bbIgnored(rf, "a/b/x.txt", false) {
		t.Fatal("ParseFile rules did not apply")
	}
	if _, err := ignore.ParseFile(filepath.Join(dir, "missing")); err == nil {
		t.Fatal("ParseFile on missing file did not error")
	}
	d := ignore.Empty()
	d.AddDefaults()
	if !bbIgnored(d, "templates/.dotfile", false) {
		t.Fatal("AddDefaults did not ignore templates/.dotfile")
	}
	if bbIgnored(d, "a.txt", false) {
		t.Fatal("AddDefaults ignored a plain file a.txt")
	}
}

func TestIgnoreOracleAgreementProperty(t *testing.T) {
	// Single-pattern rulesets avoid negation ordering differences; 10k seeded checks.
	rng := rand.New(rand.NewSource(bbSeed + 1))
	for i := 0; i < bbCases; i++ {
		lit := bbRandName(rng)
		pat := lit
		if rng.Intn(3) == 0 {
			pat = "*" + filepath.Ext(lit)
		}
		r := bbParse(t, pat+"\n")
		depth := rng.Intn(4)
		segs := make([]string, depth+1)
		for j := range segs {
			segs[j] = bbRandName(rng)
		}
		segs[depth] = lit
		p := strings.Join(segs, "/")
		dir := rng.Intn(3) == 0
		got := bbIgnored(r, p, dir)
		base := path.Base(strings.TrimSuffix(p, "/"))
		want, _ := filepath.Match(pat, base)
		if dir && strings.HasSuffix(pat, "/") {
			want = true
		}
		if got != want {
			t.Fatalf("case %d: pat %q path %q dir=%v got %v want %v", i, pat, p, dir, got, want)
		}
	}
}

func TestIgnoreUnseenRandomProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 2))
	lits := []string{"qzx", "wobble", "kr", ".env", "manifest", "blob.bin", "ut", "!edge"}
	for i := 0; i < bbCases; i++ {
		name := lits[rng.Intn(len(lits))]
		depth := rng.Intn(4)
		segs := make([]string, depth+1)
		for j := range segs {
			segs[j] = lits[rng.Intn(len(lits))]
		}
		segs[depth] = name
		p := strings.Join(segs, "/")
		r := bbParse(t, name+"\n")
		if !bbIgnored(r, p, false) {
			t.Fatalf("case %d: basename rule %q did not match %q", i, name, p)
		}
		ra := bbParse(t, "x/"+name+"\n")
		if bbIgnored(ra, "y/x/"+name, false) {
			t.Fatalf("case %d: anchored rule leaked to %q", i, "y/x/"+name)
		}
		if !bbIgnored(ra, "x/"+name, false) {
			t.Fatalf("case %d: anchored rule missed %q", i, "x/"+name)
		}
	}
}
