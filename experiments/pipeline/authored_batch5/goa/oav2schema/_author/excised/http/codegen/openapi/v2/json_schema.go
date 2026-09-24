// This file builds the JSON schemas placed in one Swagger 2.0 document. Each
// build keeps its definitions and assigned type names in its own builder.
package openapiv2

import (
	_ "fmt"

	_ "example.internal/apikit/v3/codegen"
	"example.internal/apikit/v3/expr"
	"example.internal/apikit/v3/http/codegen/openapi"
)

type (
	// schemaBuilder builds every schema used by one Swagger document.
	schemaBuilder struct {
		definitions     map[string]*openapi.Schema
		definitionNames map[*expr.ResultTypeExpr]string
		values          openapi.Values
	}
)

// newSchemaBuilder starts a schema build with no definitions or assigned names.
func newSchemaBuilder(values openapi.Values) *schemaBuilder {
	panic("excised: newSchemaBuilder")
}

// BuildAttributeSchema returns the JSON schema for at. The returned schema
// includes every named definition referenced by at.
func BuildAttributeSchema(api *expr.APIExpr, at *expr.AttributeExpr, generator *expr.ExampleGenerator) *openapi.Schema {
	panic("excised: BuildAttributeSchema")
}

// resultTypeRefWithPrefix returns a reference to the requested result view. It
// adds the definition to this builder the first time the result is used.
func (b *schemaBuilder) resultTypeRefWithPrefix(api *expr.APIExpr, mt *expr.ResultTypeExpr, view, prefix string, gen *expr.ExampleGenerator) string {
	panic("excised: schemaBuilder.resultTypeRefWithPrefix")
}

// projectedResultTypeRefWithPrefix adds a result type that was already
// projected for one HTTP response.
func (b *schemaBuilder) projectedResultTypeRefWithPrefix(api *expr.APIExpr, source, projected *expr.ResultTypeExpr, prefix string, gen *expr.ExampleGenerator) string {
	panic("excised: schemaBuilder.projectedResultTypeRefWithPrefix")
}

// typeRefWithPrefix returns a reference to a user type. It adds the definition
// to this builder the first time the type is used.
func (b *schemaBuilder) typeRefWithPrefix(api *expr.APIExpr, ut *expr.UserTypeExpr, prefix string, gen *expr.ExampleGenerator) string {
	panic("excised: schemaBuilder.typeRefWithPrefix")
}

// generateResultTypeDefinition adds the requested result view unless this
// build already has a definition with the same name.
func (b *schemaBuilder) generateResultTypeDefinition(api *expr.APIExpr, mt *expr.ResultTypeExpr, view string, gen *expr.ExampleGenerator) {
	panic("excised: schemaBuilder.generateResultTypeDefinition")
}

// generateTypeDefinitionWithName adds the user type under typeName unless this
// build already has a definition with that name.
func (b *schemaBuilder) generateTypeDefinitionWithName(api *expr.APIExpr, ut *expr.UserTypeExpr, typeName string, gen *expr.ExampleGenerator) {
	panic("excised: schemaBuilder.generateTypeDefinitionWithName")
}

// typeSchema builds a schema for t and adds any named definitions it uses to
// this builder.
func (b *schemaBuilder) typeSchema(api *expr.APIExpr, t expr.DataType, gen *expr.ExampleGenerator) *openapi.Schema {
	panic("excised: schemaBuilder.typeSchema")
}

// typeSchemaWithPrefix builds a schema for t and adds prefix to new named
// definitions created while walking the type.
func (b *schemaBuilder) typeSchemaWithPrefix(api *expr.APIExpr, t expr.DataType, prefix string, gen *expr.ExampleGenerator) *openapi.Schema {
	panic("excised: schemaBuilder.typeSchemaWithPrefix")
}

// attributeTypeSchemaWithPrefix builds a schema for at, including its
// validation rules, and adds prefix to new named definitions.
func (b *schemaBuilder) attributeTypeSchemaWithPrefix(api *expr.APIExpr, at *expr.AttributeExpr, prefix string, gen *expr.ExampleGenerator) *openapi.Schema {
	panic("excised: schemaBuilder.attributeTypeSchemaWithPrefix")
}

// buildAttributeSchema fills schema with the type, example, description, and
// validation rules from at.
func (b *schemaBuilder) buildAttributeSchema(api *expr.APIExpr, schema *openapi.Schema, at *expr.AttributeExpr, gen *expr.ExampleGenerator) *openapi.Schema {
	panic("excised: schemaBuilder.buildAttributeSchema")
}

// initSchemaValidation copies the validation rules from at into schema.
func initSchemaValidation(schema *openapi.Schema, at *expr.AttributeExpr) {
	panic("excised: initSchemaValidation")
}

// renamedResultType returns rt with name without changing the design result.
func renamedResultType(rt *expr.ResultTypeExpr, name string) *expr.ResultTypeExpr {
	panic("excised: renamedResultType")
}

// buildResultTypeSchema fills schema with the requested result view.
func (b *schemaBuilder) buildResultTypeSchema(api *expr.APIExpr, mt *expr.ResultTypeExpr, view string, schema *openapi.Schema, gen *expr.ExampleGenerator) {
	panic("excised: schemaBuilder.buildResultTypeSchema")
}
