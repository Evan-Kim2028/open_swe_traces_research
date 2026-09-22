# Closure — goval

`codegen/go_value.go` — authored-default rendering: reflect-driven value →
Go source expressions, with named-type layout unwrap, pointer local
declarations, union branch constructors, map ordering, custom Go-type checks.

Symbols stubbed: `RenderGoValue`, `planGoTypeWithAttributor`,
`goValueRenderer.{render,renderPrimitive,renderArray,renderMap,renderObject,
renderUnion,localName}`, `orderedMapValues`, `concreteGoValue`,
`renderPrimitiveLiteral`, `renderAnyValue`, `goValueIsNil`,
`renderTypedCustomValue`, `validateCustomDefault`, `authoredObjectField`,
`reflectedMapValue`, `underlyingPrimitive`.

Tests removed: `codegen/go_value_test.go` deleted (pins every commitment);
`codegen/go_transform_helpers_test.go` and `http/codegen/default_value_test.go`
deleted (exercise the renderer end-to-end); `TestClientCLIFiles` trimmed from
`http/codegen/client_cli_test.go`; `TestGoTransform`,
`TestGoTransformUsesDesignNilabilityForCustomTypes`,
`TestGoTransformDefaultUsesFinalCustomTypeImportAlias`,
`TestGoTransformReturnsFirstDefaultError` trimmed from
`codegen/go_transform_test.go` (all panic on the stubs via transform
defaults).
