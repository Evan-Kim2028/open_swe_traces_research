# Contract (L2) — codecnum

`EncodeInt`/`DecodeInt` write and read exactly 8 big-endian bytes whose byte
order sorts ascending in `int64` order (the sign bit is remapped — see
`signMask`); `EncodeIntDesc`/`DecodeIntDesc` produce and consume the
descending-order counterpart. `EncodeUint`/`DecodeUint` are 8-byte
big-endian in unsigned order and `EncodeUintDesc`/`DecodeUintDesc` the
descending form. Every `Decode*` returns the unconsumed tail of its input
as the first result — decoding `encode(v) ++ suffix` yields `v` and
`suffix` — and returns a nil tail plus an error when the input is too
short. `EncodeVarint`/`EncodeUvarint` emit standard protobuf varints
(zig-zag for the signed form) and are not mem-comparable; their decoders
report distinct errors for an over-64-bit value and a truncated input.
`EncodeComparableUvarint`/`EncodeComparableVarint` produce a tagged
big-endian form that is lexically ordered, non-negative varints sharing the
uvarint encoding; the comparable decoders return the original value (and,
for multi-byte forms, leave exactly the unconsumed suffix) and reject
malformed input (a tag byte below the valid range, a truncated body) with
an error. `EncodeIntToCmpUint` is the monotone int64→uint64
remap and `DecodeCmpUintToInt` its inverse.

## Coverage

| hidden test | contract sentence |
|---|---|
| `TestDetail01` | `EncodeInt`/`DecodeInt` are 8-byte big-endian ascending in `int64` order |
| `TestDetail02` | `EncodeIntDesc`/`DecodeIntDesc` order descending |
| `TestDetail03` | `EncodeUint`/`EncodeUintDesc` are 8-byte big-endian ascending/descending in unsigned order |
| `TestDetail04` | every `Decode*` returns the leftover tail and errors with a nil tail on short input |
| `TestDetail05` | `EncodeVarint`/`EncodeUvarint` are standard protobuf varints (zig-zag signed) and not mem-comparable |
| `TestDetail06` | varint decoders distinguish an over-64-bit value from a truncated input with different errors |
| `TestDetail07` | `EncodeComparableUvarint` is lexically ordered and round-trips |
| `TestDetail08` | `EncodeComparableVarint` orders mixed-sign values lexically and shares the uvarint form for non-negatives |
| `TestDetail09` | `DecodeComparableUvarint` errors on an out-of-range first byte and consumes exactly the encoded length |
| `TestDetail10` | `DecodeComparableVarint` returns the original value for negatives and non-negatives (multi-byte forms leave exactly the suffix) and rejects truncated input |
| `TestDetail11` | each `Decode*` of matching `Encode*` output returns the original value and the untouched suffix |
| `TestDetail12` | `EncodeIntToCmpUint` is the monotone sign remap and `DecodeCmpUintToInt` inverts it |
