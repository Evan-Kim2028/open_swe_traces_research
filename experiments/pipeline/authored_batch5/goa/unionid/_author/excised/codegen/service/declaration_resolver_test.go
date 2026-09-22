// This file verifies that service transformations resolve every named type
// through the frozen package catalog as recursion crosses explicit package
// locations.
package service

import (
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"example.internal/apikit/v3/codegen"
	_ "example.internal/apikit/v3/dsl"
	_ "example.internal/apikit/v3/eval"
	"example.internal/apikit/v3/expr"
)


// TestDeclarationResolverQualifiesRelocatedConsumersWithoutRenamingLocalType
// verifies errors and interceptor fields use their actual package owner.
func TestDeclarationResolverQualifiesRelocatedConsumersWithoutRenamingLocalType(t *testing.T) {
	service := &expr.ServiceExpr{Name: "Collisions"}
	local := resolverUserType("Fault", expr.String)
	relocated := resolverUserType("fault", expr.String)
	relocated.Attribute().AddMeta("struct:pkg:path", "errors")
	container := resolverUserType("Container", &expr.Object{
		{Name: "fault", Attribute: &expr.AttributeExpr{Type: relocated}},
	})
	container.Attribute().AddMeta("struct:pkg:path", "types")

	generation := mustTestGeneration(t, "generated.local/gen", nil)
	servicePackage := mustClaimTestPackage(t, generation, servicePackagePath(generation.GenPkg(), service))
	localDeclaration, err := servicePackage.DeclareUserType(local)
	require.NoError(t, err)
	errorConstructor := codegen.NewPreferredName(
		codegen.NameFunction,
		"MakeFault",
		codegen.ExportedName,
		serviceNameOrder{role: serviceErrorConstructorNameRole, subject: "fault"},
	)
	require.NoError(t, servicePackage.DeclareName(errorConstructor))
	errorsPackage := mustClaimTestPackage(t, generation, "generated.local/gen/errors")
	_, err = errorsPackage.DeclareUserType(relocated)
	require.NoError(t, err)
	typesPackage := mustClaimTestPackage(t, generation, "generated.local/gen/types")
	_, err = typesPackage.DeclareUserType(container)
	require.NoError(t, err)
	require.NoError(t, generation.Freeze())

	resolver := newServiceResolver(
		generation,
		aliasesForTest(
			t,
			servicePackagePath(generation.GenPkg(), service),
			"generated.local/gen/errors",
			"generated.local/gen/types",
		),
		service.Name,
		servicePackagePath(generation.GenPkg(), service),
		servicePackagePath(generation.GenPkg(), service),
	)
	require.Equal(t, "Fault", localDeclaration.Name())
	require.Equal(t, "Fault", resolver.Ref(&expr.AttributeExpr{Type: local}, ""))
}

// TestDeclarationResolverUsesFinalCustomTypeImportAlias verifies that service
// fields keep their complete custom Go type when an import name changes.
func TestDeclarationResolverUsesFinalCustomTypeImportAlias(t *testing.T) {
	const (
		servicePath = "generated.local/gen/values"
		customPath  = "example.com/custom/wire"
	)
	attribute := &expr.AttributeExpr{
		Type: expr.String,
		Meta: expr.MetaExpr{
			"struct:field:type": {"map[wire.Key]wire.Value", customPath, "wire"},
		},
	}
	generation := mustTestGeneration(t, "generated.local/gen", nil)
	pkg := mustClaimTestPackage(t, generation, servicePath)
	require.NoError(t, pkg.DeclareImport(codegen.NewImport("wire", customPath)))
	require.NoError(t, pkg.RequireImport(codegen.NewImport("wire", "example.com/fixed/wire")))
	require.NoError(t, generation.Freeze())
	require.Equal(t, "wire2", pkg.ImportName(customPath))

	resolver := newServiceResolver(
		generation,
		&importAliases{generation: generation},
		"Values",
		servicePath,
		servicePath,
	)
	require.Equal(t, "map[wire2.Key]wire2.Value", resolver.Name(attribute, "", false, true))
}

// TestDeclarationResolverPanicsWhenPlanOmittedType verifies render analysis
// fails immediately instead of allocating a missing declaration.
func TestDeclarationResolverPanicsWhenPlanOmittedType(t *testing.T) {
	service := &expr.ServiceExpr{Name: "Missing"}
	generation := mustTestGeneration(t, "generated.local/gen", nil)
	mustClaimTestPackage(t, generation, servicePackagePath(generation.GenPkg(), service))
	require.NoError(t, generation.Freeze())
	resolver := newServiceResolver(
		generation,
		aliasesForTest(t, servicePackagePath(generation.GenPkg(), service)),
		service.Name,
		servicePackagePath(generation.GenPkg(), service),
		servicePackagePath(generation.GenPkg(), service),
	)
	missing := resolverUserType("Missing", expr.String)
	require.PanicsWithValue(
		t,
		"resolve user type \"Missing\" for service \"Missing\" in package \"generated.local/gen/missing\": user type \"Missing\" has no declaration in generated package \"generated.local/gen/missing\"",
		func() {
			resolver.Name(&expr.AttributeExpr{Type: missing}, "", false, true)
		},
	)
}


// aliasesForTest builds the same frozen full-path qualifier table used by
// service analysis for the package paths exercised by a focused resolver test.
func aliasesForTest(t *testing.T, paths ...string) *importAliases {
	t.Helper()
	generation := mustTestGeneration(t, "generated.local/gen", nil)
	packages := make([]*codegen.GeneratedPackage, len(paths))
	for index, importPath := range paths {
		packages[index] = mustClaimTestPackage(t, generation, importPath)
	}
	for _, pkg := range packages {
		for _, importPath := range paths {
			if importPath != pkg.ImportPath() {
				require.NoError(t, pkg.DeclareImport(codegen.NewImport(codegen.Goify(path.Base(importPath), false), importPath)))
			}
		}
	}
	require.NoError(t, generation.Freeze())
	return &importAliases{generation: generation}
}

// resolverUserType constructs one exact declaration for resolver tests.
func resolverUserType(name string, dataType expr.DataType) *expr.UserTypeExpr {
	return &expr.UserTypeExpr{
		AttributeExpr: &expr.AttributeExpr{Type: dataType},
		TypeName:      name,
		UID:           "resolver-test#" + name,
	}
}

// transformSource combines an inline transformation with every recursive
// helper so tests can assert the complete code emitted for one conversion.
func transformSource(code string, helpers []*codegen.TransformFunctionData) string {
	var source strings.Builder
	source.WriteString(code)
	for _, helper := range helpers {
		source.WriteString(helper.Code)
	}
	return source.String()
}
