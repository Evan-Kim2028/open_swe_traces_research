# Difficulty — hexdump

predicted_flip: L1
details: 4

Missed edges: the `%s`-on-non-byte-slice quirk (`[%!s(uint64=1)]`,
`[][]byte` → `[a bc]`); hex only at element-kind `uint8`; `XXX_` field
skipping; top-level nil handling via the Ptr branch.

Hardness driver: the file is tiny — a solver who writes any plausible
reflection printer passes the doc examples but will most likely emit
`[1 22]` for `[]uint64` instead of the faithful `%!s` garbage the real
code produces.
