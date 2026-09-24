// This file defines result types and views and records the original
// declaration used when a result type is copied.
package expr

import (
	_ "fmt"
	_ "mime"
	_ "strings"

	_ "example.internal/apikit/v3/eval"
)

const (
	// DefaultView is the name of the default result type view.
	DefaultView = "default"

	// ViewMetaKey is the key used to store the view name in the attribute meta.
	ViewMetaKey = "view"
)

type (
	// ResultTypeExpr is a user type which describes views used to
	// render responses.
	ResultTypeExpr struct {
		// A result type is a user type
		*UserTypeExpr
		// Identifier is the RFC 6838 result type media type identifier.
		Identifier string
		// ContentType identifies the value written to the response
		// "Content-Type" header. Deprecated.
		ContentType string
		// Views list the supported views indexed by name.
		Views []*ViewExpr
		// origin is the earliest result type declaration copied to create this
		// result type.
		origin UserType
	}

	// ViewExpr defines which fields to render when building a response. The view
	// is an object whose field names must match the names of the parent result
	// type field names. The field definitions are inherited from the parent
	// result type but may be overridden.
	ViewExpr struct {
		// Set of properties included in view
		*AttributeExpr
		// Name of view
		Name string
		// Parent result Type
		Parent *ResultTypeExpr
	}
)

var (
	// ErrorResultIdentifier is the result type identifier used for error
	// responses.
	ErrorResultIdentifier = "application/vnd.apikit.error"

	// ErrorResult is the built-in result type for error responses.
	ErrorResult = &ResultTypeExpr{
		UserTypeExpr: &UserTypeExpr{
			AttributeExpr: &AttributeExpr{
				Type:        errorResultType,
				Description: "Error response result type",
				Validation:  &ValidationExpr{Required: []string{"name", "id", "message", "temporary", "timeout", "fault"}},
				finalized:   true,
			},
			TypeName: "error",
		},
		Identifier: ErrorResultIdentifier,
		Views:      []*ViewExpr{errorResultView},
	}

	errorResultType = &Object{
		{"name", &AttributeExpr{
			Type:         String,
			Description:  "Name is the name of this class of errors.",
			Meta:         MetaExpr{"struct:error:name": nil},
			UserExamples: []*ExampleExpr{{Value: "bad_request"}},
		}},
		{"id", &AttributeExpr{
			Type:         String,
			Description:  "ID is a unique identifier for this particular occurrence of the problem.",
			UserExamples: []*ExampleExpr{{Value: "123abc"}},
		}},
		{"message", &AttributeExpr{
			Type:         String,
			Description:  "Message is a human-readable explanation specific to this occurrence of the problem.",
			UserExamples: []*ExampleExpr{{Value: "parameter 'p' must be an integer"}},
		}},
		{"temporary", &AttributeExpr{
			Type:        Boolean,
			Description: "Is the error temporary?",
		}},
		{"timeout", &AttributeExpr{
			Type:        Boolean,
			Description: "Is the error a timeout?",
		}},
		{"fault", &AttributeExpr{
			Type:        Boolean,
			Description: "Is the error a server-side fault?",
		}},
	}

	errorResultView = &ViewExpr{
		AttributeExpr: &AttributeExpr{Type: errorResultType},
		Name:          DefaultView,
	}
)

// NewResultTypeExpr creates a result type definition but does not
// execute the DSL.
func NewResultTypeExpr(name, identifier string, fn func()) *ResultTypeExpr {
	panic("excised: NewResultTypeExpr")
}

// IsErrorResult reports whether dataType is Apikit's built-in service error type
// or a generator copy made from it.
func IsErrorResult(dataType DataType) bool {
	panic("excised: IsErrorResult")
}

// CanonicalIdentifier returns the result type identifier sans suffix
// which is what the DSL uses to store and lookup result types.
func CanonicalIdentifier(identifier string) string {
	panic("excised: CanonicalIdentifier")
}

// Kind implements DataKind.
func (*ResultTypeExpr) Kind() Kind { return ResultTypeKind }

// Dup creates a deep copy of the result type given a deep copy of its attribute.
func (rt *ResultTypeExpr) Dup(att *AttributeExpr) UserType {
	panic("excised: ResultTypeExpr.Dup")
}

// Origin returns the earliest result type declaration from which rt was
// copied. Result types override their embedded user-type origin so later copies
// still point to the original result declaration.
func (rt *ResultTypeExpr) Origin() UserType {
	panic("excised: ResultTypeExpr.Origin")
}

