// Hidden black-box suite for copyst. Exported API only: Copy.
package copystructure_test

import (
	"math/rand"
	"os"
	"reflect"
	"strconv"
	"testing"
	"unsafe"

	"example.internal/helm/internal/copystructure"
)

const HiddenSeed int64 = 20260919

func hiddenSeed() int64 {
	if s := os.Getenv("HIDDEN_SEED"); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
	}
	return HiddenSeed
}

func TestDetail01_CopyNilYieldsEmptyMap(t *testing.T) {
	got, err := copystructure.Copy(nil)
	if err != nil {
		t.Fatal(err)
	}
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("Copy(nil) type %T want map[string]any", got)
	}
	if m == nil {
		t.Fatal("Copy(nil) must be a non-nil empty map")
	}
	if len(m) != 0 {
		t.Fatalf("Copy(nil) len=%d", len(m))
	}
}

func TestDetail02_ScalarsAndArraysReturnedAsIs(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 200; i++ {
		switch rng.Intn(6) {
		case 0:
			v := rng.Int()
			got, err := copystructure.Copy(v)
			if err != nil {
				t.Fatal(err)
			}
			if got != v {
				t.Fatalf("int %v -> %v", v, got)
			}
		case 1:
			v := rng.Float64()
			got, err := copystructure.Copy(v)
			if err != nil {
				t.Fatal(err)
			}
			if got != v {
				t.Fatalf("float %v -> %v", v, got)
			}
		case 2:
			v := rng.Intn(2) == 0
			got, err := copystructure.Copy(v)
			if err != nil {
				t.Fatal(err)
			}
			if got != v {
				t.Fatalf("bool %v -> %v", v, got)
			}
		case 3:
			v := strconv.Itoa(rng.Intn(1e6))
			got, err := copystructure.Copy(v)
			if err != nil {
				t.Fatal(err)
			}
			if got != v {
				t.Fatalf("string %q -> %v", v, got)
			}
		case 4:
			v := [3]int{rng.Intn(9), rng.Intn(9), rng.Intn(9)}
			got, err := copystructure.Copy(v)
			if err != nil {
				t.Fatal(err)
			}
			if got != v {
				t.Fatalf("array %v -> %v", v, got)
			}
		default:
			v := uint64(rng.Uint64())
			got, err := copystructure.Copy(v)
			if err != nil {
				t.Fatal(err)
			}
			if got != v {
				t.Fatalf("uint64 %v -> %v", v, got)
			}
		}
	}
}

func TestDetail03_InterfaceNilPreservedDynamicCopied(t *testing.T) {
	var boxed any = nil
	got, err := copystructure.Copy(boxed)
	if err != nil {
		t.Fatal(err)
	}
	// Copy(nil) is the empty-map special case (detail 1). A typed-nil interface
	// stored in a map/slice is tested in details 4/6. Here a non-nil interface
	// wrapping a map must copy the dynamic value.
	inner := map[string]any{"k": 1}
	var iface any = inner
	got, err = copystructure.Copy(iface)
	if err != nil {
		t.Fatal(err)
	}
	gm, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("dynamic map copy type %T", got)
	}
	gm["k"] = 99
	if inner["k"] != 1 {
		t.Fatal("dynamic value must be deep-copied, not shared")
	}
}

func TestDetail04_MapNilAndNilInterfaceValues(t *testing.T) {
	var n map[string]any
	got, err := copystructure.Copy(n)
	if err != nil {
		t.Fatal(err)
	}
	if !isTypedNil(got) {
		t.Fatalf("nil map must stay nil, got %#v", got)
	}
	src := map[string]any{
		"a": 1,
		"b": map[string]any{"z": "q"},
		"n": any(nil),
	}
	got, err = copystructure.Copy(src)
	if err != nil {
		t.Fatal(err)
	}
	out, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("type %T", got)
	}
	if _, has := out["n"]; !has {
		t.Fatal("nil interface value must be preserved as a key")
	}
	if out["n"] != nil {
		t.Fatalf("nil interface value got %#v", out["n"])
	}
	nested := out["b"].(map[string]any)
	nested["z"] = "mut"
	if src["b"].(map[string]any)["z"] != "q" {
		t.Fatal("map values must be deep-copied")
	}
	if reflect.TypeOf(out) != reflect.TypeOf(src) {
		t.Fatalf("map type %v want %v", reflect.TypeOf(out), reflect.TypeOf(src))
	}
}

