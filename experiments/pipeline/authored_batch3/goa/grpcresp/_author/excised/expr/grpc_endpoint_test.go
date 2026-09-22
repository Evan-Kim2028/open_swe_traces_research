// This file verifies gRPC endpoint preparation and validation, including the
// native primitive contract required by request and response metadata.
package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"example.internal/apikit/v3/expr"
	"example.internal/apikit/v3/expr/testdata"
)


func TestGRPCEndpointStreamingPayloadKeepsInitialRequest(t *testing.T) {
	root := expr.RunDSL(t, testdata.GRPCEndpointWithStreamingPayloadInitialRequest)
	grpcSvc := root.API.GRPC.Service("Service")
	require.NotNil(t, grpcSvc)
	require.Len(t, grpcSvc.GRPCEndpoints, 1)

	endpoint := grpcSvc.GRPCEndpoints[0]
	req := expr.AsObject(endpoint.Request.Type)
	require.NotNil(t, req)
	require.NotNil(t, req.Attribute("repository_id"))
	require.NotNil(t, req.Attribute("version_ref"))
	require.True(t, endpoint.Metadata.IsEmpty())
}

func TestGRPCEndpointLegacyStreamCompat(t *testing.T) {
	cases := []struct {
		Name     string
		DSL      func()
		Expected bool
	}{
		{"method-level", testdata.GRPCEndpointStreamCompat, true},
		{"service-level", testdata.GRPCEndpointStreamCompatServiceLevel, true},
		{"not-set", testdata.GRPCEndpointWithStreamingPayloadInitialRequest, false},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := expr.RunDSL(t, c.DSL)
			grpcSvc := root.API.GRPC.Service("Service")
			require.NotNil(t, grpcSvc)
			require.Len(t, grpcSvc.GRPCEndpoints, 1)
			require.Equal(t, c.Expected, grpcSvc.GRPCEndpoints[0].LegacyStreamCompat())
		})
	}
}
