# Exported API — reflectfmt

Package `util/pkg/reflectutils` (importable as `example.internal/clustkit/util/pkg/reflectutils`).

- `func ReflectRecursive(v reflect.Value, visitor visitorFunc, options *ReflectOptions) error` — call `visitor` on `v` and every recursive sub-value; `SkipReflection` skips the subtree.
- `func JSONMergeStruct(dest, src interface{})` — merge `src` into `dest` through a JSON round-trip.
- `func BuildTypeName(t reflect.Type) string` — human type name (`*T`, `[]T`, `map[K]V`).
- `func IsPrimitiveValue(v reflect.Value) bool` — primitive-kind predicate (note: strings and slices are NOT primitive here).
- `func FormatValue(value interface{}) string` — string form of a value.
- `func IsMethodNotFound(err error) bool` / `type MethodNotFoundError`.
- `var SkipReflection` sentinel; `ReflectOptions{JSONNames, DeprecatedDoubleVisit}` — untouched.

Production callers: `pkg/configbuilder/buildconfigfile.go`, `pkg/flagbuilder/build_flags.go`, `upup/pkg/fi/cloudup/{loader,populate_cluster_spec,spec_builder}.go`, `pkg/nodelabels/builder.go`.
