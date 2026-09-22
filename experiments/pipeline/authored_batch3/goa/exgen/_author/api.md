# Exported API — exgen

```go
func (a *AttributeExpr) Example(r *ExampleGenerator) any
func NewLength(a *AttributeExpr, r *ExampleGenerator) int
```

Plus unexported helpers: `hasLengthValidation`, `hasEnumValidation`,
`hasFormatValidation`, `hasPatternValidation`, `hasMinMaxValidation`,
`byLength`, `byEnum`, `byFormat`, `byPattern`, `patgen`, `byMinMax`,
`checkPattern`, `checkMinMaxValue`.

## Pre-existing callers

`expr.Example` methods on every data type call `AttributeExpr.Example`;
service codegen embeds the produced values into generated example code
and docs.
