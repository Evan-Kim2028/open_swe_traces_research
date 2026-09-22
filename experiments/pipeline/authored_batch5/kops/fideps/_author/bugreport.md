# Bug report

Task dependency inference is broken: `FindTaskDependencies` no longer maps task keys to their
dependency keys (edges come back empty or fatal on valid deps), the `HasDependencies`
interface is ignored in favour of reflection, nil dependencies aren't skipped, task-valued
struct fields aren't discovered by the reflective walk, struct `Resource` fields are reported
as unhandled, and `NotADependency` no longer opts out.

Expected: declared deps via `HasDependencies`, reflective discovery of `Task`/`HasDependencies`
struct fields, key-based edge map, nil-dep skip, `Resource`-struct skip.

Reproduce with:

```
go test -count=1 ./upup/pkg/fi/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the
internet; work only from the repository.
