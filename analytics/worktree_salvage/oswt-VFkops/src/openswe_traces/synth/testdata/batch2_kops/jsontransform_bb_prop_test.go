package jsonutils_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"example.internal/kops/pkg/jsonutils"
)

func hiddenSeed() int64 {
	if s := os.Getenv("HIDDEN_SEED"); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
	}
	return 20260919
}

// Detail 1: root object visited at path ""; map keys append ".<key>".
func TestDetail01_PathConvention(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 60; i++ {
		tr := jsonutils.NewTransformer()
		var paths []string
		tr.AddObjectTransform(func(path string, value map[string]any) error {
			paths = append(paths, path)
			return nil
		})
		key := fmt.Sprintf("k%d", rng.Intn(1000))
		nested := fmt.Sprintf("n%d", rng.Intn(1000))
		doc := map[string]any{key: map[string]any{nested: "v"}}
		if err := tr.Transform(doc); err != nil {
			t.Fatal(err)
		}
		if len(paths) != 2 || paths[0] != "" || paths[1] != "."+key {
			t.Fatalf("i=%d paths %v want [\"\" \".%s\"]", i, paths, key)
		}
		var spaths []string
		tr2 := jsonutils.NewTransformer()
		tr2.AddStringTransform(func(path string, value string) (string, error) {
			spaths = append(spaths, path)
			return value, nil
		})
		if err := tr2.Transform(doc); err != nil {
			t.Fatal(err)
		}
		if len(spaths) != 1 || spaths[0] != "."+key+"."+nested {
			t.Fatalf("i=%d string path %v want .%s.%s", i, spaths, key, nested)
		}
	}
}

// Detail 2: slice callbacks and elements both use "<path>[]" — element
// index never appears.
func TestDetail02_SlicePathSuffix(t *testing.T) {
	tr := jsonutils.NewTransformer()
	var slicePaths, elemPaths []string
	tr.AddSliceTransform(func(path string, value []any) ([]any, error) {
		slicePaths = append(slicePaths, path)
		return value, nil
	})
	tr.AddStringTransform(func(path string, value string) (string, error) {
		elemPaths = append(elemPaths, path)
		return value, nil
	})
	doc := map[string]any{"a": map[string]any{"b": []any{"x", "y", "z"}}}
	if err := tr.Transform(doc); err != nil {
		t.Fatal(err)
	}
	if len(slicePaths) != 1 || slicePaths[0] != ".a.b[]" {
		t.Fatalf("slice paths %v want [.a.b[]]", slicePaths)
	}
	if len(elemPaths) != 3 {
		t.Fatalf("elem paths %v", elemPaths)
	}
	for _, p := range elemPaths {
		if p != ".a.b[]" {
			t.Fatalf("element path %q want .a.b[]", p)
		}
	}
}

// Detail 3: object transforms fire BEFORE descending (pre-order), in
// registration order.
func TestDetail03_ObjectTransformsPreOrder(t *testing.T) {
	tr := jsonutils.NewTransformer()
	var order []string
	tr.AddObjectTransform(func(path string, value map[string]any) error {
		order = append(order, "obj1:"+path)
		return nil
	})
	tr.AddObjectTransform(func(path string, value map[string]any) error {
		order = append(order, "obj2:"+path)
		return nil
	})
	tr.AddStringTransform(func(path string, value string) (string, error) {
		order = append(order, "str:"+path)
		return value, nil
	})
	doc := map[string]any{"a": map[string]any{"s": "v"}}
	if err := tr.Transform(doc); err != nil {
		t.Fatal(err)
	}
	want := []string{"obj1:", "obj2:", "obj1:.a", "obj2:.a", "str:.a.s"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("order %v want %v", order, want)
	}
}

// Detail 4: slice transforms fire BEFORE descending; each sees the
// previous transform's output; returned slice replaces visited one and is
// stored in the parent.
func TestDetail04_SliceTransformsChainAndReplace(t *testing.T) {
	tr := jsonutils.NewTransformer()
	tr.AddSliceTransform(func(path string, value []any) ([]any, error) {
		out := append([]any{}, value...)
		out = append(out, "added1")
		return out, nil
	})
	var secondSaw []any
	tr.AddSliceTransform(func(path string, value []any) ([]any, error) {
		secondSaw = append([]any{}, value...)
		return value, nil
	})
	tr.AddStringTransform(func(path string, value string) (string, error) {
		return value + "!", nil
	})
	doc := map[string]any{"a": []any{"x"}}
	if err := tr.Transform(doc); err != nil {
		t.Fatal(err)
	}
	if len(secondSaw) != 2 || secondSaw[1] != "added1" {
		t.Fatalf("second transform saw %v want [x added1]", secondSaw)
	}
	got := doc["a"].([]any)
	if len(got) != 2 || got[0] != "x!" || got[1] != "added1!" {
		t.Fatalf("parent slice %v want [x! added1!]", got)
	}
}

// Detail 5: string transforms chain sequentially on each string leaf;
// final value stored back.
func TestDetail05_StringTransformsChain(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 60; i++ {
		tr := jsonutils.NewTransformer()
		mark := fmt.Sprintf("M%d", rng.Intn(100))
		tr.AddStringTransform(func(path string, value string) (string, error) {
			return mark + value, nil
		})
		tr.AddStringTransform(func(path string, value string) (string, error) {
			return value + mark, nil
		})
		doc := map[string]any{"k": "v"}
		if err := tr.Transform(doc); err != nil {
			t.Fatal(err)
		}
		if doc["k"] != mark+"v"+mark {
			t.Fatalf("i=%d doc[k]=%q want %q", i, doc["k"], mark+"v"+mark)
		}
	}
}

