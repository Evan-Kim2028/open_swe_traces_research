// Hidden black-box property suite for the confparse unit.
// Same-package access is part of the API (api.md): Parse, ParseWithChecks,
// ParseFile, ParseFileWithChecks, ParseFileWithChecksDigest, and the token
// type's exported accessors.
// One property per DETAILS.md line; seeded random inputs, no worked examples.
package conf

import (
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

const bbConfHiddenSeed = 20260919

func bbConfSeed() int64 {
	if s := os.Getenv("HIDDEN_SEED"); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
	}
	return bbConfHiddenSeed
}

func bbConfRng(t *testing.T) *rand.Rand {
	t.Helper()
	return rand.New(rand.NewSource(bbConfSeed()))
}

func bbConfWrite(t *testing.T, dir, name, content string) string {
	t.Helper()
	fp := filepath.Join(dir, name)
	if err := os.WriteFile(fp, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return fp
}

// Detail 1: scalar typing — quoted/bare strings->string, integers->int64,
// decimals->float64, booleans->bool, Zulu datetimes->time.Time.
func TestDetail01_ScalarTyping(t *testing.T) {
	m, err := Parse(`
		s1: "hello world"
		s2: 'single quoted'
		s3: barestring
		i1: 42
		i2: -17
		f1: 3.14
		f2: -0.5
		b1: true
		b2: off
		dt: 2016-05-04T03:02:01Z
	`)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range map[string]any{"s1": "hello world", "s2": "single quoted", "s3": "barestring"} {
		if m[k] != v {
			t.Fatalf("%s=%v (%T) want %v string", k, m[k], m[k], v)
		}
	}
	if v, ok := m["i1"].(int64); !ok || v != 42 {
		t.Fatalf("i1=%v (%T) want int64 42", m["i1"], m["i1"])
	}
	if v, ok := m["i2"].(int64); !ok || v != -17 {
		t.Fatalf("i2=%v (%T) want int64 -17", m["i2"], m["i2"])
	}
	if v, ok := m["f1"].(float64); !ok || v != 3.14 {
		t.Fatalf("f1=%v (%T) want float64", m["f1"], m["f1"])
	}
	if v, ok := m["f2"].(float64); !ok || v != -0.5 {
		t.Fatalf("f2=%v (%T) want float64", m["f2"], m["f2"])
	}
	if v, ok := m["b1"].(bool); !ok || !v {
		t.Fatalf("b1=%v (%T) want true", m["b1"], m["b1"])
	}
	if v, ok := m["b2"].(bool); !ok || v {
		t.Fatalf("b2=%v (%T) want false", m["b2"], m["b2"])
	}
	if v, ok := m["dt"].(time.Time); !ok || !v.Equal(time.Date(2016, 5, 4, 3, 2, 1, 0, time.UTC)) {
		t.Fatalf("dt=%v (%T) want time.Time", m["dt"], m["dt"])
	}
	// Random ints/floats round-trip through the parser.
	rng := bbConfRng(t)
	for i := 0; i < 40; i++ {
		iv := rng.Int63n(1 << 60)
		fv := float64(rng.Int63n(1<<30)) / float64(1+rng.Int63n(1000))
		mm, err := Parse("k1: " + strconv.FormatInt(iv, 10) + "\nk2: " + strconv.FormatFloat(fv, 'f', 6, 64))
		if err != nil {
			t.Fatalf("iter %d: %v", i, err)
		}
		if v, ok := mm["k1"].(int64); !ok || v != iv {
			t.Fatalf("iter %d: k1=%v want int64 %d", i, mm["k1"], iv)
		}
		if _, ok := mm["k2"].(float64); !ok {
			t.Fatalf("iter %d: k2=%v (%T) want float64", i, mm["k2"], mm["k2"])
		}
	}
}

// Detail 2: boolean spellings {true,yes,on}/{false,no,off} case-insensitive;
// other spellings error.
func TestDetail02_BooleanSpellings(t *testing.T) {
	trues := []string{"true", "yes", "on", "TRUE", "Yes", "ON", "tRuE", "yEs", "oN"}
	falses := []string{"false", "no", "off", "FALSE", "No", "OFF", "fAlSe", "nO", "oFf"}
	for i, s := range trues {
		m, err := Parse("k: " + s)
		if err != nil {
			t.Fatalf("%q: %v", s, err)
		}
		if v, ok := m["k"].(bool); !ok || !v {
			t.Fatalf("%q -> %v (%T) want true", s, m["k"], m["k"])
		}
		_ = i
	}
	for _, s := range falses {
		m, err := Parse("k: " + s)
		if err != nil {
			t.Fatalf("%q: %v", s, err)
		}
		if v, ok := m["k"].(bool); !ok || v {
			t.Fatalf("%q -> %v (%T) want false", s, m["k"], m["k"])
		}
	}
	// Other spellings are not booleans.
	for _, s := range []string{"y", "n", "t", "f", "1", "0", "enabled", "disabled", "yess", "onn", "nope", "of", "truee"} {
		m, err := Parse("k: " + s)
		if err != nil {
			continue // an error is also acceptable per DETAILS ("other spellings error").
		}
		if v, ok := m["k"].(bool); ok {
			t.Fatalf("%q parsed as bool %v — not a valid spelling", s, v)
		}
	}
}

// Detail 3: integer size suffixes — bare k m g t p e = 1000^(1..6);
// kb/ki/kib mb/mi/mib gb/gi/gib tb/ti/tib pb/pi/pib eb/ei/eib = 1024^(1..6);
// case-insensitive; unrecognized suffixes lex as plain strings.
func TestDetail03_IntegerSuffixes(t *testing.T) {
	pows := []struct {
		dec, bin int64
	}{
		{1000, 1024},
		{1000 * 1000, 1024 * 1024},
		{1000 * 1000 * 1000, 1024 * 1024 * 1024},
		{1000 * 1000 * 1000 * 1000, 1024 * 1024 * 1024 * 1024},
		{1000 * 1000 * 1000 * 1000 * 1000, 1024 * 1024 * 1024 * 1024 * 1024},
		{1000 * 1000 * 1000 * 1000 * 1000 * 1000, 1024 * 1024 * 1024 * 1024 * 1024 * 1024},
	}
	dec := []string{"k", "m", "g", "t", "p", "e"}
	bin := [][]string{
		{"kb", "ki", "kib"},
		{"mb", "mi", "mib"},
		{"gb", "gi", "gib"},
		{"tb", "ti", "tib"},
		{"pb", "pi", "pib"},
		{"eb", "ei", "eib"},
	}
	rng := bbConfRng(t)
	for i, sfx := range dec {
		n := int64(1 + rng.Intn(9))
		for _, form := range []string{sfx, strings.ToUpper(sfx), strings.Title(sfx)} {
			m, err := Parse("k: " + strconv.FormatInt(n, 10) + form)
			if err != nil {
				t.Fatalf("%d%s: %v", n, form, err)
			}
			if v, ok := m["k"].(int64); !ok || v != n*pows[i].dec {
				t.Fatalf("%d%s -> %v (%T) want %d", n, form, m["k"], m["k"], n*pows[i].dec)
			}
		}
		for _, sfx2 := range bin[i] {
			for _, form := range []string{sfx2, strings.ToUpper(sfx2), strings.Title(sfx2)} {
				m, err := Parse("k: " + strconv.FormatInt(n, 10) + form)
				if err != nil {
					t.Fatalf("%d%s: %v", n, form, err)
				}
				if v, ok := m["k"].(int64); !ok || v != n*pows[i].bin {
					t.Fatalf("%d%s -> %v (%T) want %d", n, form, m["k"], m["k"], n*pows[i].bin)
				}
			}
		}
	}
	// Unrecognized suffixes lex as plain strings.
	for _, s := range []string{"5x", "5b", "5z", "9kx", "3mbz"} {
		m, err := Parse("k: " + s)
		if err != nil {
			continue
		}
		if v, ok := m["k"].(string); !ok || v != s {
			t.Fatalf("%q -> %v (%T) want string %q", s, m["k"], m["k"], s)
		}
	}
}

// Detail 4: integers exceeding int64 range error "out of the range";
// malformed integers error "expected integer".
func TestDetail04_IntegerErrors(t *testing.T) {
	for _, s := range []string{"9223372036854775808", "99999999999999999999", "18446744073709551616"} {
		_, err := Parse("k: " + s)
		if err == nil {
			t.Fatalf("%q should error", s)
		}
		if !strings.Contains(err.Error(), "out of the range") {
			t.Fatalf("%q err=%q want 'out of the range'", s, err)
		}
	}
	// Suffix shapes the lexer doesn't recognize as integers don't error:
	// "5kbb"/"5kibi" lex to nil, "7eibe" lexes as a bare string.
	for _, c := range []struct {
		s    string
		want any
	}{
		{"5kbb", nil}, {"5kibi", nil}, {"7eibe", "7eibe"},
	} {
		m, err := Parse("k: " + c.s)
		if err != nil {
			t.Fatalf("%q err=%v", c.s, err)
		}
		if m["k"] != c.want {
			t.Fatalf("%q -> %v want %v", c.s, m["k"], c.want)
		}
	}
}

// Detail 5: datetime must match 2006-01-02T15:04:05Z exactly or error
// "expected Zulu formatted DateTime".
func TestDetail05_ZuluDatetime(t *testing.T) {
	m, err := Parse("k: 2033-11-20T09:10:11Z")
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2033, 11, 20, 9, 10, 11, 0, time.UTC)
	if v, ok := m["k"].(time.Time); !ok || !v.Equal(want) {
		t.Fatalf("k=%v (%T) want %v", m["k"], m["k"], want)
	}
	for _, s := range []string{
		"2033-11-20 09:10:11Z",   // space not T
		"2033-11-20T09:10:11",  // no Z
		"2033-11-20T09:10:11z", // lower z
		"2033-11-2T09:10:11Z",  // short day
		"2033-11-20T9:10:11Z",  // short hour
		"203-11-20T09:10:11Z",  // short year
		"2033-11-20T09:10:11+00:00",
	} {
		if _, err := Parse("k: " + s); err == nil {
			t.Fatalf("%q should error (not exact Zulu)", s)
		}
	}
}

