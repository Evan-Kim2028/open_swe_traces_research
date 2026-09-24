# Exported API — compat

```go
func (p Primitive) IsCompatible(val any) bool
func (a *Array) IsCompatible(val any) bool
func (o *Object) IsCompatible(val any) bool
func (m *Map) IsCompatible(val any) bool
func (u *Union) IsCompatible(val any) bool
func (u *Union) GetTypeKey() string
func (u *Union) GetValueKey() string
```

## Pre-existing callers

`IsCompatible` is the `DataType` interface method used by DSL evaluation
to check that authored values (defaults, enum values, examples) match
their declared types. Union envelope keys are read by transport encoders
and the example generator.
