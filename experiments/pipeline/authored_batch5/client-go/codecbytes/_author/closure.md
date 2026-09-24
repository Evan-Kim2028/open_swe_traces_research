# Closure — codecbytes

Package: `util/codec`. File: `bytes.go` (195 lines).

Removed (bodies stubbed): `EncodeBytes`, `decodeBytes`, `DecodeBytes`,
`reallocBytes`.

Kept: `encGroupSize`/`encMarker`/`encPad` constants, `pads`,
`fastReverseBytes`/`safeReverseBytes`/`reverseBytes`/`supportsUnaligned`
(the descending-decode helpers — `decodeBytes`' `reverse` path is never
reached from the exported API in this tree, so they stay visible as dead
code), `wordSize`.

Tests deleted: none — `util/codec` has no `_test.go` files in this tree.
