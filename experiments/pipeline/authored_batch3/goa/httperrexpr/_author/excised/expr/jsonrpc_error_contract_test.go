// This file verifies that JSON-RPC error mappings remain separate from HTTP
// mappings and use only error codes allowed by JSON-RPC 2.0.
package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "example.internal/apikit/v3/dsl"
	"example.internal/apikit/v3/expr"
)

func TestJSONRPCAPIErrorMappingOverridesHTTPDefault(t *testing.T) {
	root := expr.RunDSL(t, func() {
		API("jobs", func() {
			Error("busy", String)
			HTTP(func() {
				Path("/v1")
				Response("busy", StatusServiceUnavailable)
			})
			JSONRPC(func() {
				Response("busy", 7001)
			})
		})
		Service("jobs", func() {
			Method("run", func() {
				Error("busy")
				HTTP(func() {
					POST("/run")
				})
				JSONRPC(func() {})
			})
			Method("inspect", func() {
				JSONRPC(func() {})
			})
		})
	})

	httpEndpoint := root.API.HTTP.Service("jobs").Endpoint("run")
	jsonrpcService := root.API.JSONRPC.Service("jobs")
	jsonrpcEndpoint := jsonrpcService.Endpoint("run")

	require.Equal(t, StatusServiceUnavailable, httpEndpoint.HTTPErrors[0].Response.StatusCode)
	require.Equal(t, 7001, jsonrpcEndpoint.HTTPErrors[0].Response.StatusCode)
	require.Empty(t, jsonrpcService.Endpoint("inspect").HTTPErrors)
	require.Equal(t, "/v1", root.API.JSONRPC.Path)
}

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


func TestJSONRPCErrorCodeDefaultsToInternalError(t *testing.T) {
	root := expr.RunDSL(t, func() {
		Service("jobs", func() {
			Method("run", func() {
				Error("failed", String)
				JSONRPC(func() {
					Response("failed", func() {})
				})
			})
		})
	})
	response := root.API.JSONRPC.Service("jobs").Endpoint("run").HTTPErrors[0].Response

	require.Equal(t, RPCInternalError, response.StatusCode)
	require.False(t, response.StatusCodeSet)
}


func TestJSONRPCErrorCodeMayBeReusedByDifferentMethods(t *testing.T) {
	root := expr.RunDSL(t, func() {
		API("jobs", func() {
			Error("busy", String)
			Error("failed", String)
			JSONRPC(func() {
				Response("busy", 7001)
				Response("failed", 7001)
			})
		})
		Service("jobs", func() {
			Method("run", func() {
				Error("busy")
				JSONRPC(func() {})
			})
			Method("inspect", func() {
				Error("failed")
				JSONRPC(func() {})
			})
		})
	})

	service := root.API.JSONRPC.Service("jobs")
	require.Equal(t, 7001, service.Endpoint("run").HTTPErrors[0].Response.StatusCode)
	require.Equal(t, "busy", service.Endpoint("run").HTTPErrors[0].Name)
	require.Equal(t, 7001, service.Endpoint("inspect").HTTPErrors[0].Response.StatusCode)
	require.Equal(t, "failed", service.Endpoint("inspect").HTTPErrors[0].Name)
}

func TestJSONRPCMethodErrorMappingReplacesAPIDefault(t *testing.T) {
	root := expr.RunDSL(t, func() {
		API("jobs", func() {
			Error("busy", String)
			JSONRPC(func() {
				Response("busy", 7001)
			})
		})
		Service("jobs", func() {
			Method("run", func() {
				Error("busy")
				Error("failed", String)
				JSONRPC(func() {
					Response("busy", 7002)
					Response("failed", 7001)
				})
			})
		})
	})

	errors := root.API.JSONRPC.Service("jobs").Endpoint("run").HTTPErrors
	require.Len(t, errors, 2)
	require.Equal(t, "busy", errors[0].Name)
	require.Equal(t, 7002, errors[0].Response.StatusCode)
	require.Equal(t, "failed", errors[1].Name)
	require.Equal(t, 7001, errors[1].Response.StatusCode)
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
