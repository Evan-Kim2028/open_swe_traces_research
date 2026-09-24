// This file verifies HTTP client CLI generation consumes stable, non-empty
// examples for body, parameter, header, cookie, array, and map flags.
package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"example.internal/apikit/v3/codegen"
	"example.internal/apikit/v3/codegen/cli"
	ctestdata "example.internal/apikit/v3/codegen/example/testdata"
	"example.internal/apikit/v3/codegen/testutil"
	"example.internal/apikit/v3/dsl"
	"example.internal/apikit/v3/expr"
	"example.internal/apikit/v3/http/codegen/testdata"
)


func TestClientCLIFilesOmitFilesOnlyServer(t *testing.T) {
	root := codegen.RunDSL(t, ctestdata.ServerHostingServiceWithFileServerDSL)
	plan := linkedHTTPPlanForRoot(t, root)

	require.NotEmpty(t, plan.ServerFiles())
	require.Empty(t, plan.ClientCLIFiles())
}

// TestClientCLIFlagPresenceGolden shows how generated HTTP commands preserve
// explicit empty values while applying every authored zero-valued default.
func TestClientCLIFlagPresenceGolden(t *testing.T) {
	root := expr.RunDSL(t, func() {
		dsl.Service("FlagPresence", func() {
			dsl.Method("check", func() {
				dsl.Payload(func() {
					dsl.Field(1, "required", dsl.String)
					dsl.Field(2, "optional", dsl.String)
					dsl.Field(3, "empty", dsl.String, func() {
						dsl.Default("")
					})
					dsl.Field(4, "disabled", dsl.Boolean, func() {
						dsl.Default(false)
					})
					dsl.Field(5, "zero", dsl.Int, func() {
						dsl.Default(0)
					})
					dsl.Required("required")
				})
				dsl.HTTP(func() {
					dsl.GET("/")
					dsl.Param("required")
					dsl.Param("optional")
					dsl.Param("empty")
					dsl.Param("disabled")
					dsl.Param("zero")
				})
			})
		})
	})
	files := linkedHTTPPlanForRoot(t, root).ClientCLIFiles()
	require.Len(t, files, 2)

	parser := codegen.SectionsCode(t, files[0].Section("parse-endpoint"))
	testutil.AssertGo(t, "testdata/golden/client_cli_flag-presence-parse.go.golden", parser)
	builder := codegen.SectionsCode(t, files[1].Section("cli-build-payload"))
	testutil.AssertGo(t, "testdata/golden/client_cli_flag-presence-build.go.golden", builder)
}

// TestJSONRPCRequestIDCLIFlagPresence checks that a defaulted request ID uses
// an ordinary string flag while an optional ID keeps omission distinct from an
// explicitly empty string.
func TestJSONRPCRequestIDCLIFlagPresence(t *testing.T) {
	tests := []struct {
		name          string
		defaultID     bool
		wantPresence  bool
		wantParamType string
	}{
		{"defaulted", true, false, "string"},
		{"optional", false, true, "*string"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := expr.RunDSL(t, func() {
				dsl.Service("Requests", func() {
					dsl.JSONRPC(func() {
						dsl.POST("/rpc")
					})
					dsl.Method("send", func() {
						dsl.Payload(func() {
							if test.defaultID {
								dsl.ID("id", dsl.String, func() {
									dsl.Default("default-id")
								})
								return
							}
							dsl.ID("id", dsl.String)
						})
						dsl.JSONRPC(func() {})
					})
				})
			})
			plan, generation, servicePlan := plannedHTTPPlan(t, root, true)
			parser := plan.cliParsers[root.API.Servers[0].Name]
			require.NotNil(t, parser)
			require.Equal(t, test.wantPresence, parser.Declarations.PresenceFlagType != nil)

			require.NoError(t, generation.Freeze())
			require.NoError(t, servicePlan.Link())
			require.NoError(t, plan.Link())

			service := plan.services.Get("Requests")
			require.NotNil(t, service)
			command := buildSubcommandData(service, service.Endpoint("send"))
			require.Len(t, command.Flags, 1)
			require.Equal(t, test.defaultID, command.Flags[0].HasDefault)
			require.Equal(t, test.wantPresence, command.Flags[0].TracksPresence)
			require.NotNil(t, command.BuildFunction)
			require.Equal(t, []string{test.wantParamType}, command.BuildFunction.FormalParamTypes)
		})
	}
}

