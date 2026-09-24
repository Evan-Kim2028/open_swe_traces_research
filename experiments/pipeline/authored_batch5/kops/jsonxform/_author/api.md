# Exported API — jsonxform

Package `pkg/jsonutils` (importable as `example.internal/clustkit/pkg/jsonutils`).

In-place JSON-tree transformation with path tracking.

- `NewTransformer()` — empty transformer.
- `AddStringTransform(fn)` / `AddObjectTransform(fn)` / `AddSliceTransform(fn)` — register
  callbacks run at every string / map / slice node; multiple transforms compose in order.
- `Transform(v map[string]any)` — walks the tree, applying registered transforms in place.
- Path syntax: dotted keys (`spec.template`), slices append `[]` to the path (the slice
  transform AND its elements see `path[]`).
- Object transforms run BEFORE descending into the map's children; slice transforms run
  before element visits and may replace the whole slice.
- `SortSlice(s []any)` — returns a copy sorted by each element's JSON encoding
  (marshal errors propagate).

Example: a string transform receives `path="metadata.name"` for the `name` field; slice
elements of `containers` are visited at `spec.containers[]`.
