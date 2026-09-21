package stringorset_test

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"reflect"
	"sort"
	"strconv"
	"testing"

	"example.internal/kops/pkg/util/stringorset"
)

func hiddenSeed() int64 {
	if s := os.Getenv("HIDDEN_SEED"); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
	}
	return 20260919
}

func randVals(rng *rand.Rand, n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("v%d-%d", rng.Intn(50), i)
	}
	return out
}

func marshal(t *testing.T, s stringorset.StringOrSet) string {
	t.Helper()
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(b)
}

// Detail 1: Set(v) always marshals an array — including 0 and 1 elements.
func TestDetail01_SetAlwaysArray(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 100; i++ {
		n := rng.Intn(4)
		v := randVals(rng, n)
		got := marshal(t, stringorset.Set(v))
		if got[0] != '[' {
			t.Fatalf("i=%d Set(%v) marshaled %q want array", i, v, got)
		}
		var arr []string
		if err := json.Unmarshal([]byte(got), &arr); err != nil {
			t.Fatalf("i=%d unmarshal %q: %v", i, got, err)
		}
		sort.Strings(arr)
		want := append([]string{}, v...)
		sort.Strings(want)
		if !reflect.DeepEqual(arr, want) {
			t.Fatalf("i=%d array %v want %v", i, arr, want)
		}
	}
	if got := marshal(t, stringorset.Set([]string{"only"})); got != `["only"]` {
		t.Fatalf("Set single = %q want [\"only\"]", got)
	}
	if got := marshal(t, stringorset.Set(nil)); got != `[]` {
		t.Fatalf("Set(nil) = %q want []", got)
	}
}

// Detail 2: Of(v...) marshals by cardinality — exactly-1 -> bare string,
// >=2 -> array.
func TestDetail02_OfCardinality(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 100; i++ {
		n := 1 + rng.Intn(5)
		v := randVals(rng, n)
		got := marshal(t, stringorset.Of(v...))
		if n == 1 {
			want, _ := json.Marshal(v[0])
			if got != string(want) {
				t.Fatalf("i=%d Of single = %q want %q", i, got, want)
			}
		} else {
			if got[0] != '[' {
				t.Fatalf("i=%d Of(%v) = %q want array", i, v, got)
			}
		}
	}
}

// Detail 3: all constructors deduplicate input.
func TestDetail03_Deduplicate(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 100; i++ {
		n := 1 + rng.Intn(5)
		v := randVals(rng, n)
		// Duplicate some elements.
		v = append(v, v[rng.Intn(n)])
		s := stringorset.Set(v)
		if len(s.Value()) != n {
			t.Fatalf("i=%d Set(%v) held %d values want %d", i, v, len(s.Value()), n)
		}
		s2 := stringorset.Of(v...)
		if len(s2.Value()) != n {
			t.Fatalf("i=%d Of(%v) held %d", i, v, len(s2.Value()))
		}
	}
}

// Detail 4: empty value marshals as [] — not "", not null; String("")
// holds one empty string -> "".
func TestDetail04_EmptyMarshal(t *testing.T) {
	if got := marshal(t, stringorset.Of()); got != `[]` {
		t.Fatalf("Of() = %q want []", got)
	}
	if got := marshal(t, stringorset.Set(nil)); got != `[]` {
		t.Fatalf("Set(nil) = %q want []", got)
	}
	if got := marshal(t, stringorset.String("")); got != `""` {
		t.Fatalf("String(\"\") = %q want \"\"", got)
	}
	var zero stringorset.StringOrSet
	if got := marshal(t, zero); got != `[]` {
		t.Fatalf("zero = %q want []", got)
	}
}

// Detail 5: unmarshalling a JSON array sets force-array — a parsed ["a"]
// re-marshals as ["a"].
func TestDetail05_ArrayRoundTripForcesArray(t *testing.T) {
	var s stringorset.StringOrSet
	if err := json.Unmarshal([]byte(`["a"]`), &s); err != nil {
		t.Fatal(err)
	}
	if got := marshal(t, s); got != `["a"]` {
		t.Fatalf("re-marshal = %q want [\"a\"]", got)
	}
	// Scalar round-trip stays scalar.
	var s2 stringorset.StringOrSet
	if err := json.Unmarshal([]byte(`"a"`), &s2); err != nil {
		t.Fatal(err)
	}
	if got := marshal(t, s2); got != `"a"` {
		t.Fatalf("scalar re-marshal = %q want \"a\"", got)
	}
}

// Detail 6: malformed JSON array swallowed (nil error, force-array set,
// elements unchanged); malformed string propagates.
func TestDetail06_MalformedAsymmetry(t *testing.T) {
	s := stringorset.Of("x")
	if err := s.UnmarshalJSON([]byte(`["a"`)); err != nil {
		t.Fatalf("malformed array returned error %v", err)
	}
	if got := marshal(t, s); got[0] != '[' {
		t.Fatalf("force-array not set after malformed array: %q", got)
	}
	if len(s.Value()) != 1 || s.Value()[0] != "x" {
		t.Fatalf("elements changed by malformed array: %v", s.Value())
	}
	s2 := stringorset.Of("x")
	if err := s2.UnmarshalJSON([]byte(`"a`)); err == nil {
		t.Fatalf("malformed string swallowed")
	}
}

// Detail 7: Value() and String() present elements in sorted order.
func TestDetail07_SortedValues(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 100; i++ {
		v := randVals(rng, 2+rng.Intn(8))
		s := stringorset.Set(v)
		got := s.Value()
		want := append([]string{}, v...)
		sort.Strings(want)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("i=%d Value()=%v want %v", i, got, want)
		}
		str := s.String()
		if !sort.StringsAreSorted(want) || str == "" {
			t.Fatalf("i=%d String()=%q", i, str)
		}
	}
}

// Detail 8: Equal compares only the element set, ignoring
// forceEncodeAsArray.
func TestDetail08_EqualIgnoresEncodingFlag(t *testing.T) {
	a := stringorset.Set([]string{"x", "y"})
	b := stringorset.Of("x", "y")
	if !a.Equal(b) || !b.Equal(a) {
		t.Fatalf("Set vs Of unequal with same elements")
	}
	var c stringorset.StringOrSet
	if err := c.UnmarshalJSON([]byte(`["x","y"]`)); err != nil {
		t.Fatal(err)
	}
	if !a.Equal(c) {
		t.Fatalf("unmarshaled array vs Set unequal")
	}
	if a.Equal(stringorset.Of("x")) {
		t.Fatalf("subset equal")
	}
	if a.Equal(stringorset.Of("x", "y", "z")) {
		t.Fatalf("superset equal")
	}
}
