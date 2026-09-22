# Closure — errcontract

`expr/error_contract.go` — service error contract equivalence: detached
effective-attribute construction (References/Bases materialized, mutable
contract values cloned, recursive user types reconnected by origin), then
order-insensitive structural equivalence over types, validations, defaults,
and metadata, plus qualifier-diff reporting.

Symbols stubbed: `equivalentErrorAttributes`, `differingErrorQualifierSettings`,
`effectiveErrorAttribute`, `effectiveErrorCopier.{attribute,dataTypes,dataType}`,
`cloneErrorValidation`, `cloneErrorContractValue`,
`equivalentErrorAttributeNodes`, `equivalentErrorValidation`,
`equivalentStringSet`, `equivalentValueSet`, `equivalentErrorMetadata`.

Tests removed: `expr/error_contract_test.go` deleted (pins every
commitment). Trimmed functions that reach the stubs through eval-time error
mapping:

- `expr/attached_service_test.go`: `TestEvaluateAttachedGRPCServiceIgnoresPackageRootAPIErrors`, `TestEvaluateAttachedGRPCServiceUsesOwningRootForAPIErrors`, `TestEvaluateAttachedJSONRPCErrorDoesNotReadPackageRoot`
- `expr/default_value_validation_test.go`: `TestErrorDefaultReportsOnceWhenInherited`
- `expr/grpc_endpoint_test.go`: `TestGRPCEndpointValidation`
- `expr/http_endpoint_test.go`: `TestHTTPRouteValidation`
- `expr/http_error_test.go`: `TestHTTPErrorResponseValidation`
- `expr/jsonrpc_error_contract_test.go`: `TestJSONRPCAPIErrorMappingOverridesHTTPDefault`, `TestJSONRPCErrorCodeAcceptsAllowedValues`, `TestJSONRPCErrorCodeDefaultsToInternalError`, `TestJSONRPCErrorCodeMayBeReusedByDifferentMethods`, `TestJSONRPCErrorCodeRejectsReservedValues`, `TestJSONRPCMethodErrorMappingReplacesAPIDefault`
- `expr/jsonrpc_notification_contract_test.go`: `TestJSONRPCNotificationContract`
- `expr/jsonrpc_response_metadata_test.go`: `TestJSONRPCErrorResponseRejectsHTTPMetadata`, `TestResponseMetadataPreservesSupportedMappings`
- `expr/service_test.go`: `TestEquivalentInlineMethodErrorsShareOrigin`, `TestIncompatibleInlineMethodErrorsAreRejected`, `TestRepeatedStandardErrorsMustUseSameQualifiers`
- `expr/transport_error_contract_test.go`: `TestGRPCInheritedErrorMappingRejectsIncompatibleError`, `TestGRPCInheritedErrorMappingUsesEffectiveInheritedContract`, `TestGRPCInheritedErrorMappingUsesMethodError`, `TestHTTPInheritedErrorMappingAcceptsEquivalentValidationOrder`, `TestHTTPInheritedErrorMappingAcceptsServiceReference`, `TestHTTPInheritedErrorMappingRejectsIncompatibleError`, `TestHTTPInheritedErrorMappingUsesEffectiveInheritedContract`, `TestHTTPInheritedErrorMappingUsesMethodError`, `TestInheritedErrorMappingRejectsDifferentEffectiveBases`, `TestInheritedErrorMappingRejectsExplicitQualifierOverride`, `TestInheritedErrorMappingRetainsQualifiers`, `TestServiceErrorMappingsUseMethodError`, `TestServiceTransportErrorResponseMayDefineUnusedDefault`
- `expr/transport_error_response_dsl_test.go`: `TestGRPCErrorResponseCallbackUsesDeclaredError`, `TestHTTPErrorResponseCallbackUsesDeclaredError`, `TestHTTPMethodErrorResponseCallbackAcceptsTag`, `TestJSONRPCErrorResponseCallbackUsesDeclaredError`
- `http/codegen/error_body_description_test.go`: `TestErrorBodyDescriptionNamesSingleError`
- `http/codegen/handler_test.go`: `TestHandlerInit`
- `http/codegen/idempotency_test.go`: `TestIdempotentHTTPEndpointCodegen`
- `http/codegen/openapi_order_independence_test.go`: `TestOpenAPIOrderIndependence`
- `http/codegen/plan_test.go`: `TestEndpointConstructorsUsePackageDeclarations`, `TestHTTPFilePlansIncludeGeneratedUses`, `TestHTTPPlanReservesFixedImportsWrittenByThePackage`
- `http/codegen/service_data_purity_test.go`: `TestAnalyzeLeavesDesignExpressionsUnchanged`
- `http/codegen/service_data_union_order_test.go`: `TestHTTPServiceDataReusesOneAuthoredUnionAcrossErrors`
