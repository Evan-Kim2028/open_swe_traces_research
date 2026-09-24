# Contract — manifest

Untyped manifest objects and the visitor in `pkg/kubemanifest`. Every
commitment below is covered by a hidden test; every hidden test maps to a
commitment.

## Commitments

1. **`LoadObjectsFrom`** skips sections whose only lines are empty or
   `#`-comments and loads the rest in order. Covered by `TestDetail01`.
2. **`ObjectList.ToYAML` (shape).** Emits each non-empty object with a
   document separator between them; `IsEmptyObject` entries are dropped;
   the output round-trips through `LoadObjectsFrom`. Exact separator bytes
   are an implementation detail. Covered by `TestDetail02`.
3. **Typed getters** return `""` for missing OR wrong-typed fields;
   `GetName`/`GetNamespace` read `metadata.name`/`metadata.namespace`.
   Covered by `TestDetail03`.
4. **`Reparse`** navigates intermediate fields as maps and re-marshals the
   leaf into the target; a missing field or non-map value errors naming the
   offending field. Covered by `TestDetail04`.
5. **`Object.Set`** requires intermediate path segments to already exist as
   maps — it does not create them; the leaf is stored as a
   yaml-round-tripped map copy. Covered by `TestDetail05`.
6. **`visit` mutation.** Mutators write back into the parent map/slice;
   path elements are field names and `[i]` for slice indexes; `[]string` is
   silently skipped; other concrete types error. Covered by
   `TestDetail06`.
7. **`visitorBase`** callbacks are no-ops — a bare embed walks without
   error, and a visitor overriding only `VisitString` still sees nested
   strings. Covered by `TestDetail07`.
8. **Root cannot be replaced (shape).** The object root is always a map and
   map visits carry no mutator, so root replacement is unreachable through
   `accept`; the mutator plumbing does reach a scalar root (the path the
   fatal guard protects). The fatal-exit itself is not directly reachable
   and is asserted only as "root mutator never invoked for a map root".
   Covered by `TestDetail08`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | yes |
| TestDetail02 | 2 | no — shape only (separator exists, empty dropped, round-trips) |
| TestDetail03 | 3 | yes |
| TestDetail04 | 4 | partially — error names the field; text not pinned |
| TestDetail05 | 5 | partially — no-create asserted; round-trip copy asserted |
| TestDetail06 | 6 | partially — `[]string` skip and `[i]` paths asserted |
| TestDetail07 | 7 | yes |
| TestDetail08 | 8 | partially — fatal unreachable; reachable shape asserted (see note) |
