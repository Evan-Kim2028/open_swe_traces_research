# Commitments — errdiff

1. An error name must be a valid identifier and must not collide with
   another error name on the same scope; duplicates and empty names are
   validation errors. In-tree coverage: deleted tests only. Inferable:
   partially.
2. Two error expressions are equivalent when their effective attributes
   compare equal — including attributes merged from bases and
   references — not merely when the declared attribute is equal.
   In-tree coverage: deleted tests only. Inferable: no — effective
   attribute materialization is the hidden step.
3. The effective attribute flattens an object type's fields plus
   inherited bases/references into a standalone comparison shape.
   Ordering differences in merged fields do not change equivalence.
   In-tree coverage: deleted tests only. Inferable: no.
4. Dup copies the error name, description, and a deep copy of the
   attribute so mutations on the copy never leak into the source.
   In-tree coverage: deleted tests only. Inferable: partially.
5. Finalize resolves the error's attribute type (including recursive
   user-type references) before transports read it. In-tree coverage:
   none directly. Inferable: partially.
6. Temporary/timeout/fault flag helpers derive from the error name's
   canonical classification. In-tree coverage: none. Inferable: no.
