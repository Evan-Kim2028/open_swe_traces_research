// This file verifies expression type conversion, compatibility, and example
// behavior, including the tagged representation produced for union values.
package expr

import "testing"









func TestPrimitiveIsCompatible(t *testing.T) {
	var (
		b    = bool(true)
		i    = int(1)
		i8   = int8(2)
		i16  = int16(3)
		i32  = int32(4)
		ui   = uint(5)
		ui8  = uint8(6)
		ui16 = uint16(7)
		ui32 = uint32(8)
		i64  = int64(9)
		ui64 = uint64(10)
		f32  = float32(10.1)
		f64  = float64(20.2)
		s    = string("string")
		bs   = []byte("bytes")
		ss   = []string{"foo", "bar"}
		is   = []int{1, 2}
	)
	cases := map[string]struct {
		p        Primitive
		values   []any
		expected bool
	}{
		"any": {
			p:        Any,
			values:   []any{b},
			expected: true,
		},
		"boolean compatible": {
			p:        Boolean,
			values:   []any{b},
			expected: true,
		},
		"boolean not compatible": {
			p:        Boolean,
			values:   []any{i, i8, i16, i32, ui, ui8, ui16, ui32, i64, ui64, f32, f64, s, bs},
			expected: false,
		},
		"int compatible": {
			p:        Int,
			values:   []any{i, i8, i16, i32, ui, ui8, ui16, ui32},
			expected: true,
		},
		"int not compatible": {
			p:        Int,
			values:   []any{b, i64, ui64, f32, f64, s, bs},
			expected: false,
		},
		"int32 compatible": {
			p:        Int32,
			values:   []any{i, i8, i16, i32, ui, ui8, ui16, ui32},
			expected: true,
		},
		"int32 not compatible": {
			p:        Int32,
			values:   []any{b, i64, ui64, f32, f64, s, bs},
			expected: false,
		},
		"int64 compatible": {
			p:        Int64,
			values:   []any{i, i8, i16, i32, ui, ui8, ui16, ui32, i64, ui64},
			expected: true,
		},
		"int64 not compatible": {
			p:        Int64,
			values:   []any{b, f32, f64, s, bs},
			expected: false,
		},
		"uint compatible": {
			p:        UInt,
			values:   []any{i, i8, i16, i32, ui, ui8, ui16, ui32},
			expected: true,
		},
		"uint not compatible": {
			p:        UInt,
			values:   []any{b, i64, ui64, f32, f64, s, bs},
			expected: false,
		},
		"uint32 compatible": {
			p:        UInt32,
			values:   []any{i, i8, i16, i32, ui, ui8, ui16, ui32},
			expected: true,
		},
		"uint32 not compatible": {
			p:        UInt32,
			values:   []any{b, i64, ui64, f32, f64, s, bs},
			expected: false,
		},
		"uint64 compatible": {
			p:        UInt64,
			values:   []any{i, i8, i16, i32, ui, ui8, ui16, ui32, i64, ui64},
			expected: true,
		},
		"uint64 not compatible": {
			p:        UInt64,
			values:   []any{b, f32, f64, s, bs},
			expected: false,
		},
		"float32 compatible": {
			p:        Float32,
			values:   []any{i, i8, i16, i32, ui, ui8, ui16, ui32, i64, ui64, f32, f64},
			expected: true,
		},
		"float32 not compatible": {
			p:        Float32,
			values:   []any{b, s, bs},
			expected: false,
		},
		"float64 compatible": {
			p:        Float64,
			values:   []any{i, i8, i16, i32, ui, ui8, ui16, ui32, i64, ui64, f32, f64},
			expected: true,
		},
		"float64 not compatible": {
			p:        Float64,
			values:   []any{b, s, bs},
			expected: false,
		},
		"string compatible": {
			p:        String,
			values:   []any{s},
			expected: true,
		},
		"string not compatible": {
			p:        String,
			values:   []any{b, i, i8, i16, i32, ui, ui8, ui16, ui32, i64, ui64, f32, f64, bs},
			expected: false,
		},
		"bytes compatible": {
			p:        Bytes,
			values:   []any{s, bs},
			expected: true,
		},
		"bytes not compatible": {
			p:        Bytes,
			values:   []any{b, i, i8, i16, i32, ui, ui8, ui16, ui32, i64, ui64, f32, f64},
			expected: false,
		},
		"not supported types": {
			p:        Boolean,
			values:   []any{ss, is},
			expected: false,
		},
	}

	for k, tc := range cases {
		for _, value := range tc.values {
			if actual := tc.p.IsCompatible(value); tc.expected != actual {
				t.Errorf("%s: got %#v, expected %#v", k, actual, tc.expected)
			}
		}
	}
}

