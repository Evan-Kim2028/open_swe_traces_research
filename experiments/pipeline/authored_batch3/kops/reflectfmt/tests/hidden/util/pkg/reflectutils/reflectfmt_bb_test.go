// Package reflectutils_test is the hidden black-box suite for reflectfmt.
// One TestDetailNN per DETAILS.md commitment. Exported API only.
package reflectutils_test

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	ru "example.internal/clustkit/util/pkg/reflectutils"
)

type visit struct {
	path      string
	fieldSet  bool
	isNilLike bool
}

func collect(t *testing.T, v interface{}, opts *ru.ReflectOptions, hook func(path string) error) []visit {
	t.Helper()
	if opts == nil {
		opts = &ru.ReflectOptions{}
	}
	var visits []visit
	err := ru.ReflectRecursive(reflect.ValueOf(v),
		func(path *ru.FieldPath, field *reflect.StructField, val reflect.Value) error {
			vis := visit{path: path.String(), fieldSet: field != nil}
			if val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
				vis.isNilLike = val.IsNil()
			}
			visits = append(visits, vis)
			if hook != nil {
				return hook(path.String())
			}
			return nil
		}, opts)
	if err != nil {
		t.Fatalf("ReflectRecursive: %v", err)
	}
	return visits
}

func paths(visits []visit) []string {
	var out []string
	for _, v := range visits {
		out = append(out, v.path)
	}
	return out
}

// Detail 1 (Inferable: doc): visitor invoked on the root value first, with a
// nil struct-field and empty path, before any descent.
func TestDetail01(t *testing.T) {
	type S struct{ A int }
	visits := collect(t, &S{A: 1}, nil, nil)
	if len(visits) < 2 {
		t.Fatalf("too few visits: %v", paths(visits))
	}
	if visits[0].path != "" || visits[0].fieldSet {
		t.Fatalf("first visit path=%q fieldSet=%v, want root (empty path, nil field)", visits[0].path, visits[0].fieldSet)
	}
	// the field visit happens after the root visit (descent order)
	foundA := -1
	for i, v := range visits {
		if v.path == "A" {
			foundA = i
		}
	}
	if foundA <= 0 {
		t.Fatalf("field A not visited after root: %v", paths(visits))
	}
}

// Detail 2 (Inferable: doc): SkipReflection prunes the subtree but siblings
// continue; any other error aborts the walk.
func TestDetail02(t *testing.T) {
	type Inner struct{ X int }
	type S struct {
		A Inner
		B int
	}
	visits := collect(t, &S{A: Inner{X: 1}, B: 2}, nil, func(p string) error {
		if p == "A" {
			return ru.SkipReflection
		}
		return nil
	})
	for _, p := range paths(visits) {
		if p == "A.X" {
			t.Fatalf("SkipReflection did not prune subtree: %v", paths(visits))
		}
	}
	found := false
	for _, p := range paths(visits) {
		if p == "B" {
			found = true
		}
	}
	if !found {
		t.Fatalf("sibling B not visited after SkipReflection: %v", paths(visits))
	}

	myErr := errors.New("boom")
	var visits2 []visit
	err := ru.ReflectRecursive(reflect.ValueOf(&S{}),
		func(path *ru.FieldPath, field *reflect.StructField, v reflect.Value) error {
			visits2 = append(visits2, visit{path: path.String()})
			if path.String() == "A" {
				return myErr
			}
			return nil
		}, &ru.ReflectOptions{})
	if err == nil {
		t.Fatal("non-skip error did not abort")
	}
	for _, v := range visits2 {
		if v.path == "B" {
			t.Fatalf("walk continued after error: %v", paths(visits2))
		}
	}
}

// Detail 3 (Inferable: partially): unexported struct fields are skipped
// entirely — never visited.
func TestDetail03(t *testing.T) {
	type S struct {
		Exported   int
		unexported int
	}
	visits := collect(t, &S{Exported: 1, unexported: 2}, nil, nil)
	for _, p := range paths(visits) {
		if p == "unexported" {
			t.Fatalf("unexported field visited: %v", paths(visits))
		}
	}
	found := false
	for _, p := range paths(visits) {
		if p == "Exported" {
			found = true
		}
	}
	if !found {
		t.Fatalf("exported field not visited: %v", paths(visits))
	}
}

