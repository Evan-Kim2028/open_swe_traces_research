// This file verifies package-level name allocation and generated type identity.
package codegen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"example.internal/apikit/v3/expr"
)

type exactHasher string

// Hash returns the exact map identity supplied by the test.
func (h exactHasher) Hash() string {
	return string(h)
}

func TestNameScope_Freeze(t *testing.T) {
	scope := NewNameScope()
	existing := &expr.UserTypeExpr{
		AttributeExpr: &expr.AttributeExpr{Type: expr.String},
		TypeName:      "Existing",
		UID:           "existing",
	}
	require.Equal(t, "Existing", scope.GoTypeName(&expr.AttributeExpr{Type: existing}))

	scope.Freeze()
	scope.Freeze()
	require.Equal(t, "Existing", scope.GoTypeName(&expr.AttributeExpr{Type: existing}))
	require.Equal(t, "Next", scope.PeekUnique("Next"))
	require.Equal(t, "Next", scope.Name("Next"))
	require.Panics(t, func() {
		scope.Unique("Next")
	})
	require.Panics(t, func() {
		scope.HashedUnique(&expr.UserTypeExpr{
			AttributeExpr: &expr.AttributeExpr{Type: expr.String},
			TypeName:      "Next",
			UID:           "next",
		}, "Next")
	})
	require.Panics(t, func() {
		scope.GoTypeName(&expr.AttributeExpr{Type: &expr.UserTypeExpr{
			AttributeExpr: &expr.AttributeExpr{Type: expr.String},
			TypeName:      "Indirect",
			UID:           "indirect",
		}})
	})
}

func TestNameScope_Unique(t *testing.T) {
	sequence := []struct {
		Input    string
		Suffix   []string
		Expected string
	}{
		{Input: "a", Expected: "a"},
		{Input: "a", Expected: "a2"},
		{Input: "a", Expected: "a3"},
		{Input: "a", Expected: "a4"},
		{Input: "b", Expected: "b"},
		{Input: "c", Expected: "c"},
		{Input: "hel", Expected: "hel"},
		{Input: "hel", Suffix: []string{"lo"}, Expected: "hello"},
		{Input: "hello", Expected: "hello2"},
		{Input: "hello", Suffix: []string{"1"}, Expected: "hello1"},
		{Input: "hello", Suffix: []string{"1"}, Expected: "hello12"},
		{Input: "hello", Suffix: []string{"2"}, Expected: "hello22"},
		{Input: "hello", Suffix: []string{"2"}, Expected: "hello23"},
		{Input: "hello,world", Expected: "hello,world"},
		{Input: "hello,world1", Expected: "hello,world1"},
		{Input: "hello,world2", Expected: "hello,world2"},
		{Input: "hello", Suffix: []string{",world"}, Expected: "hello,world3"},
	}

	scope := NewNameScope()
	for i, v := range sequence {
		if got := scope.Unique(v.Input, v.Suffix...); v.Expected != got {
			t.Errorf("#%v, expected %v, got %v", i, v.Expected, got)
		}
	}
}

func TestNameScope_HashedUniqueUsesExactHash(t *testing.T) {
	scope := NewNameScope()
	require.Equal(t, "First", scope.HashedUnique(exactHasher("shared"), "First"))
	require.Equal(t, "First", scope.HashedUnique(exactHasher("shared"), "Ignored"))
	require.Equal(t, "First2", scope.HashedUnique(exactHasher("distinct"), "First"))
}

func TestNameScope_GoFullTypeName_UsesScopedNameWhenQualified(t *testing.T) {
	scope := NewNameScope()

	// Simulate the service generator reserving/using "Request" for a different
	// identifier before naming a user type that also wants to be "Request".
	scope.Unique("Request")

	ut := &expr.UserTypeExpr{
		AttributeExpr: &expr.AttributeExpr{Type: expr.String},
		TypeName:      "Request",
		UID:           "t",
	}
	att := &expr.AttributeExpr{Type: ut}

	if got, want := scope.GoTypeName(att), "Request2"; got != want {
		t.Fatalf("expected scoped type name %q, got %q", want, got)
	}
	if got, want := scope.GoFullTypeName(att, "svc"), "svc.Request2"; got != want {
		t.Fatalf("expected qualified scoped name %q, got %q", want, got)
	}

	fresh := NewNameScope()
	if got, want := fresh.GoFullTypeName(att, "svc"), "svc.Request"; got != want {
		t.Fatalf("expected qualified base name %q with fresh scope, got %q", want, got)
	}
}








