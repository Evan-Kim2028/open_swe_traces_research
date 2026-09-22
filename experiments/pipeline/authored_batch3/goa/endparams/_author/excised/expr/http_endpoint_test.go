package expr_test

import (
	"testing"

	"example.internal/apikit/v3/dsl"
	"example.internal/apikit/v3/expr"
	"example.internal/apikit/v3/expr/testdata"
)


func TestHTTPEndpointPrepare(t *testing.T) {
	cases := map[string]struct {
		DSL     func()
		Headers []string
		Params  []string
		Cookies []string
		Error   string
	}{
		"valid": {
			DSL:    testdata.ValidRouteDSL,
			Params: []string{"base_id", "id"},
		},
		"with parent": {
			DSL:     testdata.EndpointWithParentDSL,
			Headers: []string{"pheader", "header"},
			Params:  []string{"pparam", "param"},
			Cookies: []string{"pcookie", "cookie"},
		},
		"with parent revert": {
			DSL:     testdata.EndpointWithParentRevertDSL,
			Headers: []string{"pheader", "header"},
			Params:  []string{"pparam", "param"},
			Cookies: []string{"pcookie", "cookie"},
		},
		"error": {
			DSL:   testdata.EndpointRecursiveParentDSL,
			Error: "service \"Parent\": Parent service Child is also child\nservice \"Child\": Parent service Parent is also child",
		},
	}
	for n, c := range cases {
		t.Run(n, func(t *testing.T) {
			if c.Error == "" {
				root := expr.RunDSL(t, c.DSL)
				e := root.API.HTTP.Services[len(root.API.HTTP.Services)-1].HTTPEndpoints[0]

				ht := expr.AsObject(e.Headers.Type)
				if len(*ht) != len(c.Headers) {
					t.Errorf("got %d headers, expected %d", len(*ht), len(c.Headers))
				} else {
					for _, n := range c.Headers {
						if ht.Attribute(n) == nil {
							t.Errorf("header %q is missing", n)
						}
					}
				}

				ct := expr.AsObject(e.Cookies.Type)
				if len(*ct) != len(c.Cookies) {
					t.Errorf("got %d cookies, expected %d", len(*ct), len(c.Cookies))
				} else {
					for _, n := range c.Cookies {
						if ct.Attribute(n) == nil {
							t.Errorf("cookie %q is missing", n)
						}
					}
				}

				pt := expr.AsObject(e.Params.Type)
				if len(*pt) != len(c.Params) {
					t.Errorf("got %d params, expected %d", len(*pt), len(c.Params))
				} else {
					for _, n := range c.Params {
						if pt.Attribute(n) == nil {
							t.Errorf("param %q is missing", n)
						}
					}
				}
			} else {
				err := expr.RunInvalidDSL(t, c.DSL)
				got := stripValidationLocations(err.Error())
				if got != c.Error {
					t.Errorf("got error %q, expected %q", got, c.Error)
				}
			}
		})
	}
}


func TestHTTPWebSocketViewedResultValidation(t *testing.T) {
	tests := []struct {
		name           string
		collection     bool
		collectionView string
		method         func(*expr.ResultTypeExpr)
		err            string
	}{
		{
			name: "client stream with caller-selected view",
			method: func(result *expr.ResultTypeExpr) {
				dsl.StreamingPayload(dsl.String)
				dsl.Result(result)
			},
			err: `service "Service" HTTP endpoint "Method": Endpoint cannot choose a result view at runtime when the method defines StreamingPayload because the WebSocket connection starts before the result view is known. Select a view in Result or StreamingResult.`,
		},
		{
			name: "bidirectional stream with caller-selected view",
			method: func(result *expr.ResultTypeExpr) {
				dsl.StreamingPayload(dsl.String)
				dsl.StreamingResult(result)
			},
			err: `service "Service" HTTP endpoint "Method": Endpoint cannot choose a result view at runtime when the method defines StreamingPayload because the WebSocket connection starts before the result view is known. Select a view in Result or StreamingResult.`,
		},
		{
			name:       "client stream with caller-selected collection view",
			collection: true,
			method: func(result *expr.ResultTypeExpr) {
				dsl.StreamingPayload(dsl.String)
				dsl.Result(result)
			},
			err: `service "Service" HTTP endpoint "Method": Endpoint cannot choose a result view at runtime when the method defines StreamingPayload because the WebSocket connection starts before the result view is known. Select a view in Result or StreamingResult.`,
		},
		{
			name: "client stream with fixed view",
			method: func(result *expr.ResultTypeExpr) {
				dsl.StreamingPayload(dsl.String)
				dsl.Result(result, func() {
					dsl.View("tiny")
				})
			},
		},
		{
			name: "bidirectional stream with fixed view",
			method: func(result *expr.ResultTypeExpr) {
				dsl.StreamingPayload(dsl.String)
				dsl.StreamingResult(result, func() {
					dsl.View("tiny")
				})
			},
		},
		{
			name:       "client stream with fixed collection view",
			collection: true,
			method: func(result *expr.ResultTypeExpr) {
				dsl.StreamingPayload(dsl.String)
				dsl.Result(result, func() {
					dsl.View("tiny")
				})
			},
		},
		{
			name:           "client stream with view fixed by collection type",
			collection:     true,
			collectionView: "tiny",
			method: func(result *expr.ResultTypeExpr) {
				dsl.StreamingPayload(dsl.String)
				dsl.Result(result)
			},
		},
		{
			name: "server stream with caller-selected view",
			method: func(result *expr.ResultTypeExpr) {
				dsl.StreamingResult(result)
			},
		},
		{
			name: "client stream without views",
			method: func(*expr.ResultTypeExpr) {
				dsl.StreamingPayload(dsl.String)
				dsl.Result(dsl.String)
			},
		},
		{
			name: "bidirectional stream without views",
			method: func(*expr.ResultTypeExpr) {
				dsl.StreamingPayload(dsl.String)
				dsl.StreamingResult(dsl.String)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			design := httpWebSocketViewedResultDSL(test.collection, test.collectionView, test.method)
			if test.err == "" {
				expr.RunDSL(t, design)
				return
			}
			err := expr.RunInvalidDSL(t, design)
			if got := stripValidationLocations(err.Error()); got != test.err {
				t.Errorf("got %q, expected %q", got, test.err)
			}
		})
	}
}

