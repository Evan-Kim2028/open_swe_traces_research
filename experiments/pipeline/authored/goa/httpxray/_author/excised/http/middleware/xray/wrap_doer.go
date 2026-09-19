package xray

import (
	"context"
	"net/http"

	goahttp "example.internal/apikit/v3/http"
	"example.internal/apikit/v3/middleware"
	"example.internal/apikit/v3/middleware/xray"
)

// xrayDoer is a goahttp.Doer middleware that will create xray subsegments for
// traced requests.
type xrayDoer struct {
	wrapped goahttp.Doer
}

// WrapDoer wraps a apikit HTTP Doer and creates xray subsegments for traced
// requests.
func WrapDoer(doer goahttp.Doer) goahttp.Doer { panic("excised: WrapDoer") }

// Do calls through to the wrapped Doer, creating subsegments as appropriate.
func (r *xrayDoer) Do(req *http.Request) (*http.Response, error) { panic("excised: Do") }

func _keepExcisedImports() {
	_ = context.Background
	_ = middleware.TraceIDKey
	_ = xray.SegKey
}
