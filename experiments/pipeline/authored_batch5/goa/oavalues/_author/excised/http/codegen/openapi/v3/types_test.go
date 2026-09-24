// This file verifies OpenAPI v3 schema construction and stable example
// generation for primitive, collection, object, and transport body types.
package openapiv3

import (
	_ "encoding/json"
	"hash/fnv"
	"strings"
	"testing"

	_ "example.internal/apikit/v3/codegen"
	_ "example.internal/apikit/v3/dsl"
	"example.internal/apikit/v3/expr"
	"example.internal/apikit/v3/http/codegen/openapi"
	_ "example.internal/apikit/v3/http/codegen/openapi/v3/testdata/dsls"
)

// describes a type for comparison in tests.
type typ struct {
	Type                 string
	Format               string
	Props                []attr
	SkipProps            bool
	AdditionalProperties *additionalPropsType // nil means no additionalProperties check
}

// additionalPropsType describes additionalProperties for testing
type additionalPropsType struct {
	Type  string               // "string", "array", "object", "" (for reference)
	Items *additionalPropsType // for array items
	Ref   string               // for references like "#/components/schemas/MapData"
}

type attr struct {
	Name string
	Val  typ
}

// types mapped by response code.
type rt map[int]typ

// helpers
var (
	tempty  typ
	tstring = typ{Type: "string"}
	tuuid   = typ{Type: "string", Format: "uuid"}
	tbinary = typ{Type: "string", Format: "binary"}
	tint    = typ{Type: "integer"}
	tarray  = typ{Type: "array"}
)

func tobj(attrs ...any) typ {
	res := typ{Type: "object"}
	if len(attrs) == 0 {
		res.SkipProps = true
	}
	for i := 0; i < len(attrs); i += 2 {
		res.Props = append(res.Props, attr{Name: attrs[i].(string), Val: attrs[i+1].(typ)})
	}
	return res
}

func tmap() typ {
	return typ{Type: "object", Props: []attr{{Name: "map", Val: typ{Type: "object"}}}}
}

func (tt typ) Prop(n string) (typ, bool) {
	for _, att := range tt.Props {
		if att.Name == n {
			return att.Val, true
		}
	}
	return tempty, false
}


func matchesSchema(t *testing.T, ctx string, s *openapi.Schema, types map[string]*openapi.Schema, tt typ) {
	matchesSchemaWithPrefix(t, ctx, s, types, tt, "")
}
func matchesSchemaWithPrefix(t *testing.T, ctx string, s *openapi.Schema, types map[string]*openapi.Schema, tt typ, prefix string) {
	if s == nil {
		if tt.Type != "" {
			t.Errorf("%s: %sgot type Empty, expected %q", ctx, prefix, tt.Type)
		}
		return
	}
	if s.Ref != "" {
		var ok bool
		s, ok = types[nameFromRef(s.Ref)]
		if !ok {
			t.Errorf("could not find type for ref %q", s.Ref)
			return
		}
	}
	if tt.Type != string(s.Type) {
		t.Errorf("%s: %sgot type %q, expected %q", ctx, prefix, s.Type, tt.Type)
	}
	if tt.Format != "" {
		if s.Format != tt.Format {
			t.Errorf("%s: %sgot format %q, expected %q", ctx, prefix, s.Format, tt.Format)
		}
	}
	if tt.Type == "object" {
		if tt.SkipProps {
			return
		}
		for n, v := range s.Properties {
			p, ok := tt.Prop(n)
			if !ok {
				t.Errorf("%s: %sgot unexpected field %q", ctx, prefix, n)
				continue
			}
			matchesSchemaWithPrefix(t, ctx, v, types, p, n+": ")
		}

		// Check additionalProperties
		if tt.AdditionalProperties != nil {
			validateAdditionalProperties(t, ctx, s.AdditionalProperties, types, tt.AdditionalProperties, prefix)
		}
	}
}


func validateAdditionalProperties(t *testing.T, ctx string, addProps any, types map[string]*openapi.Schema, expected *additionalPropsType, prefix string) {
	if addProps == nil {
		t.Errorf("%s: %sexpected additionalProperties to be set", ctx, prefix)
		return
	}

	// Check if additionalProperties is a schema
	schema, ok := addProps.(*openapi.Schema)
	if !ok {
		t.Errorf("%s: %sexpected additionalProperties to be schema, got %T", ctx, prefix, addProps)
		return
	}

	validateAdditionalPropsSchema(t, ctx, schema, types, expected, prefix+"additionalProperties: ")
}

