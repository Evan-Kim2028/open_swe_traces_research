# Commitments — unionid

1. `NewUnionDeclarationID` panics when the attribute's type is not a OneOf
   union. In-tree coverage: none. Inferable: partially — the package's
   fail-fast norm covers panicking on a violated precondition; panic-vs-error
   is a choice.
2. Two attributes carrying the same authored attribute and the same emitted
   definition produce equal declaration IDs; attributes backed by different
   authored OneOf declarations do not. In-tree coverage: none. Inferable: doc —
   the type comment states copies of one authored attribute share an identity
   while separate declarations do not.
3. `NewUnionTypeID` is deterministic: equal union expressions yield equal
   keys on every call. In-tree coverage: none. Inferable: yes — a "repeatable
   key" is the stated purpose.
4. Distinct emitted definitions yield distinct keys: the encoding is
   unambiguous, i.e. no two different key parts can concatenate to the same
   key. In-tree coverage: none. Inferable: partially — unambiguity is
   committed, the length-prefixed spelling is not (assert shape only).
5. Two unions that differ only in branch order produce different keys.
   In-tree coverage: `TestNameScope_GoTypeNameDistinguishesUnionBranchOrder`
   (removed). Inferable: partially — order-sensitivity follows from "every
   detail that changes the generated definition".
6. Branch value names contribute to the key; renaming a branch changes it.
   In-tree coverage: none. Inferable: yes.
7. The key includes the union's effective JSON envelope keys (type key and
   value key). In-tree coverage: none. Inferable: doc — stated in the
   `NewUnionTypeID` comment.
8. The key includes whether each branch value may be nil under the emitted
   Go layout (pointer-ness driven by the design, e.g. defaults). In-tree
   coverage: `TestUnionTypeIDIncludesDefaultDrivenPointerShape` (removed).
   Inferable: doc — "whether the value may be nil" is listed in the comment.
9. The key includes `struct:field:type` metadata values on branch
   attributes. In-tree coverage: none. Inferable: doc — "field type
   metadata" is listed in the comment.
10. Primitive branches contribute their Go-native type spelling; user-type
    branches contribute the Goified type name and the type's package import
    path when the type is emitted in a different package. In-tree coverage:
    `TestNameScope_GoTypeNameDistinguishesUnionBranchPackages` (removed).
    Inferable: partially — "package location" is documented; the exact
    components are a choice.
11. User-type branches contribute the referenced type's shape, so two user
    types with equal names but different structure yield different keys; a
    user type that occurs twice inside one key is written as a back-reference
    to its first occurrence. In-tree coverage:
    `TestUnionTypeIDIncludesGeneratedUserTypeShape`,
    `TestUnionTypeIDEncodesRecursiveGeneratedUserTypeShape` (removed).
    Inferable: partially — shape-inclusion follows from "every detail that
    changes a generated Go branch type"; the back-reference encoding does not.
12. Recursive unions terminate: a union reachable from itself is written as
    a back-reference instead of recursing forever. In-tree coverage:
    `TestUnionTypeIDEncodesRecursiveGeneratedUserTypeShape` (removed).
    Inferable: yes — termination is mandatory; encoding is not asserted.
13. Object (inline struct) branches contribute each field's Goified field
    name, serialized struct tag, and pointer-ness. In-tree coverage: none.
    Inferable: partially.
14. Array and map branches contribute their element/key shapes recursively;
    nested unions recurse through the same machinery. In-tree coverage:
    `TestUnionTypeID` (removed). Inferable: yes.
15. Pointer sharing that is not emitted in the Go definition does not change
    the key. In-tree coverage:
    `TestUnionTypeIDIgnoresNonEmittedPointerSharing` (removed). Inferable:
    partially — only emitted details are committed.
