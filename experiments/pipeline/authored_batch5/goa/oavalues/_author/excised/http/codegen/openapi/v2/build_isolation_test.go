// This file checks that Swagger builds do not share or change each other's schemas.
package openapiv2_test

import (
	_ "encoding/json"
	_ "sync"
	_ "testing"

	_ "github.com/stretchr/testify/require"

	"example.internal/apikit/v3/dsl"
	_ "example.internal/apikit/v3/expr"
	"example.internal/apikit/v3/http/codegen/openapi"
	_ "example.internal/apikit/v3/http/codegen/openapi/v2"
)



// definitionWithProperty finds the returned schema that contains property.
func definitionWithProperty(definitions map[string]*openapi.Schema, property string) *openapi.Schema {
	for _, definition := range definitions {
		if _, ok := definition.Properties[property]; ok {
			return definition
		}
	}
	return nil
}

// schemaBuildDSL returns an API whose Shared result contains only field.
func schemaBuildDSL(field string) func() {
	return func() {
		shared := dsl.Type("Shared", func() {
			dsl.Attribute(field, dsl.String)
		})
		dsl.API("test", func() {
			dsl.Server("test", func() {
				dsl.Host("localhost", func() {
					dsl.URI("https://goa.design")
				})
			})
		})
		dsl.Service("testService", func() {
			dsl.Method("show", func() {
				dsl.Result(shared)
				dsl.HTTP(func() {
					dsl.GET("/")
				})
			})
		})
	}
}
