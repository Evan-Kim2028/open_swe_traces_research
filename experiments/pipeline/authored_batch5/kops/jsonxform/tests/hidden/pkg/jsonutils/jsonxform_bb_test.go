package jsonutils

import (
	"errors"
	"strings"
	"testing"
)

// TestDetail01: Transform mutates the input map in place — string callbacks
// replace values by writing back into the parent container.
func TestDetail01(t *testing.T) {
	tr := NewTransformer()
	tr.AddStringTransform(func(path, v string) (string, error) { return v + "!", nil })
	m := map[string]any{
		"k": "v",
		"n": map[string]any{"s": "x"},
	}
	if err := tr.Transform(m); err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if m["k"] != "v!" {
		t.Fatalf("top-level string not replaced: %v", m["k"])
	}
	if m["n"].(map[string]any)["s"] != "x!" {
		t.Fatalf("nested string not replaced: %v", m["n"])
	}
}

// TestDetail02: paths are dot-joined key names; the documented example is
// "metadata.name" for a nested field. (The leading-dot form for top-level
// keys is an implementation detail — only that the path identifies the key
// is asserted for those.)
func TestDetail02(t *testing.T) {
	var paths []string
	tr := NewTransformer()
	tr.AddStringTransform(func(path, v string) (string, error) {
		paths = append(paths, path)
		return v, nil
	})
	m := map[string]any{
		"metadata": map[string]any{"name": "x"},
		"top":      "y",
	}
	if err := tr.Transform(m); err != nil {
		t.Fatalf("Transform: %v", err)
	}
	var nested, top string
	for _, p := range paths {
		if strings.HasSuffix(p, "name") && strings.Contains(p, "metadata") {
			nested = p
		}
		if strings.HasSuffix(p, "top") {
			top = p
		}
	}
	// the leading-dot prefix on paths is an implementation detail — assert
	// the path identifies the full key chain
	if !strings.HasSuffix(nested, "metadata.name") {
		t.Fatalf("nested path = %q, want a dotted chain ending metadata.name (paths: %v)", nested, paths)
	}
	if top == "" {
		t.Fatalf("top-level key path missing (paths: %v)", paths)
	}
}

// TestDetail03: slice elements are visited at the slice path with "[]"
// appended (per api.md, no index).
func TestDetail03(t *testing.T) {
	var slicePaths, elemPaths []string
	tr := NewTransformer()
	tr.AddSliceTransform(func(path string, v []any) ([]any, error) {
		slicePaths = append(slicePaths, path)
		return v, nil
	})
	tr.AddStringTransform(func(path, v string) (string, error) {
		elemPaths = append(elemPaths, path)
		return v, nil
	})
	m := map[string]any{
		"spec": map[string]any{
			"containers": []any{"a", "b"},
		},
	}
	if err := tr.Transform(m); err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if len(slicePaths) != 1 || !strings.HasSuffix(slicePaths[0], "containers[]") {
		t.Fatalf("slice paths = %v", slicePaths)
	}
	if len(elemPaths) != 2 {
		t.Fatalf("element visits = %v", elemPaths)
	}
	for _, p := range elemPaths {
		if p != slicePaths[0] {
			t.Fatalf("element path %q != slice path %q", p, slicePaths[0])
		}
	}
}

// TestDetail04: the type whitelist is map[string]any, []any, int64, float64,
// bool, string, nil — other types error.
func TestDetail04(t *testing.T) {
	ok := map[string]any{
		"m": map[string]any{},
		"s": []any{"x"},
		"i": int64(3),
		"f": float64(1.5),
		"b": true,
		"t": "str",
		"n": nil,
	}
	if err := NewTransformer().Transform(ok); err != nil {
		t.Fatalf("whitelisted types rejected: %v", err)
	}
	for _, bad := range []any{int(3), []string{"a"}, struct{ X int }{1}, int32(2)} {
		m := map[string]any{"k": bad}
		if err := NewTransformer().Transform(m); err == nil {
			t.Fatalf("%T accepted", bad)
		}
	}
}

