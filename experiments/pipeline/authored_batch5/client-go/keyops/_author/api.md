# Exported API — keyops

Package `kv` (module `example.internal/kvstore/v2`) — key successor helpers
plus the replica-read enum.

`NextKey(k []byte) []byte`, `PrefixNextKey(k []byte) []byte`,
`CmpKey(a, b []byte) int`, `StrKey(k []byte) string`.
`ReplicaReadType` (`ReplicaReadLeader`, `ReplicaReadFollower`,
`ReplicaReadMixed`, `ReplicaReadLearner`, `ReplicaReadPreferLeader`) with
`IsFollowerRead() bool` and `String() string`.

Callers: `internal/locate` (region scans), `wirerpc` (request flags),
`kvclient`. In-tree tests removed: `kv/key_test.go` (covered
`PrefixNextKey`).
