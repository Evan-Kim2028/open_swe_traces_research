// Hidden black-box tests for unit "attachsvc".
//
// TestDetailNN numbers match DETAILS.md lines 1..9.
// Inferable:no lines assert shape only (presence/structure/relations),
// never the committed literal.

package expr_test

import (
	"strings"
	"testing"

	. "example.internal/apikit/v3/dsl"
	"example.internal/apikit/v3/expr"

	"github.com/stretchr/testify/require"
)

func bbAttachedSvc(name string) *expr.ServiceExpr {
	for _, s := range expr.Root.Services {
		if s.Name == name {
			return s
		}
	}
	return nil
}

// Detail 1 (Inferable: partially): each service must be a member of the
// design's service list, the design's name lookup must return the SAME
// service, and a service may not be provided twice. Assert foreign and
// duplicated services are rejected, members accepted.
func TestDetail01(t *testing.T) {
	expr.RunDSL(t, func() {
		Service("bbone", func() { Method("m", func() {}) })
		Service("bbtwo", func() { Method("m", func() {}) })
	})
	svc := bbAttachedSvc("bbone")
	require.NotNil(t, svc)
	require.NoError(t, expr.Root.EvaluateAttachedServices([]*expr.ServiceExpr{svc}))

	// a service object that is not in the design's list
	err := expr.Root.EvaluateAttachedServices(
		[]*expr.ServiceExpr{{Name: "bbone"}})
	require.Error(t, err, "same-name foreign service object rejected")

	// the same service provided twice
	err = expr.Root.EvaluateAttachedServices(
		[]*expr.ServiceExpr{svc, svc})
	require.Error(t, err, "duplicate service rejected")

	// a service absent from the design entirely
	err = expr.Root.EvaluateAttachedServices(
		[]*expr.ServiceExpr{{Name: "bbghost"}})
	require.Error(t, err, "non-member service rejected")
}

// Detail 2 (Inferable: partially): every method of a selected service
// must point back at that service.
func TestDetail02(t *testing.T) {
	expr.RunDSL(t, func() {
		Service("bbs", func() {
			Method("a", func() {})
			Method("b", func() {})
		})
	})
	svc := bbAttachedSvc("bbs")
	require.NoError(t, expr.Root.EvaluateAttachedServices([]*expr.ServiceExpr{svc}))
	for _, m := range svc.Methods {
		require.Same(t, svc, m.Service, "method %q bound back to service", m.Name)
	}
}

// Detail 3 (Inferable: no): each named type must be a member of the
// design's type list. Assert a member type is accepted and a foreign type
// is rejected.
func TestDetail03(t *testing.T) {
	expr.RunDSL(t, func() {
		Type("BbT", func() { Attribute("f", String) })
		Service("bbs", func() { Method("m", func() {}) })
	})
	svc := bbAttachedSvc("bbs")
	ut := expr.Root.UserType("BbT")
	require.NotNil(t, ut)
	require.NoError(t, expr.Root.EvaluateAttachedServices(
		[]*expr.ServiceExpr{svc}, ut))

	err := expr.Root.EvaluateAttachedServices(
		[]*expr.ServiceExpr{svc},
		&expr.UserTypeExpr{TypeName: "BbForeign",
			AttributeExpr: &expr.AttributeExpr{Type: expr.String}})
	require.Error(t, err, "non-member type rejected")
}

// Detail 4 (Inferable: no): HTTP collection keeps only transports whose
// ServiceExpr is selected and verifies each transport's Root equals the
// transport tree being collected (HTTP root vs the JSON-RPC embedded
// root). Assert selecting one service collects only its transports, for
// both plain HTTP and JSON-RPC services.
func TestDetail04(t *testing.T) {
	expr.RunDSL(t, func() {
		Service("bbsel", func() {
			Method("m", func() { HTTP(func() { GET("/bbsel") }) })
		})
		Service("bbother", func() {
			Method("m", func() { HTTP(func() { GET("/bbother") }) })
		})
		Service("bbj", func() {
			JSONRPC(func() { POST("/bbjrpc") })
			Method("jm", func() { JSONRPC(func() {}) })
		})
		Service("bbjother", func() {
			JSONRPC(func() { POST("/bbjrpc2") })
			Method("jm", func() { JSONRPC(func() {}) })
		})
	})
	require.NoError(t, expr.Root.EvaluateAttachedServices(
		[]*expr.ServiceExpr{bbAttachedSvc("bbsel")}))
	require.NoError(t, expr.Root.EvaluateAttachedServices(
		[]*expr.ServiceExpr{bbAttachedSvc("bbj")}))
}

// Detail 5 (Inferable: partially): every collected endpoint points back
// at its transport and uses a method declared on the transport's service;
// file servers likewise.
func TestDetail05(t *testing.T) {
	expr.RunDSL(t, func() {
		Service("bbs", func() {
			Method("m", func() { HTTP(func() { GET("/bbx") }) })
			Files("/bbstatic/f.txt", "/tmp/bbf.txt")
		})
	})
	svc := bbAttachedSvc("bbs")
	require.NoError(t, expr.Root.EvaluateAttachedServices([]*expr.ServiceExpr{svc}))

	var tsvc *expr.HTTPServiceExpr
	for _, s := range expr.Root.API.HTTP.Services {
		if s.ServiceExpr == svc {
			tsvc = s
		}
	}
	require.NotNil(t, tsvc)
	for _, e := range tsvc.HTTPEndpoints {
		require.Same(t, tsvc, e.Service, "endpoint points back at transport")
		require.Same(t, svc, e.MethodExpr.Service,
			"endpoint method declared on the transport's service")
	}
	for _, f := range tsvc.FileServers {
		require.Same(t, tsvc, f.Service, "file server points back at transport")
	}
}

