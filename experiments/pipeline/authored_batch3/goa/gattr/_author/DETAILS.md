# Commitments — gattr

1. Copy returns an attribute whose type graph shares NO mutable nodes
   with the input: user types, objects, arrays, maps, unions, attribute
   metadata, validations, defaults, and examples are all detached.
   In-tree coverage: none directly. Inferable: partially.
2. Shared subtrees stay shared: two references to the same attribute or
   user type produce ONE copy reachable from both paths — the copy graph
   preserves the original sharing topology. In-tree coverage: none.
   Inferable: no — this is the point of the type but invisible in the
   signature.
3. Original maps a copied attribute back to the attribute it was copied
   from, and returns nil for attributes that were never copied.
   In-tree coverage: none. Inferable: partially.
4. Recursive types copy correctly: a user type that refers to itself
   produces a copy whose self-reference points at the copy, not the
   original. In-tree coverage: none. Inferable: no.
5. Mutable leaf values (maps, slices, pointers inside defaults,
   examples, meta, and validation values) are deep-copied via
   reflection; cycles inside reflected values are rejected rather than
   silently aliased. In-tree coverage: none. Inferable: no.
6. Immutable scalars (strings, numbers, bools) are copied by value.
   In-tree coverage: none. Inferable: yes.
