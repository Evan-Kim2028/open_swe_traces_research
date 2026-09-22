# Commitments — unionid

1. NewUnionDeclarationID pairs the AUTHORED attribute (via
   AuthoredAttribute) with the union type ID so transport copies of the
   same attribute resolve to one declaration. In-tree coverage: trimmed
   tests. Inferable: partially.
2. NewUnionTypeID is a repeatable key covering the union's effective
   JSON envelope keys (GetTypeKey/GetValueKey), every branch name, and
   every detail that changes generated Go: nilability, struct:field:type
   meta, branch types, field names/tags/pointer shape for object
   branches, package location for user types. In-tree coverage: trimmed
   tests. Inferable: no — the field set is implementation detail.
3. Keys are length-prefixed ("len:value") so different component lists
   cannot produce ambiguous concatenations. In-tree coverage: trimmed
   tests. Inferable: no.
4. Recursive references to the same object/union/user type within one
   ID emit a back-reference marker plus index instead of infinite
   recursion; the map is popped after the subtree finishes (siblings
   get independent numbering). In-tree coverage: trimmed tests.
   Inferable: no.
5. User types contribute both their Goified name and their package
   location (empty when none) plus a recursive attribute ID. In-tree
   coverage: trimmed tests. Inferable: no.
6. Unknown branch types panic rather than silently producing a weak
   key. In-tree coverage: none. Inferable: partially.