// Detail 6: $name lookup walks map contexts innermost->outermost, skipping
// array contexts.
func TestDetail06_ScopedVariableLookup(t *testing.T) {
	m, err := Parse(`
		a: 11
		outer {
			b: $a
			inner {
				a: 22
				c: $a
				d: $b
			}
		}
		arr [
			{ e: $a }
		]
	`)
	if err != nil {
		t.Fatal(err)
	}
	outer := m["outer"].(map[string]any)
	if outer["b"] != int64(11) {
		t.Fatalf("outer.b=%v want 11 (outermost)", outer["b"])
	}
	inner := outer["inner"].(map[string]any)
	if inner["c"] != int64(22) {
		t.Fatalf("inner.c=%v want 22 (innermost wins)", inner["c"])
	}
	if inner["d"] != int64(11) {
		t.Fatalf("inner.d=%v want 11 ($b walks out)", inner["d"])
	}
	arr := m["arr"].([]any)
	el := arr[0].(map[string]any)
	if el["e"] != int64(11) {
		t.Fatalf("arr[0].e=%v want 11 (array ctx skipped)", el["e"])
	}
}

// Detail 7: unresolved names fall back to the process environment; the env
// value is re-parsed as name=value config text, yielding typed results.
func TestDetail07_EnvFallback(t *testing.T) {
	rng := bbConfRng(t)
	iv := rng.Int63n(1 << 40)
	name := "BBCONF_ENV_" + strconv.Itoa(rng.Intn(1<<20))
	t.Setenv(name, strconv.FormatInt(iv, 10))
	m, err := Parse("k: $" + name)
	if err != nil {
		t.Fatal(err)
	}
	if m["k"] != iv {
		t.Fatalf("k=%v (%T) want int64 %d (env re-parsed)", m["k"], m["k"], iv)
	}
	// String env.
	t.Setenv(name, "somestring")
	m, err = Parse("k: $" + name)
	if err != nil || m["k"] != "somestring" {
		t.Fatalf("string env: %v err=%v", m, err)
	}
	// Bool env re-parsed.
	t.Setenv(name, "yes")
	m, err = Parse("k: $" + name)
	if err != nil || m["k"] != true {
		t.Fatalf("bool env: %v err=%v", m, err)
	}
}

