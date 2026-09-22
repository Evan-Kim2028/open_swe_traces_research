package stringorset

import (
	"strings"
	"testing"
)

// TestDetail01 — MarshalJSON: array when forced or len>1; bare string for a single
// non-forced value; empty non-forced set encodes [].
func TestDetail01(t *testing.T) {
	cases := []struct {
		s    StringOrSet
		want string
	}{
		{Of("a", "b"), `["a","b"]`},
		{Of("a"), `"a"`},
		{String("a"), `"a"`},
		{Of(), `[]`},
		{Set([]string{"a"}), `["a"]`},
	}
	for i, c := range cases {
		b, err := c.s.MarshalJSON()
		if err != nil {
			t.Fatalf("case %d: MarshalJSON: %v", i, err)
		}
		if string(b) != c.want {
			t.Errorf("case %d: MarshalJSON = %s, want %s", i, b, c.want)
		}
	}
}

// TestDetail02 — UnmarshalJSON keys on the first byte '['; array payload sets forced-array
// (so ["x"] round-trips as an array), string payload clears it; malformed array swallowed.
func TestDetail02(t *testing.T) {
	// ["x"] round-trips as an array, not a bare string.
	var s StringOrSet
	if err := s.UnmarshalJSON([]byte(`["x"]`)); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	b, err := s.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if string(b) != `["x"]` {
		t.Errorf(`["x"] did not round-trip as an array: %s`, b)
	}

	// A string payload clears the forced-array state.
	var s2 StringOrSet
	if err := s2.UnmarshalJSON([]byte(`["a","b"]`)); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if err := s2.UnmarshalJSON([]byte(`"solo"`)); err != nil {
		t.Fatalf("UnmarshalJSON string: %v", err)
	}
	b, err = s2.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if string(b) != `"solo"` {
		t.Errorf("string payload did not clear forced-array state: %s", b)
	}

	// Malformed array payload is swallowed (returns nil).
	var s3 StringOrSet
	if err := s3.UnmarshalJSON([]byte(`[`)); err != nil {
		t.Errorf("malformed array payload should be swallowed, got %v", err)
	}

	// Non-'[' payloads take the string path; a non-string JSON value errors there.
	var s4 StringOrSet
	if err := s4.UnmarshalJSON([]byte(`{"a":1}`)); err == nil {
		t.Error("expected error unmarshalling a JSON object into StringOrSet")
	}
}

// TestDetail03 — String() comma-joins the sorted values with no spaces.
func TestDetail03(t *testing.T) {
	if got := Of("b", "a", "c").String(); got != "a,b,c" {
		t.Errorf("String() = %q, want sorted comma-join \"a,b,c\"", got)
	}
	if got := Of().String(); strings.Contains(got, " ") || strings.Contains(got, ",") {
		t.Errorf("empty String() = %q, should contain no commas/spaces", got)
	}
}

// TestDetail04 — Value() returns a sorted copy (mutating it does not affect the source).
func TestDetail04(t *testing.T) {
	s := Of("b", "a")
	v := s.Value()
	if len(v) != 2 || v[0] != "a" || v[1] != "b" {
		t.Fatalf("Value() = %v, want sorted [a b]", v)
	}
	v[0] = "zz"
	if s.String() != "a,b" {
		t.Errorf("mutating Value() result changed the source: %q", s.String())
	}
}

// TestDetail05 — constructor forcing: Set always array-encodes; Of only when len>1;
// String never array-encodes.
func TestDetail05(t *testing.T) {
	checks := []struct {
		name string
		s    StringOrSet
		want string
	}{
		{"Set single", Set([]string{"a"}), `["a"]`},
		{"Of single", Of("a"), `"a"`},
		{"String", String("a"), `"a"`},
		{"Of pair", Of("x", "y"), `["x","y"]`},
	}
	for _, c := range checks {
		b, err := c.s.MarshalJSON()
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if string(b) != c.want {
			t.Errorf("%s: MarshalJSON = %s, want %s", c.name, b, c.want)
		}
	}
}

// TestDetail06 — Equal is order-insensitive set equality (independent of encoding flag).
func TestDetail06(t *testing.T) {
	if !Of("a", "b").Equal(Of("b", "a")) {
		t.Error("Equal should be order-insensitive")
	}
	if Of("a", "b").Equal(Of("a")) {
		t.Error("Equal should distinguish different sets")
	}
	if !String("a").Equal(Set([]string{"a"})) {
		t.Error("Equal should ignore the forced-array flag")
	}
}
