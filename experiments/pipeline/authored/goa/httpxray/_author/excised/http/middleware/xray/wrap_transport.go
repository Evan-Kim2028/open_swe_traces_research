package xray

import (
	"context"
	"net/http"

	"example.internal/apikit/v3/middleware"
	"example.internal/apikit/v3/middleware/xray"
)

// xrayTransport wraps an http RoundTripper to add a tracing subsegment of the
// request's context segment.
type xrayTransport struct {
	wrapped http.RoundTripper
}

// WrapTransport wraps a http RoundTripper with a RoundTripper which creates
// subsegments of the segment in each request's context. The subsegments
// created this way have their namespace set to "remote". The request's ctx
// must be set and contain the current request segment as set by the xray
// middleware.
//
// Example of how to wrap http.Client's transport:
//
//	httpClient := &http.Client{
//	   Transport: WrapTransport(http.DefaultTransport),
//	}
func WrapTransport(rt http.RoundTripper) http.RoundTripper { panic("excised: WrapTransport") }

// RoundTrip wraps the original RoundTripper.RoundTrip to create xray tracing
// segments.
func (t *xrayTransport) RoundTrip(req *http.Request) (*http.Response, error) { panic("excised: RoundTrip") }

func _keepExcisedImports() {
	_ = context.Background
	_ = middleware.TraceIDKey
	_ = xray.SegKey
}
