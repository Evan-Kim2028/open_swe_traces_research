package conf

import (
	"strings"
	"testing"
)

// collectLex drains the item stream until itemEOF or itemError.
func collectLex(input string) []item {
	lx := lex(input)
	var items []item
	for i := 0; i < 64; i++ {
		it := lx.nextItem()
		items = append(items, it)
		if it.typ == itemEOF || it.typ == itemError {
			break
		}
	}
	return items
}

func lexHasErr(items []item) bool {
	for _, it := range items {
		if it.typ == itemError {
			return true
		}
	}
	return false
}

func lexErrItem(items []item) *item {
	for i := range items {
		if items[i].typ == itemError {
			return &items[i]
		}
	}
	return nil
}

// valueItems returns the items after the leading key, i.e. the value
// stream for input shaped like "key = <x>".
func valueItems(items []item) []item {
	if len(items) == 0 || items[0].typ != itemKey {
		return items
	}
	return items[1:]
}

// TestDetail01: whitespace may precede a value but a newline before the
// value is an error.
func TestDetail01(t *testing.T) {
	items := collectLex("key =   value")
	if lexHasErr(items) {
		t.Fatalf("leading whitespace rejected: %v", items)
	}
	items = collectLex("key =\nvalue")
	e := lexErrItem(items)
	if e == nil || !strings.Contains(strings.ToLower(e.val), "new line") {
		t.Fatalf("newline before value: %v, want error mentioning new line", items)
	}
}

// TestDetail02: value dispatch — array, map, quoted string, negative
// number, block string, digit-scan, '.' error, bare string fallback.
func TestDetail02(t *testing.T) {
	cases := []struct {
		in   string
		want itemType
	}{
		{"key = [a]", itemArrayStart},
		{"key = {a:1}", itemMapStart},
		{"key = 'x'", itemString},
		{`key = "x"`, itemString},
		{"key = -5", itemInteger},
		{"key = (a\n)\n", itemString},
		{"key = 5", itemInteger},
		{"key = bare", itemString},
	}
	for _, tc := range cases {
		items := valueItems(collectLex(tc.in))
		if lexHasErr(items) || len(items) == 0 || items[0].typ != tc.want {
			t.Fatalf("%q: got %v, want first value %v", tc.in, items, tc.want)
		}
	}
	if e := lexErrItem(collectLex("key = .5")); e == nil {
		t.Fatal("'.'-starting value did not error")
	}
}

// TestDetail03: a digit-started token that hits a rune that is neither
// digit, '.', '-', suffix, nor a legal terminator is reclassified as a
// string.
func TestDetail03(t *testing.T) {
	items := valueItems(collectLex("key = 12abc"))
	if len(items) < 1 || items[0].typ != itemString || items[0].val != "12abc" {
		t.Fatalf("12abc: %v, want String '12abc'", items)
	}
}

// TestDetail04: a '-' inside a digit-started token enters date mode; the
// date must match the full Zulu form — any literal-position mismatch
// errors.
func TestDetail04(t *testing.T) {
	items := valueItems(collectLex("key = 2022-01-02T03:04:05Z"))
	if len(items) < 1 || items[0].typ != itemDatetime || items[0].val != "2022-01-02T03:04:05Z" {
		t.Fatalf("zulu date: %v, want Datetime", items)
	}
	for _, in := range []string{
		"key = 2022-1-02",               // missing digit mid-date
		"key = 2022-01-02T03:04:05+02:00", // non-Zulu zone
		"key = 2022-01-02",              // date without time
	} {
		if e := lexErrItem(collectLex(in)); e == nil {
			t.Fatalf("%q: want error", in)
		}
	}
}

// TestDetail05: floats need at least one digit before AND after the '.'.
func TestDetail05(t *testing.T) {
	items := valueItems(collectLex("key = 1.5"))
	if len(items) < 1 || items[0].typ != itemFloat {
		t.Fatalf("1.5: %v, want Float", items)
	}
	if e := lexErrItem(collectLex("key = 1.")); e == nil {
		t.Fatal("'1.' did not error")
	}
	if e := lexErrItem(collectLex("key = .5")); e == nil {
		t.Fatal("'.5' did not error")
	}
}

// TestDetail06: a second '.' inside a float re-routes to IP lexing, which
// accepts digits, '.', ':', '-' and emits itemString (shape per DETAILS).
func TestDetail06(t *testing.T) {
	for _, in := range []string{"key = 127.0.0.1:4222", "key = 1.2.3"} {
		items := valueItems(collectLex(in))
		if len(items) < 1 || items[0].typ != itemString {
			t.Fatalf("%q: %v, want itemString", in, items)
		}
		if !strings.Contains(items[0].val, ".") {
			t.Fatalf("%q: value %q lost its dots", in, items[0].val)
		}
	}
}

