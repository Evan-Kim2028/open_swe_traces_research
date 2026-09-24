// This file defines the JSON schema values shared by the OpenAPI generators.
// It also converts Apikit examples into the fields visible in those schemas.
package openapi

import (
	_ "encoding/base64"
	"encoding/json"
	"reflect"
	_ "strconv"

	"example.internal/apikit/v3/expr"
)

type (
	// Schema represents an instance of a JSON schema.
	// See http://json-schema.org/documentation.html
	Schema struct {
		Schema string `json:"$schema,omitempty" yaml:"$schema,omitempty"`
		// Core schema
		ID           string             `json:"id,omitempty" yaml:"id,omitempty"`
		Title        string             `json:"title,omitempty" yaml:"title,omitempty"`
		Type         Type               `json:"type,omitempty" yaml:"type,omitempty"`
		Items        *Schema            `json:"items,omitempty" yaml:"items,omitempty"`
		Properties   map[string]*Schema `json:"properties,omitempty" yaml:"properties,omitempty"`
		Defs         map[string]*Schema `json:"$defs,omitempty" yaml:"$defs,omitempty"`
		Description  string             `json:"description,omitempty" yaml:"description,omitempty"`
		DefaultValue any                `json:"default,omitempty" yaml:"default,omitempty"`
		Example      any                `json:"example,omitempty" yaml:"example,omitempty"`

		// Hyper schema
		Media     *Media  `json:"media,omitempty" yaml:"media,omitempty"`
		ReadOnly  bool    `json:"readOnly,omitempty" yaml:"readOnly,omitempty"`
		PathStart string  `json:"pathStart,omitempty" yaml:"pathStart,omitempty"`
		Links     []*Link `json:"links,omitempty" yaml:"links,omitempty"`
		Ref       string  `json:"$ref,omitempty" yaml:"$ref,omitempty"`

		// Validation
		Enum                 []any    `json:"enum,omitempty" yaml:"enum,omitempty"`
		Format               string   `json:"format,omitempty" yaml:"format,omitempty"`
		Pattern              string   `json:"pattern,omitempty" yaml:"pattern,omitempty"`
		ExclusiveMinimum     *float64 `json:"exclusiveMinimum,omitempty" yaml:"exclusiveMinimum,omitempty"`
		Minimum              *float64 `json:"minimum,omitempty" yaml:"minimum,omitempty"`
		ExclusiveMaximum     *float64 `json:"exclusiveMaximum,omitempty" yaml:"exclusiveMaximum,omitempty"`
		Maximum              *float64 `json:"maximum,omitempty" yaml:"maximum,omitempty"`
		MinLength            *int     `json:"minLength,omitempty" yaml:"minLength,omitempty"`
		MaxLength            *int     `json:"maxLength,omitempty" yaml:"maxLength,omitempty"`
		MinItems             *int     `json:"minItems,omitempty" yaml:"minItems,omitempty"`
		MaxItems             *int     `json:"maxItems,omitempty" yaml:"maxItems,omitempty"`
		Required             []string `json:"required,omitempty" yaml:"required,omitempty"`
		AdditionalProperties any      `json:"additionalProperties,omitempty" yaml:"additionalProperties,omitempty"`

		// Content (JSON Schema 2020-12), used by OpenAPI 3.2 documents to
		// describe string-encoded data such as JSON-encoded SSE event fields.
		ContentMediaType string  `json:"contentMediaType,omitempty" yaml:"contentMediaType,omitempty"`
		ContentSchema    *Schema `json:"contentSchema,omitempty" yaml:"contentSchema,omitempty"`

		// Union
		AnyOf []*Schema `json:"anyOf,omitempty" yaml:"anyOf,omitempty"`

		// Extensions defines the OpenAPI extensions.
		Extensions map[string]any `json:"-" yaml:"-"`
	}

	// Type is the JSON type enum.
	Type string

	// Media represents a "media" field in a JSON hyper schema.
	Media struct {
		BinaryEncoding string `json:"binaryEncoding,omitempty" yaml:"binaryEncoding,omitempty"`
		Type           string `json:"type,omitempty" yaml:"type,omitempty"`
	}

	// Link represents a "link" field in a JSON hyper schema.
	Link struct {
		Title        string  `json:"title,omitempty" yaml:"title,omitempty"`
		Description  string  `json:"description,omitempty" yaml:"description,omitempty"`
		Rel          string  `json:"rel,omitempty" yaml:"rel,omitempty"`
		Href         string  `json:"href,omitempty" yaml:"href,omitempty"`
		Method       string  `json:"method,omitempty" yaml:"method,omitempty"`
		Schema       *Schema `json:"schema,omitempty" yaml:"schema,omitempty"`
		TargetSchema *Schema `json:"targetSchema,omitempty" yaml:"targetSchema,omitempty"`
		ResultType   string  `json:"mediaType,omitempty" yaml:"mediaType,omitempty"`
		EncType      string  `json:"encType,omitempty" yaml:"encType,omitempty"`
	}

	// These types are used in marshalJSON() to avoid recursive call of json.Marshal().
	_Schema Schema
)

