# Closure — goval

`codegen/go_value.go` — Go value rendering: per-kind literals, container
rendering under planned layouts, custom-type defaults, union
constructors, reflection helpers.

Symbols stubbed: `goValueRenderer.render`, `renderPrimitive`,
`renderArray`, `renderMap`, `renderObject`, `renderUnion`, `localName`,
`orderedMapValues`, `concreteGoValue`, `renderPrimitiveLiteral`,
`renderAnyValue`, `goValueIsNil`, `renderTypedCustomValue`,
`validateCustomDefault`, `authoredObjectField`, `reflectedMapValue`,
`underlyingPrimitive`.

Tests removed: `codegen/go_value_test.go` deleted.