func validateAdditionalPropsSchema(t *testing.T, ctx string, schema *openapi.Schema, types map[string]*openapi.Schema, expected *additionalPropsType, prefix string) {
	// Handle reference case
	if expected.Ref != "" {
		if schema.Ref == "" {
			t.Errorf("%s: %sexpected reference to %s, but got inline schema", ctx, prefix, expected.Ref)
			return
		}
		if schema.Ref != expected.Ref {
			t.Errorf("%s: %sexpected reference %s, got %s", ctx, prefix, expected.Ref, schema.Ref)
		}
		return
	}

	// Resolve reference if present
	if schema.Ref != "" {
		typeName := nameFromRef(schema.Ref)
		resolvedSchema, ok := types[typeName]
		if !ok {
			t.Errorf("%s: %scould not resolve reference %s", ctx, prefix, schema.Ref)
			return
		}
		schema = resolvedSchema
	}

	// Check type
	if string(schema.Type) != expected.Type {
		t.Errorf("%s: %sexpected type %s, got %s", ctx, prefix, expected.Type, schema.Type)
	}

	// Check array items if expected
	if expected.Items != nil {
		if schema.Items == nil {
			t.Errorf("%s: %sexpected array items to be set", ctx, prefix)
		} else {
			validateAdditionalPropsSchema(t, ctx, schema.Items, types, expected.Items, prefix+"items: ")
		}
	}
}



func TestHashAttribute(t *testing.T) {
	type (
		testAttr struct {
			name string
			att  *expr.AttributeExpr
		}

		hashBehavior int

		testGroup struct {
			name     string
			attrs    []testAttr
			behavior hashBehavior
		}
	)

	const (
		uniqueHashes hashBehavior = iota
		identicalHashes
	)

	var (
		metaNotGenerate = expr.MetaExpr{"openapi:generate": []string{"false"}}
		metaEmpty       = expr.MetaExpr{}
	)

	cases := []testGroup{
		{
			name:     "Primitive types",
			behavior: uniqueHashes,
			attrs: []testAttr{
				{name: "bool", att: &expr.AttributeExpr{Type: expr.Boolean}},
				{name: "int", att: &expr.AttributeExpr{Type: expr.Int}},
				{name: "int32", att: &expr.AttributeExpr{Type: expr.Int32}},
				{name: "int64", att: &expr.AttributeExpr{Type: expr.Int64}},
				{name: "uint", att: &expr.AttributeExpr{Type: expr.UInt}},
				{name: "uint32", att: &expr.AttributeExpr{Type: expr.UInt32}},
				{name: "uint64", att: &expr.AttributeExpr{Type: expr.UInt64}},
				{name: "float32", att: &expr.AttributeExpr{Type: expr.Float32}},
				{name: "float64", att: &expr.AttributeExpr{Type: expr.Float64}},
				{name: "string", att: &expr.AttributeExpr{Type: expr.String}},
				{name: "bytes", att: &expr.AttributeExpr{Type: expr.Bytes}},
				{name: "any", att: &expr.AttributeExpr{Type: expr.Any}},
			},
		}, {
			name:     "Collection types",
			behavior: uniqueHashes,
			attrs: []testAttr{
				{name: "array-bool", att: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.Boolean}}}},
				{name: "array-int", att: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.Int}}}},
				{name: "map-str-int", att: &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.Int}}}},
				{name: "map-str-str", att: &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.String}}}},
			},
		}, {
			name:     "Objects with validation rules",
			behavior: uniqueHashes,
			attrs: []testAttr{
				{name: "no-validation", att: newObj("foo", false)},
				{name: "required-validation", att: newObj("foo", true)},
				{name: "pattern-validation", att: &expr.AttributeExpr{
					Type: expr.String,
					Validation: &expr.ValidationExpr{
						Pattern: "^[a-z]+$",
					},
				}},
				{name: "enum-validation", att: &expr.AttributeExpr{
					Type: expr.String,
					Validation: &expr.ValidationExpr{
						Values: []any{"foo", "bar"},
					},
				}},
			},
		}, {
			name:     "Result types with different views",
			behavior: uniqueHashes,
			attrs: []testAttr{
				{name: "no-view", att: newRT("id", newObj("foo", true))},
				{name: "default-view", att: newRTWithView("id", newObj("foo", true), "default")},
				{name: "tiny-view", att: newRTWithView("id", newObj("foo", true), "tiny")},
			},
		}, {
			name:     "Objects with openapi:generate:false metadata",
			behavior: identicalHashes,
			attrs: []testAttr{
				{name: "obj-with-skipped-field", att: newObj2Meta("foo", "bar", expr.String, expr.String, metaEmpty, metaNotGenerate)},
				{name: "obj-without-skipped-field", att: newObj("foo", false)},
			},
		}, {
			name:     "Complex map types",
			behavior: uniqueHashes,
			attrs: []testAttr{
				{name: "map-int-array", att: &expr.AttributeExpr{Type: &expr.Map{
					KeyType:  &expr.AttributeExpr{Type: expr.Int},
					ElemType: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}},
				}}},
				{name: "map-array-int", att: &expr.AttributeExpr{Type: &expr.Map{
					KeyType:  &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}},
					ElemType: &expr.AttributeExpr{Type: expr.Int},
				}}},
			},
		}, {
			name:     "Nested user types",
			behavior: uniqueHashes,
			attrs: []testAttr{
				{name: "single-nest", att: newUserType("foo", newObj("bar", false))},
				{name: "double-nest", att: newUserType("foo", newUserType("bar", newObj("baz", false)))},
			},
		}, {
			name:     "Recursive types",
			behavior: identicalHashes,
			attrs: []testAttr{
				{name: "recursive-1", att: newRecursiveType("foo")},
				{name: "recursive-2", att: newRecursiveType("foo")},
			},
		},
	}

	h := fnv.New64()
	sf := newSchemafier(expr.NewExampleGenerator(expr.NewFakerRandomizerFactory("test")), openapi.Values{})

	for _, group := range cases {
		t.Run(group.name, func(t *testing.T) {
			seen := make(map[uint64][]string)

			// Collect all hashes in this group
			for _, attr := range group.attrs {
				hash := sf.hashAttribute(attr.att, h)
				seen[hash] = append(seen[hash], attr.name)
			}

			switch group.behavior {
			case uniqueHashes:
				// Verify all hashes are different
				for hash, names := range seen {
					if len(names) > 1 {
						t.Errorf("expected unique hashes but got collision between %v (hash: %d)",
							names, hash)
					}
				}
			case identicalHashes:
				// Verify all hashes are the same
				if len(seen) > 1 {
					t.Errorf("expected identical hashes but got different ones: %v", seen)
				}
			}
		})
	}
}

