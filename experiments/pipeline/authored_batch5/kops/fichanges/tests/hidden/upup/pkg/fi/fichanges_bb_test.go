package fi

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func bbStr(s string) *string { return &s }

type bbD1 struct {
	P *string
	M map[string]string
	S string
}

// TestDetail01: a nil POINTER field in e means "don't care"; other nil kinds
// (maps, slices, scalars) in e are real expected values that still compare.
func TestDetail01(t *testing.T) {
	a := &bbD1{P: bbStr("actual"), M: map[string]string{"k": "v"}, S: "old"}
	e := &bbD1{P: nil, M: nil, S: "new"}
	changes := &bbD1{P: bbStr("sentinel"), M: map[string]string{"x": "y"}, S: "sent"}

	if !BuildChanges(a, e, changes) {
		t.Fatal("expected changes to be detected")
	}
	if changes.P == nil || *changes.P != "sentinel" {
		t.Fatalf("nil pointer in e should be skipped, changes.P = %v", changes.P)
	}
	if changes.M != nil {
		t.Fatalf("nil map in e is a real value and should be copied: %v", changes.M)
	}
	if changes.S != "new" {
		t.Fatalf("differing scalar not copied: %q", changes.S)
	}
}

// TestDetail02: a nil a copies every non-nil e field into changes and
// reports changed.
func TestDetail02(t *testing.T) {
	var a *bbD1
	e := &bbD1{P: bbStr("x"), M: map[string]string{"k": "v"}, S: "new"}
	changes := &bbD1{}
	if !BuildChanges(a, e, changes) {
		t.Fatal("nil a should always report changed")
	}
	if changes.P == nil || *changes.P != "x" {
		t.Fatalf("non-nil e field not copied: %v", changes.P)
	}
	if len(changes.M) != 1 || changes.M["k"] != "v" {
		t.Fatalf("map field not copied: %v", changes.M)
	}
	if changes.S != "new" {
		t.Fatalf("scalar field not copied: %q", changes.S)
	}
}

type bbD3 struct {
	V      string
	hidden string
}

// TestDetail03: unexported fields are skipped entirely.
func TestDetail03(t *testing.T) {
	a := &bbD3{V: "same", hidden: "a"}
	e := &bbD3{V: "same", hidden: "e"}
	changes := &bbD3{V: "sentinel"}
	if BuildChanges(a, e, changes) {
		t.Fatal("unexported field difference counted as a change")
	}
	if changes.V != "sentinel" {
		t.Fatal("equal exported field was touched")
	}
}

type bbCWID struct {
	id  *string
	tag string
}

func (c *bbCWID) CompareWithID() *string { return c.id }

type bbD4 struct {
	R *bbCWID
}

// TestDetail04: CompareWithID values are equal only when both IDs are
// non-nil and equal; nil IDs fall through to ordinary deep equality.
func TestDetail04(t *testing.T) {
	// equal IDs, different payloads -> equal, no change recorded
	a := &bbD4{R: &bbCWID{id: bbStr("1"), tag: "a"}}
	e := &bbD4{R: &bbCWID{id: bbStr("1"), tag: "e"}}
	changes := &bbD4{}
	if BuildChanges(a, e, changes) {
		t.Fatal("equal CompareWithID ids reported as changed")
	}
	if changes.R != nil {
		t.Fatal("equal field was copied")
	}

	// different IDs -> changed
	e = &bbD4{R: &bbCWID{id: bbStr("2"), tag: "e"}}
	if !BuildChanges(a, e, changes) {
		t.Fatal("different CompareWithID ids not detected")
	}
	if changes.R != e.R {
		t.Fatal("changed field not copied from e")
	}

	// nil IDs fall through to deep equality: differing payloads -> changed
	changes = &bbD4{}
	a = &bbD4{R: &bbCWID{id: nil, tag: "a"}}
	e = &bbD4{R: &bbCWID{id: nil, tag: "e"}}
	if !BuildChanges(a, e, changes) {
		t.Fatal("nil IDs with differing payloads should not compare equal")
	}

	// nil IDs, identical payloads -> equal
	changes = &bbD4{}
	e = &bbD4{R: &bbCWID{id: nil, tag: "a"}}
	if BuildChanges(a, e, changes) {
		t.Fatal("nil IDs with identical payloads reported as changed")
	}
}

type bbRes struct {
	data  string
	ready bool
}

func (r *bbRes) Open() (io.Reader, error) { return strings.NewReader(r.data), nil }
func (r *bbRes) IsReady() bool            { return r.ready }

type bbErrRes struct{}