func TestDetail05_PointerNilOrRewrap(t *testing.T) {
	var p *int
	got, err := copystructure.Copy(p)
	if err != nil {
		t.Fatal(err)
	}
	if !isTypedNil(got) {
		t.Fatalf("nil pointer must stay nil, got %#v", got)
	}
	v := 7
	gp, err := copystructure.Copy(&v)
	if err != nil {
		t.Fatal(err)
	}
	op, ok := gp.(*int)
	if !ok {
		t.Fatalf("type %T", gp)
	}
	if op == &v {
		t.Fatal("pointer must be re-wrapped, not shared")
	}
	if *op != 7 {
		t.Fatalf("pointee %d", *op)
	}
	*op = 1
	if v != 7 {
		t.Fatal("pointee must be deep-copied")
	}
}

func TestDetail06_SliceLenAndCapNilElems(t *testing.T) {
	var n []any
	got, err := copystructure.Copy(n)
	if err != nil {
		t.Fatal(err)
	}
	if !isTypedNil(got) {
		t.Fatalf("nil slice must stay nil, got %#v", got)
	}
	s := make([]any, 2, 8)
	s[0] = "a"
	s[1] = any(nil)
	got, err = copystructure.Copy(s)
	if err != nil {
		t.Fatal(err)
	}
	out := got.([]any)
	if len(out) != 2 {
		t.Fatalf("len %d", len(out))
	}
	if cap(out) != 8 {
		t.Fatalf("cap must be preserved, got %d want 8", cap(out))
	}
	if out[1] != nil {
		t.Fatalf("nil interface elem got %#v", out[1])
	}
	out[0] = "mut"
	if s[0] != "a" {
		t.Fatal("slice elems must be copied")
	}
}

func TestDetail07_StructFieldWiseUnexportedPanics(t *testing.T) {
	type exp struct {
		A int
		B string
	}
	in := exp{A: 3, B: "x"}
	got, err := copystructure.Copy(in)
	if err != nil {
		t.Fatal(err)
	}
	out := got.(exp)
	if out != in {
		t.Fatalf("struct copy %v", out)
	}
	type hid struct{ a int }
	defer func() {
		if recover() == nil {
			t.Fatal("unexported struct field must panic")
		}
	}()
	_, _ = copystructure.Copy(hid{a: 1})
}

func TestDetail08_FuncChanSharedUnsupportedKindErrors(t *testing.T) {
	fn := func() {}
	got, err := copystructure.Copy(fn)
	if err != nil {
		t.Fatal(err)
	}
	if reflect.ValueOf(got).Pointer() != reflect.ValueOf(fn).Pointer() {
		t.Fatal("func must be returned as-is (shared)")
	}
	ch := make(chan int, 1)
	got, err = copystructure.Copy(ch)
	if err != nil {
		t.Fatal(err)
	}
	if got.(chan int) != ch {
		t.Fatal("chan must be returned as-is (shared)")
	}
	up := unsafe.Pointer(&fn)
	got, err = copystructure.Copy(up)
	if err != nil {
		t.Fatal(err)
	}
	if got != up {
		t.Fatal("unsafe.Pointer must be returned as-is")
	}
	_, err = copystructure.Copy(uintptr(7))
	if err != nil {
		if !stringsContains(err.Error(), "unsupported type") {
			t.Fatalf("unsupported wording: %v", err)
		}
	}
}

func isTypedNil(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return rv.IsNil()
	default:
		return false
	}
}

func stringsContains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		})())
}
