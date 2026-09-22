# Commitments — namedecl

1. Name() panics until the owning generation freezes the declaration.
   In-tree coverage: trimmed tests. Inferable: yes — documented.
2. NewPreferredName goifies the requested spelling to match the
   visibility (exported vs unexported). In-tree coverage: trimmed
   tests. Inferable: partially.
3. A dependent name's preferred spelling is prefix + base preferred (or
   base final once frozen) + suffix. In-tree coverage: trimmed tests.
   Inferable: no — the frozen-base switch is a subtle detail.
4. validateNameDeclaration requires a valid kind, a valid visibility
   (unless dependent or exact), a non-empty preferred name, and a valid
   Go identifier for exact names. In-tree coverage: trimmed tests.
   Inferable: partially.
5. comparePackageNameOrders compares order values by package path, then
   type name, then the order's own ComparePackageName — never by
   discovery order or pointer identity. In-tree coverage: trimmed
   tests. Inferable: no.
6. validatePackageNameOrder requires a named, non-pointer type whose
   fields are all stable value types (arrays, structs, bools, ints,
   uints, floats, complexes, strings). In-tree coverage: trimmed tests.
   Inferable: no — the stable-type whitelist is arbitrary.
7. packagePath panics when the declaration has no owning generated
   package. In-tree coverage: trimmed tests. Inferable: partially.
8. Exact names keep their spelling verbatim; preferred names may gain a
   numeric suffix at freeze. In-tree coverage: trimmed tests.
   Inferable: doc.
