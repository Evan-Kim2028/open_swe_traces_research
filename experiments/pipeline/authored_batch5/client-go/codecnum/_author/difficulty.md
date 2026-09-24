# Difficulty — codecnum

predicted_flip: L2
details: 12

Missed edges: comparable-varint tag arithmetic (`8-len` vs `247+len`), the
239 single-byte cutoff, negative seeding to `MaxUint64` plus the range check
that turns wrong-tag decodes into "invalid", descending = bitwise complement
of ascending, leftover-slice returns on partial input.

Hardness driver: three different integer encodings share one file (sign-flip
big-endian, zig-zag varint, tag-prefixed comparable); the tag byte values are
easy to off-by-one and the negative-decode validity rule is easy to invert.