func TestArrayIsCompatible(t *testing.T) {
	var (
		b  = true
		i  = 1
		ia = [2]int{1, 2}
		is = []int{3, 4}
	)
	cases := map[string]struct {
		typ      DataType
		values   []any
		expected bool
	}{
		"compatible": {
			typ:      Int,
			values:   []any{ia, is},
			expected: true,
		},
		"not array and slice": {
			typ:      String,
			values:   []any{b, i},
			expected: false,
		},
		"array but not compatible": {
			typ:      String,
			values:   []any{ia},
			expected: false,
		},
		"slice but not compatible": {
			typ:      String,
			values:   []any{is},
			expected: false,
		},
	}

	for k, tc := range cases {
		array := Array{
			ElemType: &AttributeExpr{
				Type: tc.typ,
			},
		}
		for _, value := range tc.values {
			if actual := array.IsCompatible(value); tc.expected != actual {
				t.Errorf("%s: got %#v, expected %#v", k, actual, tc.expected)
			}
		}
	}
}

func TestObjectRename(t *testing.T) {
	cases := map[string]struct {
		old, new string
		expected []string
	}{
		"renamed": {
			old:      "foo",
			new:      "qux",
			expected: []string{"qux", "bar"},
		},
		"unmatched": {
			old:      "baz",
			new:      "qux",
			expected: []string{"foo", "bar"},
		},
	}

	for k, tc := range cases {
		object := &Object{
			&NamedAttributeExpr{
				Name: "foo",
				Attribute: &AttributeExpr{
					Type: String,
				},
			},
			&NamedAttributeExpr{
				Name: "bar",
				Attribute: &AttributeExpr{
					Type: String,
				},
			},
		}
		object.Rename(tc.old, tc.new)
		for _, s := range tc.expected {
			if att := object.Attribute(s); att == nil {
				t.Errorf("%s: %s not found", k, s)
			}
		}
	}
}

func TestObjectIsCompatible(t *testing.T) {
	var (
		b = true
		i = 1
		s = struct {
			Foo string
		}{
			Foo: "foo",
		}
		m = map[int]string{}
	)
	cases := map[string]struct {
		values   []any
		expected bool
	}{
		"compatible": {
			values:   []any{s, m},
			expected: true,
		},
		"not comatible": {
			values:   []any{b, i},
			expected: false,
		},
	}

	object := Object{}
	for k, tc := range cases {
		for _, value := range tc.values {
			if actual := object.IsCompatible(value); tc.expected != actual {
				t.Errorf("%s: got %#v, expected %#v", k, actual, tc.expected)
			}
		}
	}
}

func TestMapIsCompatible(t *testing.T) {
	var (
		b   = true
		i   = 1
		ism = map[int]string{
			1: "foo",
		}
		ssm = map[string]string{
			"bar": "bar",
		}
		iim = map[int]int{
			2: 2,
		}
	)
	cases := map[string]struct {
		values   []any
		expected bool
	}{
		"compatible": {
			values:   []any{ism},
			expected: true,
		},
		"not comatible": {
			values:   []any{b, i},
			expected: false,
		},
		"map but not comatible": {
			values:   []any{ssm, iim},
			expected: false,
		},
	}

	m := Map{
		KeyType: &AttributeExpr{
			Type: Int,
		},
		ElemType: &AttributeExpr{
			Type: String,
		},
	}
	for k, tc := range cases {
		for _, value := range tc.values {
			if actual := m.IsCompatible(value); tc.expected != actual {
				t.Errorf("%s: got %#v, expected %#v", k, actual, tc.expected)
			}
		}
	}
}

