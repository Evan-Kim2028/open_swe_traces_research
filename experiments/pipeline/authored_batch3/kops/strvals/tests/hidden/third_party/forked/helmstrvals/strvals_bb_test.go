// Package helmstrvals_test is the hidden black-box suite for strvals.
// One TestDetailNN per DETAILS.md commitment. Exported API only:
// ParseInto, ParseIntoString, ErrNotList, MaxIndex, MaxNestedNameLevel.
package helmstrvals_test

import (
	"strings"
	"testing"

	sv "example.internal/clustkit/third_party/forked/helmstrvals"
)

func mustParse(t *testing.T, s string) map[string]interface{} {
	t.Helper()
	dest := map[string]interface{}{}
	if err := sv.ParseInto(s, dest); err != nil {
		t.Fatalf("ParseInto(%q): %v", s, err)
	}
	return dest
}

// Detail 1 (Inferable: yes): a=b sets dest["a"]; a.b=c nests; , separates
// pairs.
func TestDetail01(t *testing.T) {
	dest := mustParse(t, "a=b")
	if dest["a"] != "b" {
		t.Fatalf("a=b: %v", dest)
	}
	dest = mustParse(t, "a.b=c")
	inner, ok := dest["a"].(map[string]interface{})
	if !ok || inner["b"] != "c" {
		t.Fatalf("a.b=c: %v", dest)
	}
	dest = mustParse(t, "a=b,c=d")
	if dest["a"] != "b" || dest["c"] != "d" {
		t.Fatalf("a=b,c=d: %v", dest)
	}
}

// Detail 2 (Inferable: partially): `key=` at end of input stores "" and
// succeeds.
func TestDetail02(t *testing.T) {
	dest := mustParse(t, "a=")
	if v, ok := dest["a"]; !ok || v != "" {
		t.Fatalf("a= stored %v (present=%v)", v, ok)
	}
}

// Detail 3 (Inferable: partially): a key ending on `,` or at EOF with no `=`
// is an error mentioning that it has no value.
func TestDetail03(t *testing.T) {
	for _, s := range []string{"a", "a,b=c", "a.b"} {
		dest := map[string]interface{}{}
		err := sv.ParseInto(s, dest)
		if err == nil {
			t.Fatalf("%q did not error", s)
		}
		if !strings.Contains(err.Error(), "no value") {
			t.Fatalf("%q error lacks 'no value': %v", s, err)
		}
	}
}

// Detail 4 (Inferable: partially): a[i]=v assigns into a list at index i,
// growing it with nils to length i+1 when too short.
func TestDetail04(t *testing.T) {
	dest := mustParse(t, "a[0]=x")
	l, ok := dest["a"].([]interface{})
	if !ok || len(l) != 1 || l[0] != "x" {
		t.Fatalf("a[0]=x: %v", dest)
	}
	dest = mustParse(t, "a[2]=v")
	l, ok = dest["a"].([]interface{})
	if !ok || len(l) != 3 || l[2] != "v" || l[0] != nil || l[1] != nil {
		t.Fatalf("a[2]=v: %v", dest)
	}
}

// Detail 5 (Inferable: doc): negative index or index > MaxIndex is an error.
func TestDetail05(t *testing.T) {
	for _, s := range []string{"a[-1]=x", "a[65537]=x"} {
		dest := map[string]interface{}{}
		if err := sv.ParseInto(s, dest); err == nil {
			t.Fatalf("%q did not error", s)
		}
	}
	if sv.MaxIndex != 65536 {
		t.Fatalf("MaxIndex = %v", sv.MaxIndex)
	}
}

// Detail 6 (Inferable: no): a.b[i].c=v nests maps inside list elements; an
// existing nil slot becomes a fresh container rather than failing. Asserted
// shape: list elements can hold nested maps and a nil slot is replaced.
// (Under the reference implementation a single un-revisited index loses the
// nested element entirely — a quirk consistent with the DETAILS claim only
// for the re-touched form used here.)
func TestDetail06(t *testing.T) {
	dest := mustParse(t, "a.b[0].c=v,a.b[0].d=w")
	a, ok := dest["a"].(map[string]interface{})
	if !ok {
		t.Fatalf("a not a map: %v", dest)
	}
	b, ok := a["b"].([]interface{})
	if !ok || len(b) != 1 {
		t.Fatalf("a.b not a 1-list: %v", a)
	}
	elem, ok := b[0].(map[string]interface{})
	if !ok || elem["c"] != "v" || elem["d"] != "w" {
		t.Fatalf("a.b[0] not a nested map: %v", b)
	}
	dest = mustParse(t, "a[1].x=v,a[1].y=w")
	l, ok := dest["a"].([]interface{})
	if !ok || len(l) != 2 {
		t.Fatalf("a[1].x=v,...: %v", dest)
	}
	if l[0] != nil {
		t.Fatalf("slot 0 not nil: %v", l)
	}
	m, ok := l[1].(map[string]interface{})
	if !ok || m["x"] != "v" || m["y"] != "w" {
		t.Fatalf("nil slot did not become a container: %v", l)
	}
}

