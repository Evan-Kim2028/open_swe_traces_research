# Closure — keyflags

Package: `kv`. File: `kv/keyflags.go` (260 lines).

Removed (all bodies stubbed): the 15 `KeyFlags` predicates
(`HasAssertExist`, `HasAssertNotExist`, `HasAssertUnknown`,
`HasAssertionFlags`, `HasPresumeKeyNotExists`, `HasLocked`, `HasNeedLocked`,
`HasLockedValueExists`, `HasNeedCheckExists`, `HasPrewriteOnly`,
`HasIgnoredIn2PC`, `HasReadable`, `HasNeedConstraintCheckInPrewrite`,
`AndPersistent`, `HasNewlyInserted`) plus `ApplyFlagsOps`.

Kept: the `flag*`/`persistentFlags` constants, the `FlagsOp` enum and its doc
comments (the per-op semantics table stays readable — it documents intent,
not the coupled bit writes), `KeyFlags`, `FlagBytes`.

Tests deleted: none — `kv/key_test.go` covers `PrefixNextKey` only and still
passes.
