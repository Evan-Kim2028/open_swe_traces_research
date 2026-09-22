# Closure — fideps

Package: `upup/pkg/fi` (`example.internal/clustkit/upup/pkg/fi`).

Files: `upup/pkg/fi/topological_sort.go` (5 funcs).

Removed functions (bodies stubbed): `NotADependency.GetDependencies`, `FindTaskDependencies`,
`reflectForDependencies`, `FindDependencies`, `getDependencies`.

Exported entry point(s): `FindTaskDependencies` — builds the dependency graph the executor
topologically sorts before running tasks.

Test files removed in excision: none (kept tests don't reach the closure).