func (r *bbErrRes) Open() (io.Reader, error) { return nil, errors.New("bb open failure") }

type bbD5 struct {
	R Resource
}

// TestDetail05: Resource fields compare via ResourcesMatch; HasIsReady is
// consulted on the EXPECTED side only — a not-ready expected resource is a
// change without comparing bytes.
func TestDetail05(t *testing.T) {
	// expected not ready -> change even though bytes are identical
	a := &bbD5{R: &bbRes{data: "x", ready: true}}
	e := &bbD5{R: &bbRes{data: "x", ready: false}}
	changes := &bbD5{}
	if !BuildChanges(a, e, changes) {
		t.Fatal("not-ready expected resource did not count as a change")
	}
	if changes.R != e.R {
		t.Fatal("changed field not copied from e")
	}

	// actual not ready, expected ready: readiness of a is ignored; equal
	// bytes means no change
	a = &bbD5{R: &bbRes{data: "x", ready: false}}
	e = &bbD5{R: &bbRes{data: "x", ready: true}}
	changes = &bbD5{}
	if BuildChanges(a, e, changes) {
		t.Fatal("readiness of the actual resource was consulted")
	}

	// different bytes, both ready -> changed
	a = &bbD5{R: &bbRes{data: "a", ready: true}}
	e = &bbD5{R: &bbRes{data: "e", ready: true}}
	if !BuildChanges(a, e, &bbD5{}) {
		t.Fatal("different resource bytes not detected")
	}
}

type bbD6 struct {
	M map[string]string
	N map[string]*bbCWID
	L []string
	P []*bbCWID
}

// TestDetail06: map/slice equality requires identical nil-ness and length,
// then compares elementwise with the custom equality.
func TestDetail06(t *testing.T) {
	cases := []struct {
		name    string
		a, e    *bbD6
		changed bool
	}{
		{"nil-vs-empty-map", &bbD6{M: nil}, &bbD6{M: map[string]string{}}, true},
		{"equal-map", &bbD6{M: map[string]string{"k": "v"}}, &bbD6{M: map[string]string{"k": "v"}}, false},
		{"map-value-diff", &bbD6{M: map[string]string{"k": "v"}}, &bbD6{M: map[string]string{"k": "w"}}, true},
		{"map-len-diff", &bbD6{M: map[string]string{"k": "v"}}, &bbD6{M: map[string]string{"k": "v", "k2": "v"}}, true},
		{"map-key-diff", &bbD6{M: map[string]string{"k1": "v"}}, &bbD6{M: map[string]string{"k2": "v"}}, true},
		{"nil-vs-empty-slice", &bbD6{L: nil}, &bbD6{L: []string{}}, true},
		{"equal-slice", &bbD6{L: []string{"a", "b"}}, &bbD6{L: []string{"a", "b"}}, false},
		{"slice-elem-diff", &bbD6{L: []string{"a"}}, &bbD6{L: []string{"b"}}, true},
		{"slice-len-diff", &bbD6{L: []string{"a"}}, &bbD6{L: []string{"a", "b"}}, true},
		// elementwise recursion uses the custom equality: equal IDs => equal
		{"map-elem-cwid", &bbD6{N: map[string]*bbCWID{"x": {id: bbStr("1"), tag: "a"}}},
			&bbD6{N: map[string]*bbCWID{"x": {id: bbStr("1"), tag: "e"}}}, false},
		{"slice-elem-cwid", &bbD6{P: []*bbCWID{{id: bbStr("1"), tag: "a"}}},
			&bbD6{P: []*bbCWID{{id: bbStr("1"), tag: "e"}}}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := BuildChanges(tc.a, tc.e, &bbD6{}); got != tc.changed {
				t.Fatalf("changed = %v, want %v", got, tc.changed)
			}
		})
	}
}

// TestDetail07: errors from resource comparison inside equalFieldValues are
// fatal (klog.Fatalf) — they cannot be returned because the signature has no
// error return. Verified via a child process: a comparison error must kill
// the process rather than be silently swallowed.
func TestDetail07(t *testing.T) {
	if os.Getenv("BB_FATAL_CHILD") == "1" {
		a := &bbD5{R: &bbRes{data: "x", ready: true}}
		e := &bbD5{R: &bbErrRes{}}
		BuildChanges(a, e, &bbD5{})
		os.Exit(0)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestDetail07$")
	cmd.Env = append(os.Environ(), "BB_FATAL_CHILD=1")
	if err := cmd.Run(); err == nil {
		t.Fatal("a resource comparison error was silently swallowed")
	}
}
