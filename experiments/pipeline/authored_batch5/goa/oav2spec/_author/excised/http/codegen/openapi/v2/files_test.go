// This file renders complete Swagger 2.0 documents from prepared HTTP designs
// and compares the JSON and YAML output produced with run-owned example state.
package openapiv2_test

import (
	_ "bytes"
	"errors"
	_ "fmt"
	_ "path/filepath"
	"testing"
	_ "text/template"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/stretchr/testify/require"

	_ "example.internal/apikit/v3/codegen/testutil"
	"example.internal/apikit/v3/dsl"
	"example.internal/apikit/v3/expr"
	_ "example.internal/apikit/v3/http/codegen/openapi"
	openapiv2 "example.internal/apikit/v3/http/codegen/openapi/v2"
	_ "example.internal/apikit/v3/http/codegen/testdata"
)




func TestNamedPrimitiveParamsAndHeadersUseOpenAPIBaseTypes(t *testing.T) {
	root := expr.RunDSL(t, func() {
		var UUID = dsl.Type("UUID", dsl.String, func() {
			dsl.Format(dsl.FormatUUID)
		})
		var Time = dsl.Type("Time", dsl.String, func() {
			dsl.Format(dsl.FormatDateTime)
		})
		var UploadStatus = dsl.ResultType("application/vnd.upload-status", func() {
			dsl.Attributes(func() {
				dsl.Attribute("expires", Time, "RFC3339 expiration timestamp.")
				dsl.Attribute("offset", dsl.Int64, "Current upload offset in bytes.")
			})
		})

		dsl.API("test", func() {
			dsl.Server("test", func() {
				dsl.Host("localhost", func() {
					dsl.URI("http://localhost:80")
				})
			})
		})

		dsl.Service("repro", func() {
			dsl.Method("show", func() {
				dsl.Payload(func() {
					dsl.Attribute("ids", dsl.ArrayOf(UUID), "UUID filter values.")
				})
				dsl.Result(UploadStatus)
				dsl.HTTP(func() {
					dsl.GET("/repro")
					dsl.Param("ids")
					dsl.Response(dsl.StatusOK, func() {
						dsl.Header("expires:Upload-Expires")
						dsl.Header("offset:Upload-Offset")
						dsl.Body(dsl.Empty)
					})
				})
			})
		})
	})

	spec, err := openapiv2.NewV2(root, root.API.Servers[0].Hosts[0])
	require.NoError(t, err)

	path, ok := spec.Paths["/repro"]
	require.True(t, ok)

	get := path.(*openapiv2.Path).Get
	require.Len(t, get.Parameters, 1)
	require.Equal(t, "array", get.Parameters[0].Type)
	require.NotNil(t, get.Parameters[0].Items)
	require.Equal(t, "string", get.Parameters[0].Items.Type)
	require.Equal(t, "uuid", get.Parameters[0].Items.Format)

	headers := get.Responses["200"].Headers
	require.Equal(t, "string", headers["Upload-Expires"].Type)
	require.Equal(t, "date-time", headers["Upload-Expires"].Format)
	require.Equal(t, "integer", headers["Upload-Offset"].Type)
	require.Equal(t, "int64", headers["Upload-Offset"].Format)
}

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