// Detail 4 (Inferable: no): with JSONNames a field reports its json tag name
// — including the literal "-" — falling back to the Go name when the tag is
// absent or empty. Asserted shape: the reported path element is the tag
// value, not the Go name.
func TestDetail04(t *testing.T) {
	type S struct {
		Tagged   int `json:"tagged_name"`
		Dash     int `json:"-"`
		Empty    int `json:""`
		Plain    int
	}
	visits := collect(t, &S{Tagged: 1, Dash: 2, Empty: 3, Plain: 4}, &ru.ReflectOptions{JSONNames: true}, nil)
	got := map[string]bool{}
	for _, p := range paths(visits) {
		got[p] = true
	}
	if !got["tagged_name"] {
		t.Fatalf("json tag name not reported: %v", paths(visits))
	}
	if got["Tagged"] {
		t.Fatalf("go name used despite tag: %v", paths(visits))
	}
	// `json:"-"` is still visited under some tag-derived name (never the go name)
	if got["Dash"] {
		t.Fatalf("go name used despite json:\"-\" tag: %v", paths(visits))
	}
	// empty tag falls back to the go name
	if !got["Empty"] {
		t.Fatalf("empty-tag field not visited under go name: %v", paths(visits))
	}
	if !got["Plain"] {
		t.Fatalf("untagged field not visited under go name: %v", paths(visits))
	}
}

// Detail 5 (Inferable: partially): map entries are visited under a MapKey
// element rendering [key]; slices/arrays use ArrayIndex rendering [n].
// Asserted via the rendered path strings.
func TestDetail05(t *testing.T) {
	type S struct {
		M map[string]int
		L []int
	}
	visits := collect(t, &S{M: map[string]int{"k1": 7}, L: []int{8, 9}}, nil, nil)
	got := map[string]bool{}
	for _, p := range paths(visits) {
		got[p] = true
	}
	if !got["M[k1]"] {
		t.Fatalf("map key not rendered as [key]: %v", paths(visits))
	}
	if !got["L[0]"] || !got["L[1]"] {
		t.Fatalf("slice indices not rendered as [n]: %v", paths(visits))
	}
}

// Detail 6 (Inferable: no): a nil pointer or interface is visited once but
// not descended. Asserted shape: the nil appears as a visit and produces no
// children.
func TestDetail06(t *testing.T) {
	type S struct {
		P *int
		I interface{}
	}
	visits := collect(t, &S{}, nil, nil)
	sawP, sawI := false, false
	for _, v := range visits {
		if v.path == "P" {
			sawP = true
			if !v.isNilLike {
				t.Fatal("nil pointer not seen as nil")
			}
		}
		if v.path == "I" {
			sawI = true
			if !v.isNilLike {
				t.Fatal("nil interface not seen as nil")
			}
		}
		if strings.HasPrefix(v.path, "P.") || strings.HasPrefix(v.path, "I.") {
			t.Fatalf("descended into nil: %v", paths(visits))
		}
	}
	if !sawP || !sawI {
		t.Fatalf("nil members not visited: %v", paths(visits))
	}
}

// Detail 7 (Inferable: doc): DeprecatedDoubleVisit calls the visitor once per
// field with the *reflect.StructField set, before descending into the value.
func TestDetail07(t *testing.T) {
	type S struct{ A int }
	visits := collect(t, &S{A: 1}, &ru.ReflectOptions{DeprecatedDoubleVisit: true}, nil)
	withField, withoutField := 0, 0
	for _, v := range visits {
		if v.path == "A" {
			if v.fieldSet {
				withField++
			} else {
				withoutField++
			}
		}
	}
	if withField == 0 || withoutField == 0 {
		t.Fatalf("double visit not observed (withField=%d withoutField=%d): %v", withField, withoutField, paths(visits))
	}
}

// Detail 8 (Inferable: doc): IsPrimitiveValue is true for bool/int/uint/
// float/complex kinds; false for string, slice, array, ptr, interface, chan,
// func, map, struct.
func TestDetail08(t *testing.T) {
	type row struct {
		v    interface{}
		want bool
	}
	rows := []row{
		{true, true}, {int(0), true}, {int8(0), true}, {uint(0), true},
		{uint64(0), true}, {float32(0), true}, {float64(0), true},
		{complex64(0), true}, {complex128(0), true},
		{"s", false}, {[]int{}, false}, {[2]int{}, false},
		{(*int)(nil), false},
		{make(chan int), false},
		{func() {}, false}, {map[string]int{}, false},
		{struct{}{}, false},
	}
	for _, r := range rows {
		if got := ru.IsPrimitiveValue(reflect.ValueOf(r.v)); got != r.want {
			t.Fatalf("IsPrimitiveValue(%T) = %v, want %v", r.v, got, r.want)
		}
	}
	// interface kind is not primitive either
	var iface interface{} = 1
	if ru.IsPrimitiveValue(reflect.ValueOf(&iface).Elem()) {
		t.Fatal("interface kind reported primitive")
	}
}

