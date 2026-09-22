// This file verifies that reusable transport error mappings never replace the
// service error contract selected by an endpoint method.
package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "example.internal/apikit/v3/dsl"
	"example.internal/apikit/v3/expr"
)














func TestMethodTransportErrorResponseRequiresSelectedError(t *testing.T) {
	for _, test := range []struct {
		name string
		dsl  func()
	}{
		{
			name: "HTTP method",
			dsl: unselectedErrorResponseDSL(
				func() {},
				func() {
					HTTP(func() {
						POST("/run")
						Response(StatusServiceUnavailable, "busy")
					})
				},
			),
		},
		{
			name: "gRPC method",
			dsl: unselectedErrorResponseDSL(
				func() {},
				func() { GRPC(func() { Response("busy", CodeUnavailable) }) },
			),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, test.dsl)
			require.ErrorContains(t, err, "does not match an error defined in")
		})
	}
}

func qualifierErrorMappingDSL(transport string, qualifier func()) func() {
	return func() {
		API("errors", func() {
			Error("busy", qualifier)
			if transport == "HTTP" {
				HTTP(func() { Response(StatusServiceUnavailable, "busy") })
			} else {
				GRPC(func() { Response("busy", CodeUnavailable) })
			}
		})
		Service("Jobs", func() {
			Method("Run", func() {
				Error("busy")
				if transport == "HTTP" {
					HTTP(func() { POST("/run") })
				} else {
					GRPC(func() {})
				}
			})
		})
	}
}

func explicitQualifierOverrideDSL(transport string) func() {
	return func() {
		API("errors", func() {
			Error("busy", Temporary)
			if transport == "HTTP" {
				HTTP(func() { Response(StatusServiceUnavailable, "busy") })
			} else {
				GRPC(func() { Response("busy", CodeUnavailable) })
			}
		})
		Service("Jobs", func() {
			Method("Run", func() {
				Error("busy", ErrorResult)
				if transport == "HTTP" {
					HTTP(func() { POST("/run") })
				} else {
					GRPC(func() {})
				}
			})
		})
	}
}

func unselectedErrorResponseDSL(serviceTransport, methodTransport func()) func() {
	return func() {
		API("errors", func() {
			Error("busy", Temporary)
		})
		Service("Jobs", func() {
			serviceTransport()
			Method("Run", func() {
				methodTransport()
			})
		})
	}
}

func serviceDefaultErrorResponseDSL(transport string) func() {
	return func() {
		API("errors", func() {
			Error("busy", Temporary)
		})
		Service("Jobs", func() {
			if transport == "HTTP" {
				HTTP(func() { Response(StatusServiceUnavailable, "busy") })
			} else {
				GRPC(func() { Response("busy", CodeUnavailable) })
			}
			Method("Selected", func() {
				Error("busy")
				if transport == "HTTP" {
					HTTP(func() { POST("/selected") })
				} else {
					GRPC(func() {})
				}
			})
			Method("Unselected", func() {
				if transport == "HTTP" {
					HTTP(func() { POST("/unselected") })
				} else {
					GRPC(func() {})
				}
			})
		})
	}
}

var equivalentHTTPErrorMappingDSL = func() {
	API("errors", func() {
		Error("bad_request", String)
		HTTP(func() { Response(StatusBadRequest, "bad_request") })
	})
	Service("Errors", func() {
		Method("Show", func() {
			Error("bad_request", String)
			HTTP(func() { GET("/") })
		})
	})
}

var incompatibleHTTPErrorMappingDSL = func() {
	API("errors", func() {
		Error("bad_request", String)
		HTTP(func() { Response(StatusBadRequest, "bad_request") })
	})
	Service("Errors", func() {
		Method("Show", func() {
			Error("bad_request", Int)
			HTTP(func() { GET("/") })
		})
	})
}

var incompatibleHTTPErrorValidationDSL = func() {
	API("errors", func() {
		Error("bad_request", String, func() { MinLength(2) })
		HTTP(func() { Response(StatusBadRequest, "bad_request") })
	})
	Service("Errors", func() {
		Method("Show", func() {
			Error("bad_request", String, func() { MinLength(3) })
			HTTP(func() { GET("/") })
		})
	})
}

var equivalentHTTPErrorValidationOrderDSL = func() {
	API("errors", func() {
		Error("bad_request", String, func() { Enum("first", "second") })
		HTTP(func() { Response(StatusBadRequest, "bad_request") })
	})
	Service("Errors", func() {
		Method("Show", func() {
			Error("bad_request", String, func() { Enum("second", "first") })
			HTTP(func() { GET("/") })
		})
	})
}

