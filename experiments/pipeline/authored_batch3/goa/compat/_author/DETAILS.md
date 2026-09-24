# Commitments — compat

1. Primitive.IsCompatible accepts any value for Any; bool only for
   Boolean; every Go int/uint width smaller than 64 bits for any
   integer or float primitive; int64/uint64 only for Int64/UInt64/float;
   float32/64 only for float primitives; string for String or Bytes;
   []byte only for Bytes. In-tree coverage: trimmed
   TestPrimitiveIsCompatible. Inferable: no — the width-widening matrix
   is an arbitrary policy table.
2. Array.IsCompatible rejects non-array/slice Go values and checks every
   element recursively against the element type. In-tree coverage:
   trimmed tests. Inferable: partially.
3. Object.IsCompatible accepts Go maps and structs. In-tree coverage:
   trimmed tests. Inferable: partially.
4. Map.IsCompatible rejects non-map values and checks every key and
   element recursively. In-tree coverage: trimmed tests. Inferable:
   partially.
5. Union.IsCompatible requires exactly a two-key map holding the
   discriminator tag and the value; the tag must name a union member
   and the value must be compatible with that member's type. In-tree
   coverage: trimmed tests. Inferable: no — the two-key tagged-envelope
   shape is implementation-only.
6. GetTypeKey returns the union's TypeKey field, defaulting to "type";
   GetValueKey returns ValueKey, defaulting to "value". In-tree
   coverage: trimmed tests. Inferable: doc — defaults named in the doc
   comment.
