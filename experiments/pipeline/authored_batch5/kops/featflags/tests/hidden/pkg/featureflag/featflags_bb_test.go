package featureflag

import (
	"os"
	"strings"
	"testing"
)

// TestDetail01: Enabled precedence — explicit enabled beats defaultValue;
// both nil means false.
func TestDetail01(t *testing.T) {
	d := new("BBDetail1Default", Bool(true))
	if !d.Enabled() {
		t.Fatal("default true not honored")
	}
	n := new("BBDetail1Nil", nil)
	if n.Enabled() {
		t.Fatal("nil default should be disabled")
	}
	ParseFlags("+BBDetail1Nil")
	if !n.Enabled() {
		t.Fatal("explicit enable did not beat nil default")
	}
	ParseFlags("-BBDetail1Default")
	if d.Enabled() {
		t.Fatal("explicit disable did not beat default true")
	}
}

// TestDetail02: new is idempotent per key — same flag returned, first
// non-nil default wins.
func TestDetail02(t *testing.T) {
	f1 := new("BBDetail2", Bool(true))
	f2 := new("BBDetail2", Bool(false))
	if f1 != f2 {
		t.Fatal("re-registration returned a different flag")
	}
	if !f2.Enabled() {
		t.Fatal("second registration overwrote the first default")
	}
	g1 := new("BBDetail2NilFirst", nil)
	g2 := new("BBDetail2NilFirst", Bool(true))
	if g1 != g2 {
		t.Fatal("re-registration returned a different flag")
	}
	if !g2.Enabled() {
		t.Fatal("later non-nil default did not fill a nil default")
	}
}

// TestDetail03: ParseFlags grammar — +Name, -Name, and bare Name (= enable);
// whitespace trimmed; empty items skipped.
func TestDetail03(t *testing.T) {
	a := new("BBDetail3A", nil)
	b := new("BBDetail3B", Bool(true))
	c := new("BBDetail3C", nil)
	ParseFlags("  +BBDetail3A , -BBDetail3B ,, BBDetail3C  ")
	if !a.Enabled() {
		t.Fatal("+Name did not enable")
	}
	if b.Enabled() {
		t.Fatal("-Name did not disable")
	}
	if !c.Enabled() {
		t.Fatal("bare Name did not enable")
	}
}

// TestDetail04: unknown flag names are logged and skipped — not an error.
func TestDetail04(t *testing.T) {
	k := new("BBDetail4", Bool(true))
	ParseFlags("+BBDetail4NoSuchFlag, -BBDetail4AlsoUnknown")
	if !k.Enabled() {
		t.Fatal("unrelated flag disturbed by unknown names")
	}
}

// TestDetail05: parse order — later items override earlier ones.
func TestDetail05(t *testing.T) {
	a := new("BBDetail5A", nil)
	ParseFlags("+BBDetail5A,-BBDetail5A")
	if a.Enabled() {
		t.Fatal("later - did not override earlier +")
	}
	b := new("BBDetail5B", nil)
	ParseFlags("-BBDetail5B,+BBDetail5B")
	if !b.Enabled() {
		t.Fatal("later + did not override earlier -")
	}
}

// TestDetail06: the env var is parsed once at package init; later env
// mutation has no effect until ParseFlags is called again.
func TestDetail06(t *testing.T) {
	f := new("BBDetail6", Bool(true))
	os.Setenv(Name, "-BBDetail6")
	defer os.Unsetenv(Name)
	if !f.Enabled() {
		t.Fatal("env mutation after init took effect without ParseFlags")
	}
	ParseFlags(os.Getenv(Name))
	if f.Enabled() {
		t.Fatal("explicit ParseFlags did not apply the env value")
	}
}

// TestDetail07: Get returns the flag for a known name; for an unknown name it
// returns a non-nil error that names the flag (shape only — exact text is an
// implementation detail).
func TestDetail07(t *testing.T) {
	new("BBDetail7", nil)
	if _, err := Get("BBDetail7"); err != nil {
		t.Fatalf("Get on a known flag errored: %v", err)
	}
	_, err := Get("BBDetail7Unknown")
	if err == nil {
		t.Fatal("Get on an unknown flag succeeded")
	}
	if !strings.Contains(err.Error(), "BBDetail7Unknown") {
		t.Fatalf("error does not name the missing flag: %v", err)
	}
}