// TestDetail07: number suffixes k/K/m/M/g/G/t/T/p/P/e/E with an optional
// b/B/i/I run stay numbers only when followed by a terminator; ']' is NOT
// a terminator so [1k] yields the string "1k".
func TestDetail07(t *testing.T) {
	for _, v := range []string{"1k", "1KiB", "1M", "1Gi"} {
		items := valueItems(collectLex("key = " + v))
		if len(items) < 1 || items[0].typ != itemInteger || items[0].val != v {
			t.Fatalf("%q: %v, want Integer %q", v, items, v)
		}
	}
	// ']' is not a number terminator: the token falls back to string.
	items := collectLex("key = [1k]")
	var strs []item
	for _, it := range items {
		if it.typ == itemString {
			strs = append(strs, it)
		}
	}
	if len(strs) != 1 || strs[0].val != "1k" {
		t.Fatalf("[1k]: %v, want one String '1k'", items)
	}
	// A suffix followed by a non-terminator letter falls back to string.
	items = valueItems(collectLex("key = 1kx"))
	if len(items) < 1 || items[0].typ != itemString || items[0].val != "1kx" {
		t.Fatalf("1kx: %v, want String '1kx'", items)
	}
}

// TestDetail08: bare strings terminate on newline, eof, ';', ',', ']', '}',
// whitespace, or '\''; a backslash inside a bare string triggers escape
// processing.
func TestDetail08(t *testing.T) {
	// ';' terminates: 'bar' lexes as the next key.
	items := collectLex("key = foo;bar")
	if len(items) < 3 || items[1].typ != itemString || items[1].val != "foo" ||
		items[2].typ != itemKey || items[2].val != "bar" {
		t.Fatalf("foo;bar: %v", items)
	}
	// ',' and ']' terminate inside arrays.
	items = collectLex("key = [a,b]")
	var vals []string
	for _, it := range items {
		if it.typ == itemString {
			vals = append(vals, it.val)
		}
	}
	if len(vals) != 2 || vals[0] != "a" || vals[1] != "b" {
		t.Fatalf("[a,b]: %v", items)
	}
	// Escape processing inside a bare string.
	items = valueItems(collectLex(`key = a\tb`))
	if len(items) < 1 || items[0].typ != itemString || items[0].val != "a\tb" {
		t.Fatalf(`a\tb: %v, want String with real tab`, items)
	}
}

// TestDetail09: on bare-string termination the order is escaped-parts to
// string, then bool check, then $-variable check, then string. Bools are
// true/false/on/off/yes/no case-insensitively; variables strip the '$'.
func TestDetail09(t *testing.T) {
	for _, v := range []string{"true", "false", "on", "off", "yes", "no", "YES", "True"} {
		items := valueItems(collectLex("key = " + v))
		if len(items) < 1 || items[0].typ != itemBool {
			t.Fatalf("%q: %v, want Bool", v, items)
		}
	}
	items := valueItems(collectLex("key = $foo"))
	if len(items) < 1 || items[0].typ != itemVariable || items[0].val != "foo" {
		t.Fatalf("$foo: %v, want Variable 'foo' (dollar stripped)", items)
	}
	// Escaped parts emit a string BEFORE the bool/variable checks.
	items = valueItems(collectLex(`key = tru\x65`))
	if len(items) < 1 || items[0].typ != itemString || items[0].val != "true" {
		t.Fatalf(`tru\x65: %v, want String 'true' not Bool`, items)
	}
	items = valueItems(collectLex(`key = $f\x6fo`))
	if len(items) < 1 || items[0].typ != itemString {
		t.Fatalf(`$f\x6fo: %v, want String not Variable`, items)
	}
}

// TestDetail10: single-quoted strings are raw — no escape interpretation;
// double-quoted strings process escapes; the closing quote is not part of
// the value.
func TestDetail10(t *testing.T) {
	items := valueItems(collectLex(`key = 'a\nb'`))
	if len(items) < 1 || items[0].typ != itemString || items[0].val != `a\nb` {
		t.Fatalf("single-quoted: %v, want raw 'a\\nb'", items)
	}
	items = valueItems(collectLex(`key = "a\nb"`))
	if len(items) < 1 || items[0].typ != itemString || items[0].val != "a\nb" {
		t.Fatalf("double-quoted: %v, want 'a<NL>b'", items)
	}
}

// TestDetail11: the only escapes are \xNN, \t, \n, \r, \" and \\;
// anything else errors, and a newline inside \x errors.
func TestDetail11(t *testing.T) {
	for in, want := range map[string]string{
		`key = "\x41"`: "A",
		`key = "a\tb"`: "a\tb",
		`key = "a\nb"`: "a\nb",
		`key = "a\rb"`: "a\rb",
		`key = "a\"b"`: `a"b`,
		`key = "a\\b"`: `a\b`,
	} {
		items := valueItems(collectLex(in))
		if len(items) < 1 || items[0].typ != itemString || items[0].val != want {
			t.Fatalf("%q: %v, want String %q", in, items, want)
		}
	}
	for _, in := range []string{`key = "\q"`, `key = "a\xb"`, "key = \"a\\x\nb\""} {
		if e := lexErrItem(collectLex(in)); e == nil {
			t.Fatalf("%q: want error", in)
		}
	}
}