var incompatibleHTTPNamedErrorMappingDSL = func() {
	firstError := Type("FirstError", func() { Attribute("message", String) })
	secondError := Type("SecondError", func() { Attribute("message", String) })
	API("errors", func() {
		Error("bad_request", firstError)
		HTTP(func() { Response(StatusBadRequest, "bad_request") })
	})
	Service("Errors", func() {
		Method("Show", func() {
			Error("bad_request", secondError)
			HTTP(func() { GET("/") })
		})
	})
}

var inheritedHTTPObjectErrorMappingDSL = func() {
	stringError := Type("StringError", func() { Attribute("header") })
	API("errors", func() {
		Error("bad_request", stringError)
		HTTP(func() {
			Response("bad_request", StatusBadRequest, func() { Header("header") })
		})
	})
	Service("Errors", func() {
		Error("bad_request")
		Method("Show", func() { HTTP(func() { GET("/") }) })
	})
}

var equivalentGRPCErrorMappingDSL = func() {
	API("errors", func() {
		Error("bad_request", String)
		GRPC(func() { Response("bad_request", CodeInvalidArgument) })
	})
	Service("Errors", func() {
		Method("Show", func() {
			Error("bad_request", String)
			GRPC(func() {})
		})
	})
}

var incompatibleGRPCErrorMappingDSL = func() {
	API("errors", func() {
		Error("bad_request", String)
		GRPC(func() { Response("bad_request", CodeInvalidArgument) })
	})
	Service("Errors", func() {
		Method("Show", func() {
			Error("bad_request", Int)
			GRPC(func() {})
		})
	})
}

var equivalentServiceErrorMappingDSL = func() {
	Service("Errors", func() {
		Error("bad_request", String)
		HTTP(func() { Response(StatusBadRequest, "bad_request") })
		GRPC(func() { Response("bad_request", CodeInvalidArgument) })
		Method("Show", func() {
			Error("bad_request", String)
			HTTP(func() { GET("/") })
			GRPC(func() {})
		})
	})
}

var equivalentHTTPInheritedErrorMappingDSL = func() {
	base := Type("HTTPErrorBase", func() {
		Attribute("message", String, func() {
			Default("invalid")
			Meta("struct:field:name", "Message")
		})
		Required("message")
	})
	API("errors", func() {
		Error("bad_request", &expr.Object{}, func() { Extend(base) })
		HTTP(func() { Response(StatusBadRequest, "bad_request") })
	})
	Service("Errors", func() {
		Method("Show", func() {
			Error("bad_request", &expr.Object{}, func() {
				Attribute("message", String, func() {
					Default("invalid")
					Meta("struct:field:name", "Message")
				})
				Required("message")
			})
			HTTP(func() { GET("/") })
		})
	})
}

var equivalentGRPCInheritedErrorMappingDSL = func() {
	base := Type("GRPCErrorBase", func() {
		Attribute("message", String, func() { Meta("rpc:tag", "1") })
		Required("message")
	})
	API("errors", func() {
		Error("bad_request", &expr.Object{}, func() { Extend(base) })
		GRPC(func() { Response("bad_request", CodeInvalidArgument) })
	})
	Service("Errors", func() {
		Method("Show", func() {
			Error("bad_request", &expr.Object{}, func() {
				Attribute("message", String, func() { Meta("rpc:tag", "1") })
				Required("message")
			})
			GRPC(func() {})
		})
	})
}

var incompatibleHTTPInheritedErrorMappingDSL = func() {
	stringBase := Type("HTTPStringErrorBase", func() { Attribute("value", String) })
	integerBase := Type("HTTPIntegerErrorBase", func() { Attribute("value", Int) })
	API("errors", func() {
		Error("bad_request", &expr.Object{}, func() { Extend(stringBase) })
		HTTP(func() { Response(StatusBadRequest, "bad_request") })
	})
	Service("Errors", func() {
		Method("Show", func() {
			Error("bad_request", &expr.Object{}, func() { Extend(integerBase) })
			HTTP(func() { GET("/") })
		})
	})
}

var incompatibleGRPCInheritedErrorMappingDSL = func() {
	stringBase := Type("GRPCStringErrorBase", func() {
		Attribute("value", String, func() { Meta("rpc:tag", "1") })
	})
	integerBase := Type("GRPCIntegerErrorBase", func() {
		Attribute("value", Int, func() { Meta("rpc:tag", "1") })
	})
	API("errors", func() {
		Error("bad_request", &expr.Object{}, func() { Extend(stringBase) })
		GRPC(func() { Response("bad_request", CodeInvalidArgument) })
	})
	Service("Errors", func() {
		Method("Show", func() {
			Error("bad_request", &expr.Object{}, func() { Extend(integerBase) })
			GRPC(func() {})
		})
	})
}
