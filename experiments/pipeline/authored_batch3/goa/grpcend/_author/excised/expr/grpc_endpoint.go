// This file prepares, validates, and finalizes the gRPC transport contract for
// one service method.
package expr

import (
	_ "fmt"
	_ "slices"

	"example.internal/apikit/v3/eval"
)

type (
	// GRPCEndpointExpr describes a gRPC endpoint. It embeds a MethodExpr
	// and adds gRPC specific properties.
	GRPCEndpointExpr struct {
		eval.DSLFunc
		// MethodExpr is the underlying method expression.
		MethodExpr *MethodExpr
		// Service is the parent service.
		Service *GRPCServiceExpr
		// Request is the message passed to the gRPC method.
		Request *AttributeExpr
		// StreamingRequest is the message passed to the gRPC method through a
		// stream.
		StreamingRequest *AttributeExpr
		// Responses is the success gRPC response from the method.
		Response *GRPCResponseExpr
		// GRPCErrors is the list of all the possible error gRPC responses.
		GRPCErrors []*GRPCErrorExpr
		// Metadata is the metadata to be sent in a gRPC request.
		Metadata *MappedAttributeExpr
		// Requirements is the list of security requirements for the gRPC endpoint.
		Requirements []*SecurityExpr
		// Meta is a set of key/value pairs with semantic that is
		// specific to each generator, see dsl.Meta.
		Meta MetaExpr
	}
)

const (
	// streamCompatMetaKey is the meta key that makes generated servers also
	// accept clients speaking the legacy stream protocol which predates the
	// typed stream envelope.
	streamCompatMetaKey = "grpc:stream:compat"
	// streamCompatLegacy is the only supported value of the stream
	// compatibility meta and selects the legacy metadata-based protocol.
	streamCompatLegacy = "v1"
)

// Name of gRPC endpoint
func (e *GRPCEndpointExpr) Name() string {
	panic("excised: GRPCEndpointExpr.Name")
}

// Description of gRPC endpoint
func (e *GRPCEndpointExpr) Description() string {
	panic("excised: GRPCEndpointExpr.Description")
}

// EvalName returns the generic expression name used in error messages.
func (e *GRPCEndpointExpr) EvalName() string {
	panic("excised: GRPCEndpointExpr.EvalName")
}

// Prepare initializes the Request and Response if nil.
func (e *GRPCEndpointExpr) Prepare() {
	panic("excised: GRPCEndpointExpr.Prepare")
}

// LegacyStreamCompat reports whether the generated server must also accept
// clients that speak the legacy stream protocol which carries the one-shot
// method payload in gRPC request metadata instead of a typed initial stream
// frame. It is enabled by setting Meta("grpc:stream:compat", "v1") on the
// method, the service or the API.
func (e *GRPCEndpointExpr) LegacyStreamCompat() bool {
	panic("excised: GRPCEndpointExpr.LegacyStreamCompat")
}

// Validate validates the endpoint expression by checking if the request
// and responses contains the "rpc:tag" in the meta. It also makes sure
// that there is only one response per status code.
func (e *GRPCEndpointExpr) Validate() error {
	panic("excised: GRPCEndpointExpr.Validate")
}

// validateErrorMappings ensures inherited gRPC response policy describes the
// same concrete error value returned by the endpoint method.
func (e *GRPCEndpointExpr) validateErrorMappings() *eval.ValidationErrors {
	panic("excised: GRPCEndpointExpr.validateErrorMappings")
}

func validateGRPCUnionShapes(att *AttributeExpr, parent eval.Expression, verr *eval.ValidationErrors, seenUnions map[*Union]struct{}, seenAttrs map[*AttributeExpr]struct{}) {
	panic("excised: validateGRPCUnionShapes")
}

// Finalize ensures the request and response attributes are initialized.
func (e *GRPCEndpointExpr) Finalize() {
	panic("excised: GRPCEndpointExpr.Finalize")
}

// validateMessage validates the gRPC message. It compares the given message
// with the service type (Payload or Result) and ensures all the attributes
// defined in the message type are found in the service type and the attributes
// are set with unique "rpc:tag" numbers.
//
// msgAtt is the Request/Response message attribute. validateMessage assumes
// that the msgAtt is not Empty.
// serviceAtt is the Payload/Result attribute.
// e is the endpoint expression.
// req if true indicates the Request message is being validated.
func validateMessage(msgAtt, serviceAtt *AttributeExpr, e *GRPCEndpointExpr, req bool) *eval.ValidationErrors {
	panic("excised: validateMessage")
}

// validateRPCTags verifies whether every attribute in the object type has
// "rpc:tag" set in the meta and the tag numbers are unique.
func validateRPCTags(fields *Object, e *GRPCEndpointExpr) *eval.ValidationErrors {
	panic("excised: validateRPCTags")
}

// validateMetadata validates the gRPC metadata. It compares the given metadata
// with the service type (Payload or Result) and ensures all the attributes
// defined in the metadata type are found in the service type.
//
// metAtt is the Request/Response metadata attribute. validateMetadata assumes
// that the metAtt is not Empty.
// serviceAtt is the Payload/Result attribute.
// e is the endpoint expression.
// req if true indicates the Request metadata is being validated.
func validateMetadata(metAtt *MappedAttributeExpr, serviceAtt *AttributeExpr, e *GRPCEndpointExpr, req bool) *eval.ValidationErrors {
	panic("excised: validateMetadata")
}

// getSecurityAttributes returns the attributes that describes a security
// scheme from a method expression.
func getSecurityAttributes(m *MethodExpr) []string {
	panic("excised: getSecurityAttributes")
}

// streamCompatValue returns the value of the stream compatibility meta by
// looking up the endpoint, method, service and API expressions in that order.
func (e *GRPCEndpointExpr) streamCompatValue() (string, bool) {
	panic("excised: GRPCEndpointExpr.streamCompatValue")
}

// validateStreamCompat validates the stream compatibility meta if set. The
// legacy stream protocol carries the one-shot method payload in gRPC metadata
// which can only encode primitive values and arrays of primitive values.
func (e *GRPCEndpointExpr) validateStreamCompat() *eval.ValidationErrors {
	panic("excised: GRPCEndpointExpr.validateStreamCompat")
}

// isMetadataEncodable reports whether values of the given type can be carried
// in gRPC metadata, that is whether they can be encoded to and decoded from
// header strings.
func isMetadataEncodable(dt DataType) bool {
	panic("excised: isMetadataEncodable")
}
