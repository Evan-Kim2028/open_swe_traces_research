package systemd_test

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"testing"

	"example.internal/kops/pkg/systemd"
)

func hiddenSeed() int64 {
	if s := os.Getenv("HIDDEN_SEED"); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
	}
	return 20260919
}

// Detail 1: allowed punctuation passes through unescaped.
func TestDetail01_AllowedPunctuationPassthrough(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	allowed := "!#$%&()*+,-./:;<>=?@[]^_`{|}~"
	for i := 0; i < 200; i++ {
		var b strings.Builder
		for j := 0; j < 1+rng.Intn(8); j++ {
			b.WriteByte(allowed[rng.Intn(len(allowed))])
		}
		arg := "a" + b.String() + "z"
		got := systemd.EscapeCommand([]string{arg})
		if got != arg {
			t.Fatalf("i=%d EscapeCommand(%q)=%q", i, arg, got)
		}
	}
	// Alphanumerics too.
	if got := systemd.EscapeCommand([]string{"abcXYZ019"}); got != "abcXYZ019" {
		t.Fatalf("alnum escaped: %q", got)
	}
}

// Detail 2: only a space triggers quoting the whole arg in "..."; quote,
// apostrophe and backslash do NOT trigger quoting.
func TestDetail02_OnlySpaceTriggersQuoting(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 100; i++ {
		arg := fmt.Sprintf("word%d part%d", rng.Intn(100), rng.Intn(100))
		got := systemd.EscapeCommand([]string{arg})
		if got != `"`+arg+`"` {
			t.Fatalf("i=%d space arg=%q got %q want quoted", i, arg, got)
		}
	}
	// Quotes/apostrophes/backslashes escape in place, no wrapping.
	cases := []struct{ in, want string }{
		{`a"b`, `a\"b`},
		{`a'b`, `a\'b`},
		{`a\b`, `a\\b`},
	}
	for i, c := range cases {
		got := systemd.EscapeCommand([]string{c.in})
		if got != c.want {
			t.Fatalf("i=%d EscapeCommand(%q)=%q want %q", i, c.in, got, c.want)
		}
		if strings.HasPrefix(got, `"`) && strings.HasSuffix(got, `"`) && len(got) > len(c.in) {
			t.Fatalf("i=%d non-space char triggered quoting: %q", i, got)
		}
	}
}

// Detail 3: any other byte falls back to \x + lowercase hex.
func TestDetail03_HexFallback(t *testing.T) {
	// Bytes outside the allowed set and outside " ' \ space.
	for _, b := range []byte{0x01, 0x7f, 0x80, 0xff, '\n', '\t', '(', ')'} {
		arg := "a" + string([]byte{b}) + "z"
		got := systemd.EscapeCommand([]string{arg})
		if b == '(' || b == ')' {
			if got != arg {
				t.Fatalf("allowed %q escaped to %q", string(b), got)
			}
			continue
		}
		want := fmt.Sprintf("a\\x%02xz", b)
		if got != want {
			t.Fatalf("byte %#x: EscapeCommand=%q want %q", b, got, want)
		}
	}
}

// Detail 4: Set — first use creates section; later same-name Sets append
// (duplicates kept); "key=value\n" format.
func TestDetail04_SetAppendsDuplicates(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 60; i++ {
		m := &systemd.Manifest{}
		sec := fmt.Sprintf("S%d", rng.Intn(10))
		k1, v1 := fmt.Sprintf("k%d", rng.Intn(10)), fmt.Sprintf("v%d", rng.Intn(10))
		k2, v2 := fmt.Sprintf("k%d", rng.Intn(10)), fmt.Sprintf("v%d", rng.Intn(10))
		m.Set(sec, k1, v1)
		m.Set(sec, k1, v2) // duplicate key, different value — kept
		m.Set(sec, k2, v2)
		out := m.Render()
		if !strings.Contains(out, "["+sec+"]\n") {
			t.Fatalf("i=%d missing section header in %q", i, out)
		}
		body := out[strings.Index(out, "["+sec+"]\n"):]
		if !strings.Contains(body, k1+"="+v1+"\n") || !strings.Contains(body, k1+"="+v2+"\n") || !strings.Contains(body, k2+"="+v2+"\n") {
			t.Fatalf("i=%d entries missing in %q", i, out)
		}
		// Append order preserved.
		if strings.Index(body, k1+"="+v1) > strings.Index(body, k1+"="+v2) {
			t.Fatalf("i=%d entries reordered in %q", i, out)
		}
	}
}

// Detail 5: SetSection content emitted verbatim immediately after header,
// only when non-empty, before that section's entries.
func TestDetail05_RawContentBeforeEntries(t *testing.T) {
	m := &systemd.Manifest{}
	m.SetSection("A", "raw line 1\nraw line 2\n")
	m.Set("A", "k", "v")
	out := m.Render()
	head := strings.Index(out, "[A]\n")
	raw := strings.Index(out, "raw line 1\nraw line 2\n")
	entry := strings.Index(out, "k=v\n")
	if head < 0 || raw < 0 || entry < 0 {
		t.Fatalf("missing pieces: %q", out)
	}
	if !(head < raw && raw < entry) {
		t.Fatalf("order wrong (want header < raw < entry): %q", out)
	}
	// Empty content is skipped entirely.
	m2 := &systemd.Manifest{}
	m2.SetSection("B", "")
	m2.Set("B", "k", "v")
	out2 := m2.Render()
	if strings.Contains(out2, "[B]\n\n") {
		t.Fatalf("empty content emitted blank: %q", out2)
	}
}

// Detail 6: exactly one blank line between sections; none after the last.
func TestDetail06_SectionSeparator(t *testing.T) {
	m := &systemd.Manifest{}
	m.Set("First", "a", "1")
	m.Set("Second", "b", "2")
	m.Set("Third", "c", "3")
	out := m.Render()
	if strings.Contains(out, "\n\n\n") {
		t.Fatalf("double blank separator: %q", out)
	}
	if !strings.Contains(out, "a=1\n\n[Second]") || !strings.Contains(out, "b=2\n\n[Third]") {
		t.Fatalf("missing single blank separators: %q", out)
	}
	if strings.HasSuffix(out, "\n\n") {
		t.Fatalf("trailing blank after last section: %q", out)
	}
}

// Detail 7: UnitFileExtensionValid — suffix match over the 11-extension
// list.
func TestDetail07_UnitFileExtensionValid(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 100; i++ {
		ext := systemd.UnitExtensions[rng.Intn(len(systemd.UnitExtensions))]
		name := fmt.Sprintf("unit-%d%s", i, ext)
		if !systemd.UnitFileExtensionValid(name) {
			t.Fatalf("i=%d %q rejected", i, name)
		}
	}
	for i, name := range []string{
		"my-unit.not-valid", "unit.serviceX", "unit.conf", "x.service.bak",
		"unit", "unit.serv",
	} {
		if systemd.UnitFileExtensionValid(name) {
			t.Fatalf("i=%d %q accepted", i, name)
		}
	}
}
