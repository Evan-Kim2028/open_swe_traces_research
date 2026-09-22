# Exported API — propvalues

`CustomProperty.DefaultValue` is an untyped payload whose concrete shape
depends on `ValueType`; the `DefaultValue*` accessors hand callers the
right Go type for each property kind.

- `CustomProperty.DefaultValueString` — string payload for string,
  single_select and url types.
- `CustomProperty.DefaultValueStrings` — []string payload for
  multi_select.
- `CustomProperty.DefaultValueBool` — bool payload for true_false.
