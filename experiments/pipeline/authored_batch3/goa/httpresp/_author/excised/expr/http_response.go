package expr

import (
	_ "fmt"
	_ "strings"

	"example.internal/apikit/v3/eval"
)

const (
	StatusContinue           = 100 // RFC 7231, 6.2.1
	StatusSwitchingProtocols = 101 // RFC 7231, 6.2.2
	StatusProcessing         = 102 // RFC 2518, 10.1

	StatusOK                   = 200 // RFC 7231, 6.3.1
	StatusCreated              = 201 // RFC 7231, 6.3.2
	StatusAccepted             = 202 // RFC 7231, 6.3.3
	StatusNonAuthoritativeInfo = 203 // RFC 7231, 6.3.4
	StatusNoContent            = 204 // RFC 7231, 6.3.5
	StatusResetContent         = 205 // RFC 7231, 6.3.6
	StatusPartialContent       = 206 // RFC 7233, 4.1
	StatusMultiStatus          = 207 // RFC 4918, 11.1
	StatusAlreadyReported      = 208 // RFC 5842, 7.1
	StatusIMUsed               = 226 // RFC 3229, 10.4.1

	StatusMultipleChoices  = 300 // RFC 7231, 6.4.1
	StatusMovedPermanently = 301 // RFC 7231, 6.4.2
	StatusFound            = 302 // RFC 7231, 6.4.3
	StatusSeeOther         = 303 // RFC 7231, 6.4.4
	StatusNotModified      = 304 // RFC 7232, 4.1
	StatusUseProxy         = 305 // RFC 7231, 6.4.5

	StatusTemporaryRedirect = 307 // RFC 7231, 6.4.7
	StatusPermanentRedirect = 308 // RFC 7538, 3

	StatusBadRequest                   = 400 // RFC 7231, 6.5.1
	StatusUnauthorized                 = 401 // RFC 7235, 3.1
	StatusPaymentRequired              = 402 // RFC 7231, 6.5.2
	StatusForbidden                    = 403 // RFC 7231, 6.5.3
	StatusNotFound                     = 404 // RFC 7231, 6.5.4
	StatusMethodNotAllowed             = 405 // RFC 7231, 6.5.5
	StatusNotAcceptable                = 406 // RFC 7231, 6.5.6
	StatusProxyAuthRequired            = 407 // RFC 7235, 3.2
	StatusRequestTimeout               = 408 // RFC 7231, 6.5.7
	StatusConflict                     = 409 // RFC 7231, 6.5.8
	StatusGone                         = 410 // RFC 7231, 6.5.9
	StatusLengthRequired               = 411 // RFC 7231, 6.5.10
	StatusPreconditionFailed           = 412 // RFC 7232, 4.2
	StatusRequestEntityTooLarge        = 413 // RFC 7231, 6.5.11
	StatusRequestURITooLong            = 414 // RFC 7231, 6.5.12
	StatusUnsupportedMediaType         = 415 // RFC 7231, 6.5.13
	StatusRequestedRangeNotSatisfiable = 416 // RFC 7233, 4.4
	StatusExpectationFailed            = 417 // RFC 7231, 6.5.14
	StatusTeapot                       = 418 // RFC 7168, 2.3.3
	StatusUnprocessableEntity          = 422 // RFC 4918, 11.2
	StatusLocked                       = 423 // RFC 4918, 11.3
	StatusFailedDependency             = 424 // RFC 4918, 11.4
	StatusUpgradeRequired              = 426 // RFC 7231, 6.5.15
	StatusPreconditionRequired         = 428 // RFC 6585, 3
	StatusTooManyRequests              = 429 // RFC 6585, 4
	StatusRequestHeaderFieldsTooLarge  = 431 // RFC 6585, 5
	StatusUnavailableForLegalReasons   = 451 // RFC 7725, 3

	StatusInternalServerError           = 500 // RFC 7231, 6.6.1
	StatusNotImplemented                = 501 // RFC 7231, 6.6.2
	StatusBadGateway                    = 502 // RFC 7231, 6.6.3
	StatusServiceUnavailable            = 503 // RFC 7231, 6.6.4
	StatusGatewayTimeout                = 504 // RFC 7231, 6.6.5
	StatusHTTPVersionNotSupported       = 505 // RFC 7231, 6.6.6
	StatusVariantAlsoNegotiates         = 506 // RFC 2295, 8.1
	StatusInsufficientStorage           = 507 // RFC 4918, 11.5
	StatusLoopDetected                  = 508 // RFC 5842, 7.2
	StatusNotExtended                   = 510 // RFC 2774, 7
	StatusNetworkAuthenticationRequired = 511 // RFC 6585, 6
)

const (
	RPCParseError     = -32700 // JSON-RPC 2.0, 5.1
	RPCInvalidRequest = -32600
	RPCMethodNotFound = -32601
	RPCInvalidParams  = -32602
	RPCInternalError  = -32603
)

type (
	// HTTPResponseExpr defines a HTTP response including its status code,
	// headers and result type.
	HTTPResponseExpr struct {
		// HTTP status
		StatusCode int
		// StatusCodeSet is true when the design explicitly chose StatusCode.
		// It distinguishes code zero from an omitted code.
		StatusCodeSet bool
		// Response description
		Description string
		// Headers describe the HTTP response headers.
		Headers *MappedAttributeExpr
		// Cookies describe the HTTP response cookies.
		Cookies *MappedAttributeExpr
		// Response body if any
		Body *AttributeExpr
		// Response Content-Type header value
		ContentType string
		// Tag the value a field of the result must have for this
		// response to be used.
		Tag [2]string
		// Parent expression, one of EndpointExpr, ServiceExpr or
		// RootExpr.
		Parent eval.Expression
		// Meta is a list of key/value pairs
		Meta MetaExpr
	}
)

// EvalName returns the generic definition name used in error messages.
func (r *HTTPResponseExpr) EvalName() string {
	panic("excised: HTTPResponseExpr.EvalName")
}

// Prepare makes sure the response is initialized even if not done explicitly
// by
func (r *HTTPResponseExpr) Prepare() {
	panic("excised: HTTPResponseExpr.Prepare")
}

// Validate checks that the response definition is consistent: its status is set
// and the result type definition if any is valid.
func (r *HTTPResponseExpr) Validate(e *HTTPEndpointExpr) *eval.ValidationErrors {
	panic("excised: HTTPResponseExpr.Validate")
}

// Finalize sets the response result type from its type if the type is a result
// type and no result type is already specified.
func (r *HTTPResponseExpr) Finalize(a *HTTPEndpointExpr, svcAtt *AttributeExpr) {
	panic("excised: HTTPResponseExpr.Finalize")
}

// Dup creates a copy of the response expression.
func (r *HTTPResponseExpr) Dup() *HTTPResponseExpr {
	panic("excised: HTTPResponseExpr.Dup")
}

// mapUnmappedAttrs maps any unmapped attributes in ErrorResult type to the
// response headers. Unmapped attributes refer to the attributes in ErrorResult
// type that are not mapped to response body or headers. Such unmapped
// attributes are mapped to special apikit headers in the form of
// "Apikit-Attribute(-<Attribute Name>)".
func (r *HTTPResponseExpr) mapUnmappedAttrs(svcAtt *AttributeExpr) {
	panic("excised: HTTPResponseExpr.mapUnmappedAttrs")
}

// bodyAllowedForStatus reports whether a given response status code
// permits a body. See RFC 2616, section 4.4.
// See https://golang.org/src/net/http/transfer.go
func bodyAllowedForStatus(status int) bool {
	panic("excised: bodyAllowedForStatus")
}
