# Exported API — strvals

Package `third_party/forked/helmstrvals` (importable as `example.internal/clustkit/third_party/forked/helmstrvals`).

- `func ParseInto(s string, dest map[string]interface{}) error` — parse a strvals line and merge the result into `dest`; existing keys are overwritten.
- `func ParseIntoString(s string, dest map[string]interface{}) error` — same, but every parsed value stays a string.
- `var ErrNotList`, `var MaxIndex`, `var MaxNestedNameLevel` — exported package knobs the parser consults.

Production callers: `cmd/kops/toolbox_template.go` (the `--set` flag handling for `kops toolbox template`).