// TestDetail05: object transforms run BEFORE children are visited — a map
// mutation is visible to child visits; slice transforms may replace the
// whole slice before element visits.
func TestDetail05(t *testing.T) {
	var seen []string
	tr := NewTransformer()
	tr.AddObjectTransform(func(path string, m map[string]any) error {
		if strings.HasSuffix(path, "outer") {
			m["injected"] = "added"
		}
		return nil
	})
	tr.AddStringTransform(func(path, v string) (string, error) {
		seen = append(seen, path)
		return v, nil
	})
	m := map[string]any{"outer": map[string]any{"inner": "x"}}
	if err := tr.Transform(m); err != nil {
		t.Fatalf("Transform: %v", err)
	}
	foundInjected := false
	for _, p := range seen {
		if strings.HasSuffix(p, "injected") {
			foundInjected = true
		}
	}
	if !foundInjected {
		t.Fatalf("injected key not seen by children: %v", seen)
	}

	// slice transform may replace the whole slice before elements visit
	tr2 := NewTransformer()
	var elemSeen []string
	tr2.AddSliceTransform(func(path string, v []any) ([]any, error) {
		return []any{"replaced"}, nil
	})
	tr2.AddStringTransform(func(path, v string) (string, error) {
		elemSeen = append(elemSeen, v)
		return v, nil
	})
	m2 := map[string]any{"l": []any{"a", "b"}}
	if err := tr2.Transform(m2); err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if len(elemSeen) != 1 || elemSeen[0] != "replaced" {
		t.Fatalf("elements of replaced slice = %v", elemSeen)
	}
}

// TestDetail06: string transforms compose left-to-right; an error aborts the
// walk.
func TestDetail06(t *testing.T) {
	tr := NewTransformer()
	tr.AddStringTransform(func(path, v string) (string, error) { return v + "a", nil })
	tr.AddStringTransform(func(path, v string) (string, error) { return v + "b", nil })
	m := map[string]any{"k": "x"}
	if err := tr.Transform(m); err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if m["k"] != "xab" {
		t.Fatalf("composition order = %q, want xab", m["k"])
	}

	second := false
	tr = NewTransformer()
	tr.AddStringTransform(func(path, v string) (string, error) { return "", errors.New("boom") })
	tr.AddStringTransform(func(path, v string) (string, error) {
		second = true
		return v, nil
	})
	if err := tr.Transform(map[string]any{"k": "x"}); err == nil {
		t.Fatal("transform error not propagated")
	}
	if second {
		t.Fatal("later transform ran after an error")
	}
}

// TestDetail07: SortSlice returns a copy sorted by JSON encoding — strings
// before numbers before objects (by first byte); input is not mutated.
func TestDetail07(t *testing.T) {
	in := []any{map[string]any{"a": 1.0}, "str", float64(2), float64(10)}
	out, err := SortSlice(in)
	if err != nil {
		t.Fatalf("SortSlice: %v", err)
	}
	// JSON encodings: "str" (0x22), 10 ("10"=0x31), 2 ("2"=0x32), {"a":1} (0x7b)
	want := []any{"str", float64(10), float64(2), map[string]any{"a": 1.0}}
	if len(out) != len(want) {
		t.Fatalf("len = %d", len(out))
	}
	for i := range want {
		if !jsonEqual(out[i], want[i]) {
			t.Fatalf("sorted[%d] = %v, want %v (out=%v)", i, out[i], want[i], out)
		}
	}
	// input is a copy — original order preserved
	if _, isMap := in[0].(map[string]any); !isMap {
		t.Fatal("SortSlice mutated its input")
	}
}

func jsonEqual(a, b any) bool {
	switch av := a.(type) {
	case string:
		bv, ok := b.(string)
		return ok && av == bv
	case float64:
		bv, ok := b.(float64)
		return ok && av == bv
	default:
		bm, ok := b.(map[string]any)
		if !ok {
			return false
		}
		am, _ := a.(map[string]any)
		if len(am) != len(bm) {
			return false
		}
		for k, v := range am {
			if !jsonEqual(v, bm[k]) {
				return false
			}
		}
		return true
	}
}

// TestDetail08: primitives pass through unchanged when no transform
// consumes them.
func TestDetail08(t *testing.T) {
	m := map[string]any{
		"f": float64(1.5),
		"b": true,
		"i": int64(7),
		"n": nil,
	}
	if err := NewTransformer().Transform(m); err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if m["f"] != float64(1.5) || m["b"] != true || m["i"] != int64(7) || m["n"] != nil {
		t.Fatalf("primitives changed: %v", m)
	}
}
