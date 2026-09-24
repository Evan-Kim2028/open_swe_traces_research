# Exported API — tfliterals

Package `upup/pkg/fi/cloudup/terraformWriter` (importable as `example.internal/clustkit/upup/pkg/fi/cloudup/terraformWriter`).

- `type Literal{String string}` — one HCL expression; implements `json.Marshaler` and the terraform `element` protocol (`Write`, `IsSingleValue`).
- Constructors: `LiteralFunctionExpression(fn, args...)`, `LiteralSelfLink(type,name)`, `LiteralData(type,name,prop)`, `LiteralProperty(type,name,prop)`, `LiteralTokens(tokens...)`, `LiteralFromIntValue(i)`, `LiteralFromStringValue(s)`, `LiteralWithIndex(s)`, `LiteralBinaryExpression(l,op,r)`, `LiteralIndexExpression(coll,idx)`, `LiteralListExpression(args...)`, `LiteralEmptyStrConditionalExpression(empty,value)`.
- `func SortLiterals(v []*Literal)`, `func dedupLiterals(v []*Literal) ([]*Literal, error)` (unexported helper, used by `GetOutputs`).

Production callers: every `*tasks` package (`upup/pkg/fi/cloudup/{do,scaleway,aws,gce,...}tasks`), `util/pkg/vfs/*_terraform.go`.