// Detail 9 (Inferable: partially): FormatValue — nil and nil-pointer print a
// null-ish sentinel; ints/floats/bools print %v; strings print %q (quoted);
// fmt.Stringer uses String(); everything else %#v. Asserted: the committed
// renderings; nil and nil-pointer render identically (the choice itself is
// not pinned).
func TestDetail09(t *testing.T) {
	if ru.FormatValue(nil) != ru.FormatValue((*int)(nil)) {
		t.Fatalf("nil and nil-pointer render differently: %q vs %q",
			ru.FormatValue(nil), ru.FormatValue((*int)(nil)))
	}
	if ru.FormatValue(nil) == "" {
		t.Fatal("nil renders empty")
	}
	if got := ru.FormatValue(42); got != "42" {
		t.Fatalf("int: %q", got)
	}
	if got := ru.FormatValue(1.5); got != "1.5" {
		t.Fatalf("float: %q", got)
	}
	if got := ru.FormatValue(true); got != "true" {
		t.Fatalf("bool: %q", got)
	}
	if got := ru.FormatValue("hi"); got != `"hi"` {
		t.Fatalf("string not %q-quoted: %q", "%q", got)
	}
	if got := ru.FormatValue(fmt.Stringer(stringerImpl{})); got != "STR" {
		t.Fatalf("Stringer: %q", got)
	}
	if got := ru.FormatValue(map[string]int{"a": 1}); !strings.Contains(got, "map") {
		t.Fatalf("fallback not verbose: %q", got)
	}
}

type stringerImpl struct{}

func (stringerImpl) String() string { return "STR" }

// Detail 10 (Inferable: yes): BuildTypeName composes */[]/map[k]v over
// t.Name(); unknown kinds fall back to t.Name().
func TestDetail10(t *testing.T) {
	cases := map[reflect.Type]string{
		reflect.TypeOf(0):                    "int",
		reflect.TypeOf((*int)(nil)):          "*int",
		reflect.TypeOf([]string(nil)):        "[]string",
		reflect.TypeOf(map[string]int(nil)):  "map[string]int",
		reflect.TypeOf((**int)(nil)):         "**int",
		reflect.TypeOf([]*map[string]int{}):  "[]*map[string]int",
	}
	for typ, want := range cases {
		if got := ru.BuildTypeName(typ); got != want {
			t.Fatalf("BuildTypeName(%v) = %q, want %q", typ, got, want)
		}
	}
	// unknown kinds (chan) must not panic — some string falls out
	_ = ru.BuildTypeName(reflect.TypeOf(make(chan int)))
}

// Detail 11 (Inferable: partially): JSONMergeStruct merges only JSON-visible
// fields (JSON marshal+unmarshal); a marshal failure is fatal (process exit),
// not an error return.
func TestDetail11(t *testing.T) {
	if os.Getenv("BB_FATAL_CHILD") == "1" {
		type S struct{ A int }
		// func fields cannot marshal -> klog.Fatalf path
		ru.JSONMergeStruct(&S{}, map[string]interface{}{"a": func() {}})
		return
	}
	type S struct {
		A        int `json:"a"`
		B        int `json:"-"`
		Exported int `json:"exported"`
	}
	dest := &S{A: 1, B: 5, Exported: 9}
	ru.JSONMergeStruct(dest, &S{A: 2, B: 7, Exported: 3})
	if dest.A != 2 || dest.Exported != 3 {
		t.Fatalf("visible fields not merged: %+v", dest)
	}
	if dest.B != 5 {
		t.Fatalf("json:- field was merged: %+v", dest)
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestDetail11")
	cmd.Env = append(os.Environ(), "BB_FATAL_CHILD=1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("unmarshalable src did not abort process: %s", out)
	}
}

// Detail 12 (Inferable: no): IsMethodNotFound type-asserts
// *MethodNotFoundError — a wrapped error is NOT detected.
func TestDetail12(t *testing.T) {
	mnf := &ru.MethodNotFoundError{Name: "m"}
	if !ru.IsMethodNotFound(mnf) {
		t.Fatal("bare MethodNotFoundError not detected")
	}
	if ru.IsMethodNotFound(fmt.Errorf("wrap: %w", mnf)) {
		t.Fatal("wrapped MethodNotFoundError detected — should use plain assertion")
	}
	if ru.IsMethodNotFound(errors.New("x")) {
		t.Fatal("unrelated error detected")
	}
	if ru.IsMethodNotFound(nil) {
		t.Fatal("nil detected")
	}
}
