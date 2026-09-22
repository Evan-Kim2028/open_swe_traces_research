// This file verifies that JSON-RPC responses contain only JSON-RPC message
// data. HTTP response headers and cookies cannot belong to one message in a
// batch or server-sent-event stream.
package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"example.internal/apikit/v3/dsl"
	"example.internal/apikit/v3/expr"
)

// TestJSONRPCSuccessResponseRejectsHTTPMetadata checks the method result
// mapping, which is the only level where a success response can be declared.
func TestJSONRPCSuccessResponseRejectsHTTPMetadata(t *testing.T) {
	for _, test := range []struct {
		name    string
		mapping func()
		wantErr string
	}{
		{
			name:    "header",
			mapping: func() { dsl.Header("value:X-Value") },
			wantErr: "JSON-RPC success response cannot map result attributes to HTTP headers",
		},
		{
			name:    "cookie",
			mapping: func() { dsl.Cookie("value:result") },
			wantErr: "JSON-RPC success response cannot map result attributes to HTTP cookies",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, jsonRPCSuccessResponseMetadataDSL(test.mapping))
			require.ErrorContains(t, err, test.wantErr)
		})
	}
}



// jsonRPCSuccessResponseMetadataDSL defines one result response with the given
// transport mapping.
func jsonRPCSuccessResponseMetadataDSL(mapping func()) func() {
	return func() {
		dsl.Service("records", func() {
			dsl.Method("fetch", func() {
				dsl.Result(func() {
					dsl.Attribute("value", dsl.String)
				})
				dsl.JSONRPC(func() {
					dsl.Response(mapping)
				})
			})
		})
	}
}

// jsonRPCErrorResponseMetadataDSL defines one error mapping at the requested
// inheritance level and selects that error from one JSON-RPC method.
func jsonRPCErrorResponseMetadataDSL(scope string, mapping func()) func() {
	return func() {
		errorType := func() {
			dsl.Attribute("detail", dsl.String)
		}
		dsl.API("records", func() {
			if scope == "API" {
				dsl.Error("failed", errorType)
				dsl.JSONRPC(func() {
					dsl.Response("failed", 7001, mapping)
				})
			}
		})
		dsl.Service("records", func() {
			if scope == "service" {
				dsl.Error("failed", errorType)
				dsl.JSONRPC(func() {
					dsl.Response("failed", 7001, mapping)
				})
			}
			dsl.Method("fetch", func() {
				if scope == "API" || scope == "service" {
					dsl.Error("failed")
				} else {
					dsl.Error("failed", errorType)
				}
				dsl.JSONRPC(func() {
					if scope == "method" {
						dsl.Response("failed", 7001, mapping)
					}
				})
			})
		})
	}
}
