# Contract — fichanges

Reflection-based change detection between an actual object and an expected
spec (`BuildChanges`/`equalFieldValues`). Every commitment below is covered
by a hidden test; every hidden test maps to a commitment.

## Commitments

1. **Nil-pointer means don't care.** A nil pointer field in `e` is skipped;
   other nil kinds in `e` (maps, slices, scalars) are real expected values
   and are compared/copied normally. Covered by `TestDetail01`.
2. **Nil `a`.** A nil `a` (interface holding a nil pointer) copies every
   non-nil `e` field into `changes` and reports changed. Covered by
   `TestDetail02`.
3. **Unexported fields skipped.** Fields whose `PkgPath` is non-empty are
   ignored entirely — differences in them produce no change and no panic.
   Covered by `TestDetail03`.
4. **`CompareWithID` equality.** Values implementing `CompareWithID` compare
   equal only when both IDs are non-nil and equal; nil IDs fall through to
   deep equality of the whole value. Covered by `TestDetail04`.
5. **Resource readiness.** `Resource` fields compare via `ResourcesMatch`;
   `HasIsReady` is consulted on the expected side only — a not-ready expected
   resource is a change without comparing bytes, and a not-ready actual
   resource does not affect the comparison. Covered by `TestDetail05`.
6. **Container equality.** Maps and slices require identical nil-ness and
   length, then compare elementwise using the custom equality (so
   `CompareWithID` semantics apply to elements). Covered by `TestDetail06`.
7. **Comparison errors are fatal (shape).** An error from resource
   comparison inside `equalFieldValues` is not silently swallowed — it
   terminates the process (`klog.Fatalf`), since the function has no error
   return. Verified via a child process asserting non-zero exit. Covered by
   `TestDetail07`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | yes |
| TestDetail02 | 2 | yes |
| TestDetail03 | 3 | yes |
| TestDetail04 | 4 | partially — nil-ID fallthrough asserted via differing payloads |
| TestDetail05 | 5 | partially — expected-side-only readiness asserted |
| TestDetail06 | 6 | yes |
| TestDetail07 | 7 | partially — asserts death/non-swallow only, not the mechanism |
