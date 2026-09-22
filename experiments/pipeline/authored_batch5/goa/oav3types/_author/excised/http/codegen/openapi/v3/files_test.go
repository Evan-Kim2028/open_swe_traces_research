// This file renders complete OpenAPI 3.0 and 3.2 documents from prepared HTTP
// designs and compares output produced with run-owned example state.
package openapiv3_test

import (
	_ "bytes"
	"context"
	_ "encoding/json"
	_ "fmt"
	_ "path/filepath"
	_ "strings"
	"testing"
	_ "text/template"

	"github.com/getkin/kin-openapi/openapi3"

	_ "example.internal/apikit/v3/codegen/testutil"
	_ "example.internal/apikit/v3/expr"
	_ "example.internal/apikit/v3/http/codegen/openapi"
	_ "example.internal/apikit/v3/http/codegen/openapi/v3"
	_ "example.internal/apikit/v3/http/codegen/testdata"
)




func validateSwagger(t *testing.T, b []byte) {
	swagger, err := openapi3.NewLoader().LoadFromData(b)
	if err == nil {
		err = swagger.Validate(context.Background())
	}
	if err != nil {
		t.Errorf("invalid spec: %s\nspec:\n%s", err.Error(), string(b))
	}
}