// TestClientCLIBuildNameMatchesDeclaration verifies released plugins can read
// the final payload builder name without choosing that name themselves.
func TestClientCLIBuildNameMatchesDeclaration(t *testing.T) {
	root := expr.RunDSL(t, testdata.MultiSimpleDSL)
	plan := linkedHTTPPlanForRoot(t, root)
	files := plan.ClientCLIFiles()
	require.Greater(t, len(files), 1)

	build, ok := files[1].SectionTemplates[1].Data.(*cli.BuildFunctionData)
	require.True(t, ok)
	endpoint := plan.services.Get("ServiceMultiSimple1").Endpoint("MethodMultiSimplePayload")
	require.Equal(t, endpoint.CLIPayloadDeclaration.Name(), build.Name)
}

// TestClientCLITransportNamesMatchDeclarations checks the released multipart
// and stream helper names exposed to plugin templates.
func TestClientCLITransportNamesMatchDeclarations(t *testing.T) {
	plan := linkedHTTPPlanForRoot(t, releasedHTTPNamesRoot(t))
	service := plan.services.Get("Names")

	multipart := buildSubcommandData(service, service.Endpoint("Multipart"))
	require.NotNil(t, multipart.MultipartFuncDeclaration)
	require.Equal(t, multipart.MultipartFuncDeclaration.Name(), multipart.MultipartFuncName)

	stream := buildSubcommandData(service, service.Endpoint("Raw"))
	require.NotNil(t, stream.BuildStreamPayloadDeclaration)
	require.Equal(t, stream.BuildStreamPayloadDeclaration.Name(), stream.BuildStreamPayload)
}

func TestEmptyBodyCLIUsesPayloadFieldExample(t *testing.T) {
	root := expr.RunDSL(t, testdata.PayloadBodyPrimitiveFieldEmptyDSL)
	plan := linkedHTTPPlanForRoot(t, root)
	endpoint := plan.services.Get("ServiceBodyPrimitiveArrayUser").Endpoints[0]
	require.NotNil(t, endpoint.Payload.Request.PayloadInit)
	require.Len(t, endpoint.Payload.Request.PayloadInit.ClientArgs, 1)
	example := endpoint.Payload.Request.PayloadInit.ClientArgs[0].Example

	require.IsType(t, []string{}, example)
	require.NotEmpty(t, example)
}

// TestClientCLINestedBodyWithoutValidationEmitsNoChecks verifies that a body
// with no top-level checks does not add validation to the payload builder.
func TestClientCLINestedBodyWithoutValidationEmitsNoChecks(t *testing.T) {
	root := expr.RunDSL(t, testdata.PayloadBodyUserInnerDSL)
	plan := linkedHTTPPlanForRoot(t, root)
	files := plan.ClientCLIFiles()
	require.NotEmpty(t, files)
	service := plan.services.Get("ServiceBodyUserInner")
	endpoint := service.Endpoints[0]
	require.NotNil(t, endpoint.Payload.Request.PayloadInit)
	require.Len(t, endpoint.Payload.Request.PayloadInit.ClientArgs, 1)
	arg := endpoint.Payload.Request.PayloadInit.ClientArgs[0]
	require.Empty(t, arg.Validate)
	require.NotNil(t, arg.CLIPlan)
	_, builder := buildFlags(service, endpoint)
	require.NotNil(t, builder)
	require.Len(t, builder.Fields, 1)
	require.NotContains(t, builder.Fields[0].Init, "goa.")
}