// Detail 8: env values lex like inline text — "3xyz"/"3Gyz" are strings;
// "'3xyz'" parses the quotes too, yielding "3xyz".
func TestDetail08_EnvNumberStrings(t *testing.T) {
	name := "BBCONF_D8"
	t.Setenv(name, "3xyz")
	m, err := Parse("k: $" + name)
	if err != nil || m["k"] != "3xyz" {
		t.Fatalf("env 3xyz: %v err=%v want \"3xyz\"", m, err)
	}
	t.Setenv(name, "3Gyz")
	m, err = Parse("k: $" + name)
	if err != nil || m["k"] != "3Gyz" {
		t.Fatalf("env 3Gyz: %v err=%v want \"3Gyz\"", m, err)
	}
	t.Setenv(name, "'3xyz'")
	m, err = Parse("k: $" + name)
	if err != nil || m["k"] != "3xyz" {
		t.Fatalf("env '3xyz': %v err=%v want \"3xyz\"", m, err)
	}
}

// Detail 9: missing reference -> "variable reference for '<name>' on line
// <n> can not be found"; env-parse failure -> "could not be parsed".
func TestDetail09_ReferenceErrors(t *testing.T) {
	_, err := Parse("k: $BBCONF_MISSING_XYZ")
	if err == nil {
		t.Fatal("missing ref should error")
	}
	if !strings.Contains(err.Error(), "variable reference for 'BBCONF_MISSING_XYZ'") ||
		!strings.Contains(err.Error(), "can not be found") {
		t.Fatalf("missing ref err=%q", err)
	}
	if !strings.Contains(err.Error(), "line 1") {
		t.Fatalf("missing ref err lacks line: %q", err)
	}
	// Line number tracks the reference site.
	_, err = Parse("a: 1\nb: 2\nc: $BBCONF_MISSING_XYZ")
	if err == nil || !strings.Contains(err.Error(), "line 3") {
		t.Fatalf("missing ref line: %v", err)
	}
	// Env value that fails re-parse -> "could not be parsed".
	t.Setenv("BBCONF_BAD_ENV", "{unclosed")
	_, err = Parse("k: $BBCONF_BAD_ENV")
	if err == nil {
		t.Fatal("bad env value should error")
	}
	if !strings.Contains(err.Error(), "could not be parsed") {
		t.Fatalf("env parse err=%q want 'could not be parsed'", err)
	}
}