// httpWebSocketViewedResultDSL defines a result with two response shapes and
// lets each test choose how the method streams it.
func httpWebSocketViewedResultDSL(collection bool, collectionView string, method func(*expr.ResultTypeExpr)) func() {
	return func() {
		result := dsl.ResultType("application/vnd.websocket-view", func() {
			dsl.Attribute("name", dsl.String)
			dsl.View("tiny", func() {
				dsl.Attribute("name")
			})
		})
		if collection {
			if collectionView == "" {
				result = dsl.CollectionOf(result)
			} else {
				result = dsl.CollectionOf(result, func() {
					dsl.View(collectionView)
				})
			}
		}
		dsl.Service("Service", func() {
			dsl.Method("Method", func() {
				method(result)
				dsl.HTTP(func() {
					dsl.GET("/")
				})
			})
		})
	}
}

func TestHTTPEndpointParentRequired(t *testing.T) {
	root := expr.RunDSL(t, testdata.EndpointHasParent)
	svc := root.Service("Child")
	if svc == nil {
		t.Fatal(`unexpected error, service "Child" not found`)
	}
	m := svc.Method("Method")
	if m == nil || m.Payload == nil {
		t.Fatal(`unexpected error, method "Method" or its payload not found`)
	}
	if !m.Payload.IsRequired("ancestor_id") {
		t.Errorf(`expected "ancestor_id" is required, but not so`)
	}
	if !m.Payload.IsRequired("parent_id") {
		t.Errorf(`expected "parent_id" is required, but not so`)
	}
}

func TestHTTPEndpointFinalization(t *testing.T) {
	cases := map[string]struct {
		DSL          func()
		ExpectedBody expr.DataType
	}{
		"body-as-extend-type": {
			DSL:          testdata.FinalizeEndpointBodyAsExtendedTypeDSL,
			ExpectedBody: testdata.FinalizeEndpointBodyAsExtendedType,
		},
		"body-as-prop-with-extend-type": {
			DSL:          testdata.FinalizeEndpointBodyAsPropWithExtendedTypeDSL,
			ExpectedBody: testdata.FinalizeEndpointBodyAsPropWithExtendedType,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			root := expr.RunDSL(t, tc.DSL)
			e := root.API.HTTP.Services[0].HTTPEndpoints[0]

			if tc.ExpectedBody != nil {
				if e.Body == nil {
					t.Errorf("got endpoint without body, expected endpoint with body")
					return
				}
				bodyObj := *expr.AsObject(e.Body.Type)
				expectedBodyObj := *expr.AsObject(tc.ExpectedBody)
				if len(bodyObj) != len(expectedBodyObj) {
					t.Errorf("got %d, expected %d attribute(s) in endpoint body", len(bodyObj), len(expectedBodyObj))
				} else {
					for i := range expectedBodyObj {
						if bodyObj[i].Name != expectedBodyObj[i].Name {
							t.Errorf("got %q, expected %q attribute in endpoint body", bodyObj[i].Name, expectedBodyObj[i].Name)
						}
					}
				}
			}
		})
	}
}

func TestHTTPAuthorizationMapping(t *testing.T) {
	cases := []struct {
		Name           string
		DSL            func()
		ExpectedHeader string
	}{{
		Name:           "explicit",
		DSL:            testdata.ExplicitAuthHeaderDSL,
		ExpectedHeader: "token",
	}, {
		Name:           "implicit",
		DSL:            testdata.ImplicitAuthHeaderDSL,
		ExpectedHeader: "Authorization",
	},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			root := expr.RunDSL(t, tc.DSL)
			e := root.API.HTTP.Services[0].HTTPEndpoints[0]
			if e.Headers == nil {
				t.Errorf("got endpoint without header, expected endpoint with HTTP header")
				return
			}
			if len(*expr.AsObject(e.Headers.Type)) != 1 {
				t.Errorf("got %d, expected 1 attribute in endpoint headers", len(*expr.AsObject(e.Headers.Type)))
				return
			}
			n := e.Headers.ElemName("token")
			if n != tc.ExpectedHeader {
				t.Errorf("got %q, expected %q attribute in endpoint headers", n, tc.ExpectedHeader)
			}
		})
	}
}
