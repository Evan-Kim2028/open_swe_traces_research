# Closure — codecnum

Package: `util/codec`. File: `number.go` (305 lines, 18 funcs).

Removed (all bodies stubbed): `EncodeIntToCmpUint`, `DecodeCmpUintToInt`,
`EncodeInt`, `EncodeIntDesc`, `DecodeInt`, `DecodeIntDesc`, `EncodeUint`,
`EncodeUintDesc`, `DecodeUint`, `DecodeUintDesc`, `EncodeVarint`,
`DecodeVarint`, `EncodeUvarint`, `DecodeUvarint`, `EncodeComparableVarint`,
`EncodeComparableUvarint`, `DecodeComparableUvarint`, `DecodeComparableVarint`.

Kept: `signMask`, `negativeTagEnd`/`positiveTagStart` constants,
`errDecodeInsufficient`/`errDecodeInvalid` sentinels (error VALUES remain;
only the functions that return them are stubbed).

Tests deleted: none — `util/codec` has no `_test.go` files in this tree.