// Detail 10: a 2a$-prefixed reference yields literal "$"+ref string, no
// lookup.
func TestDetail10_BcryptReference(t *testing.T) {
	m, err := Parse("password: $2a$11$someseedvalue")
	if err != nil {
		t.Fatal(err)
	}
	if m["password"] != "$2a$11$someseedvalue" {
		t.Fatalf("password=%v want literal $2a$11$someseedvalue", m["password"])
	}
	// Even though it looks like a reference, no lookup happens — value kept.
	rng := bbConfRng(t)
	suffix := strings.Repeat("x", rng.Intn(10))
	m, err = Parse("p: $2a$" + suffix)
	if err != nil || m["p"] != "$2a$"+suffix {
		t.Fatalf("p=%v err=%v want literal", m["p"], err)
	}
}

// Detail 11: env-expansion reference cycles error "variable reference
// cycle for '<name>'".
func TestDetail11_ReferenceCycles(t *testing.T) {
	t.Setenv("BBCONF_CYCLE_A", "$BBCONF_CYCLE_A")
	_, err := Parse("k: $BBCONF_CYCLE_A")
	if err == nil {
		t.Fatal("self-referencing env cycle should error")
	}
	if !strings.Contains(err.Error(), "variable reference cycle") {
		t.Fatalf("cycle err=%q", err)
	}
	// Two-hop cycle through env.
	t.Setenv("BBCONF_CY1", "$BBCONF_CY2")
	t.Setenv("BBCONF_CY2", "$BBCONF_CY1")
	_, err = Parse("k: $BBCONF_CY1")
	if err == nil || !strings.Contains(err.Error(), "variable reference cycle") {
		t.Fatalf("two-hop cycle err=%v", err)
	}
}

