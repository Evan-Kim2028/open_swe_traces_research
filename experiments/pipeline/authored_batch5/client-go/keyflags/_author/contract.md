# Contract (L2) — keyflags

`KeyFlags` is a `uint16` whose top bit is reserved; no flag operation ever
sets it. The assertion pair is two bits with four states: neither set, the
exist bit alone, the nonexist bit alone, or both (unknown). The exist and
nonexist predicates are true only in their exclusive states, the unknown
predicate requires both bits, and the any-assertion predicate requires at
least one. The presume predicate reads either the live or the previous
presume bit. Ops apply left to right: the presume-set op also sets the
need-check bit, and the presume-clear op removes both. The locked-value
ops set or clear the locked-value bit. Assert ops keep the pair consistent:
setting one side clears the other, the unknown op sets both, and the none
op clears both. `AndPersistent` keeps only the persistent flag set — all
other bits drop. Unknown ops leave the value unchanged. The nine named
predicates are single-bit tests: false on a zero value, true after the op
that sets their bit.

| test | commitment |
| --- | --- |
| `TestDetail01` | No flag op ever sets the reserved top bit, applied together or individually. |
| `TestDetail02` | The assertion pair truth table: exclusive states read exclusively, both bits read unknown, any bit reads as an assertion, and the none op clears the pair. |
| `TestDetail03` | The presume predicate is true when either the live or the previous presume bit is set, false on a zero value. |
| `TestDetail04` | Ops apply left to right; the presume-set op also sets need-check and the presume-clear op clears both, with the later op winning. |
| `TestDetail05` | The locked-value-set op sets the locked-value bit and the not-exists op clears it; the cross-flag clearing of the constraint bit is internal and not asserted. |
| `TestDetail06` | Setting one assert side clears the opposite bit, the unknown op sets both bits, and the none op clears both. |
| `TestDetail07` | `AndPersistent` keeps exactly the persistent flag set and drops every other bit. |
| `TestDetail08` | Unknown ops leave the flag value unchanged — shape: the returned value equals the input. |
| `TestDetail09` | Each named predicate is a single-bit test: false on zero, true after its corresponding set op. |