// ID returns the identifier of the result type.
func (rt *ResultTypeExpr) ID() string {
	panic("excised: ResultTypeExpr.ID")
}

// Name returns the result type name.
func (rt *ResultTypeExpr) Name() string { return rt.TypeName }

// Rename changes the result type name and starts a new generated declaration
// origin at rt.
func (rt *ResultTypeExpr) Rename(name string) {
	panic("excised: ResultTypeExpr.Rename")
}

// View returns the view with the given name.
func (rt *ResultTypeExpr) View(name string) *ViewExpr {
	panic("excised: ResultTypeExpr.View")
}

// HasMultipleViews returns true if the result type has more than one view.
func (rt *ResultTypeExpr) HasMultipleViews() bool {
	panic("excised: ResultTypeExpr.HasMultipleViews")
}

// ViewHasAttribute returns true if the result type view has the given
// attribute.
func (rt *ResultTypeExpr) ViewHasAttribute(view, attr string) bool {
	panic("excised: ResultTypeExpr.ViewHasAttribute")
}

// Finalize builds the default view if not explicitly defined and finalizes
// the underlying UserTypeExpr.
func (rt *ResultTypeExpr) Finalize() {
	panic("excised: ResultTypeExpr.Finalize")
}

// useExplicitView projects the result type using the view explicitly set on the
// attribute if any.
func (rt *ResultTypeExpr) useExplicitView() {
	panic("excised: ResultTypeExpr.useExplicitView")
}

// ensureDefaultView builds the default view if not explicitly defined.
func (rt *ResultTypeExpr) ensureDefaultView() {
	panic("excised: ResultTypeExpr.ensureDefaultView")
}

// Project creates a ResultTypeExpr containing the fields defined in the view
// expression of m named after the view argument.
//
// The resulting result type defines a default view. The result type identifier is
// computed by adding a parameter called "view" to the original identifier. The
// value of the "view" parameter is the name of the view.
//
// Project returns an error if the view does not exist for the given result type
// or any result type that makes up its attributes recursively. Note that
// individual attributes may use a different view. In this case Project uses
// that view and returns an error if it isn't defined on the attribute type.
func Project(rt *ResultTypeExpr, view string) (*ResultTypeExpr, error) {
	panic("excised: Project")
}

// project computes the projection of rt for view. seen memoizes projected
// types keyed by (type hash, view). It caches types only - never field
// attributes - so that sibling fields referencing the same type each keep
// their own AttributeExpr (description and meta would otherwise leak across
// fields). projectSingle registers its projection before computing the fields
// so that recursive references resolve to the in-flight projection and the
// recursion terminates.
func project(rt *ResultTypeExpr, view string, seen map[string]UserType) (*ResultTypeExpr, error) {
	panic("excised: project")
}

func projectSingle(rt *ResultTypeExpr, view string, seen map[string]UserType) (*ResultTypeExpr, error) {
	panic("excised: projectSingle")
}

func projectCollection(rt *ResultTypeExpr, view string, seen map[string]UserType) (*ResultTypeExpr, error) {
	panic("excised: projectCollection")
}

// projectedUserType makes a synthesized result type use the same repeatable
// example sequence as source. A view-specific type authored in the design keeps
// its media-type-derived UID instead.
func projectedUserType(source UserType, name, uid string, attribute *AttributeExpr) *UserTypeExpr {
	panic("excised: projectedUserType")
}

// projectRecursive computes the projected attribute for the field described
// by at within a result type being projected with view. vat is the matching
// view attribute. It always returns a fresh attribute: projected types are
// shared through seen but the attributes wrapping them never are, so that
// per-field metadata does not leak across fields of the same type.
func projectRecursive(at *AttributeExpr, vat *NamedAttributeExpr, view string, seen map[string]UserType) (*AttributeExpr, error) {
	panic("excised: projectRecursive")
}

// projectIdentifier computes the projected result type identifier by adding the
// "view" param. We need the projected result type identifier to be different so
// that looking up projected result types from ProjectedResultTypes works
// correctly. It's also good for clients.
func (rt *ResultTypeExpr) projectIdentifier(view string) string {
	panic("excised: ResultTypeExpr.projectIdentifier")
}

// EvalName returns the generic definition name used in error messages.
func (v *ViewExpr) EvalName() string {
	panic("excised: ViewExpr.EvalName")
}

// hashTypeAndView computes the projection cache key for the given type and
// view. Two types with the same key project to the same type for the view.
func hashTypeAndView(t DataType, view string) string {
	panic("excised: hashTypeAndView")
}
