// This file verifies OpenAPI v2 construction from evaluated HTTP endpoint
// designs, including request and response example ownership.
package openapiv2

import (
	_ "encoding/json"
	_ "testing"

	_ "github.com/stretchr/testify/require"
	_ "example.internal/apikit/v3/codegen"
	dsl "example.internal/apikit/v3/dsl"
	_ "example.internal/apikit/v3/expr"
	_ "example.internal/apikit/v3/http/codegen/openapi"
	_ "gopkg.in/yaml.v3"
)








var noSecurityOverridesAPISecurityDSL = func() {
	var JWTAuth = dsl.JWTSecurity("jwt")

	dsl.API("test", func() {
		dsl.Security(JWTAuth)
	})

	dsl.Service("test", func() {
		dsl.Method("secure", func() {
			dsl.Payload(func() {
				dsl.Token("token", dsl.String)
				dsl.Required("token")
			})
			dsl.HTTP(func() {
				dsl.GET("/secure")
			})
		})
		dsl.Method("public", func() {
			dsl.NoSecurity()
			dsl.HTTP(func() {
				dsl.GET("/public")
			})
		})
	})
}

var localizedValuesDSL = func() {
	dsl.API("messages", func() {
		dsl.Title("Original API")
		dsl.Description("Original API description")
	})
	dsl.Service("messages", func() {
		dsl.Description("Original service description")
		dsl.Method("show", func() {
			dsl.Description("Original method description")
			dsl.HTTP(func() {
				dsl.GET("/messages")
			})
		})
	})
}

var visibleSecuritySchemesDSL = func() {
	var (
		VisibleAuth       = dsl.JWTSecurity("visible_auth")
		HiddenMethodAuth  = dsl.JWTSecurity("hidden_method_auth")
		HiddenServiceAuth = dsl.JWTSecurity("hidden_service_auth")
	)

	dsl.Service("visible", func() {
		dsl.Method("read", func() {
			dsl.Security(VisibleAuth)
			dsl.Payload(func() {
				dsl.Token("token", dsl.String)
			})
			dsl.HTTP(func() {
				dsl.GET("/visible")
			})
		})
	})
	dsl.Service("mixed", func() {
		dsl.Method("hidden", func() {
			dsl.Meta("openapi:generate", "false")
			dsl.Security(HiddenMethodAuth)
			dsl.Payload(func() {
				dsl.Token("token", dsl.String)
			})
			dsl.HTTP(func() {
				dsl.GET("/hidden-method")
			})
		})
	})
	dsl.Service("hidden", func() {
		dsl.Meta("openapi:generate", "false")
		dsl.Method("read", func() {
			dsl.Security(HiddenServiceAuth)
			dsl.Payload(func() {
				dsl.Token("token", dsl.String)
			})
			dsl.HTTP(func() {
				dsl.GET("/hidden-service")
			})
		})
	})
}

var noSecurityOverridesServiceSecurityDSL = func() {
	var JWTAuth = dsl.JWTSecurity("jwt")

	dsl.API("test", func() {})

	dsl.Service("test", func() {
		dsl.Security(JWTAuth)
		dsl.Method("service-secure", func() {
			dsl.Payload(func() {
				dsl.Token("token", dsl.String)
				dsl.Required("token")
			})
			dsl.HTTP(func() {
				dsl.GET("/service-secure")
			})
		})
		dsl.Method("service-public", func() {
			dsl.NoSecurity()
			dsl.HTTP(func() {
				dsl.GET("/service-public")
			})
		})
	})
}

var streamingResponseStatusDSL = func() {
	dsl.Service("streaming", func() {
		dsl.Method("sse", func() {
			dsl.StreamingResult(dsl.String)
			dsl.HTTP(func() {
				dsl.GET("/sse")
				dsl.ServerSentEvents()
			})
		})
		dsl.Method("websocket", func() {
			dsl.StreamingResult(dsl.String)
			dsl.HTTP(func() {
				dsl.GET("/websocket")
			})
		})
	})
}

