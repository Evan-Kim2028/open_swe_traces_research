# Contract — fideps

Dependency inference between `Task[T]`s via the `HasDependencies` interface
or reflection. Every commitment below is covered by a hidden test; every
hidden test maps to a commitment.

## Commitments

1. **`NotADependency` opts out.** Its `GetDependencies` returns nil, and a
   task embedding it reports no dependencies even when it has task fields —
   embedding skips the reflective scan. Covered by `TestDetail01`.
2. **Interface wins.** A task implementing `HasDependencies` reports exactly
   its declared deps; its struct fields are not scanned. Covered by
   `TestDetail02`.
3. **Deps are map keys.** Dependencies are reported as the task's key in the
   task map; every task gets an entry (empty for no deps). Covered by
   `TestDetail03`.
4. **Nil skipped, missing fatal (shape).** Typed and untyped nil deps are
   silently skipped; a non-nil dep absent from the task map terminates the
   process rather than being silently accepted (verified via child process).
   Covered by `TestDetail04`.
5. **Reflective walk.** The task's own root struct, primitives, and strings
   are ignored; the walk descends through ptr/interface/slice/map to find
   task values. Covered by `TestDetail05`.
6. **Struct precedence.** A struct field implementing both `HasDependencies`
   and `Task` contributes its declared deps AND itself; a `Resource`-only
   struct is ignored; any other non-task struct type is an unhandled type —
   fatal rather than silently ignored (verified via child process). Covered
   by `TestDetail06`.
7. **`FindDependencies`.** The same inference applies to an arbitrary
   non-task object. Covered by `TestDetail07`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | yes |
| TestDetail02 | 2 | yes |
| TestDetail03 | 3 | yes |
| TestDetail04 | 4 | partially — asserts skipped/fatal, not the mechanism |
| TestDetail05 | 5 | yes |
| TestDetail06 | 6 | partially — precedence asserted; "error" asserted as process death |
| TestDetail07 | 7 | yes |
