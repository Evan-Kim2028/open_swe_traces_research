# Exported API — goval

```go
type GoValueCode struct {
    Declarations []string
    Expression   string
}
type UnionConstructorResolver func(attribute *expr.AttributeExpr, branch string) (string, error)
type GoTypeLayoutResolver interface {
    GoTypeLayout(attribute *expr.AttributeExpr, policy GoLayoutPolicy) (LinkedGoType, error)
}
func RenderGoValue(attribute *expr.AttributeExpr, value any, layout LinkedGoType, pointer bool, resolveUnion UnionConstructorResolver, localPrefix string) (GoValueCode, error)
```

Unexported: `goValueRenderer.render`/`renderPrimitive`/`renderArray`/
`renderMap`/`renderObject`/`renderUnion`/`localName`,
`orderedMapValues`, `concreteGoValue`, `renderPrimitiveLiteral`,
`renderAnyValue`, `goValueIsNil`, `renderTypedCustomValue`,
`validateCustomDefault`, `authoredObjectField`, `reflectedMapValue`,
`underlyingPrimitive`.

## Pre-existing callers

Every generated default value, enum rendering, and fixture expression
goes through RenderGoValue — including struct:field:type custom types
and OneOf defaults that need a planned branch constructor.
