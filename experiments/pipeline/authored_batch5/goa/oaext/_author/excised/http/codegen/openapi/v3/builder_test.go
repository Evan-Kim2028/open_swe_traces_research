// This file verifies OpenAPI v3 operation construction, body schemas, and
// examples produced from evaluated HTTP endpoint designs.
package openapiv3

import (
	_ "encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"example.internal/apikit/v3/codegen"
	dsl "example.internal/apikit/v3/dsl"
	_ "example.internal/apikit/v3/expr"
	"example.internal/apikit/v3/http/codegen/openapi"
	_ "example.internal/apikit/v3/http/codegen/openapi/v3/testdata/dsls"
	_ "gopkg.in/yaml.v3"
)



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







func TestBuildServersKeepsVariableValuesSeparate(t *testing.T) {
	root := codegen.RunDSL(t, serverVariablesDSL)
	servers := buildServers(root.API.Servers, openapi.Version30, openapi.Values{})
	require.Len(t, servers, 1)

	region := servers[0].Variables["region"]
	require.Equal(t, []any{"west", "east"}, region.Enum)
	require.Equal(t, "west", region.Default)
	require.Equal(t, "Deployment region", region.Description)

	stage := servers[0].Variables["stage"]
	require.Equal(t, []any{"test", "production"}, stage.Enum)
	require.Equal(t, "production", stage.Default)
	require.Equal(t, "Deployment stage", stage.Description)
}

type param struct {
	Name        string
	In          string
	Description string
	Style       string
	Required    bool
	Type        typ
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

var serverVariablesDSL = func() {
	dsl.API("server variables", func() {
		dsl.Server("public", func() {
			dsl.Host("production", func() {
				dsl.URI("https://{region}.{stage}.example.com")
				dsl.Variable("region", dsl.String, "Deployment region", func() {
					dsl.Default("west")
					dsl.Enum("west", "east")
				})
				dsl.Variable("stage", dsl.String, "Deployment stage", func() {
					dsl.Default("production")
					dsl.Enum("test", "production")
				})
			})
		})
	})
	dsl.Service("status", func() {
		dsl.Method("read", func() {
			dsl.HTTP(func() {
				dsl.GET("/status")
			})
		})
	})
}

type requestBody struct {
	Description string
	Type        typ
	Required    bool
}

type response struct {
	Description string
	Type        typ
	Headers     map[string]param
}

type responses map[string]response



func matchesParameter(t *testing.T, p *ParameterRef, types map[string]*openapi.Schema, expected param) {
	matchesParameterHeader(t, p, types, expected, "parameter")
}
func matchesParameterHeader(t *testing.T, p *ParameterRef, types map[string]*openapi.Schema, expected param, title string) {
	if p.Value == nil {
		t.Errorf("no value for %s", title)
		return
	}
	if p.Ref != "" {
		t.Errorf("got ref %q for %s %q, expected none", p.Ref, title, p.Value.Name)
	}
	v := p.Value
	if v.Name != expected.Name {
		t.Errorf("got %s name %q, expected %q", title, v.Name, expected.Name)
	}
	if v.In != expected.In {
		t.Errorf("got %s in %q, expected %q", title, v.In, expected.In)
	}
	if v.Description != expected.Description {
		t.Errorf("got %s description %q, expected %q", title, v.Description, expected.Description)
	}
	if v.Style != expected.Style {
		t.Errorf("got %s style %q, expected %q", title, v.Style, expected.Style)
	}
	if v.Required != expected.Required {
		t.Errorf("got %s required %v, expected %v", title, v.Required, expected.Required)
	}
	matchesSchema(t, fmt.Sprintf("%s %q", title, v.Name), v.Schema, types, expected.Type)
	if v.Content != nil {
		t.Errorf("got content %#v, expected none", v.Content)
	}
}

func matchesRequestBody(t *testing.T, b *RequestBodyRef, types map[string]*openapi.Schema, expected *requestBody) {
	if b == nil {
		if expected != nil {
			t.Error("request body is nil")
		}
		return
	}
	if b.Value == nil {
		t.Error("no value for request body")
		return
	}
	if b.Ref != "" {
		t.Errorf("got ref %q for request body, expected none", b.Ref)
	}
	v := b.Value
	if v.Description != expected.Description {
		t.Errorf("got request body description %q, expected %q", v.Description, expected.Description)
	}
	if v.Required != expected.Required {
		t.Errorf("got request body required %v, expected %v", v.Required, expected.Required)
	}
	ct, ok := v.Content["application/json"]
	if !ok {
		t.Error("missing request content, expected application/json")
		return
	}
	matchesSchema(t, "request body", ct.Schema, types, expected.Type)
}

func matchesResponse(t *testing.T, r *ResponseRef, types map[string]*openapi.Schema, expected response) {
	if r.Value == nil {
		t.Error("no value for response")
		return
	}
	if r.Ref != "" {
		t.Errorf("got ref %q for response, expected none", r.Ref)
	}
	v := r.Value
	if v.Description == nil && expected.Description != "" {
		t.Errorf("got no response description, expected %q", expected.Description)
	} else if *v.Description != expected.Description {
		t.Errorf("got response description %q, expected %q", *v.Description, expected.Description)
	}
	if len(v.Headers) != len(expected.Headers) {
		t.Errorf("got %d response header(s), expected %d", len(v.Headers), len(expected.Headers))
		return
	}
	for n, h := range v.Headers {
		exp, ok := expected.Headers[n]
		if !ok {
			t.Errorf("response header %q not expected", n)
		}
		matchesHeader(t, h, types, exp)
	}
	if expected.Type.Type != "" {
		ct, ok := v.Content["application/json"]
		if !ok {
			t.Error("missing response content, expected application/json")
			return
		}
		matchesSchema(t, "response body", ct.Schema, types, expected.Type)
	}
}

func matchesHeader(t *testing.T, h *HeaderRef, types map[string]*openapi.Schema, expected param) {
	if h.Value == nil {
		t.Error("no value for header")
		return
	}
	if h.Ref != "" {
		t.Errorf("got ref %q for header, expected none", h.Ref)
	}
	v := h.Value
	par := &ParameterRef{Value: &Parameter{
		Description:     v.Description,
		Style:           v.Style,
		Explode:         v.Explode,
		AllowEmptyValue: v.AllowEmptyValue,
		AllowReserved:   v.AllowReserved,
		Deprecated:      v.Deprecated,
		Required:        v.Required,
		Schema:          v.Schema,
		Example:         v.Example,
		Examples:        v.Examples,
		Content:         v.Content,
		Extensions:      v.Extensions,
		In:              "header",
	}}
	matchesParameterHeader(t, par, types, expected, "header")
}

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
