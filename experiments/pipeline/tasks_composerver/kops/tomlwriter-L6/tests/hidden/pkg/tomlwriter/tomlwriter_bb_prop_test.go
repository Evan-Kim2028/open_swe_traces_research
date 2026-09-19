// Package tomlwriter_test is a hidden black-box property suite for tomlwriter.
// Exported API only (api.md): NewTree, (*Tree).Table, SetPath, String.
// Seed 20260919; >=10k cases per file.
//
// Contract (contract.md) -> property coverage table:
//
//	"empty document is empty string" -> TestTomlEmptyAndRootTableProperty
//	"scalars before subtables, sorted keys" -> TestTomlScalarsOrderProperty, TestTomlUnseenRandomProperty
//	"table header at root" -> TestTomlEmptyAndRootTableProperty
//	"quoting and escape sequences" -> TestTomlQuotingContractProperty
//	"already-quoted keys not re-escaped" -> TestTomlQuotingContractProperty
//	"empty key is \"\"" -> TestTomlQuotingContractProperty
//	"control-char \\u form" -> TestTomlQuotingContractProperty
//	"scalar path element does not descend" -> TestTomlSetPathBehaviorProperty
//	"leaf replaces table" -> TestTomlSetPathBehaviorProperty
//	"non string/int64/bool panics" -> TestTomlSetPathUnsupportedTypeProperty
package tomlwriter_test

import (
	"fmt"
	"math/rand"
	"regexp"
	"sort"
	"strings"
	"testing"
	"unicode"

	tw "example.internal/clustkit/pkg/tomlwriter"
)

const bbSeed = 20260919
const bbCases = 10000

func bbBareKey(k string) bool {
	if k == "" {
		return false
	}
	for _, r := range k {
		switch {
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_', r == '-':
			continue
		default:
			return false
		}
	}
	return true
}

