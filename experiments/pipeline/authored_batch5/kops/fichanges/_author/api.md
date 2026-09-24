# Exported API — fichanges

Package `upup/pkg/fi` (importable as `example.internal/clustkit/upup/pkg/fi`).

Reflection-based change detection between an actual object and an expected spec.

- `func BuildChanges(a, e, changes interface{}) bool` — `a`, `e`, `changes` are pointers to
  the SAME struct type. For each exported field: nil pointer in `e` means "don't care"
  (skipped); a nil `a` copies every non-nil `e` field; otherwise differing fields are copied
  from `e` into `changes`. Returns whether any field changed. Panics on type mismatch.
- `func equalFieldValues(a, e reflect.Value) bool` — custom equality: maps and slices
  recurse elementwise; ptr/interface values implementing `CompareWithID` compare equal when
  their non-nil IDs match; `Resource` values compare via `ResourcesMatch` (and a not-ready
  `HasIsReady` expected resource is never equal); otherwise `reflect.DeepEqual`.
- `func equalMapValues(a, e reflect.Value)` — nil-ness, length, then recursive per-key
  compare over a's keys.
- `func equalSlice(a, e reflect.Value)` — nil-ness, length, then recursive per-index compare.

Example: expected `{Name: ptr("x"), Size: nil}` vs actual `{Name: "y", Size: 3}` → changes
gets `Name="x"` only; `Size` untouched.
