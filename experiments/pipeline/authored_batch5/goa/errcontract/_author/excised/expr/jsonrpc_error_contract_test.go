// This file verifies that JSON-RPC error mappings remain separate from HTTP
// mappings and use only error codes allowed by JSON-RPC 2.0.
package expr_test

import (
	_ "fmt"
	"testing"

	"github.com/stretchr/testify/require"

	. "example.internal/apikit/v3/dsl"
	"example.internal/apikit/v3/expr"
)


func TestJSONRPCPreparePreservesJSONRPCErrorDefaults(t *testing.T) {
	previousRoot := expr.Root
	t.Cleanup(func() {
		expr.Root = previousRoot
	})
	httpMapping := &expr.HTTPErrorExpr{
		Name: "busy",
		Response: &expr.HTTPResponseExpr{
			StatusCode: StatusServiceUnavailable,
		},
	}
	jsonrpcMapping := &expr.HTTPErrorExpr{
		Name: "busy",
		Response: &expr.HTTPResponseExpr{
			StatusCode: 7001,
		},
	}
	expr.Root = &expr.RootExpr{
		API: &expr.APIExpr{
			HTTP: &expr.HTTPExpr{
				Errors: []*expr.HTTPErrorExpr{httpMapping},
			},
			JSONRPC: &expr.JSONRPCExpr{
				HTTPExpr: expr.HTTPExpr{
					Errors: []*expr.HTTPErrorExpr{jsonrpcMapping},
				},
			},
		},
	}

	expr.Root.API.JSONRPC.Prepare()

	require.Equal(t, []*expr.HTTPErrorExpr{jsonrpcMapping}, expr.Root.API.JSONRPC.Errors)
}

func TestJSONRPCAPIErrorMappingRequiresReusableError(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		API("jobs", func() {
			JSONRPC(func() {
				Response("busy", 7001)
			})
		})
		Service("jobs", func() {
			Method("inspect", func() {
				JSONRPC(func() {})
			})
		})
	})

	require.ErrorContains(t, err, `Error "busy" does not match an error defined in the API`)
}






func jsonRPCErrorCodeDSL(code int, useCodeDSL bool) func() {
	return func() {
		Service("jobs", func() {
			Method("run", func() {
				Error("failed", String)
				JSONRPC(func() {
					if useCodeDSL {
						Response("failed", func() {
							Code(code)
						})
						return
					}
					Response("failed", code)
				})
			})
		})
	}
}
