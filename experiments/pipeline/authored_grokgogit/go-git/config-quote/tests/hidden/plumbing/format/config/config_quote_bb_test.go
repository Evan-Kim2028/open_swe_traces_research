package config

import (
	"bytes"
	"strings"
	"testing"
)

func cqEncode(t *testing.T, cfg *Config) string {
	t.Helper()
	var buf bytes.Buffer
	if err := NewEncoder(&buf).Encode(cfg); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	return buf.String()
}

func cqLines(t *testing.T, cfg *Config) []string {
	t.Helper()
	out := cqEncode(t, cfg)
	return strings.Split(strings.TrimRight(out, "\n"), "\n")
}

// TestDetail01 (yes): a section that has options is written as `[name]` on
// its own line, then its options, then its subsections.
func TestDetail01(t *testing.T) {
	out := cqEncode(t, &Config{Sections: Sections{
		{
			Name:    "sec",
			Options: Options{{Key: "k1", Value: "v1"}},
			Subsections: Subsections{
				{Name: "sub", Options: Options{{Key: "k2", Value: "v2"}}},
			},
		},
	}})
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	want := []string{"[sec]", "\tk1 = v1", "[sec \"sub\"]", "\tk2 = v2"}
	if len(lines) < len(want) {
		t.Fatalf("output lines = %q", lines)
	}
	for i, w := range want {
		if lines[i] != w {
			t.Fatalf("line %d = %q, want %q (all %q)", i, lines[i], w, lines)
		}
	}
}

// TestDetail02 (partially): a section with no options and only subsections
// omits the bare `[name]` header.
func TestDetail02(t *testing.T) {
	lines := cqLines(t, &Config{Sections: Sections{
		{
			Name: "sec",
			Subsections: Subsections{
				{Name: "sub", Options: Options{{Key: "k", Value: "v"}}},
			},
		},
	}})
	for _, l := range lines {
		if l == "[sec]" {
			t.Fatalf("bare [sec] header emitted for option-less section: %q", lines)
		}
	}
	found := false
	for _, l := range lines {
		if l == `[sec "sub"]` {
			found = true
		}
	}
	if !found {
		t.Fatalf("no [sec \"sub\"] header: %q", lines)
	}
}

// TestDetail03 (shape — Inferable: no): a subsection header is
// `[section "name"]` with the subsection name escaped — `"` becomes `\"`
// and `\` becomes `\\`.
func TestDetail03(t *testing.T) {
	for _, c := range []struct{ name, want string }{
		{`a"b`, `[sec "a\"b"]`},
		{`a\b`, `[sec "a\\b"]`},
		{`plain`, `[sec "plain"]`},
	} {
		lines := cqLines(t, &Config{Sections: Sections{
			{
				Name:    "sec",
				Options: Options{{Key: "k", Value: "v"}},
				Subsections: Subsections{
					{Name: c.name, Options: Options{{Key: "k", Value: "v"}}},
				},
			},
		}})
		found := false
		for _, l := range lines {
			if l == c.want {
				found = true
			}
		}
		if !found {
			t.Fatalf("subsection %q: no header %q in %q", c.name, c.want, lines)
		}
	}
}

// TestDetail04 (yes): each option is written as a tab, the key, ` = `,
// the value, and a newline.
func TestDetail04(t *testing.T) {
	lines := cqLines(t, &Config{Sections: Sections{
		{Name: "s", Options: Options{{Key: "key", Value: "value"}}},
	}})
	if len(lines) < 2 || lines[1] != "\tkey = value" {
		t.Fatalf("lines = %q, want option line `\\tkey = value`", lines)
	}
}

// TestDetail05 (shape — Inferable: no): the value is wrapped in double
// quotes when it contains any of `# ; " tab newline backslash`, or when it
// has a leading or trailing space.
func TestDetail05(t *testing.T) {
	for _, c := range []struct{ value, want string }{
		{"a#b", `k = "a#b"`},
		{"a;b", `k = "a;b"`},
		{`a"b`, `k = "a\"b"`},
		{"a\tb", `k = "a\tb"`},
		{"a\nb", `k = "a\nb"`},
		{`a\b`, `k = "a\\b"`},
		{" a", `k = " a"`},
		{"a ", `k = "a "`},
		{"plain", `k = plain`},
		{"a'b", `k = a'b`},
	} {
		out := cqEncode(t, &Config{Sections: Sections{
			{Name: "s", Options: Options{{Key: "k", Value: c.value}}},
		}})
		if !strings.Contains(out, "\t"+c.want+"\n") {
			t.Fatalf("value %q: no line %q in %q", c.value, "\t"+c.want, out)
		}
	}
}

// TestDetail06 (shape — Inferable: no): inside a quoted value `"`→`\"`,
// `\`→`\\`, newline→`\n`, tab→`\t`, backspace→`\b`.
func TestDetail06(t *testing.T) {
	// Backspace is only observable inside quotes, so force quoting with '#'
	// and check the backspace escape specifically.
	out := cqEncode(t, &Config{Sections: Sections{
		{Name: "s", Options: Options{{Key: "k", Value: "a\bb#"}}},
	}})
	if !strings.Contains(out, `k = "a\bb#"`) {
		t.Fatalf("backspace escape missing: %q", out)
	}

	// A value that is quotable only because of a newline uses \n inside.
	out = cqEncode(t, &Config{Sections: Sections{
		{Name: "s", Options: Options{{Key: "k", Value: "x\ny"}}},
	}})
	if !strings.Contains(out, `k = "x\ny"`) {
		t.Fatalf("newline escape missing: %q", out)
	}
}

// TestDetail07 (shape — Inferable: no): values with other unusual bytes
// (control characters, non-ASCII) are written unquoted when they miss the
// trigger set.
func TestDetail07(t *testing.T) {
	for _, v := range []string{"a\x01b", "héllo", "a\x7fb"} {
		out := cqEncode(t, &Config{Sections: Sections{
			{Name: "s", Options: Options{{Key: "k", Value: v}}},
		}})
		if !strings.Contains(out, "\tk = "+v+"\n") {
			t.Fatalf("value %q not emitted verbatim/unquoted: %q", v, out)
		}
	}
}

// TestDetail08 (yes): duplicate keys are emitted in order, each on its own
// line.
func TestDetail08(t *testing.T) {
	out := cqEncode(t, &Config{Sections: Sections{
		{Name: "s", Options: Options{
			{Key: "k", Value: "1"},
			{Key: "k", Value: "2"},
			{Key: "k", Value: "3"},
		}},
	}})
	want := "\tk = 1\n\tk = 2\n\tk = 3\n"
	if !strings.Contains(out, want) {
		t.Fatalf("duplicate keys not in order: %q", out)
	}
}