func newObj(n string, req bool) *expr.AttributeExpr {
	attr := &expr.AttributeExpr{
		Type:       &expr.Object{{Name: n, Attribute: &expr.AttributeExpr{Type: expr.String}}},
		Validation: &expr.ValidationExpr{},
	}
	if req {
		attr.Validation.Required = []string{n}
	}
	return attr
}

func newObj2Meta(n, o string, t, u expr.DataType, l, m expr.MetaExpr, reqs ...string) *expr.AttributeExpr {
	attr := &expr.AttributeExpr{
		Type: &expr.Object{
			{Name: n, Attribute: &expr.AttributeExpr{Type: t, Meta: l}},
			{Name: o, Attribute: &expr.AttributeExpr{Type: u, Meta: m}},
		},
		Validation: &expr.ValidationExpr{},
	}
	attr.Validation.Required = append(attr.Validation.Required, reqs...)
	return attr
}

func newRT(id string, att *expr.AttributeExpr) *expr.AttributeExpr {
	return &expr.AttributeExpr{
		Type: &expr.ResultTypeExpr{
			Identifier: id,
			UserTypeExpr: &expr.UserTypeExpr{
				AttributeExpr: att,
			},
		},
	}
}

// Helper function for result types with views
func newRTWithView(id string, att *expr.AttributeExpr, view string) *expr.AttributeExpr {
	rt := newRT(id, att)
	rt.Type.(*expr.ResultTypeExpr).Meta = expr.MetaExpr{
		expr.ViewMetaKey: []string{view},
	}
	return rt
}

// Helper function for user types
func newUserType(name string, att *expr.AttributeExpr) *expr.AttributeExpr {
	return &expr.AttributeExpr{
		Type: &expr.UserTypeExpr{
			AttributeExpr: att,
			TypeName:      name,
		},
	}
}

// Helper function for recursive types
func newRecursiveType(name string) *expr.AttributeExpr {
	// Create a user type that references itself
	ut := &expr.UserTypeExpr{
		TypeName: name,
	}
	att := &expr.AttributeExpr{
		Type: &expr.Object{
			&expr.NamedAttributeExpr{
				Name: "self",
				Attribute: &expr.AttributeExpr{
					Type: ut,
				},
			},
		},
	}
	ut.AttributeExpr = att
	return &expr.AttributeExpr{Type: ut}
}

// nameFromRef does the reverse of toRef: it returns the type name from its
// JSON Schema reference.
func nameFromRef(ref string) string {
	elems := strings.Split(ref, "/")
	return elems[len(elems)-1]
}
