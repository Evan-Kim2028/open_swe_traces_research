# Commitments — goval

1. render dispatches on the attribute's type — primitive, array, map,
   object, union — through the layout's planned shape, and wraps the
   expression in a named local declaration when the value must be
   stored through a pointer. In-tree coverage: deleted tests only.
   Inferable: partially.
2. renderPrimitive honors struct:field:type meta: identical native name
   renders the literal; a custom type validates the authored Go type
   (pkg path + type name) and emits TypeName(literal). In-tree
   coverage: deleted tests only. Inferable: no.
3. renderPrimitiveLiteral writes kind-appropriate literals: bool via
   FormatBool, ints via FormatInt (signed ints accepted for unsigned
   only when non-negative), floats via FormatFloat 'g' rejecting
   NaN/Inf with 32-bit precision for Float32, strings via Quote, bytes
   as []byte("...") or a hex list, any via renderAnyValue. In-tree
   coverage: deleted tests only. Inferable: no — per-kind conversion
   matrix is detail.
4. renderArray/renderMap render every element/entry under the planned
   element/key layout; map keys sort by their rendered literal so
   output is source-order independent. In-tree coverage: deleted tests
   only. Inferable: no.
5. renderObject writes only fields present in the authored value,
   looked up through maps or Go structs (GoifyAtt'd field names), using
   the per-field planned layout. In-tree coverage: deleted tests only.
   Inferable: partially.
6. renderUnion reads the envelope's type-key and value-key, finds the
   named branch, and calls the resolved constructor with the rendered
   branch argument; missing envelope keys or unknown branches are
   errors. In-tree coverage: deleted tests only. Inferable: no.
7. concreteGoValue unwraps interfaces and pointers and rejects nil;
   goValueIsNil reports nil after the same unwrapping. In-tree
   coverage: deleted tests only. Inferable: partially.
8. renderAnyValue emits JSON-compatible Go values with explicit
   container types ([]any{...}, map[string]any{...} with sorted keys,
   quoted keys). In-tree coverage: deleted tests only. Inferable:
   partially.
9. validateCustomDefault requires the authored value's Go type to match
   the declared custom type's package path and name exactly.
   underlyingPrimitive unwraps user types to their base primitive.
   In-tree coverage: none. Inferable: no.
10. authoredObjectField finds a field in a map (string or any keys) or
    struct; reflectedMapValue matches string keys among any-typed keys.
    In-tree coverage: none. Inferable: partially.
