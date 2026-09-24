# Closure — archive

Package: `internal/archive`. File: `archive.go`.

Removed (11 functions stubbed): `SupportedFormats`, `ApplyUmask`,
`ApplyUmaskDir`, `ResolveTreeish`, `ResolveRef`, `WriteTarArchive`,
`WriteZipArchive`, `MatchesPathFilter`, `GetTarCommitID`,
`WriteArchive`, `HasInvalidPrefix`.

Kept: error vars, `PAXGlobalHeader`/`DefaultUmask`/
`maxTarSymlinkTargetSize` constants, doc comments. Disjoint from
`treeobj` (the TreeWalker is consumed intact) and `sigcodec` (blob
reading is consumed intact).

Tests deleted: `internal/archive/archive_test.go` (1).
