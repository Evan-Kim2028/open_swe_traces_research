# Exported API — evalctx

```
func Register(r Root) error
func (s Stack) Current() Expression
func (c *DSLContext) Error() string
func (c *DSLContext) Roots() ([]Root, error)
func (c *DSLContext) Record(err *Error)
```

Register rejects a second root with the same EvalName and records that root's package paths for later location skipping. Roots returns registered roots ordered so that a root's dependencies appear after it, and errors on a dependency cycle. Record appends to the context multi-error. Stack.Current is the last pushed expression or nil.

## Pre-existing callers

RunDSL (Roots, Errors); ReportError (Record, Stack.Current); dsl init-time Register; eval tests.
