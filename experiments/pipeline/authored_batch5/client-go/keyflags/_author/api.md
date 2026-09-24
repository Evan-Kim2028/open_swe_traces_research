# Exported API — keyflags

Package `kv` (module `example.internal/kvstore/v2`) — per-key metadata bits
carried by the in-transaction buffer.

`type KeyFlags uint16`; `type FlagsOp uint32` (21 ops, `SetPresumeKeyNotExists`
… `SetPreviousPresumeKNE`). Predicates: `HasAssertExist`, `HasAssertNotExist`,
`HasAssertUnknown`, `HasAssertionFlags`, `HasPresumeKeyNotExists`,
`HasLocked`, `HasNeedLocked`, `HasLockedValueExists`, `HasNeedCheckExists`,
`HasPrewriteOnly`, `HasIgnoredIn2PC`, `HasReadable`,
`HasNeedConstraintCheckInPrewrite`, `HasNewlyInserted`, `AndPersistent`.
Mutator: `func ApplyFlagsOps(origin KeyFlags, ops ...FlagsOp) KeyFlags`.

Callers: `internal/unionstore` (memdb flag plumbing), `txnkv/transaction`
(2pc/pessimistic paths). In-tree tests removed: none.
