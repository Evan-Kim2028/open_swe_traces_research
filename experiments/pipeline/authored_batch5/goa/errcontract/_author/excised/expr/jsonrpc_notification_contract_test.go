// This file checks the design rules for one-way JSON-RPC calls.
package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"example.internal/apikit/v3/dsl"
	"example.internal/apikit/v3/expr"
)

const jsonRPCInternalError = -32603


// TestJSONRPCRequestIDBodyContract checks that the request ID exists only in
// the JSON-RPC envelope and cannot also be mapped into params.
func TestJSONRPCRequestIDBodyContract(t *testing.T) {
	t.Run("computed params omit ID", func(t *testing.T) {
		root := expr.RunDSL(t, jsonRPCNotificationDSL(func() {
			dsl.Payload(func() {
				dsl.ID("request_id")
				dsl.Attribute("message", dsl.String)
				dsl.Required("request_id", "message")
			})
			dsl.JSONRPC(func() {})
		}))
		body := root.API.JSONRPC.Services[0].HTTPEndpoints[0].Body
		require.Nil(t, body.Find("request_id"))
		require.NotNil(t, body.Find("message"))
		require.False(t, body.IsRequired("request_id"))
		require.True(t, body.IsRequired("message"))
	})

	t.Run("ID-only payload has no params body", func(t *testing.T) {
		root := expr.RunDSL(t, jsonRPCNotificationDSL(func() {
			dsl.Payload(func() {
				dsl.ID("request_id")
				dsl.Required("request_id")
			})
			dsl.JSONRPC(func() {})
		}))
		body := root.API.JSONRPC.Services[0].HTTPEndpoints[0].Body
		require.Equal(t, expr.Empty, body.Type)
	})

	t.Run("explicit params reject ID", func(t *testing.T) {
		err := expr.RunInvalidDSL(t, jsonRPCNotificationDSL(func() {
			dsl.Payload(func() {
				dsl.ID("request_id")
				dsl.Attribute("message", dsl.String)
			})
			dsl.JSONRPC(func() {
				dsl.Body(func() {
					dsl.Attribute("request_id")
					dsl.Attribute("message")
				})
			})
		}))
		require.Contains(t, err.Error(), `JSON-RPC request ID field "request_id" cannot also appear in params`)
	})
}

// TestJSONRPCRequestIDTransportContract checks that the envelope ID is not
// also read from another part of the HTTP request.
func TestJSONRPCRequestIDTransportContract(t *testing.T) {
	tests := []struct {
		name    string
		mapping func()
		wantErr string
	}{
		{name: "query parameter", mapping: func() { dsl.Param("request_id") }, wantErr: `cannot also be mapped as an HTTP parameter`},
		{name: "header", mapping: func() { dsl.Header("request_id") }, wantErr: `cannot also be mapped as an HTTP header`},
		{name: "cookie", mapping: func() { dsl.Cookie("request_id") }, wantErr: `cannot also be mapped as an HTTP cookie`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, jsonRPCNotificationDSL(func() {
				dsl.Payload(func() {
					dsl.ID("request_id")
				})
				dsl.JSONRPC(func() {
					test.mapping()
				})
			}))
			require.Contains(t, err.Error(), test.wantErr)
		})
	}

	t.Run("path parameter", func(t *testing.T) {
		err := expr.RunInvalidDSL(t, func() {
			dsl.Service("records", func() {
				dsl.JSONRPC(func() {
					dsl.POST("/rpc/{request_id}")
				})
				dsl.Method("lookup", func() {
					dsl.Payload(func() {
						dsl.ID("request_id")
					})
					dsl.JSONRPC(func() {
						dsl.Param("request_id")
					})
				})
			})
		})
		require.Contains(t, err.Error(), `cannot also be mapped as an HTTP parameter`)
	})

	t.Run("SSE reconnect header", func(t *testing.T) {
		err := expr.RunInvalidDSL(t, jsonRPCNotificationDSL(func() {
			dsl.Payload(func() {
				dsl.ID("request_id")
			})
			dsl.StreamingResult(dsl.String)
			dsl.JSONRPC(func() {
				dsl.ServerSentEvents(func() {
					dsl.SSERequestID("request_id")
				})
			})
		}))
		require.Contains(t, err.Error(), `cannot also be mapped to the Last-Event-ID header`)
	})

	t.Run("ordinary HTTP mapping remains valid", func(t *testing.T) {
		expr.RunDSL(t, func() {
			dsl.Service("records", func() {
				dsl.JSONRPC(func() {
					dsl.POST("/rpc")
				})
				dsl.Method("lookup", func() {
					dsl.Payload(func() {
						dsl.ID("request_id")
						dsl.Required("request_id")
					})
					dsl.HTTP(func() {
						dsl.GET("/records/{request_id}")
						dsl.Param("request_id")
					})
					dsl.JSONRPC(func() {})
				})
			})
		})
	})
}

// TestJSONRPCMethodNameContract checks that application methods do not use the
// namespace reserved by JSON-RPC itself.
func TestJSONRPCMethodNameContract(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		dsl.Service("records", func() {
			dsl.JSONRPC(func() {
				dsl.POST("/rpc")
			})
			dsl.Method("rpc.lookup", func() {
				dsl.JSONRPC(func() {})
			})
		})
	})
	require.Contains(t, err.Error(), `JSON-RPC method "rpc.lookup" cannot begin with "rpc."`)
}

// jsonRPCNotificationDSL exposes one method through the shared JSON-RPC route.
func jsonRPCNotificationDSL(method func()) func() {
	return func() {
		dsl.Service("notifications", func() {
			dsl.JSONRPC(func() {
				dsl.POST("/rpc")
			})
			dsl.Method("notify", method)
		})
	}
}
