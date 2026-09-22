package eval_test

import (
	"errors"
	_ "strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "example.internal/apikit/v3/dsl"
	"example.internal/apikit/v3/eval"
	_ "example.internal/apikit/v3/expr"
)




// mockExpr is a test implementation of the Expression interface
type mockExpr struct{}

func (m *mockExpr) EvalName() string {
	return "MockExpression"
}

// TestValidationErrors tests the ValidationErrors type methods
func TestValidationErrors(t *testing.T) {
	t.Run("Error", func(t *testing.T) {
		// Test Error() method
		expr1 := &mockExpr{}
		expr2 := &mockExpr{}

		verr := &eval.ValidationErrors{
			Errors:      []error{errors.New("error1"), errors.New("error2")},
			Expressions: []eval.Expression{expr1, expr2},
		}

		errStr := verr.Error()
		assert.Contains(t, errStr, "MockExpression: error1")
		assert.Contains(t, errStr, "MockExpression: error2")
		assert.Contains(t, errStr, "\n") // Should contain a newline between errors
	})

	t.Run("Merge", func(t *testing.T) {
		// Test Merge() method
		expr1 := &mockExpr{}
		expr2 := &mockExpr{}
		expr3 := &mockExpr{}

		verr1 := &eval.ValidationErrors{
			Errors:      []error{errors.New("error1")},
			Expressions: []eval.Expression{expr1},
		}

		verr2 := &eval.ValidationErrors{
			Errors:      []error{errors.New("error2"), errors.New("error3")},
			Expressions: []eval.Expression{expr2, expr3},
		}

		// Test merging with nil
		verrCopy := *verr1
		verrCopy.Merge(nil)
		assert.Equal(t, 1, len(verrCopy.Errors))
		assert.Equal(t, 1, len(verrCopy.Expressions))

		// Test merging with empty but non-nil ValidationErrors
		verrCopy2 := *verr1
		emptyVerr := &eval.ValidationErrors{}
		verrCopy2.Merge(emptyVerr)
		assert.Equal(t, 1, len(verrCopy2.Errors), "Merging with empty ValidationErrors should not change the error count")
		assert.Equal(t, 1, len(verrCopy2.Expressions), "Merging with empty ValidationErrors should not change the expression count")

		// Test normal merge
		verr1.Merge(verr2)
		assert.Equal(t, 3, len(verr1.Errors))
		assert.Equal(t, 3, len(verr1.Expressions))
		assert.Equal(t, expr1, verr1.Expressions[0])
		assert.Equal(t, expr2, verr1.Expressions[1])
		assert.Equal(t, expr3, verr1.Expressions[2])
	})

	t.Run("Add", func(t *testing.T) {
		// Test Add() method
		expr1 := &mockExpr{}

		verr := &eval.ValidationErrors{}
		verr.Add(expr1, "test error %s", "message")

		require.Equal(t, 1, len(verr.Errors))
		require.Equal(t, 1, len(verr.Expressions))
		assert.Equal(t, "test error message", verr.Errors[0].Error())
		assert.Equal(t, expr1, verr.Expressions[0])
	})

	t.Run("AddError_Simple", func(t *testing.T) {
		// Test AddError() with a simple error
		expr1 := &mockExpr{}
		simpleErr := errors.New("simple error")

		verr := &eval.ValidationErrors{}
		verr.AddError(expr1, simpleErr)

		require.Equal(t, 1, len(verr.Errors))
		require.Equal(t, 1, len(verr.Expressions))
		assert.Equal(t, simpleErr, verr.Errors[0])
		assert.Equal(t, expr1, verr.Expressions[0])
	})

	t.Run("AddError_ValidationErrors", func(t *testing.T) {
		// Test AddError() with another ValidationErrors
		expr1 := &mockExpr{}
		expr2 := &mockExpr{}
		expr3 := &mockExpr{}

		nestedVerr := &eval.ValidationErrors{
			Errors:      []error{errors.New("nested error 1"), errors.New("nested error 2")},
			Expressions: []eval.Expression{expr2, expr3},
		}

		verr := &eval.ValidationErrors{}
		verr.AddError(expr1, nestedVerr) // expr1 should be ignored since we're adding a ValidationErrors

		// The nested ValidationErrors should be flattened
		require.Equal(t, 2, len(verr.Errors))
		require.Equal(t, 2, len(verr.Expressions))
		assert.Equal(t, "nested error 1", verr.Errors[0].Error())
		assert.Equal(t, "nested error 2", verr.Errors[1].Error())
		assert.Equal(t, expr2, verr.Expressions[0])
		assert.Equal(t, expr3, verr.Expressions[1])
	})
}
