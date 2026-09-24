// This file checks that every package name is chosen before core generators or
// plugins write files.
package generator

import (
	_ "fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"example.internal/apikit/v3/codegen"
	"example.internal/apikit/v3/eval"
	_ "example.internal/apikit/v3/expr"
	httpdata "example.internal/apikit/v3/http/codegen/testdata"
)


func TestCommandsPlanOnlyTheirFiles(t *testing.T) {
	root := codegen.RunDSL(t, httpdata.AliasTypeDSL)

	genRun, err := newGenerationRun("gen", newDefaultRegistry())
	require.NoError(t, err)
	genResult, err := genRun.execute("generated.local/gen", []eval.Root{root})
	require.NoError(t, err)
	require.Nil(t, genResult.plan.example)
	require.NotNil(t, genResult.plan.openapi)

	exampleRun, err := newGenerationRun("example", newDefaultRegistry())
	require.NoError(t, err)
	exampleResult, err := exampleRun.execute("generated.local/gen", []eval.Root{root})
	require.NoError(t, err)
	require.NotNil(t, exampleResult.plan.example)
	require.Nil(t, exampleResult.plan.openapi)
}

// TestRenderUsesRetainedPlans proves that file rendering does not look up
// services or transports from the prepared design after planning finishes.
func TestRenderUsesRetainedPlans(t *testing.T) {
	root := codegen.RunDSL(t, httpdata.AliasTypeDSL)
	plan := mustTestPlan(
		t,
		"generated.local/gen",
		[]eval.Root{root},
		planServiceData,
		planTransportData,
	)

	plan.preparedRoots = nil
	plan.services = nil
	plan.http = nil
	plan.jsonrpcHTTP = nil
	plan.jsonrpc = nil
	plan.grpc = nil

	serviceFiles, err := serviceFiles(plan)
	require.NoError(t, err)
	require.NotEmpty(t, serviceFiles)
	transportFiles, err := transportFiles(plan)
	require.NoError(t, err)
	require.NotEmpty(t, transportFiles)
}

// TestPreparedRootsRejectFileRenderMutation proves that persistent mutations
// made by templates and file finalizers are rejected after rendering completes.
func TestPreparedRootsRejectFileRenderMutation(t *testing.T) {
	for _, phase := range []string{"template", "finalizer"} {
		t.Run(phase, func(t *testing.T) {
			root := codegen.RunDSL(t, httpdata.AliasTypeDSL)
			dir := t.TempDir()
			writeGeneratedModule(t, filepath.Join(dir, codegen.Gendir), "generated.local/gen")
			mutate := func() {
				root.API.HTTP.Services[0].HTTPEndpoints[0].Routes[0].Path = "/changed"
			}
			first := &codegen.File{
				Path: "first.txt",
				SectionTemplates: []*codegen.SectionTemplate{{
					Name:   "first",
					Source: "first",
				}},
			}
			if phase == "template" {
				first.SectionTemplates[0].Source = "{{ mutate }}"
				first.SectionTemplates[0].FuncMap = map[string]any{"mutate": func() string {
					mutate()
					return "first"
				}}
			} else {
				first.FinalizeFunc = func(_ string) error {
					mutate()
					return nil
				}
			}
			registry := testRegistry("test", func() coreGenerator {
				return coreGenerator{name: "files", Generate: func(_ *Plan) ([]*codegen.File, error) {
					return []*codegen.File{first}, nil
				}}
			})

			_, err := generate(dir, "test", false, registry)
			require.ErrorContains(t, err, "generated file renders mutated prepared design")
		})
	}
}
