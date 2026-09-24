# Exported API — goval

```go
type GoValueCode struct {
    Declarations []string  // locals that must precede Expression
    Expression   string    // final typed Go value
}

type UnionConstructorResolver func(attribute *expr.AttributeExpr, branch string) (string, error)

type GoTypeLayoutResolver interface {
    GoTypeLayout(attribute *expr.AttributeExpr, policy GoLayoutPolicy) (LinkedGoType, error)
}

func RenderGoValue(attribute *expr.AttributeExpr, value any, layout LinkedGoType, pointer bool, resolveUnion UnionConstructorResolver, localPrefix string) (GoValueCode, error)
```

## Pre-existing callers

`http/codegen/service_data.go` renders authored `DefaultValue`s into client CLI
and payload code. `codegen/go_transform.go` and `codegen/transformer.go` render
default expressions inside generated transform helpers via
`planGoTypeWithAttributor` and the `goValueRenderer` walk.
