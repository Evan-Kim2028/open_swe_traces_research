# Exported API — fivalues

Package `upup/pkg/fi` (importable as `example.internal/clustkit/upup/pkg/fi`).

- `func ValueOf[T any](*T) T` — nil-safe dereference.
- `func StringSliceValue([]*string) []string`, `func StringSlice([]string) []*string` — slice<->ptr-slice conversion.
- `func IsNilOrEmpty(*string) bool`, `func ArrayContains([]string, string) bool`.
- `func DebugPrint(interface{}) string`, `DebugAsJsonString`, `DebugAsJsonStringIndent` — dry-run rendering.
- `func ToInt64(*string) *int64`, `func ToString(*int64) *string` — nil-propagating conversions.

Production callers: every `*tasks` package (`upup/pkg/fi/cloudup/*tasks`), `dryruntarget`, `upup/pkg/fi/nodeup`.
