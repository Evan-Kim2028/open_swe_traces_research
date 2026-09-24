// This file verifies that validation planning preserves service and view
// output without reading expressions after package names are fixed.
package codegen

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"example.internal/apikit/v3/expr"
)


// TestValidationPlanImportsOnlyUsedRuntimePackages checks that standalone
// validation users receive the packages named directly by rendered checks.
func TestValidationPlanImportsOnlyUsedRuntimePackages(t *testing.T) {
	for _, test := range []struct {
		name            string
		validation      *expr.ValidationExpr
		wantPreferences []GoTypeImport
		wantImports     []GoTypeImport
	}{
		{
			name:       "no checks",
			validation: nil,
		},
		{
			name:       "pattern",
			validation: &expr.ValidationExpr{Pattern: "^[a-z]+$"},
			wantPreferences: []GoTypeImport{
				{Name: "goa", Path: "example.internal/apikit/v3/pkg"},
			},
			wantImports: []GoTypeImport{
				{Name: "goa", Path: "example.internal/apikit/v3/pkg"},
			},
		},
		{
			name: "string length",
			validation: func() *expr.ValidationExpr {
				minimum := 2
				return &expr.ValidationExpr{MinLength: &minimum}
			}(),
			wantPreferences: []GoTypeImport{
				{Path: "unicode/utf8"},
				{Name: "goa", Path: "example.internal/apikit/v3/pkg"},
			},
			wantImports: []GoTypeImport{
				{Name: "utf8", Path: "unicode/utf8"},
				{Name: "goa", Path: "example.internal/apikit/v3/pkg"},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			attribute := &expr.AttributeExpr{Type: expr.String, Validation: test.validation}
			policy := GoLayoutPolicy{UseDefault: true, SumType: true}
			layout, err := PlanGoType(attribute, GoTypePlanOptions{
				Owner:  "generated.local/gen/service",
				Policy: policy,
			})
			require.NoError(t, err)
			plan, err := NewValidationPlan(attribute, layout, ValidationPlanOptions{Required: true})
			require.NoError(t, err)
			require.Equal(t, test.wantPreferences, plan.ImportPreferences())
			require.Equal(t, test.wantPreferences, ValidationRuntimeImports(attribute, policy))
			linked, err := plan.Link(layout.Link(layout.Owner(), validationPlanTestQualifier))
			require.NoError(t, err)
			require.Equal(t, test.wantImports, linked.Imports())
		})
	}
}

// TestValidationRuntimeImportsStopAtNestedValidators verifies that a parent
// file does not reserve packages used only inside a named child validator.
func TestValidationRuntimeImportsStopAtNestedValidators(t *testing.T) {
	minimum := 1
	child := &expr.UserTypeExpr{
		TypeName: "Child",
		AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{
			{Name: "name", Attribute: &expr.AttributeExpr{
				Type:       expr.String,
				Validation: &expr.ValidationExpr{MinLength: &minimum},
			}},
		}},
	}
	parent := &expr.AttributeExpr{Type: &expr.Object{
		{Name: "child", Attribute: &expr.AttributeExpr{Type: child}},
	}}
	policy := GoLayoutPolicy{UseDefault: true, SumType: true}

	require.Equal(t, []GoTypeImport{{Name: "goa", Path: GoaImport("").Path}}, ValidationRuntimeImports(parent, policy))
}

// TestValidationPlanImportPreferencesIncludeExternalValidators checks that
// planning includes only packages containing validation functions that the
// generated checks call.
func TestValidationPlanImportPreferencesIncludeExternalValidators(t *testing.T) {
	const (
		owner      = "generated.local/gen/service"
		childOwner = "generated.local/gen/shared"
	)
	minimum := 1.0
	child := goTypeTestUserType("Child", &expr.Object{
		{Name: "count", Attribute: &expr.AttributeExpr{
			Type:       expr.Int,
			Validation: &expr.ValidationExpr{Minimum: &minimum},
		}},
	})
	generation, err := NewGeneration("generated.local/gen", nil)
	require.NoError(t, err)
	childDeclaration := declareGoTypeTestUserType(t, generation, childOwner, child)
	validator := NewExactName(NameFunction, "ValidateChild")
	require.NoError(t, generation.Package(childOwner).DeclareName(validator))
	attribute := &expr.AttributeExpr{Type: &expr.Object{
		{Name: "first", Attribute: &expr.AttributeExpr{Type: child}},
		{Name: "second", Attribute: &expr.AttributeExpr{Type: child}},
	}}
	layout, err := PlanGoType(attribute, GoTypePlanOptions{
		Owner:  owner,
		Policy: GoLayoutPolicy{UseDefault: true, SumType: true},
		Bind: goTypeTestBinder(map[expr.DataType]GoTypeBinding{
			child: {Owner: childOwner, Type: childDeclaration},
		}),
	})
	require.NoError(t, err)
	plan, err := NewValidationPlan(attribute, layout, ValidationPlanOptions{
		Required: true,
		Bind: func(ValidatorBindingRequest) (*NameDeclaration, error) {
			return validator, nil
		},
	})
	require.NoError(t, err)

	require.Equal(t, []GoTypeImport{
		{Name: "goa", Path: "example.internal/apikit/v3/pkg"},
		{Name: "shared", Path: childOwner},
	}, plan.ImportPreferences())
}

