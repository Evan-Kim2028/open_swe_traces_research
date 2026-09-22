package kubemanifest

import (
	"reflect"
	"strings"
	"testing"
)

// TestDetail01: LoadObjectsFrom skips sections whose only lines are empty
// or comments, and loads the rest.
func TestDetail01(t *testing.T) {
	doc := `# leading comment

---
kind: Pod
metadata:
  name: p1
---
# comment-only section

---
kind: Service
metadata:
  name: s1
`
	l, err := LoadObjectsFrom([]byte(doc))
	if err != nil {
		t.Fatalf("LoadObjectsFrom: %v", err)
	}
	if len(l) != 2 {
		t.Fatalf("got %d objects, want 2", len(l))
	}
	if l[0].Kind() != "Pod" || l[0].GetName() != "p1" {
		t.Fatalf("object 0 = %v %v", l[0].Kind(), l[0].GetName())
	}
	if l[1].Kind() != "Service" || l[1].GetName() != "s1" {
		t.Fatalf("object 1 = %v %v", l[1].Kind(), l[1].GetName())
	}
}

// TestDetail02: ObjectList.ToYAML emits each non-empty object separated by a
// document marker, and round-trips through LoadObjectsFrom.
func TestDetail02(t *testing.T) {
	l := ObjectList{
		NewObject(map[string]interface{}{"kind": "Pod", "metadata": map[string]interface{}{"name": "a"}}),
		NewObject(map[string]interface{}{}), // empty — dropped
		NewObject(map[string]interface{}{"kind": "Service", "metadata": map[string]interface{}{"name": "b"}}),
	}
	out, err := l.ToYAML()
	if err != nil {
		t.Fatalf("ToYAML: %v", err)
	}
	s := string(out)
	if !strings.Contains(s, "---") {
		t.Fatalf("no document separator in %q", s)
	}
	if !strings.Contains(s, "a") || !strings.Contains(s, "b") {
		t.Fatalf("missing objects in %q", s)
	}
	back, err := LoadObjectsFrom(out)
	if err != nil {
		t.Fatalf("round-trip: %v", err)
	}
	if len(back) != 2 {
		t.Fatalf("round-trip produced %d objects, want 2 (empty dropped)", len(back))
	}
}

// TestDetail03: getters return "" for missing or wrong-typed fields;
// name/namespace come from metadata.
func TestDetail03(t *testing.T) {
	o := NewObject(map[string]interface{}{
		"kind":       "Pod",
		"apiVersion": "v1",
		"metadata":   map[string]interface{}{"name": "n", "namespace": "ns"},
	})
	if o.Kind() != "Pod" || o.APIVersion() != "v1" || o.GetName() != "n" || o.GetNamespace() != "ns" {
		t.Fatalf("getters = %q %q %q %q", o.Kind(), o.APIVersion(), o.GetName(), o.GetNamespace())
	}

	missing := NewObject(map[string]interface{}{})
	if missing.Kind() != "" || missing.APIVersion() != "" || missing.GetName() != "" || missing.GetNamespace() != "" {
		t.Fatal("missing fields should be \"\"")
	}
	wrongType := NewObject(map[string]interface{}{
		"kind":     5,
		"metadata": map[string]interface{}{"name": 42, "namespace": true},
	})
	if wrongType.Kind() != "" || wrongType.GetName() != "" || wrongType.GetNamespace() != "" {
		t.Fatal("wrong-typed fields should be \"\"")
	}
}

// TestDetail04: Reparse navigates intermediate fields as maps and
// re-marshals into the target; missing/non-map intermediates error naming
// the offending field.
func TestDetail04(t *testing.T) {
	type inner struct {
		Field string `json:"field" yaml:"field"`
	}
	o := NewObject(map[string]interface{}{
		"spec":   map[string]interface{}{"sub": map[string]interface{}{"field": "v"}},
		"scalar": "notamap",
	})
	var in inner
	if err := o.Reparse(&in, "spec", "sub"); err != nil {
		t.Fatalf("Reparse: %v", err)
	}
	if in.Field != "v" {
		t.Fatalf("Reparse result = %+v", in)
	}

	if err := o.Reparse(&in, "missing", "sub"); err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("missing field error = %v", err)
	}
	if err := o.Reparse(&in, "scalar", "sub"); err == nil || !strings.Contains(err.Error(), "scalar") {
		t.Fatalf("non-map intermediate error = %v", err)
	}
}

