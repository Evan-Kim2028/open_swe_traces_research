# Commitments — defval

1. An attribute with no authored default produces no errors; a present
   default is checked under context "<ctx>default value" and nested
   attributes are walked once each — descent stops at named types.
   In-tree coverage: deleted tests only. Inferable: partially.
2. Nesting context labels are fixed: `field %q`, ` element`,
   ` map key`, ` map value`, ` OneOf branch %q`. In-tree coverage:
   deleted tests only. Inferable: no.
3. A nil authored value is "must not be nil"; interfaces and pointers
   are unwrapped to the concrete value first. In-tree coverage: deleted
   tests only. Inferable: partially.
4. Primitives must be compatible with the authored Go value — EXCEPT a
   value already carrying the exact custom field type declared in
   `struct:field:type` meta, which is converted by package path plus
   trailing type name. In-tree coverage: deleted tests only.
   Inferable: no.
5. Numeric defaults must FIT the design primitive: signed bounds per
   bit-width (0 = platform int), unsigned bounds likewise, Float32
   capped at MaxFloat32 with NaN/Inf rejected for both float widths.
   In-tree coverage: deleted tests only. Inferable: no.
6. Union-typed defaults require a map value; objects accept map or
   struct; arrays require array/slice; maps require map. In-tree
   coverage: deleted tests only. Inferable: partially.
7. Any-typed defaults accept only JSON-shaped Go values — bool, ints,
   uints, strings, finite floats, slices, and maps with string keys —
   nils anywhere are fine. In-tree coverage: deleted tests only.
   Inferable: no.
8. Enum membership compares numeric values after converting both sides
   to the design primitive's reflect type; non-numeric enums compare by
   deep equality. In-tree coverage: deleted tests only. Inferable: no.
9. Format and pattern rules apply only when the authored value is a
   string; min/max and exclusive bounds compare via float64; string
   length counts RUNES, collections and bytes count elements. In-tree
   coverage: deleted tests only. Inferable: partially — rune-vs-byte is
   the detail.
10. Object defaults: non-string map keys are reported by type, required
    fields by design name must be present, unknown fields error, and
    struct literals are matched through the generated Go field name
    (struct:field:name meta + goify). In-tree coverage: deleted tests
    only. Inferable: no.
11. Union defaults need the canonical envelope: exactly the type key and
    value key, the discriminator a string naming a declared branch, and
    the branch's own contract applied to its value. In-tree coverage:
    deleted tests only. Inferable: no.
12. Map iteration and object-field errors report in deterministic
    order: entries sorted by printed key, names sorted, invalid key
    types sorted. In-tree coverage: none. Inferable: no.
