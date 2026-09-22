# Exported API — typepred

```go
func AsObject(dt DataType) *Object
func AsArray(dt DataType) *Array
func AsMap(dt DataType) *Map
func AsUnion(dt DataType) *Union
func IsObject(dt DataType) bool
func IsArray(dt DataType) bool
func IsMap(dt DataType) bool
func IsUnion(dt DataType) bool
func IsPrimitive(dt DataType) bool
func IsAlias(dt DataType) bool
func Equal(dt, dt2 DataType) bool
func QualifiedTypeName(t DataType) string
```

`toReflectType` is unexported; it backs example/map reflection paths inside
`expr`.

## Pre-existing callers

Every DSL evaluation, validation, and codegen phase calls these predicates:
`AsObject`/`IsObject` unwrap user types to reach object attributes, `Equal`
compares field types during transform planning, and `QualifiedTypeName`
produces names like `map<string, array<int32>>` in error messages.
