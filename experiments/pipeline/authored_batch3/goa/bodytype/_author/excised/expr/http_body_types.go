// HTTP body type helpers derive request and response shapes from service types
// without changing the original design declarations.
package expr

import (
	_ "net/http"
	_ "strings"
	_ "unicode"
)

// defaultRequestHeaderAttributes returns a map keyed by the names of the
// payload attributes that should come from the request HTTP headers by default.
// This includes mapping done for certain authorization schemes (basic auth,
// Bearer, JWT, OAuth). The corresponding boolean value indicates whether the
// value maps directly to a payload attribute (true) or whether the value is used
// to compute the payload attribute (false). The only case where the value is
// computed by the generated code at this point is for basic authorization (the
// single "Authorization" header is used to compute both the username and
// password attributes).
func defaultRequestHeaderAttributes(e *HTTPEndpointExpr) map[string]bool {
	panic("excised: defaultRequestHeaderAttributes")
}

// httpRequestBody returns an attribute describing the HTTP request body of the
// given endpoint. If the DSL defines a body explicitly via the Body function
// then the corresponding attribute is used. Otherwise the attribute is computed
// by removing the attributes of the method payload used to define headers and
// parameters.
func httpRequestBody(a *HTTPEndpointExpr) *AttributeExpr {
	panic("excised: httpRequestBody")
}

// httpStreamingBody returns an attribute representing the structs being
// streamed via websocket.
func httpStreamingBody(e *HTTPEndpointExpr) *AttributeExpr {
	panic("excised: httpStreamingBody")
}

// httpResponseBody returns an attribute representing the HTTP response body for
// the given endpoint and response. If the DSL defines a body explicitly via the
// Body function then the corresponding attribute is used. Otherwise the
// attribute is computed by removing the attributes of the method payload used
// to define cookies and headers.
func httpResponseBody(a *HTTPEndpointExpr, resp *HTTPResponseExpr) *AttributeExpr {
	panic("excised: httpResponseBody")
}

// httpErrorResponseBody returns an attribute describing the response body of a
// given error. If the DSL defines a body explicitly via the Body function then
// the corresponding attribute is returned. Otherwise the attribute is computed
// by removing the attributes of the error used to define cookies, headers and
// parameters.
func httpErrorResponseBody(e *HTTPEndpointExpr, v *HTTPErrorExpr) *AttributeExpr {
	panic("excised: httpErrorResponseBody")
}

func buildHTTPResponseBody(name string, attr *AttributeExpr, resp *HTTPResponseExpr, identity ExampleIdentity) *AttributeExpr {
	panic("excised: buildHTTPResponseBody")
}

// generatedUserType preserves result-type behavior while giving a computed
// transport type its own original declaration and repeatable example sequence.
func generatedUserType(typ UserType, identity ExampleIdentity) UserType {
	panic("excised: generatedUserType")
}

// concat concatenates the given strings with "smart(?) casing".
// The concatenation algorithm is:
//
// 1) If the first string contains underscores and starts with a lower case,
// the rest of the strings are converted to lower case and concatenated with
// underscores.
// e.g. concat("my_endpoint", "Request", "BODY") => "my_endpoint_request_body"
//
// 2) If the first string contains underscores and starts with a upper case,
// the rest of the strings are converted to title case and concatenated with
// underscores.
// e.g. concat("My_endpoint", "response", "body") => "My_endpoint_Response_Body"
//
// 3) If the first string is a single word or camelcased, the rest of the
// strings are concatenated to form a valid upper camelcase.
// e.g. concat("myEndpoint", "streaming", "Body") => "MyEndpointStreamingBody"
// copyOpenAPITypeMeta copies the OpenAPI schema meta — type name, additional
// properties and extensions — from the original design type to the computed
// HTTP body type so the generated specs render them.
func copyOpenAPITypeMeta(from UserType, to *UserTypeExpr) {
	panic("excised: copyOpenAPITypeMeta")
}

func concat(strs ...string) string {
	panic("excised: concat")
}

func renameType(att *AttributeExpr, name string) {
	panic("excised: renameType")
}

// RemovePkgPath traverses the given data type and removes the "struct:pkg:path"
// metadata from all the user type attributes.
func RemovePkgPath(attr *AttributeExpr) {
	panic("excised: RemovePkgPath")
}

func removeAttributes(attr, sub *MappedAttributeExpr) {
	panic("excised: removeAttributes")
}

func removeAttribute(attr *MappedAttributeExpr, name string) {
	panic("excised: removeAttribute")
}

// extendBodyAttribute returns an attribute describing the HTTP
// request/response body type by merging any Bases and References to the parent
// attribute. This must be invoked during validation or to determine the actual
// body type by removing any headers/params/cookies.
func extendBodyAttribute(body *MappedAttributeExpr) {
	panic("excised: extendBodyAttribute")
}

// walk traverses the given data type and invokes the given function for each
// user type it finds including dt itself.
func walk(dt DataType, do func(UserType)) {
	panic("excised: walk")
}

func walkrec(dt DataType, do func(UserType), seen map[UserType]struct{}) {
	panic("excised: walkrec")
}