// Detail 6 (Inferable: no): gRPC collection applies the same endpoint
// checks (no root check, no file servers). Assert a gRPC service
// evaluates and its endpoints point back correctly.
func TestDetail06(t *testing.T) {
	expr.RunDSL(t, func() {
		Service("bbg", func() {
			GRPC(func() {})
			Method("gm", func() { GRPC(func() {}) })
		})
	})
	svc := bbAttachedSvc("bbg")
	require.NoError(t, expr.Root.EvaluateAttachedServices([]*expr.ServiceExpr{svc}))

	var gsvc *expr.GRPCServiceExpr
	for _, s := range expr.Root.API.GRPC.Services {
		if s.ServiceExpr == svc {
			gsvc = s
		}
	}
	require.NotNil(t, gsvc)
	for _, e := range gsvc.GRPCEndpoints {
		require.Same(t, svc, e.MethodExpr.Service,
			"gRPC endpoint method declared on the transport's service")
	}
}

// Detail 7 (Inferable: no): set order — types, services, methods, HTTP
// services, HTTP endpoints, HTTP file servers, JSON-RPC services,
// JSON-RPC endpoints, JSON-RPC file servers, gRPC services, gRPC
// endpoints. Observable externally only through accumulated validation
// order; assert a type violation precedes a method violation when both
// members are invalid.
func TestDetail07(t *testing.T) {
	badType := &expr.UserTypeExpr{
		TypeName: "BbBad",
		AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{
			{Name: "bbn", Attribute: &expr.AttributeExpr{
				Type:         expr.Int,
				DefaultValue: "zzbad",
			}},
		}},
	}
	svc := &expr.ServiceExpr{Name: "bbs"}
	svc.Methods = []*expr.MethodExpr{{
		Name:    "m",
		Service: svc,
		Payload: &expr.AttributeExpr{Type: &expr.Object{
			{Name: "bbp", Attribute: &expr.AttributeExpr{
				Type:         expr.Int,
				DefaultValue: "zzbad",
			}},
		}},
	}}
	r := &expr.RootExpr{
		API: &expr.APIExpr{
			HTTP:    &expr.HTTPExpr{},
			GRPC:    &expr.GRPCExpr{},
			JSONRPC: &expr.JSONRPCExpr{},
		},
		Types:    []expr.UserType{badType},
		Services: []*expr.ServiceExpr{svc},
	}
	err := r.EvaluateAttachedServices([]*expr.ServiceExpr{svc}, badType)
	require.Error(t, err)
	msg := err.Error()
	it := strings.Index(msg, "bbn")
	im := strings.Index(msg, "bbp")
	require.GreaterOrEqual(t, it, 0, "type violation reported: %s", msg)
	require.GreaterOrEqual(t, im, 0, "method violation reported: %s", msg)
	require.Less(t, it, im, "types validated before methods")
}

// Detail 8 (Inferable: partially): selected services are bound to the
// owning root before the pipeline runs. Observable: evaluating through a
// different root rejects the service.
func TestDetail08(t *testing.T) {
	expr.RunDSL(t, func() {
		Service("bbs", func() { Method("m", func() {}) })
	})
	svc := bbAttachedSvc("bbs")
	other := &expr.RootExpr{
		API: &expr.APIExpr{
			HTTP:    &expr.HTTPExpr{},
			GRPC:    &expr.GRPCExpr{},
			JSONRPC: &expr.JSONRPCExpr{},
		},
	}
	require.Error(t, other.EvaluateAttachedServices([]*expr.ServiceExpr{svc}),
		"service from another design rejected")
	require.NoError(t, expr.Root.EvaluateAttachedServices([]*expr.ServiceExpr{svc}))
}

// Detail 9 (Inferable: partially): prepare every preparer in order, then
// validate — the ROOT itself validates first, errors accumulate rather
// than short-circuit — then finalize every finalizer. Assert a root-level
// violation is reported even when the selected service is clean, and that
// two root violations accumulate.
func TestDetail09(t *testing.T) {
	expr.RunDSL(t, func() {
		Type("BbT", func() { Attribute("f", String) })
		Service("bbs", func() { Method("m", func() {}) })
	})
	svc := bbAttachedSvc("bbs")
	ut := expr.Root.UserType("BbT")
	dup := ut.Dup(&expr.AttributeExpr{Type: expr.String})

	// two root-level violations, clean service -> both reported
	expr.Root.Conversions = []*expr.TypeMap{
		{User: ut, External: struct{ A string }{}},
		{User: dup, External: struct{ A string }{}},
	}
	expr.Root.Creations = []*expr.TypeMap{
		{User: ut, External: struct{ B string }{}},
		{User: dup, External: struct{ B string }{}},
	}
	err := expr.Root.EvaluateAttachedServices([]*expr.ServiceExpr{svc})
	require.Error(t, err, "root validates even when the service is clean")
	require.Equal(t, 2, strings.Count(err.Error(), "BbT"),
		"root violations accumulate: %s", err.Error())
}
