# Closure — defval

`expr/default_value.go` — authored-default validation: reflect-driven value
checking against design types, rule application, numeric fitting, union
envelope, deterministic reporting.

Symbols stubbed: all 26 functions in the file (the
`(*AttributeExpr).validateDefaultValue*` entry points plus the
`defaultValueValidator` methods and `default*` helpers).

Tests removed: `expr/default_value_validation_test.go` deleted (pins every
commitment including rune-length, bounds, union envelope, dedupe-by-visit).
