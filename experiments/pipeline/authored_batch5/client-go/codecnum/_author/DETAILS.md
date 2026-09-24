# Details — codecnum

1. `EncodeInt`/`DecodeInt` write/read exactly 8 big-endian bytes; the encoded
   byte order sorts ascending in `int64` order — i.e. the sign bit is flipped
   before the big-endian write (`EncodeIntToCmpUint`). Inferable: yes — the
   doc comments promise ascending comparison order.
2. `EncodeIntDesc`/`DecodeIntDesc` produce/consume the bitwise complement of
   the ascending encoding, so larger `v` sorts earlier lexically. Inferable:
   partially — descending order is documented, the complement mechanism is not.
3. `EncodeUint`/`DecodeUint` are 8-byte big-endian with NO sign flip;
   `EncodeUintDesc`/`DecodeUintDesc` complement it. Inferable: partially.
4. Every `Decode*` returns the leftover slice (input minus consumed bytes) as
   its first result, and `nil` leftover plus an "insufficient bytes to decode
   value" error when the input is too short. Inferable: partially — the
   leftover-return convention is documented, the exact split is not.
5. `EncodeVarint`/`EncodeUvarint` emit standard protobuf varint (signed uses
   zig-zag) and are explicitly NOT mem-comparable. Inferable: doc — the
   comment says "not memcomparable".
6. `DecodeVarint`/`DecodeUvarint` distinguish two failures: an over-64-bit
   value reports "value larger than 64 bits"; a truncated input reports
   "insufficient bytes to decode value". Inferable: partially.
7. `EncodeComparableUvarint` stores values 0..239 in ONE byte worth `v+8`;
   larger values get a tag byte `247+len` followed by `len` big-endian bytes,
   so the encoded stream is lexically ordered. Inferable: doc — the tag
   scheme is spelled out in the function comment.
8. `EncodeComparableVarint` maps negatives onto a tag byte `8-len` (0..7)
   followed by `len` big-endian two's-complement bytes — a MORE negative value
   uses MORE bytes and a SMALLER tag; non-negatives delegate to the uvarint
   form. Inferable: partially.
9. `DecodeComparableUvarint` treats a first byte below 8 as an "invalid"
   error, bytes 8..247 as an inline value `first-8` consuming just the tag
   byte, and bytes above 247 as `first-247` big-endian bytes. Inferable:
   partially.
10. `DecodeComparableVarint` seeds negative decodes with all-ones so the
    reassembled value lands above `MaxInt64`; a negative-tagged buffer that
    decodes to `<= MaxInt64`, or a positive-tagged buffer that decodes above
    it, is an "invalid" error. Inferable: no.
11. Round-trip: each `Decode*` of the matching `Encode*` output returns the
    original value and the untouched suffix. Inferable: yes.
12. `EncodeIntToCmpUint(0)` is `0x8000000000000000` and
    `EncodeIntToCmpUint(-1)` is `0x7fffffffffffffff` — order flips at zero.
    Inferable: partially — derivable once the sign-flip rule is known.
