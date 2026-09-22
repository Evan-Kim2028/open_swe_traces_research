# Exported API — manifest

Package `pkg/kubemanifest` (importable as `example.internal/clustkit/pkg/kubemanifest`).

Untyped Kubernetes manifest objects (`map[string]interface{}` wrappers) and a visitor.

- `NewObject(data)` — wraps a map; `ToUnstructured`, `GroupVersionKind`, `FromRuntimeObject`
  convert to/from typed objects.
- `LoadObjectsFrom(contents)` — splits a multi-doc YAML on `---` sections, SKIPS sections
  with no non-comment content, unmarshals the rest into `Object`s.
- `ObjectList.ToYAML()` — serializes objects joined by `\n---\n\n`; empty objects skipped.
- `Object.ToYAML`/`MarshalJSON` — marshal the underlying map.
- `Object.accept(visitor)` — walk all scalar/map fields.
- `Object.IsEmptyObject` — `len(data)==0`.
- `Object.Kind`/`GetNamespace`/`GetName`/`APIVersion` — typed getters returning `""` when the
  field is missing or not a string (namespace/name read from `metadata`).
- `Object.Reparse(obj, fields...)` — re-marshals a nested map path into a typed struct.
- `Object.Set(newValue, fieldPath...)` — sets a nested field; intermediate segments must
  exist as maps; the value is yaml-round-tripped into a `map[string]interface{}` so further
  mutation works.

Visitor: `VisitString`/`VisitBool`/`VisitFloat64`/`VisitMap`; `visit` recurses maps and
slices, invokes the typed callback with a mutator that writes back into the container.
`[]string` elements are skipped; other types error `unhandled type in manifest: %T`.
