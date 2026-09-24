# Closure — keyops

Package: `kv`. Files: `kv/key.go` (4 funcs), `kv/store_vars.go` (2 funcs).

Removed (bodies stubbed): `NextKey`, `PrefixNextKey`, `CmpKey`, `StrKey`,
`ReplicaReadType.IsFollowerRead`, `ReplicaReadType.String`.

Kept: `KeyRange`, `LockCtx` and its helpers in `kv/kv.go` (separate
surface), the `ReplicaReadType` constants, all flag machinery in
`kv/keyflags.go` (its own unit).

Tests deleted: `kv/key_test.go` (its only case was `TestPrefixNextKey`).