// Detail 6: numbers and booleans are visited but NO callbacks fire —
// passed through unchanged.
func TestDetail06_NumbersBooleansPassThrough(t *testing.T) {
	tr := jsonutils.NewTransformer()
	calls := 0
	tr.AddStringTransform(func(path string, value string) (string, error) {
		calls++
		return "X", nil
	})
	tr.AddObjectTransform(func(path string, value map[string]any) error {
		calls++
		return nil
	})
	tr.AddSliceTransform(func(path string, value []any) ([]any, error) {
		calls++
		return value, nil
	})
	doc := map[string]any{"n": 42.5, "b": true, "i": -3.0}
	if err := tr.Transform(doc); err != nil {
		t.Fatal(err)
	}
	if calls != 1 { // only the root object transform fires
		t.Fatalf("callbacks fired %d times want 1", calls)
	}
	if doc["n"] != 42.5 || doc["b"] != true || doc["i"] != -3.0 {
		t.Fatalf("primitives mutated: %v", doc)
	}
}

// Detail 7: null (nil) values short-circuit — visited, unchanged, no
// callbacks.
func TestDetail07_NilShortCircuits(t *testing.T) {
	tr := jsonutils.NewTransformer()
	calls := 0
	tr.AddStringTransform(func(path string, value string) (string, error) {
		calls++
		return "X", nil
	})
	doc := map[string]any{"k": nil, "arr": []any{nil, "s"}}
	if err := tr.Transform(doc); err != nil {
		t.Fatal(err)
	}
	if calls != 1 { // only "s" inside arr
		t.Fatalf("string transform fired %d times want 1", calls)
	}
	if doc["k"] != nil {
		t.Fatalf("nil mutated: %v", doc["k"])
	}
	if doc["arr"].([]any)[0] != nil {
		t.Fatalf("nil in array mutated")
	}
}

// Detail 8: any other Go type aborts the walk with an error naming path
// and type.
func TestDetail08_UnknownTypeAborts(t *testing.T) {
	tr := jsonutils.NewTransformer()
	doc := map[string]any{"k": struct{ X int }{X: 1}}
	err := tr.Transform(doc)
	if err == nil {
		t.Fatalf("struct value accepted")
	}
	if !strings.Contains(err.Error(), ".k") {
		t.Fatalf("error %q lacks path .k", err.Error())
	}
	// A plain Go int (not the JSON-decoded float64) is also an unhandled type.
	tr2 := jsonutils.NewTransformer()
	if err := tr2.Transform(map[string]any{"k": 7}); err == nil {
		t.Fatalf("int value accepted")
	}
}

// Detail 9: callback error aborts immediately with that error.
func TestDetail09_CallbackErrorPropagates(t *testing.T) {
	tr := jsonutils.NewTransformer()
	boom := errors.New("boom-12345")
	calls := 0
	tr.AddStringTransform(func(path string, value string) (string, error) {
		calls++
		if calls == 1 {
			return value, boom
		}
		return value, nil
	})
	doc := map[string]any{"a": "x", "b": "y", "c": "z"}
	err := tr.Transform(doc)
	if !errors.Is(err, boom) {
		t.Fatalf("err %v want boom", err)
	}
	if calls != 1 {
		t.Fatalf("transform continued after error: %d calls", calls)
	}
}

// Detail 10: SortSlice orders by JSON-marshal text; returns a NEW slice.
func TestDetail10_SortSliceByJSONText(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 100; i++ {
		n := 2 + rng.Intn(8)
		in := make([]any, n)
		for j := range in {
			switch rng.Intn(3) {
			case 0:
				in[j] = float64(rng.Intn(100))
			case 1:
				in[j] = fmt.Sprintf("s%d", rng.Intn(100))
			case 2:
				in[j] = map[string]any{"id": float64(rng.Intn(100))}
			}
		}
		orig := append([]any{}, in...)
		out, err := jsonutils.SortSlice(in)
		if err != nil {
			t.Fatalf("i=%d SortSlice err %v", i, err)
		}
		if !reflect.DeepEqual(in, orig) {
			t.Fatalf("i=%d input mutated", i)
		}
		prev := ""
		for j, e := range out {
			b, err := json.Marshal(e)
			if err != nil {
				t.Fatalf("i=%d unmarshalable elem %v", i, e)
			}
			s := string(b)
			if j > 0 && s < prev {
				t.Fatalf("i=%d out[%d]=%q < out[%d]=%q", i, j, s, j-1, prev)
			}
			prev = s
		}
	}
	// Ordering is by raw JSON-marshal text: `"a"` (0x22) sorts before `2`
	// (0x32), and {"id":1} before {"id":2}.
	out, err := jsonutils.SortSlice([]any{float64(2), "a"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 || out[0] != "a" || out[1] != float64(2) {
		t.Fatalf("marshal-text order violated: %v", out)
	}
	out, err = jsonutils.SortSlice([]any{map[string]any{"id": float64(2)}, map[string]any{"id": float64(1)}})
	if err != nil {
		t.Fatal(err)
	}
	if out[0].(map[string]any)["id"] != float64(1) {
		t.Fatalf("object marshal order violated: %v", out)
	}
}

// Detail 11: zero registered transforms -> no-op walk returning nil.
func TestDetail11_ZeroTransformsNoOp(t *testing.T) {
	doc := map[string]any{"a": []any{"x", map[string]any{"b": 1.5}}}
	if err := jsonutils.NewTransformer().Transform(doc); err != nil {
		t.Fatalf("empty transformer errored: %v", err)
	}
	if !reflect.DeepEqual(doc, map[string]any{"a": []any{"x", map[string]any{"b": 1.5}}}) {
		t.Fatalf("doc mutated: %v", doc)
	}
}
