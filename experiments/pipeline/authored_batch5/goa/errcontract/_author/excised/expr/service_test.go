package expr_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"example.internal/apikit/v3/dsl"
	_ "example.internal/apikit/v3/eval"
	"example.internal/apikit/v3/expr"
	"example.internal/apikit/v3/expr/testdata"
)

func TestServiceExprMethod(t *testing.T) {
	var (
		methodFoo = &expr.MethodExpr{
			Name: "foo",
		}
		methodBar = &expr.MethodExpr{
			Name: "bar",
		}
	)
	cases := map[string]struct {
		name     string
		expected *expr.MethodExpr
	}{
		"exist": {
			name:     "foo",
			expected: methodFoo,
		},
		"not exist": {
			name:     "baz",
			expected: nil,
		},
	}

	for k, tc := range cases {
		s := expr.ServiceExpr{
			Methods: []*expr.MethodExpr{
				methodFoo,
				methodBar,
			},
		}
		if actual := s.Method(tc.name); actual != tc.expected {
			t.Errorf("%s: got %#v, expected %#v", k, actual, tc.expected)
		}
	}
}




func TestAPIErrorDoesNotAddErrorsToServicesOrMethods(t *testing.T) {
	root := expr.RunDSL(t, func() {
		dsl.API("jobs", func() {
			dsl.Error("busy", dsl.Temporary)
		})
		dsl.Service("jobs", func() {
			dsl.Method("run", func() {})
			dsl.Method("retry", func() {
				dsl.Error("busy")
			})
		})
	})
	service := root.Service("jobs")

	require.Empty(t, service.Errors)
	require.Empty(t, service.Method("run").Errors)
	require.Equal(t, []*expr.ErrorExpr{root.Error("busy")}, service.Method("retry").Errors)
}

func TestRepeatedAuthoredErrorTypesDoNotShareGeneratedConstructors(t *testing.T) {
	custom := dsl.Type("CustomError", func() {
		dsl.Attribute("message", dsl.String)
	})
	expr.RunDSL(t, func() {
		dsl.Service("jobs", func() {
			dsl.Error("busy", custom, dsl.Temporary)
			dsl.Method("run", func() {
				dsl.Error("busy", custom)
			})
		})
	})
}

func TestServiceExprError(t *testing.T) {
	var (
		errorFoo = &expr.ErrorExpr{
			Name: "foo",
		}
	)
	cases := map[string]struct {
		name     string
		expected *expr.ErrorExpr
	}{
		"exist in service": {
			name:     "foo",
			expected: errorFoo,
		},
		"not exist": {
			name:     "qux",
			expected: nil,
		},
	}

	s := expr.ServiceExpr{
		Errors: []*expr.ErrorExpr{
			errorFoo,
		},
	}
	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			if actual := s.Error(tc.name); actual != tc.expected {
				t.Errorf("got %#v, expected %#v", actual, tc.expected)
			}
		})
	}
}

func TestServiceExprValidate(t *testing.T) {
	cases := []struct {
		Name  string
		DSL   func()
		Error string
	}{
		{"service errors", testdata.ServiceErrorDSL, `attribute: error name "a" must be required in type "ServiceError"`},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, tc.DSL)
			assert.EqualError(t, err, tc.Error)
		})
	}
}

func TestErrorExprValidate(t *testing.T) {
	cases := []struct {
		Name  string
		DSL   func()
		Error string
	}{
		{"no error", testdata.ValidErrorsDSL, ""},
		{"invalid-struct-error-name-meta", testdata.InvalidStructErrorNameDSL,
			`attribute: type "ErrorType" defines errors error1, error2 and must identify the attribute containing the error name with ErrorName
attribute: error name "a" must be required in type "ServiceError"
attribute: duplicate error names in type "Error"
attribute: error name "a" must be a string in type "Error"
attribute: error name "a" must be required in type "Error"`,
		},
		{"shared error names across services", testdata.InvalidSharedErrorNamesDSL,
			`attribute: type "SharedError" defines errors first_error, second_error and must identify the attribute containing the error name with ErrorName`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			if tc.Error == "" {
				expr.RunDSL(t, tc.DSL)
			} else {
				err := expr.RunInvalidDSL(t, tc.DSL)
				assert.EqualError(t, err, tc.Error)
			}
		})
	}
}