// Detail 7 (Inferable: partially): a={x,y} produces a []interface{} of typed
// values; an unterminated { is an error mentioning }.
func TestDetail07(t *testing.T) {
	dest := mustParse(t, "a={x,y}")
	l, ok := dest["a"].([]interface{})
	if !ok || len(l) != 2 || l[0] != "x" || l[1] != "y" {
		t.Fatalf("a={x,y}: %v", dest)
	}
	dest = mustParse(t, "a={1,2}")
	l, ok = dest["a"].([]interface{})
	if !ok || len(l) != 2 {
		t.Fatalf("a={1,2}: %v", dest)
	}
	if l[0] != int64(1) || l[1] != int64(2) {
		t.Fatalf("list items not typed: %#v", l)
	}
	dest = map[string]interface{}{}
	err := sv.ParseInto("a={x", dest)
	if err == nil || !strings.Contains(err.Error(), "}") {
		t.Fatalf("unterminated list: %v", err)
	}
}

// Detail 8 (Inferable: partially): scalar coercion — true/false -> bool,
// null -> nil, 0 -> int64(0), non-zero-leading integers -> int64, rest stay
// strings; ParseIntoString never coerces.
func TestDetail08(t *testing.T) {
	dest := mustParse(t, "t=true,f=false,n=null,z=0,i=42,lead=007,s=abc")
	if dest["t"] != true || dest["f"] != false {
		t.Fatalf("bool coercion: %v", dest)
	}
	if v, ok := dest["n"]; !ok || v != nil {
		t.Fatalf("null coercion: %#v", v)
	}
	if dest["z"] != int64(0) || dest["i"] != int64(42) {
		t.Fatalf("int coercion: %#v", dest)
	}
	if dest["lead"] != "007" || dest["s"] != "abc" {
		t.Fatalf("string coercion: %#v", dest)
	}
	dest = map[string]interface{}{}
	if err := sv.ParseIntoString("a=1,b=true,c=null", dest); err != nil {
		t.Fatal(err)
	}
	if dest["a"] != "1" || dest["b"] != "true" || dest["c"] != "null" {
		t.Fatalf("string variant coerced: %#v", dest)
	}
}

// Detail 9 (Inferable: partially): \ escapes the following character
// verbatim in keys and values — a\.b is one key segment.
func TestDetail09(t *testing.T) {
	dest := mustParse(t, `a\.b=c`)
	if v, ok := dest["a.b"]; !ok || v != "c" {
		t.Fatalf(`a\.b=c: %v`, dest)
	}
	if _, nested := dest["a"]; nested {
		t.Fatalf(`a\.b=c produced a nested map: %v`, dest)
	}
	dest = mustParse(t, `a=b\,c`)
	if dest["a"] != "b,c" {
		t.Fatalf(`a=b\,c: %v`, dest)
	}
}

// Detail 10 (Inferable: doc): more than MaxNestedNameLevel dotted segments is
// an error.
func TestDetail10(t *testing.T) {
	if sv.MaxNestedNameLevel != 30 {
		t.Fatalf("MaxNestedNameLevel = %v", sv.MaxNestedNameLevel)
	}
	segs := make([]string, sv.MaxNestedNameLevel+2)
	for i := range segs {
		segs[i] = "a"
	}
	dest := map[string]interface{}{}
	if err := sv.ParseInto(strings.Join(segs, ".")+"=v", dest); err == nil {
		t.Fatal("over-nested key did not error")
	}
	// at the limit it still parses
	segs = make([]string, sv.MaxNestedNameLevel-1)
	for i := range segs {
		segs[i] = "a"
	}
	dest = map[string]interface{}{}
	if err := sv.ParseInto(strings.Join(segs, ".")+"=v", dest); err != nil {
		t.Fatalf("limit-level key errored: %v", err)
	}
}

// Detail 11 (Inferable: no): an empty key writes nothing — the pair is
// skipped, not an error.
func TestDetail11(t *testing.T) {
	dest := map[string]interface{}{}
	if err := sv.ParseInto("=v", dest); err != nil {
		t.Fatalf("empty key errored: %v", err)
	}
	if len(dest) != 0 {
		t.Fatalf("empty key wrote %v", dest)
	}
}

// Detail 12 (Inferable: no): a pre-existing dest entry with the wrong shape
// turns a panic into an "unable to parse"-class error rather than crashing.
func TestDetail12(t *testing.T) {
	dest := map[string]interface{}{"a": "scalar"}
	err := sv.ParseInto("a.b=c", dest)
	if err == nil {
		t.Fatal("shape conflict did not error")
	}
	if !strings.Contains(err.Error(), "a.b") && !strings.Contains(err.Error(), "parse") {
		t.Fatalf("error does not mention the offending key or parse failure: %v", err)
	}
}

// Detail 13 (Inferable: no): after a } list closes, a following non-comma
// rune is pushed back so parsing continues into the next pair. Asserted
// shape: `a={x,y}b=c` parses both pairs — the `b` after `}` begins a new key.
func TestDetail13(t *testing.T) {
	dest := mustParse(t, "a={x,y}b=c")
	l, ok := dest["a"].([]interface{})
	if !ok || len(l) != 2 {
		t.Fatalf("list not parsed: %v", dest)
	}
	if dest["b"] != "c" {
		t.Fatalf("pair after } not parsed: %v", dest)
	}
}
