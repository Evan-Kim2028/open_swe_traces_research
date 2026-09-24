# Exported API — strorset

Package `pkg/util/stringorset` (importable as `example.internal/clustkit/pkg/util/stringorset`).

`StringOrSet` holds a set of strings that marshals to a JSON string or array.

- `func Set(v []string) StringOrSet` — always encodes as a JSON array.
- `func Of(v ...string) StringOrSet` — array-encoding only when more than one value;
  `Of()` with no args yields an empty set.
- `func String(v string) StringOrSet` — single value, encodes as a JSON string.
- `func (s *StringOrSet) UnmarshalJSON(value []byte) error` — a `[`-prefixed payload decodes
  as array and marks the value array-encoded; otherwise decodes as a bare string.
- `func (s StringOrSet) String() string` — comma-joins the sorted values (`"a,b"`).
- `func (v *StringOrSet) Value() []string` — sorted slice of values.
- `func (l StringOrSet) Equal(r StringOrSet) bool` — set equality.
- `func (v StringOrSet) MarshalJSON() ([]byte, error)` — emits a JSON array when the set is
  array-forced or holds >1 value, a bare JSON string when exactly one non-forced value, and
  `[]` when empty.

Examples: `String("a")` marshals `"a"`; `Set([]string{"a","b"})` marshals `["a","b"]`;
unmarshalling `["x","y"]` then re-marshalling yields `["x","y"]` even via `Of` semantics.