// TestDetail12: block strings capture raw text until a ')' sitting on a
// line by itself — immediately preceded by '\n' and followed by '\n' or
// EOF. A ')' anywhere else is content.
func TestDetail12(t *testing.T) {
	items := valueItems(collectLex("key = (a\nb\n)\n"))
	if len(items) < 1 || items[0].typ != itemString || items[0].val != "a\nb\n" {
		t.Fatalf("block: %v, want String 'a\\nb\\n'", items)
	}
	// ')' not on its own line is content — the block never terminates.
	for _, in := range []string{"key = (a)b)", "key = (a\n)b", "key = ()"} {
		if e := lexErrItem(collectLex(in)); e == nil {
			t.Fatalf("%q: want error (unterminated block)", in)
		}
	}
}

// TestDetail13: '#' and '//' comments run to (not including) the newline,
// emitting itemCommentStart then itemText; a lone '/' is swallowed in
// post-value positions at top level, acts like a separator between array
// values, and errors at an array value position.
func TestDetail13(t *testing.T) {
	for _, c := range []string{"# hi", "// hi"} {
		items := collectLex("key = 1 " + c + "\nk2 = 2")
		var sawComment, sawText bool
		for _, it := range items {
			if it.typ == itemCommentStart {
				sawComment = true
			}
			if it.typ == itemText && strings.Contains(it.val, "hi") {
				sawText = true
			}
		}
		if !sawComment || !sawText || lexHasErr(items) {
			t.Fatalf("comment %q: %v", c, items)
		}
	}
	// Lone '/' at top-level post-value is swallowed.
	items := collectLex("key = 1 /\nkey2 = 2")
	if lexHasErr(items) {
		t.Fatalf("lone slash post-value: %v", items)
	}
	// Between array values it acts as a separator.
	items = collectLex("key = [a / b]")
	var strs []string
	for _, it := range items {
		if it.typ == itemString {
			strs = append(strs, it.val)
		}
	}
	if len(strs) != 2 || lexHasErr(items) {
		t.Fatalf("[a / b]: %v", items)
	}
	// At an array value position it errors.
	if e := lexErrItem(collectLex("key = [/a]")); e == nil {
		t.Fatal("[/a] did not error")
	}
}

// TestDetail14: after a top-level value only newline, eof, ';', ',', '}',
// or a comment are legal; anything else errors.
func TestDetail14(t *testing.T) {
	for _, in := range []string{
		"key = 1\nkey2 = 2", "key = 1, key2 = 2",
		"key = 1; key2 = 2", "key = 1 }", "key = 1 # ok\n", "key = 1",
	} {
		if e := lexErrItem(collectLex(in)); e != nil {
			t.Fatalf("%q: unexpected error %v", in, e)
		}
	}
	if e := lexErrItem(collectLex("key = 1 key2 = 2")); e == nil {
		t.Fatal("trailing token after value did not error")
	}
}

// TestDetail15: inside arrays both ',' and bare newlines separate values,
// a ',' at a value position errors, and ';' is not an array separator;
// inside maps ',', ';' and newlines all separate pairs.
func TestDetail15(t *testing.T) {
	count := func(items []item, typ itemType) int {
		n := 0
		for _, it := range items {
			if it.typ == typ {
				n++
			}
		}
		return n
	}
	// Newline separates.
	items := collectLex("key = [1\n2\n]")
	if lexHasErr(items) || count(items, itemInteger) != 2 {
		t.Fatalf("[1 NL 2 NL ]: %v", items)
	}
	// ',' at a value position errors.
	if e := lexErrItem(collectLex("key = [,1]")); e == nil {
		t.Fatal("[,1] did not error")
	}
	// ';' is not an array separator.
	if e := lexErrItem(collectLex("key = [1;2]")); e == nil {
		t.Fatal("[1;2] did not error")
	}
	// Maps: ',', ';', newline all separate pairs.
	for _, in := range []string{"key = {a:1,b:2}", "key = {a:1;b:2}", "key = {a:1\nb:2}"} {
		items := collectLex(in)
		if lexHasErr(items) || count(items, itemInteger) != 2 {
			t.Fatalf("%q: %v", in, items)
		}
	}
}

// TestDetail16: emitString joins buffered escape parts with the
// in-progress span; the emitted item records the line and a column that
// counts from the start of the logical line.
func TestDetail16(t *testing.T) {
	items := collectLex("key = value")
	if len(items) < 2 || items[1].typ != itemString || items[1].val != "value" {
		t.Fatalf("value: %v", items)
	}
	// 'value' starts at column 6 of line 1.
	if items[1].line != 1 || items[1].pos != 6 {
		t.Fatalf("pos = %d, want 6 (column in logical line)", items[1].pos)
	}
	// Escape parts concatenate with the raw span into one item.
	items = collectLex(`key = a\tb`)
	for _, it := range items {
		if it.typ == itemString {
			if it.val != "a\tb" {
				t.Fatalf("joined value %q, want 'a\\tb'", it.val)
			}
			return
		}
	}
	t.Fatalf("no string item: %v", items)
}
