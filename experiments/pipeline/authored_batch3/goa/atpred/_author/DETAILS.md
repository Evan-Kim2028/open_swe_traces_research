# Commitments — atpred

1. AllRequired recurses into the attribute's type when it is a user type
   (e.g. a Reference target) and returns the validation's Required list
   otherwise. In-tree coverage: trimmed tests. Inferable: partially —
   the user-type recursion is documented.
2. IsRequired is membership in AllRequired. In-tree coverage: trimmed
   tests. Inferable: yes.
3. IsRequiredNoDefault is IsRequired plus the child having no default
   value. In-tree coverage: trimmed tests. Inferable: yes.
4. IsPrimitivePointer requires the child attribute to exist, have an
   unaliased primitive type whose kind is neither Bytes nor Any, not be
   required, and either have no default or useDefault be false. In-tree
   coverage: trimmed tests. Inferable: no — the Bytes/Any exclusion and
   the useDefault truth table are policy.
5. unalias follows user types to their underlying attribute type
   recursively, not just one level. In-tree coverage: none directly.
   Inferable: partially.
6. HasTag/HasTagPrefix inspect the META of the object's member
   attributes — not the receiver's own Meta. TaggedAttribute returns
   the member's name and recurses into the attribute's Bases. In-tree
   coverage: trimmed tests. Inferable: no — member-vs-receiver is a
   subtle detail.
7. FieldTag returns the LAST value of "rpc:tag" meta. In-tree coverage:
   none directly. Inferable: no.
8. GetDefault returns the child's DefaultValue, falling back to the
   user type's own default when the child type is a non-object user
   type. SetDefault converts MapVal/ArrayVal to plain map/slice.
   In-tree coverage: trimmed tests. Inferable: no.
9. Find searches the object's own members, then Bases, then References.
   In-tree coverage: trimmed tests. Inferable: partially.
10. Delete removes the member from the object, removes it from required,
    and deletes the key from any map-valued user examples; on a user
    type it delegates to the type's attribute. In-tree coverage: none
    directly. Inferable: no — the example cleanup is hidden.
11. walkAttribute iterates the attribute's object members, then its
    Bases and References as attributes, recursing through user types.
    In-tree coverage: none directly. Inferable: partially.
