# Exported API — defval

Internal: `(*AttributeExpr).validateDefaultValue(s)` — invoked from root,
service, and method validation for every authored `Default` value. No new
exported surface; behavior is observable through DSL validation errors.

## Pre-existing callers

`RootExpr.Validate` walks declared types/result types/errors;
`AttributeExpr.Validate` chains the same per-attribute default check.