// TestDetail05: Object.Set requires intermediate path segments to already
// exist as maps — it does not create them; the leaf is a yaml-round-tripped
// map copy.
func TestDetail05(t *testing.T) {
	o := NewObject(map[string]interface{}{
		"spec": map[string]interface{}{"a": "b"},
	})
	if err := o.Set(map[string]interface{}{"k": "v"}, "spec", "nk"); err != nil {
		t.Fatalf("Set into existing path: %v", err)
	}
	spec := o.data["spec"].(map[string]interface{})
	nk, ok := spec["nk"].(map[string]interface{})
	if !ok || nk["k"] != "v" {
		t.Fatalf("set value = %v", spec["nk"])
	}
	// missing intermediate -> error, nothing created
	if err := o.Set(map[string]interface{}{"k": "v"}, "absent", "k"); err == nil {
		t.Fatal("Set created a missing intermediate")
	}
	if _, exists := o.data["absent"]; exists {
		t.Fatal("Set created the intermediate map")
	}
}

type bbStringVisitor struct {
	visitorBase
	paths [][]string
	vals  []string
}

func (v *bbStringVisitor) VisitString(path []string, s string, mutator func(string)) error {
	v.paths = append(v.paths, append([]string(nil), path...))
	v.vals = append(v.vals, s)
	mutator(s + "!")
	return nil
}

// TestDetail06: visit mutates in place; path elements are field names plus
// "[i]" for slice indexes; []string is skipped silently; other concrete
// types error.
func TestDetail06(t *testing.T) {
	v := &bbStringVisitor{}
	o := NewObject(map[string]interface{}{
		"metadata": map[string]interface{}{"name": "x"},
		"items":    []interface{}{map[string]interface{}{"v": "s"}, map[string]interface{}{"v": "t"}},
		"strs":     []string{"skip", "me"},
	})
	if err := o.accept(v); err != nil {
		t.Fatalf("accept: %v", err)
	}
	data := o.data
	if data["metadata"].(map[string]interface{})["name"] != "x!" {
		t.Fatalf("nested string not mutated: %v", data["metadata"])
	}
	if data["items"].([]interface{})[0].(map[string]interface{})["v"] != "s!" {
		t.Fatalf("slice element not mutated: %v", data["items"])
	}
	if !reflect.DeepEqual(data["strs"], []string{"skip", "me"}) {
		t.Fatalf("[]string field was visited/mutated: %v", data["strs"])
	}

	var sawName, sawElem bool
	for _, p := range v.paths {
		if reflect.DeepEqual(p, []string{"metadata", "name"}) {
			sawName = true
		}
		if len(p) == 3 && p[0] == "items" && strings.HasPrefix(p[1], "[") && strings.HasSuffix(p[1], "]") && p[2] == "v" {
			sawElem = true
		}
	}
	if !sawName || !sawElem {
		t.Fatalf("paths = %v", v.paths)
	}

	// unhandled concrete type errors
	bad := NewObject(map[string]interface{}{"n": 42})
	if err := bad.accept(&bbStringVisitor{}); err == nil {
		t.Fatal("int field did not error")
	}
}

// TestDetail07: visitorBase callbacks are no-ops — a visitor embedding it
// and overriding nothing walks without error, and one overriding only
// VisitString still sees nested strings.
func TestDetail07(t *testing.T) {
	o := NewObject(map[string]interface{}{
		"nested": map[string]interface{}{"s": "x", "b": true, "f": 1.5},
	})
	if err := o.accept(&visitorBase{}); err != nil {
		t.Fatalf("visitorBase accept: %v", err)
	}
	v := &bbStringVisitor{}
	if err := o.accept(v); err != nil {
		t.Fatalf("accept: %v", err)
	}
	found := false
	for _, p := range v.paths {
		if reflect.DeepEqual(p, []string{"nested", "s"}) {
			found = true
		}
	}
	if !found {
		t.Fatalf("nested string not visited through embedded base: %v", v.paths)
	}
}

// TestDetail08: top-level data cannot be replaced. The root is always a map
// (NewObject takes a map) and map visits carry no mutator, so root
// replacement is unreachable through accept — the fatal guard exists only
// for scalar roots. Asserted shape: the root mutator is never invoked for a
// map root, and the mutator plumbing works for a scalar root (the path the
// fatal guard protects).
func TestDetail08(t *testing.T) {
	// map root: the root mutator is never invoked
	rootCalled := false
	if err := visit(&bbStringVisitor{}, map[string]interface{}{"k": "v"}, nil, func(interface{}) { rootCalled = true }); err != nil {
		t.Fatalf("visit on map root: %v", err)
	}
	if rootCalled {
		t.Fatal("root mutator invoked for a map root — root replacement attempted")
	}

	// scalar root: mutator plumbing reaches the top-level mutator
	var replaced interface{}
	v := &bbStringVisitor{}
	if err := visit(v, "root-scalar", nil, func(nv interface{}) { replaced = nv }); err != nil {
		t.Fatalf("visit on scalar root: %v", err)
	}
	if replaced != "root-scalar!" {
		t.Fatalf("scalar root replacement = %v", replaced)
	}
}