// TestValidationPlanUsesFinalRuntimeImportNames proves rendered checks and
// reported imports use the same collision-safe package names.
func TestValidationPlanUsesFinalRuntimeImportNames(t *testing.T) {
	minimum := 2
	attribute := &expr.AttributeExpr{
		Type:       expr.String,
		Validation: &expr.ValidationExpr{MinLength: &minimum},
	}
	layout, err := PlanGoType(attribute, GoTypePlanOptions{
		Owner:  "generated.local/gen/service",
		Policy: GoLayoutPolicy{UseDefault: true, SumType: true},
	})
	require.NoError(t, err)
	plan, err := NewValidationPlan(attribute, layout, ValidationPlanOptions{Required: true})
	require.NoError(t, err)
	linked, err := plan.Link(layout.Link(layout.Owner(), func(importPath string) string {
		switch importPath {
		case "example.internal/apikit/v3/pkg":
			return "goa2"
		case "unicode/utf8":
			return "utf82"
		default:
			t.Fatalf("unexpected validation import %q", importPath)
			return ""
		}
	}))
	require.NoError(t, err)

	require.Equal(t, []GoTypeImport{
		{Name: "utf82", Path: "unicode/utf8"},
		{Name: "goa2", Path: "example.internal/apikit/v3/pkg"},
	}, linked.Imports())
	code := linked.Render("target", "target")
	require.Contains(t, code, "utf82.RuneCountInString")
	require.Contains(t, code, "goa2.MergeErrors")
}


// TestValidationPlanCopiesEnumValues verifies that accepted mutable enum
// values cannot change a validation program after planning.
func TestValidationPlanCopiesEnumValues(t *testing.T) {
	bytesValue := []byte{1, 2}
	arrayValue := []any{
		bytesValue,
		map[string]any{"nested": []any{"kept"}},
	}
	mapValue := map[string]any{"array": arrayValue}
	attribute := &expr.AttributeExpr{
		Type: expr.Any,
		Validation: &expr.ValidationExpr{Values: []any{
			bytesValue,
			arrayValue,
			mapValue,
		}},
	}
	layout, err := PlanGoType(attribute, GoTypePlanOptions{
		Owner:  "generated.local/gen/service",
		Policy: GoLayoutPolicy{UseDefault: true, SumType: true},
	})
	require.NoError(t, err)
	plan, err := NewValidationPlan(attribute, layout, ValidationPlanOptions{Required: true})
	require.NoError(t, err)

	bytesValue[0] = 9
	arrayValue[1].(map[string]any)["nested"].([]any)[0] = "changed"
	mapValue["added"] = true
	attribute.Validation.Values[0] = "replaced"

	require.Equal(t, []any{
		[]byte{1, 2},
		[]any{
			[]byte{1, 2},
			map[string]any{"nested": []any{"kept"}},
		},
		map[string]any{"array": []any{
			[]byte{1, 2},
			map[string]any{"nested": []any{"kept"}},
		}},
	}, plan.root.rules.values)
}

// TestNeedsValidation reports whether the validation renderer can write code
// for local rules, nested rules, and values with no rules.
func TestNeedsValidation(t *testing.T) {
	minimum := 1.0
	child := goTypeTestUserType("Child", &expr.Object{
		{Name: "count", Attribute: &expr.AttributeExpr{
			Type:       expr.Int,
			Validation: &expr.ValidationExpr{Minimum: &minimum},
		}},
	})
	tests := []struct {
		name      string
		attribute *expr.AttributeExpr
		want      bool
	}{
		{
			name: "local rule",
			attribute: &expr.AttributeExpr{
				Type:       expr.String,
				Validation: &expr.ValidationExpr{Pattern: ".+"},
			},
			want: true,
		},
		{
			name: "nested rule",
			attribute: &expr.AttributeExpr{Type: &expr.Object{
				{Name: "child", Attribute: &expr.AttributeExpr{Type: child}},
			}},
			want: true,
		},
		{
			name:      "no rules",
			attribute: &expr.AttributeExpr{Type: &expr.Object{{Name: "name", Attribute: &expr.AttributeExpr{Type: expr.String}}}},
		},
	}
	policy := GoLayoutPolicy{Pointer: true, UseDefault: true, SumType: true}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, NeedsValidation(test.attribute, policy))
		})
	}
}