func bbEscapeString(value string) string {
	var b strings.Builder
	for _, r := range value {
		switch r {
		case '\b':
			b.WriteString("\\b")
		case '\t':
			b.WriteString("\\t")
		case '\n':
			b.WriteString("\\n")
		case '\f':
			b.WriteString("\\f")
		case '\r':
			b.WriteString("\\r")
		case '"':
			b.WriteString("\\\"")
		case '\\':
			b.WriteString("\\\\")
		default:
			if r < 0x1f {
				b.WriteString(fmt.Sprintf("\\u%04X", r))
			} else {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}

func bbQuoteKey(k string) string {
	if len(k) >= 2 && k[0] == '"' && k[len(k)-1] == '"' {
		return k
	}
	if bbBareKey(k) {
		return k
	}
	if k == "" {
		return "\"\""
	}
	return "\"" + bbEscapeString(k) + "\""
}

func bbCheckTableDoc(t *testing.T, doc string) {
	t.Helper()
	lines := strings.Split(doc, "\n")
	var scalarBlock []string
	var sawTable bool
	for _, ln := range lines {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(ln), "[") {
			if len(scalarBlock) > 1 {
				cp := append([]string(nil), scalarBlock...)
				sort.Strings(cp)
				if strings.Join(cp, "|") != strings.Join(scalarBlock, "|") {
					t.Fatalf("scalar keys not sorted before table: %q", doc)
				}
			}
			scalarBlock = nil
			sawTable = true
			continue
		}
		if strings.Contains(ln, " = ") && !sawTable {
			scalarBlock = append(scalarBlock, strings.SplitN(ln, " = ", 2)[0])
		}
	}
}

func bbRandBareKey(rng *rand.Rand) string {
	const alpha = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-"
	n := 1 + rng.Intn(8)
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteByte(alpha[rng.Intn(len(alpha))])
	}
	return b.String()
}

func bbRandScalar(rng *rand.Rand) any {
	switch rng.Intn(3) {
	case 0:
		return bbRandString(rng)
	case 1:
		return int64(rng.Intn(1_000_000) - 500_000)
	default:
		return rng.Intn(2) == 0
	}
}

func bbRandString(rng *rand.Rand) string {
	n := rng.Intn(12)
	var b strings.Builder
	for i := 0; i < n; i++ {
		switch rng.Intn(20) {
		case 0:
			b.WriteByte('\n')
		case 1:
			b.WriteByte('\t')
		case 2:
			b.WriteByte('"')
		case 3:
			b.WriteByte('\\')
		case 4:
			b.WriteRune(rune(rng.Intn(0x1f)))
		default:
			b.WriteRune(rune('a' + rng.Intn(26)))
		}
	}
	return b.String()
}

func TestTomlEmptyAndRootTableProperty(t *testing.T) {
	if tw.NewTree().String() != "" {
		t.Fatal("empty tree must be empty string")
	}
	tr := tw.NewTree()
	tr.Table("plugins")
	got := tr.String()
	if !strings.Contains(got, "[plugins]") {
		t.Fatalf("root table header missing: %q", got)
	}
	if strings.TrimSpace(got) == "" {
		t.Fatal("table-only tree should not be empty")
	}
}

func TestTomlQuotingContractProperty(t *testing.T) {
	tr := tw.NewTree()
	tr.SetPath([]string{""}, "empty-key-val")
	tr.SetPath([]string{"bare_ok"}, "x")
	tr.SetPath([]string{"needs.quote"}, "y")
	tr.SetPath([]string{"space here"}, "z")
	tr.SetPath([]string{`"dotted.key"`}, "prequoted")
	tr.SetPath([]string{"esc"}, "a\tb\nc\"d\\e")
	tr.SetPath([]string{"ctrl"}, string([]byte{0x00, 0x1e}))
	doc := tr.String()
	if !strings.Contains(doc, "\"\" = ") {
		t.Fatalf("empty key not quoted: %q", doc)
	}
	if !strings.Contains(doc, "bare_ok = ") {
		t.Fatalf("bare key should stay bare: %q", doc)
	}
	if strings.Contains(doc, "needs.quote = ") && !strings.Contains(doc, "\"needs.quote\"") {
		t.Fatalf("dotted key should be quoted: %q", doc)
	}
	if !strings.Contains(doc, "\\t") || !strings.Contains(doc, "\\n") {
		t.Fatalf("escape sequences missing: %q", doc)
	}
	if !regexp.MustCompile(`\\u00[0-9A-F]{2}`).MatchString(doc) {
		t.Fatalf("control char \\u escape missing: %q", doc)
	}
	if !strings.Contains(doc, `"dotted.key" = `) {
		t.Fatalf("pre-quoted key passthrough: %q", doc)
	}
}

func TestTomlSetPathBehaviorProperty(t *testing.T) {
	tr := tw.NewTree()
	tr.SetPath([]string{"a"}, "scalar")
	tr.Table("a", "b").SetPath([]string{"nested"}, int64(1))
	doc := tr.String()
	if strings.Contains(doc, "[a.b]") {
		t.Fatalf("SetPath through scalar must not descend under a: %q", doc)
	}
	if !strings.Contains(doc, "a = ") {
		t.Fatalf("scalar a should remain: %q", doc)
	}
	if !strings.Contains(doc, "[b]") || !strings.Contains(doc, "nested = 1") {
		t.Fatalf("Table after scalar collision stays in current table: %q", doc)
	}
	tr2 := tw.NewTree()
	tr2.Table("t")
	tr2.Table("t", "inner").SetPath([]string{"leaf"}, "v")
	tr2.SetPath([]string{"t"}, "replaced")
	doc2 := tr2.String()
	if strings.Contains(doc2, "[t.inner]") {
		t.Fatalf("leaf must replace table: %q", doc2)
	}
	if !strings.Contains(doc2, "t = \"replaced\"") {
		t.Fatalf("table overwritten by scalar: %q", doc2)
	}
}

func TestTomlSetPathUnsupportedTypeProperty(t *testing.T) {
	bad := []any{3.14, []string{"a"}, map[string]int{"k": 1}, struct{}{}}
	for _, v := range bad {
		tr := tw.NewTree()
		panicked := false
		func() {
			defer func() {
				if recover() != nil {
					panicked = true
				}
			}()
			tr.SetPath([]string{"x"}, v)
		}()
		if !panicked {
			t.Fatalf("SetPath(%T) should panic", v)
		}
	}
}

func TestTomlScalarsOrderProperty(t *testing.T) {
	tr := tw.NewTree()
	tr.SetPath([]string{"z"}, int64(1))
	tr.SetPath([]string{"a"}, true)
	tr.SetPath([]string{"m"}, "mid")
	tr.Table("sub").SetPath([]string{"b"}, int64(2))
	doc := tr.String()
	bbCheckTableDoc(t, doc)
	ai := strings.Index(doc, "a = ")
	mi := strings.Index(doc, "m = ")
	zi := strings.Index(doc, "z = ")
	ti := strings.Index(doc, "[sub]")
	if ai < 0 || mi < 0 || zi < 0 || ti < 0 {
		t.Fatalf("expected keys and table: %q", doc)
	}
	if !(ai < mi && mi < zi && zi < ti) {
		t.Fatalf("scalars not lexicographic before table: %q", doc)
	}
	if !strings.HasPrefix(strings.Split(doc, "[sub]")[1], "\n  b = ") && !strings.Contains(doc, "\n  b = 2") {
		t.Fatalf("subtable should be indented: %q", doc)
	}
}

func TestTomlUnseenRandomProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed))
	for i := 0; i < bbCases; i++ {
		tr := tw.NewTree()
		nKeys := 1 + rng.Intn(6)
		keys := make([]string, nKeys)
		seen := map[string]bool{}
		for j := range keys {
			for {
				k := bbRandBareKey(rng)
				if !seen[k] {
					seen[k] = true
					keys[j] = k
					break
				}
			}
		}
		sort.Strings(keys)
		for _, k := range keys {
			tr.SetPath([]string{k}, bbRandScalar(rng))
		}
		if rng.Intn(3) == 0 {
			tbl := bbRandBareKey(rng)
			tr.Table(tbl).SetPath([]string{bbRandBareKey(rng)}, bbRandScalar(rng))
		}
		s1 := tr.String()
		s2 := tr.String()
		if s1 != s2 {
			t.Fatalf("case %d: String not byte-stable", i)
		}
		bbCheckTableDoc(t, s1)
		for _, k := range keys {
			if !strings.Contains(s1, bbQuoteKey(k)+" = ") {
				t.Fatalf("case %d: missing key %q in %q", i, k, s1)
			}
		}
		for _, r := range s1 {
			if r > unicode.MaxASCII && r != '\n' {
				t.Fatalf("case %d: non-ascii in output %q", i, s1)
			}
		}
	}
}
