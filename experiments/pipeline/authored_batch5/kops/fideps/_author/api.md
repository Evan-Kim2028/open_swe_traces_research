# Exported API — fideps

Package `upup/pkg/fi` (importable as `example.internal/clustkit/upup/pkg/fi`).

Dependency inference between `Task[T]`s.

- `type NotADependency[T]` — embeddable marker; `GetDependencies` returns nil so embedding
  types are never treated as dependencies.
- `func FindTaskDependencies[T SubContext](tasks map[string]Task[T]) map[string][]string` —
  maps each task key to the keys of its dependencies. Uses `HasDependencies.GetDependencies`
  when implemented, else reflection. Nil deps are skipped; a dep not in the map is fatal.
- `func reflectForDependencies[T](tasks, task)` — reflection path: walks the task struct.
- `func FindDependencies[T](tasks, o)` — same inference for an arbitrary object.
- `func getDependencies[T](tasks, v reflect.Value)` — recursive struct walk: primitives,
  strings, interfaces, ptrs, slices and maps are ignored (the walk descends into them);
  a struct field that is a `Task[T]` or `HasDependencies[T]` contributes deps (a
  `HasDependencies` struct contributes its declared deps AND itself when also a Task);
  a struct implementing `Resource` is ignored; any other struct type is an error.

Example: task A has field `B *BTask`; `FindTaskDependencies({"a":A,"b":B})` →
`{"a": ["b"], "b": []}`.
