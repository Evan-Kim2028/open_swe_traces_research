// This file verifies that transport error response callbacks describe the
// declared error value while successful response callbacks describe the method
// result.
package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "example.internal/apikit/v3/dsl"
	"example.internal/apikit/v3/expr"
)





func TestErrorResponseCallbackWithUnknownErrorDoesNotPanic(t *testing.T) {
	for _, test := range []struct {
		name  string
		dsl   func()
		error string
	}{
		{
			name:  "HTTP",
			error: "Body is set but Error missing is not defined",
			dsl: func() {
				Service("Errors", func() {
					Method("Run", func() {
						HTTP(func() {
							POST("/")
							Response("missing", StatusBadRequest, func() { Body("detail") })
						})
					})
				})
			},
		},
		{
			name:  "gRPC",
			error: `Error "missing" does not match an error defined in the method`,
			dsl: func() {
				Service("Errors", func() {
					Method("Run", func() {
						GRPC(func() {
							Response("missing", CodeInvalidArgument, func() {
								Message(func() { Field(1, "detail", String) })
							})
						})
					})
				})
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.NotPanics(t, func() {
				err := expr.RunInvalidDSL(t, test.dsl)
				require.ErrorContains(t, err, test.error)
			})
		})
	}
}

func TestSuccessResponseCallbackUsesMethodResult(t *testing.T) {
	root := expr.RunDSL(t, func() {
		Service("Success", func() {
			Method("Run", func() {
				Result(func() {
					Attribute("value", String)
				})
				HTTP(func() {
					POST("/")
					Response(StatusOK, func() {
						Tag("value", "complete")
						Body("value")
					})
					Response(StatusAccepted)
				})
			})
		})
	})

	response := root.API.HTTP.Services[0].HTTPEndpoints[0].Responses[0]
	require.Equal(t, [2]string{"value", "complete"}, response.Tag)
	require.Equal(t, []string{"value"}, response.Body.Meta["origin:attribute"])
}

func TestAPIErrorMappingsDoNotDeclareMethodErrors(t *testing.T) {
	root := expr.RunDSL(t, func() {
		API("Errors", func() {
			Error("busy", String)
			HTTP(func() { Response("busy", StatusServiceUnavailable) })
			JSONRPC(func() { Response("busy", 7001) })
			GRPC(func() { Response("busy", CodeUnavailable) })
		})
		Service("plain", func() {
			Method("Run", func() {
				HTTP(func() { POST("/run") })
				GRPC(func() {})
			})
		})
		Service("rpc", func() {
			JSONRPC(func() { POST("/rpc") })
			Method("Run", func() { JSONRPC(func() {}) })
		})
	})

	require.Empty(t, root.API.HTTP.Service("plain").Endpoint("Run").HTTPErrors)
	require.Empty(t, root.API.GRPC.Service("plain").Endpoint("Run").GRPCErrors)
	require.Empty(t, root.API.JSONRPC.Service("rpc").Endpoint("Run").HTTPErrors)
}

func methodHTTPErrorResponseDSL() {
	Service("Errors", func() {
		Method("Run", func() {
			Error("problem", transportErrorType())
			HTTP(func() {
				POST("/")
				Response("problem", errorHTTPResponse)
			})
		})
	})
}

func serviceHTTPErrorResponseDSL() {
	Service("Errors", func() {
		Error("problem", transportErrorType())
		HTTP(func() { Response("problem", errorHTTPResponse) })
		Method("Run", func() {
			Error("problem")
			HTTP(func() { POST("/") })
		})
	})
}

func apiHTTPErrorResponseDSL() {
	API("Errors", func() {
		HTTP(func() { Response("problem", errorHTTPResponse) })
		Error("problem", transportErrorType())
	})
	Service("Errors", func() {
		Error("problem")
		Method("Run", func() {
			Error("problem")
			HTTP(func() { POST("/") })
		})
	})
}

func methodGRPCErrorResponseDSL() {
	Service("Errors", func() {
		Method("Run", func() {
			Error("problem", transportErrorType())
			GRPC(func() { Response("problem", errorGRPCResponse) })
		})
	})
}

func serviceGRPCErrorResponseDSL() {
	Service("Errors", func() {
		Error("problem", transportErrorType())
		GRPC(func() { Response("problem", errorGRPCResponse) })
		Method("Run", func() {
			Error("problem")
			GRPC(func() {})
		})
	})
}

func apiGRPCErrorResponseDSL() {
	API("Errors", func() {
		GRPC(func() { Response("problem", errorGRPCResponse) })
		Error("problem", transportErrorType())
	})
	Service("Errors", func() {
		Error("problem")
		Method("Run", func() {
			Error("problem")
			GRPC(func() {})
		})
	})
}

func transportErrorType() func() {
	return func() {
		Field(1, "detail", String)
		Field(2, "header", String)
		Field(3, "trailer", String)
	}
}

func errorHTTPResponse() {
	Code(StatusUnprocessableEntity)
	Description("The problem response.")
	ContentType("application/problem+json")
	Header("detail:X-Detail")
	Cookie("detail:problem")
	CookieMaxAge(30)
	Body("detail")
}

func errorGRPCResponse() {
	Code(CodeInvalidArgument)
	Description("The problem response.")
	Message(func() { Attribute("detail") })
	Headers(func() { Attribute("header") })
	Trailers(func() { Attribute("trailer") })
}
