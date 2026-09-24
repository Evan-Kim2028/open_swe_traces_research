# Exported API — unionid

```go
// UnionDeclarationID identifies one authored OneOf declaration and the Go
// definition emitted for the current copy. Copies of the same authored
// attribute share an identity, while separate OneOf declarations do not.
type UnionDeclarationID struct { /* opaque */ }

// UnionTypeID identifies the Go and JSON definition emitted for a union.
// It is distinct from expr.Union.Hash, which describes design compatibility.
type UnionTypeID string

// NewUnionDeclarationID returns the identity of the OneOf stored in attribute.
func NewUnionDeclarationID(attribute *expr.AttributeExpr) UnionDeclarationID

// NewUnionTypeID returns a repeatable key for union's generated Go and JSON
// definitions.
func NewUnionTypeID(union *expr.Union) UnionTypeID
```

## Pre-existing callers

`NameScope` (`codegen/scope.go`) uses union type IDs to decide whether two
union expressions may share one generated declaration name. Service
planning (`codegen/service`), the generation phases (`codegen/generator`),
and the HTTP, gRPC and JSON-RPC transport generators collect
`NewUnionDeclarationID`s when attributing emitted union declarations to
packages. `RenderGoValue` consumes declaration IDs when emitting union
branch constructor calls for authored defaults.