func TestUnionGetTypeKey(t *testing.T) {
	cases := map[string]struct {
		typeKey  string
		expected string
	}{
		"default": {
			typeKey:  "",
			expected: "type",
		},
		"custom": {
			typeKey:  "kind",
			expected: "kind",
		},
		"discriminator": {
			typeKey:  "discriminator",
			expected: "discriminator",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			union := &Union{
				TypeName: "TestUnion",
				TypeKey:  tc.typeKey,
			}
			if actual := union.GetTypeKey(); actual != tc.expected {
				t.Errorf("got %q, expected %q", actual, tc.expected)
			}
		})
	}
}

func TestUnionExampleAndCompatibilityUseTaggedEnvelope(t *testing.T) {
	union := &Union{
		TypeName: "Outcome",
		TypeKey:  "kind",
		ValueKey: "data",
		Values: []*NamedAttributeExpr{
			{Name: "text", Attribute: &AttributeExpr{Type: String}},
			{Name: "count", Attribute: &AttributeExpr{Type: Int}},
		},
	}

	example := union.Example(NewExampleGenerator(NewFakerRandomizerFactory("test")).At(
		MethodPayloadExampleIdentity(&MethodExpr{
			Name:    "union",
			Service: &ServiceExpr{Name: "test"},
		}),
	))
	envelope, ok := example.(map[string]any)
	if !ok {
		t.Fatalf("expected tagged envelope, got %T", example)
	}
	if !union.IsCompatible(envelope) {
		t.Fatalf("generated example is not compatible: %#v", envelope)
	}
	if union.IsCompatible(map[string]any{"kind": "text", "data": 3}) {
		t.Fatal("expected mismatched branch value to be incompatible")
	}
	if union.IsCompatible("plain branch value") {
		t.Fatal("expected untagged branch value to be incompatible")
	}
}

func TestUnionGetValueKey(t *testing.T) {
	cases := map[string]struct {
		valueKey string
		expected string
	}{
		"default": {
			valueKey: "",
			expected: "value",
		},
		"custom": {
			valueKey: "data",
			expected: "data",
		},
		"payload": {
			valueKey: "payload",
			expected: "payload",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			union := &Union{
				TypeName: "TestUnion",
				ValueKey: tc.valueKey,
			}
			if actual := union.GetValueKey(); actual != tc.expected {
				t.Errorf("got %q, expected %q", actual, tc.expected)
			}
		})
	}
}

func TestUnionDupPreservesCustomKeys(t *testing.T) {
	original := &Union{
		TypeName: "TestUnion",
		TypeKey:  "kind",
		ValueKey: "data",
		Values: []*NamedAttributeExpr{
			{
				Name: "String",
				Attribute: &AttributeExpr{
					Type: String,
				},
			},
		},
	}

	duplicated := Dup(original)

	dup, ok := duplicated.(*Union)
	if !ok {
		t.Fatalf("expected *Union, got %T", duplicated)
	}

	if dup.TypeKey != original.TypeKey {
		t.Errorf("TypeKey: got %q, expected %q", dup.TypeKey, original.TypeKey)
	}
	if dup.ValueKey != original.ValueKey {
		t.Errorf("ValueKey: got %q, expected %q", dup.ValueKey, original.ValueKey)
	}
	if dup.GetTypeKey() != original.GetTypeKey() {
		t.Errorf("GetTypeKey(): got %q, expected %q", dup.GetTypeKey(), original.GetTypeKey())
	}
	if dup.GetValueKey() != original.GetValueKey() {
		t.Errorf("GetValueKey(): got %q, expected %q", dup.GetValueKey(), original.GetValueKey())
	}
}

