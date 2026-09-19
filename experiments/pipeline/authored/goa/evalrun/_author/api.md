# Exported API — evalrun

```
func RunDSL() error
func Execute(fn func(), def Expression) bool
func Current() Expression
func ReportError(fm string, vals ...any)
func IncompatibleDSL()
func InvalidArgError(expected string, actual any)
func TooFewArgError()
func TooManyArgError()
```

RunDSL walks registered roots, executes each expression's source DSL (including expressions appended during the walk, with a 100-iteration cap), then prepare, then validate, then finalize. Execute pushes the target expression, runs the DSL, pops, and reports success iff no new context errors appeared. Current is the stack top, or Top when empty. ReportError / IncompatibleDSL / argument-count helpers record errors against the current expression.

## Pre-existing callers

dsl package functions (Attribute, Type, Service, HTTP, …); expr.RunDSL / expr.RunInvalidDSL; eval tests.
