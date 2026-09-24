# Contract (L2) — keyops

`NextKey` returns the key with one `0x00` appended — the immediate successor
in byte order. `PrefixNextKey` increments the last byte and propagates the
carry left, producing a strictly greater key that never extends the slice —
a key that is all `0xFF` carries out to an empty result, as does an empty
key.
`CmpKey` is unsigned lexicographic byte order returning -1/0/+1.
`StrKey` renders the key as hex. `ReplicaReadType.IsFollowerRead` is false
for `ReplicaReadLeader` and true for `ReplicaReadFollower`.
`ReplicaReadType.String` names `leader`/`follower`/`mixed`/`learner` with
their lowercase names, renders prefer-leader as a lowercase name mentioning
leader, and renders an out-of-range value as a marker distinct from every
named kind.

## Coverage

| hidden test | contract sentence |
|---|---|
| `TestDetail01` | `NextKey` appends a single `0x00` |
| `TestDetail02` | `PrefixNextKey` increments the last byte and carries left without extending |
| `TestDetail03` | an all-`0xFF` key carries out to an empty result |
| `TestDetail04` | `PrefixNextKey` on an empty key returns empty without panicking |
| `TestDetail05` | `CmpKey` is unsigned lexicographic order |
| `TestDetail06` | `StrKey` renders the key as hex |
| `TestDetail07` | `IsFollowerRead` is false for leader and true for follower |
| `TestDetail08` | `String` names the known kinds in lowercase and marks unknowns distinctly |
