// This file renders complete Swagger 2.0 documents from prepared HTTP designs
// and compares the JSON and YAML output produced with run-owned example state.
package openapiv2_test

import (
	_ "bytes"
	"errors"
	_ "fmt"
	_ "path/filepath"
	_ "testing"
	_ "text/template"

	"github.com/getkin/kin-openapi/openapi2"
	_ "github.com/stretchr/testify/require"

	_ "example.internal/apikit/v3/codegen/testutil"
	_ "example.internal/apikit/v3/dsl"
	_ "example.internal/apikit/v3/expr"
	_ "example.internal/apikit/v3/http/codegen/openapi"
	_ "example.internal/apikit/v3/http/codegen/openapi/v2"
	_ "example.internal/apikit/v3/http/codegen/testdata"
)





// validateSwagger asserts that the given bytes contain a valid Swagger spec.
func validateSwagger(b []byte) error {
	doc := &openapi2.T{}
	if err := doc.UnmarshalJSON(b); err != nil {
		return err
	}
	if doc.Swagger == "" {
		return errors.New("nil swagger")
	}
	return nil
}
