# Closure — unionid

`codegen/union.go` — canonical identity for generated union declarations:
`UnionDeclarationID` keys one authored OneOf declaration; `UnionTypeID` is a
repeatable length-prefixed serialization of everything that changes the
emitted Go/JSON definition (envelope keys, branch names and order, branch
shape, nilability, field metadata, package location, recursion
back-references).

Symbols stubbed: `NewUnionDeclarationID`, `NewUnionTypeID`,
`writeUnionTypeID`, `writeUnionAttributeID`, `writeUnionObjectID`,
`writeUnionIDPart`.

The ID is consumed wherever a generated union is named or shared, so the
excision is wide: every in-tree test that plans or emits a union reaches it.
Tests removed (trimmed functions, by file):

- `codegen/generated_types_test.go`: `TestGeneratedPackageLookupAcrossFreeze`, `TestGeneratedPackageUnionAndUserTypeCollisionFailsRegardlessOfOrder`, `TestGeneratedPackageUnionBranchesAreIsolatedByUnion`, `TestGeneratedPackageUnionBranchesShareDeclaration`, `TestGeneratedPackageUnionFamilyRejectsExactTypeNames`, `TestGeneratedPackageUnions`, `TestGeneratedTypeFamiliesContainCanonicalNames`, `TestGenerationCatalogsAreIsolated`
- `codegen/generator/generate_union_merge_integration_test.go`: `TestGenerateUnionUserTypeSamePathMerged`
- `codegen/generator/generation_test.go`: `TestGeneratePhasesShareOneGeneration`
- `codegen/generator/service_union_package_scope_test.go`: `TestServiceRelocatedUnionNamesSpanDesignRoots`, `TestServiceUnionFamilyNamesRejectExactDeclarationCollisions`
- `codegen/go_transform_test.go`: `TestGoTransformServiceUnionField`, `TestGoTransformUnionAcrossTransportBoundary`, `TestGoTransformUnionKeepsNilSelectedBranch`, `TestGoTransformUnionTemporaryUsesNestingDepth`, `TestTransformPlanHelperEligibilityMatchesCompositeRenderers`
- `codegen/go_transform_union_test.go`: `TestGoTransformUnion`
- `codegen/go_type_plan_test.go`: `TestGoTypePlanFormatsContainersAndUnions`, `TestGoTypePlanReportsReferencePointers`, `TestGoTypePlanRetainsNestedOwners`, `TestGoTypePlanUnionReferenceOwnsOnlyItsDeclarationImport`
- `codegen/go_value_test.go`: `TestRenderGoValueUsesContainerElementLayouts`, `TestRenderGoValueUsesPlannedUnionConstructor`
- `codegen/scope_test.go`: `TestNameScopeGoTypeDefUsesValueUnions`, `TestNameScope_GoFullTypeNameRejectsUnplannedUnionAfterFreeze`, `TestNameScope_GoFullTypeNameUsesAuthoredUnionNameBeforePlanning`, `TestNameScope_GoTypeNameDistinguishesInlineObjectFieldOrder`, `TestNameScope_GoTypeNameDistinguishesUnionBranchOrder`, `TestNameScope_GoTypeNameDistinguishesUnionBranchPackages`, `TestNameScope_GoTypeNameDoesNotPlanUnionDeclarations`, `TestNameScope_GoTypeNameUsesAuthoredNameForEqualUnionDefinitions`, `TestUnionTypeID`, `TestUnionTypeIDEncodesRecursiveGeneratedUserTypeShape`, `TestUnionTypeIDIgnoresNonEmittedPointerSharing`, `TestUnionTypeIDIncludesDefaultDrivenPointerShape`, `TestUnionTypeIDIncludesGeneratedUserTypeShape`
- `codegen/service/declaration_resolver_test.go`: `TestDeclarationResolverTransformsRelocatedUnionBranches`, `TestServicesDataServiceAttributorUsesFrozenPackageDeclarations`
- `codegen/service/imports_test.go`: `TestEmittedUnionReservesFixedJSON`, `TestUnionFieldReferencesUseFixedImportAliases`, `TestUnionImportsStopAtNamedBranches`
- `codegen/service/service_data_union_nilability_test.go`: `TestBuildUnionTypeDataMarksNilableBranches`
- `codegen/service/service_data_union_order_test.go`: `TestServicePlanUnionNamesAreIndependentOfObjectOrder`
- `codegen/service/service_plan_compile_contract_test.go`: `TestServicePackageNameUsesClaimedImportPath`
- `codegen/service/service_plan_render_contract_test.go`: `TestServicePlanRenderingIsPure`
- `codegen/service/service_test.go`: `TestFilesEmitsDifferentExplicitlyNamedUnions`, `TestFilesEmitsPackageDeclarationsOnce`, `TestFilesEmitsSharedPackagesOnceAcrossRoots`, `TestGeneratedUnionBranchCollisionDoesNotCanonicalizeToRootType`, `TestNewPlansRejectSharedUnionBranchLayoutConflicts`, `TestServicesDataUsesFrozenPackageDeclarations`, `TestStructPkgPath_ExtendedUnionGeneratedInEachOwningPackage`, `TestStructPkgPath_UnionImportsJSON`, `TestStructPkgPath_UnionJSONFieldBranchesGenerateAliases`, `TestStructPkgPath_UnionNamesRemainExactAcrossServices`
- `codegen/validation_plan_test.go`: `TestValidationPlanPreservesContainersUnionsAndValidatorBindings`
- `grpc/codegen/plan_test.go`: `TestNewPlansIsIndependentOfInputOrder`
- `grpc/codegen/protobuf_descriptor_plan_test.go`: `TestPlanUsesNamesFromSupportedProtobufTools`, `TestPlanWritesLegalFieldAndOneofNames`
- `grpc/codegen/required_union_validation_test.go`: `TestRequiredUnionValidationUsesCompleteProtobufBranches`
- `grpc/codegen/streaming_test.go`: `TestStreamingPayloadEnvelopeWithUnionPayload`
- `grpc/codegen/transform_helper_test.go`: `TestTransformHelpersShareAcrossDocumentedUses`
- `http/codegen/oneof_http_codegen_test.go`: `TestClientCLIInlinesOneOfRequestValidation`, `TestClientResponseCodeProjectsSingleViewOneOfResults`
- `http/codegen/planned_name_collision_test.go`: `TestHTTPUnionPlannedNamesRejectPackageCollisions`
- `http/codegen/server_encode_test.go`: `TestEncode`
- `http/codegen/server_payload_types_test.go`: `TestPayloadConstructor`
- `http/codegen/service_data_union_order_test.go`: `TestCollectHTTPUnionTypesDeterministicAcrossObjectOrder`, `TestCollectHTTPUnionTypesRejectsConflictingShapesInOneRootRole`, `TestCollectHTTPUnionTypesRejectsUnrelatedSameShapedDeclarations`, `TestCollectHTTPUnionTypesUsesExactRootRoleNames`, `TestHTTPServiceDataReusesOneAuthoredUnionAcrossErrors`, `TestHTTPServiceDataReusesOneAuthoredUnionAcrossExplicitRequestBodies`, `TestHTTPServiceDataReusesOneAuthoredUnionAcrossMethods`, `TestHTTPServiceDataReusesUnionWhenBranchTypeAppearsEarlier`, `TestHTTPUnionBranchTypesDoNotDependOnSurroundingFieldOrder`
- `http/codegen/typedef_test.go`: `TestGoTypeDefUnionPresence`
- `http/codegen/wire_catalog_test.go`: `TestWireTypeCatalogCollectsUnionsBeforeFreeze`
- `jsonrpc/codegen/file_imports_test.go`: `TestJSONRPCFilesUsePlannedImports`
- `jsonrpc/codegen/params_golden_test.go`: `TestJSONRPCParamsGeneratedSource`
- `jsonrpc/codegen/plan_test.go`: `TestPlanSelectsPositionalParams`