// TestNameScopeForkPreservesBindingsAndAcceptsHelperNames verifies a frozen
// declaration scope can seed a separate mutable helper namespace.
func TestNameScopeForkPreservesBindingsAndAcceptsHelperNames(t *testing.T) {
	typeExpr := &expr.UserTypeExpr{
		AttributeExpr: &expr.AttributeExpr{Type: expr.String},
		TypeName:      "Value",
	}
	scope := NewNameScope()
	assert.Equal(t, "Value", scope.GoTypeName(&expr.AttributeExpr{Type: typeExpr}))
	scope.Freeze()

	fork := scope.Fork()
	assert.Equal(t, "Value", fork.GoTypeName(&expr.AttributeExpr{Type: typeExpr}))
	assert.Equal(t, "Value2", fork.Unique("Value"))
	assert.Equal(t, "helper", fork.Unique("helper"))
}






func TestNameScope_GoFullTypeName_UsesScopedRelocatedUserTypeNameWhenQualified(t *testing.T) {
	scope := NewNameScope()
	first := &expr.UserTypeExpr{
		AttributeExpr: &expr.AttributeExpr{
			Type: expr.String,
			Meta: expr.MetaExpr{"struct:pkg:path": {"types"}},
		},
		TypeName: "foo-bar",
		UID:      "first",
	}
	second := &expr.UserTypeExpr{
		AttributeExpr: &expr.AttributeExpr{
			Type: expr.String,
			Meta: expr.MetaExpr{"struct:pkg:path": {"types"}},
		},
		TypeName: "foo_bar",
		UID:      "second",
	}
	scope.GoTypeName(&expr.AttributeExpr{Type: first})
	secondAtt := &expr.AttributeExpr{Type: second}
	if got, want := scope.GoTypeName(secondAtt), "FooBar2"; got != want {
		t.Fatalf("GoTypeName() = %q, want %q", got, want)
	}
	if got, want := scope.GoFullTypeName(secondAtt, "types"), "types.FooBar2"; got != want {
		t.Errorf("GoFullTypeName() = %q, want %q", got, want)
	}
}

// TestNameScopeForkPreservesUserTypeBindings verifies that a child scope keeps
// the exact generated name chosen for an original user type and all its copies.
func TestNameScopeForkPreservesUserTypeBindings(t *testing.T) {
	scope := NewNameScope()
	original := &expr.UserTypeExpr{
		AttributeExpr: &expr.AttributeExpr{Type: expr.String},
		TypeName:      "Value",
		UID:           "value",
	}
	copy := original.Dup(expr.DupAtt(original.Attribute()))
	require.Equal(t, "Value", scope.Unique("Value"))
	scope.bindUserType(original, "Value")

	fork := scope.Fork()
	scope.Freeze()
	fork.Freeze()
	require.Equal(t, "Value", scope.GoTypeName(&expr.AttributeExpr{Type: copy}))
	require.Equal(t, "types.Value", fork.GoFullTypeName(&expr.AttributeExpr{Type: copy}, "types"))
}

func TestNameScope_PeekUnique_MatchesUniqueWithoutMutation(t *testing.T) {
	seed := func(scope *NameScope) {
		scope.Unique("a")
		scope.Unique("a")
		scope.Unique("a2")
		scope.Unique("hello")
		scope.Unique("hello1")
	}

	peek := NewNameScope()
	seed(peek)

	mutating := NewNameScope()
	seed(mutating)

	if got, want := peek.PeekUnique("a"), mutating.Unique("a"); got != want {
		t.Fatalf("expected peek %q, got %q", want, got)
	}
	if got, want := peek.PeekUnique("hel", "lo"), mutating.Unique("hel", "lo"); got != want {
		t.Fatalf("expected peek %q, got %q", want, got)
	}
	if got, want := peek.PeekUnique("hello", "1"), mutating.Unique("hello", "1"); got != want {
		t.Fatalf("expected peek %q, got %q", want, got)
	}

	// PeekUnique must not mutate the scope.
	if got, want := peek.Unique("a"), "a3"; got != want {
		t.Fatalf("expected scope unchanged, got %q", got)
	}
}

