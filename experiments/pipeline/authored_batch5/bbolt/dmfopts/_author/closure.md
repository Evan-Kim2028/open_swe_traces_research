# Closure — dmfopts

Package: `dmflakey` (`tests/dmflakey`). Files: `tests/dmflakey/dmflakey.go`.

Removed (6 functions stubbed): `WithIntervalFeatOpt`, `WithSyncFSFeatOpt`,
`flakey.DevicePath`, `flakey.Filesystem`, `createEmptyFSImage`,
`validateFSType`.

Kept: `featCfg`, `defaultFeatCfg`, `FeatOpt`, `FSType` and its constants,
the `Flakey` interface, `flakey` struct and fields, `InitFlakey`,
`AllowWrites`, `DropWrites`, `ErrorWrites`, `Teardown` (kept — their only
container-observable behavior is the device-stack error path, which a stub
also produces; nothing measurable to certify), all of `dmsetup.go` and
`loopback.go` (same reason), every import.

Tests deleted: none — `dmflakey_test.go` and the robustness suite are
root-gated through `TestMain` and pass vacuously without `-test.root`.
