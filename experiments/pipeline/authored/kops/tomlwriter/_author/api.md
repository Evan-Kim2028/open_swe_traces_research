# Exported API — tomlwriter

`NewTree() *Tree`

`(*Tree) Table(path ...string) *Tree` — create/return nested table; a path element that is already a scalar does not descend.

`(*Tree) SetPath(path []string, value any)` — only `string`, `int64`, `bool`; other types panic. A leaf overwrites a table.

`(*Tree) String() string` — byte-stable TOML: scalars before subtables, keys sorted, blank line before each `[table]`, two-space indent per level. Bare keys are `[A-Za-z0-9_-]`; empty key is `""`; keys already wrapped in `"` pass through unescaped.

Callers: containerd config rendering. In-tree tests cover empty docs, quoting, escaping, SetPath-through-scalar, overwrite, unsupported types.
