# Contract (L2) — codecbytes

`EncodeBytes` appends the memcomparable encoding of `data` to `b`: the
output is `[group][marker]` pairs where each group is 8 data bytes
zero-padded on the right and each marker is `0xFF` minus the pad count.
Empty input still emits a single all-pad group (`0x00`×8 then `0xF7`); an
input whose length is an exact multiple of 8 emits a full group marked
`0xFF` followed by the terminating all-pad group — the exact worked table
in the `EncodeBytes` doc comment is the format. `DecodeBytes` consumes
groups until the first marker that implies nonzero padding, returns the
decoded value and the leftover input as its first result, and reuses a
non-nil `buf` for the decoded output. It errors on a marker that implies
more than 8 pad bytes, on a nonzero padding byte, and on a truncated tail
— the error wording is unspecified. `DecodeBytes(EncodeBytes(nil, d))`
returns `d` verbatim with an empty leftover for every length including 0,
7, 8, 9 and 16. `reallocBytes` returns a slice with the same length and
contents that can hold `n` more bytes, reallocating only when `cap(b)` is
too small.

## Coverage

| hidden test | contract sentence |
|---|---|
| `TestDetail01` | each group is 8 data bytes zero-padded right and the marker is `0xFF` minus the pad count |
| `TestDetail02` | empty input emits one all-pad group terminated by `0xF7` |
| `TestDetail03` | an exact multiple of 8 emits a full `0xFF`-marked group plus the terminating all-pad group |
| `TestDetail04` | `DecodeBytes` stops at the first padding marker and returns the leftover input first |
| `TestDetail05` | decode errors (wording unspecified) on an impossible marker, nonzero padding, and truncation |
| `TestDetail06` | a non-nil `buf` is reused for the decoded output |
| `TestDetail07` | `DecodeBytes(EncodeBytes(nil, d))` returns `d` verbatim with an empty leftover |
| `TestDetail08` | `reallocBytes` preserves length and contents, grows capacity to hold `n` more, and does not reallocate when capacity suffices |
