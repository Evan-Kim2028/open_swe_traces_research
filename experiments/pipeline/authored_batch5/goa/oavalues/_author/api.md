# Exported API — oavalues

```go
func (v Values) WithTitle(target eval.Expression, title string) Values
func (v Values) WithDescription(target eval.Expression, description string) Values
func (v Values) WithExamples(attribute *expr.AttributeExpr, examples []*expr.ExampleExpr) Values
func (v Values) Title(target eval.Expression, fallback string) string
func (v Values) Description(target eval.Expression, fallback string) string
func (v Values) Examples(attribute *expr.AttributeExpr, fallback []*expr.ExampleExpr) []*expr.ExampleExpr
func (v Values) Example(attribute *expr.AttributeExpr, generator *expr.ExampleGenerator) any
```

`copy`, `storeExamples`, `materializeExamples`, and `storedExample` are
package-private.

## Pre-existing callers

`NewWithValues`-style builders and `DocsFromExpr`/`Example` call sites in
v2/v3 thread per-build overrides without mutating the evaluated design.