// Detail 12: include parses the file relative to the including file's dir
// and merges top-level keys into the current context; inner errors wrap as
// "error parsing include file '<name>', …"; pedantic propagates and
// preserves token items.
func TestDetail12_Includes(t *testing.T) {
	dir := t.TempDir()
	bbConfWrite(t, dir, "sub.conf", "x: 1\ny: \"two\"\n")
	main := bbConfWrite(t, dir, "main.conf", "a: 0\ninclude 'sub.conf'\nb: 3\n")
	m, err := ParseFile(main)
	if err != nil {
		t.Fatal(err)
	}
	if m["a"] != int64(0) || m["x"] != int64(1) || m["y"] != "two" || m["b"] != int64(3) {
		t.Fatalf("merged map=%v", m)
	}
	// Include inside a map merges into that map's context.
	sub2 := bbConfWrite(t, dir, "deep.conf", "inner: 7\n")
	main2 := bbConfWrite(t, dir, "main2.conf", "top { include 'deep.conf'\nother: 9 }\n")
	m, err = ParseFile(main2)
	if err != nil {
		t.Fatal(err)
	}
	top := m["top"].(map[string]any)
	if top["inner"] != int64(7) || top["other"] != int64(9) {
		t.Fatalf("nested include=%v", top)
	}
	// Missing include -> wrapped error.
	bad := bbConfWrite(t, dir, "bad.conf", "include 'does-not-exist.conf'\n")
	_, err = ParseFile(bad)
	if err == nil || !strings.Contains(err.Error(), "error parsing include file 'does-not-exist.conf'") {
		t.Fatalf("missing include err=%v", err)
	}
	// Broken include content -> wrapped error.
	bbConfWrite(t, dir, "broken.conf", "keyonly")
	main3 := bbConfWrite(t, dir, "main3.conf", "include 'broken.conf'\n")
	_, err = ParseFile(main3)
	if err == nil || !strings.Contains(err.Error(), "error parsing include file 'broken.conf'") {
		t.Fatalf("broken include err=%v", err)
	}
	// Pedantic: included values are *token with the include's source file.
	m, err = ParseFileWithChecks(main)
	if err != nil {
		t.Fatal(err)
	}
	tk, ok := m["x"].(*token)
	if !ok {
		t.Fatalf("pedantic x is %T not *token", m["x"])
	}
	if tk.SourceFile() == "" || !strings.HasSuffix(tk.SourceFile(), "sub.conf") {
		t.Fatalf("token source=%q want sub.conf", tk.SourceFile())
	}
	_ = sub2
}

// Detail 13: empty/whitespace/comments-only input -> valid empty map;
// document ending on a bare key -> "config is invalid (<fp>:<line>:<pos>)"
// with fp empty for string input.
func TestDetail13_EmptyAndBareKey(t *testing.T) {
	for _, s := range []string{"", "   \n\n  ", "# comment\n// another\n", "{\n}\n", "# c\n{\n# c2\n}\n"} {
		m, err := Parse(s)
		if err != nil {
			t.Fatalf("Parse(%q): %v", s, err)
		}
		if len(m) != 0 {
			t.Fatalf("Parse(%q)=%v want empty map", s, m)
		}
	}
	// Bare key at EOF -> config is invalid with ":line:pos".
	_, err := Parse("keyonly")
	if err == nil || !strings.Contains(err.Error(), "config is invalid") || !strings.Contains(err.Error(), ":1:") {
		t.Fatalf("bare key err=%v", err)
	}
	_, err = Parse("a: 1\nbarekey")
	if err == nil || !strings.Contains(err.Error(), ":2:") {
		t.Fatalf("bare key line2 err=%v", err)
	}
}

// Detail 14: a trailing stray '}' and full '{…}' wrapping are tolerated.
func TestDetail14_StrayBraces(t *testing.T) {
	m, err := Parse("a: 1\n}")
	if err != nil || m["a"] != int64(1) {
		t.Fatalf("trailing brace: %v err=%v", m, err)
	}
	m, err = Parse("{\na: 1\nb: 2\n}")
	if err != nil || m["a"] != int64(1) || m["b"] != int64(2) {
		t.Fatalf("wrapped braces: %v err=%v", m, err)
	}
}

