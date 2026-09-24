package openapi

import (
	_ "sort"
	_ "strings"

	"example.internal/apikit/v3/expr"
)

// Tag allows adding meta data to a single tag that is used by the Operation Object. It is
// not mandatory to have a Tag Object per tag used there.
type Tag struct {
	// Name of the tag.
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
	// Summary is a short summary of the tag (OpenAPI 3.2).
	Summary string `json:"summary,omitempty" yaml:"summary,omitempty"`
	// Description is a short description of the tag.
	// GFM syntax can be used for rich text representation.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	// Parent is the name of the parent tag (OpenAPI 3.2).
	Parent string `json:"parent,omitempty" yaml:"parent,omitempty"`
	// Kind is the kind of the tag, e.g. "nav" or "audience" (OpenAPI 3.2).
	Kind string `json:"kind,omitempty" yaml:"kind,omitempty"`
	// ExternalDocs is additional external documentation for this tag.
	ExternalDocs *ExternalDocs `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`
	// Extensions defines the OpenAPI extensions.
	Extensions map[string]any `json:"-" yaml:"-"`
}

// TagsFromExpr extracts the OpenAPI tag metadata from the given expression
// for the given specification version. The tag fields introduced by OpenAPI
// 3.2 (summary, parent, kind) are only extracted when ver is Version32 as
// earlier specification versions do not define them.
func TagsFromExpr(mdata expr.MetaExpr, ver Version) []*Tag {
	panic("excised: TagsFromExpr")
}

// TagNamesFromExpr computes the names of the OpenAPI tags specified in the
// given metadata expressions.
func TagNamesFromExpr(mdata expr.MetaExpr) (tagNames []string) {
	panic("excised: TagNamesFromExpr")
}

// parseTags builds the tags defined in the given metadata. extras enables the
// OpenAPI 3.2 only tag fields which must not appear in 2.0 and 3.0 documents.
func parseTags(mdata expr.MetaExpr, extras bool) (tags []*Tag) {
	panic("excised: parseTags")
}

type _tag Tag

// MarshalJSON returns the JSON encoding of t.
func (t Tag) MarshalJSON() ([]byte, error) {
	return MarshalJSON(_tag(t), t.Extensions)
}

// MarshalYAML returns value which marshaled in place of the original value
func (t Tag) MarshalYAML() (any, error) {
	return MarshalYAML(_tag(t), t.Extensions)
}
