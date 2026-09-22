# Closure — strorset

Package: `pkg/util/stringorset` (`example.internal/clustkit/pkg/util/stringorset`).

Files: `pkg/util/stringorset/stringorset.go` (9 funcs/methods).

Removed functions (bodies stubbed): `StringOrSet.IsEmpty`, `Set`, `Of`, `String`,
`StringOrSet.UnmarshalJSON`, `StringOrSet.String`, `StringOrSet.Value`, `StringOrSet.Equal`,
`StringOrSet.MarshalJSON`.

Exported entry point(s): `Set`/`Of`/`String` constructors plus the JSON marshalling pair —
the type is embedded in API specs (e.g. egress/proxy selectors) where a field accepts
`"name"` or `["name", ...]`.

Test files removed in excision: `stringorset_test.go`.