// TestNeedsValidationRecursiveCollections checks that walking a recursive array
// or map terminates and still finds a rule after the recursive field.
func TestNeedsValidationRecursiveCollections(t *testing.T) {
	for _, container := range []string{"array", "map"} {
		for _, constrained := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/constrained=%t", container, constrained), func(t *testing.T) {
				node := goTypeTestUserType("Node", &expr.Object{})
				element := &expr.AttributeExpr{Type: node}
				var children expr.DataType = &expr.Array{ElemType: element}
				if container == "map" {
					children = &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: element}
				}
				label := &expr.AttributeExpr{Type: expr.String}
				if constrained {
					label.Validation = &expr.ValidationExpr{Pattern: "^valid$"}
				}
				node.Attribute().Type = &expr.Object{
					{Name: "children", Attribute: &expr.AttributeExpr{Type: children}},
					{Name: "label", Attribute: label},
				}
				require.Equal(t, constrained, NeedsValidation(&expr.AttributeExpr{Type: node}, GoLayoutPolicy{SumType: true}))
			})
		}
	}
}


// TestNeedsValidationChecksEverySiblingCopy verifies that one unconstrained
// copy of a type does not hide rules on another copy of the same type.
func TestNeedsValidationChecksEverySiblingCopy(t *testing.T) {
	minLength := 2
	child := goTypeTestUserType("Child", &expr.Object{
		{Name: "value", Attribute: &expr.AttributeExpr{Type: expr.String}},
	})
	unvalidated := expr.DupAtt(&expr.AttributeExpr{Type: child})
	validated := expr.DupAtt(&expr.AttributeExpr{Type: child})
	expr.AsObject(validated.Type.(expr.UserType).Attribute().Type).Attribute("value").Validation =
		&expr.ValidationExpr{MinLength: &minLength}

	tests := []struct {
		name   string
		fields *expr.Object
	}{
		{
			name: "unvalidated copy first",
			fields: &expr.Object{
				{Name: "first", Attribute: unvalidated},
				{Name: "second", Attribute: validated},
			},
		},
		{
			name: "validated copy first",
			fields: &expr.Object{
				{Name: "first", Attribute: validated},
				{Name: "second", Attribute: unvalidated},
			},
		},
	}
	policy := GoLayoutPolicy{Pointer: true, UseDefault: true, SumType: true}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			attribute := &expr.AttributeExpr{Type: test.fields}
			require.True(t, NeedsValidation(attribute, policy))
		})
	}
}


// TestValidationPlanRejectsUnboundNestedValidator verifies planning never
// falls back to reconstructing a validator name from a user type.
func TestValidationPlanRejectsUnboundNestedValidator(t *testing.T) {
	minLength := 1
	child := goTypeTestUserType("Child", &expr.Object{
		{Name: "name", Attribute: &expr.AttributeExpr{
			Type:       expr.String,
			Validation: &expr.ValidationExpr{MinLength: &minLength},
		}},
	})
	attribute := &expr.AttributeExpr{Type: &expr.Object{
		{Name: "child", Attribute: &expr.AttributeExpr{Type: child}},
	}}
	generation, err := NewGeneration("generated.local/gen", nil)
	require.NoError(t, err)
	declaration := declareGoTypeTestUserType(t, generation, "generated.local/gen/service", child)
	layout, err := PlanGoType(attribute, GoTypePlanOptions{
		Owner:  "generated.local/gen/service",
		Policy: GoLayoutPolicy{Pointer: true, UseDefault: true, SumType: true},
		Bind: goTypeTestBinder(map[expr.DataType]GoTypeBinding{
			child: {Owner: "generated.local/gen/service", Type: declaration},
		}),
	})
	require.NoError(t, err)

	_, err = NewValidationPlan(attribute, layout, ValidationPlanOptions{Required: true})
	require.EqualError(t, err, "plan validation for field \"child\": validator binder must not be nil")
}

// validationPlanTestQualifier resolves the focused generated package aliases.
func validationPlanTestQualifier(importPath string) string {
	switch importPath {
	case "generated.local/gen/service":
		return "service"
	case "example.internal/apikit/v3/pkg":
		return "goa"
	case "unicode/utf8":
		return "utf8"
	default:
		panic(fmt.Sprintf("unexpected validation import %q", importPath))
	}
}