// Detail 15: pedantic mode wraps every stored value in *token recording
// value, key's line+pos, source file, and usedVariable; variable
// resolution marks the referent used and stores a fresh token.
func TestDetail15_PedanticTokens(t *testing.T) {
	m, err := ParseWithChecks("a: 42\nb: \"str\"\n")
	if err != nil {
		t.Fatal(err)
	}
	tk, ok := m["a"].(*token)
	if !ok {
		t.Fatalf("a is %T not *token", m["a"])
	}
	if tk.Value() != int64(42) {
		t.Fatalf("token value=%v want 42", tk.Value())
	}
	if tk.Line() != 1 {
		t.Fatalf("token line=%d want 1", tk.Line())
	}
	if tk.IsUsedVariable() {
		t.Fatal("a should not be used yet")
	}
	if tk.SourceFile() != "" {
		t.Fatalf("token source=%q want empty for string input", tk.SourceFile())
	}
	// Variable resolution: referent marked used; referree gets fresh token.
	m, err = ParseWithChecks("a: 7\nb: $a\n")
	if err != nil {
		t.Fatal(err)
	}
	tka := m["a"].(*token)
	tkb := m["b"].(*token)
	if !tka.IsUsedVariable() {
		t.Fatal("referent 'a' should be marked used")
	}
	if tkb.Value() != int64(7) {
		t.Fatalf("b token value=%v want 7", tkb.Value())
	}
	// MarshalJSON exposes the underlying value.
	jb, err := tk.MarshalJSON()
	if err != nil || string(jb) != "42" {
		t.Fatalf("MarshalJSON=%s err=%v", jb, err)
	}
	// Line/Position recorded.
	if tkb.Line() != 2 {
		t.Fatalf("b line=%d want 2", tkb.Line())
	}
	if tkb.Position() <= 0 {
		t.Fatalf("b position=%d", tkb.Position())
	}
}

// Detail 16: ParseFileWithChecksDigest returns "sha256:"+hex of the JSON
// encoding of the pedantic map; syntax-equivalent docs share a digest; any
// error -> empty digest. Unreadable file -> "error opening config file:".
func TestDetail16_ConfigDigest(t *testing.T) {
	dir := t.TempDir()
	f1 := bbConfWrite(t, dir, "a.conf", "x: 1\ny: \"s\"\n")
	f2 := bbConfWrite(t, dir, "b.conf", "# comment\n\nx: 1\ny: \"s\"\n") // syntax-equivalent.
	f3 := bbConfWrite(t, dir, "c.conf", "x: 2\ny: \"s\"\n")
	_, d1, err := ParseFileWithChecksDigest(f1)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(d1, "sha256:") || len(d1) != len("sha256:")+64 {
		t.Fatalf("digest=%q want sha256:<64hex>", d1)
	}
	_, d2, err := ParseFileWithChecksDigest(f2)
	if err != nil {
		t.Fatal(err)
	}
	if d1 != d2 {
		t.Fatalf("equivalent docs digest %q != %q", d1, d2)
	}
	_, d3, err := ParseFileWithChecksDigest(f3)
	if err != nil {
		t.Fatal(err)
	}
	if d1 == d3 {
		t.Fatalf("different docs share digest %q", d1)
	}
	// Error -> empty digest.
	bad := bbConfWrite(t, dir, "bad.conf", "keyonly")
	_, d4, err := ParseFileWithChecksDigest(bad)
	if err == nil {
		t.Fatal("bad file should error")
	}
	if d4 != "" {
		t.Fatalf("error digest=%q want empty", d4)
	}
	// Unreadable file.
	_, d5, err := ParseFileWithChecksDigest(filepath.Join(dir, "missing.conf"))
	if err == nil {
		t.Fatal("missing file should error")
	}
	if !strings.Contains(err.Error(), "missing.conf") {
		t.Fatalf("missing file err=%q", err)
	}
	if d5 != "" {
		t.Fatalf("missing digest=%q want empty", d5)
	}
}