const (
	// Array represents a JSON array.
	Array Type = "array"
	// Boolean represents a JSON boolean.
	Boolean = "boolean"
	// Integer represents a JSON number without a fraction or exponent part.
	Integer = "integer"
	// Number represents any JSON number. Number includes integer.
	Number = "number"
	// Null represents the JSON null value.
	Null = "null"
	// Object represents a JSON object.
	Object = "object"
	// String represents a JSON string.
	String = "string"
	// File is an extension used by OpenAPI to represent a file download.
	File = "file"
)

// SchemaRef is the JSON Schema draft 2020-12 meta-schema identifier.
const SchemaRef = "https://json-schema.org/draft/2020-12/schema"

// NewSchema instantiates a new JSON schema.
func NewSchema() *Schema {
	js := Schema{
		Properties: make(map[string]*Schema),
		Defs:       make(map[string]*Schema),
	}
	return &js
}

// JSON serializes the schema into JSON. It makes sure the "$schema" standard
// field is set if needed prior to delegating to the standard JSON marshaler.
func (s *Schema) JSON() ([]byte, error) {
	if s.Ref == "" {
		s.Schema = SchemaRef
	}
	return json.Marshal(s)
}

// ToString returns the string representation of the given type.
func ToString(val any) string {
	panic("excised: ToString")
}

// ToStringMap converts map[any]any to a map[string]any
// when possible.
func ToStringMap(val any) any {
	panic("excised: ToStringMap")
}

// Example returns an example value projected to the OpenAPI-visible shape of at.
func Example(at *expr.AttributeExpr, r *expr.ExampleGenerator) any {
	return ProjectExample(at, at.Example(r))
}

// ProjectExample removes values for fields that are hidden from the OpenAPI
// schema by metadata while keeping examples usable by service codegen.
func ProjectExample(at *expr.AttributeExpr, val any) any {
	panic("excised: ProjectExample")
}

// MarshalJSON returns the JSON encoding of s.
func (s *Schema) MarshalJSON() ([]byte, error) {
	return MarshalJSON((*_Schema)(s), s.Extensions)
}

// MarshalYAML returns value which marshaled in place of the original value
func (s *Schema) MarshalYAML() (any, error) {
	return MarshalYAML((*_Schema)(s), s.Extensions)
}

// Dup returns an independent copy of the schema. Callers may change any nested
// schema, collection, example, default, or extension without changing s.
func (s *Schema) Dup() *Schema {
	panic("excised: Schema.Dup")
}

// MustGenerate returns true if the meta indicates that a OpenAPI specification should be
// generated, false otherwise.
func MustGenerate(meta expr.MetaExpr) bool {
	panic("excised: MustGenerate")
}

// AdditionalPropertiesFromExpr extracts the OpenAPI additionalProperties.
func AdditionalPropertiesFromExpr(meta expr.MetaExpr) any {
	panic("excised: AdditionalPropertiesFromExpr")
}

func projectExample(t expr.DataType, val any) any {
	panic("excised: projectExample")
}

// duplicateJSONValue copies the maps, slices, arrays, pointers, and interface
// values accepted by JSON fields while preserving their concrete Go types.
func duplicateJSONValue(value any) any {
	panic("excised: duplicateJSONValue")
}

// duplicateJSONReflectValue recursively copies one reflected JSON value.
func duplicateJSONReflectValue(value reflect.Value) reflect.Value {
	panic("excised: duplicateJSONReflectValue")
}

// duplicatePointer copies one scalar schema limit while preserving nil.
func duplicatePointer[T any](value *T) *T {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func projectObjectExample(obj *expr.Object, val any) any {
	panic("excised: projectObjectExample")
}

func projectArrayExample(array *expr.Array, val any) any {
	panic("excised: projectArrayExample")
}

func projectMapExample(m *expr.Map, val any) any {
	panic("excised: projectMapExample")
}

func exampleMap(val any) (map[string]any, bool) {
	panic("excised: exampleMap")
}

func exampleSlice(val any) ([]any, bool) {
	panic("excised: exampleSlice")
}
