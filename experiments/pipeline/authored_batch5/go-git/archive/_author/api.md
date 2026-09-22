# Exported API — archive

Package `internal/archive` — git-upload-archive support: `SupportedFormats`,
`ApplyUmask`, `ApplyUmaskDir`, `ResolveTreeish`, `ResolveRef`,
`WriteTarArchive`, `WriteZipArchive`, `MatchesPathFilter`,
`GetTarCommitID`, `WriteArchive`, `HasInvalidPrefix`.

Kept visible: `PAXGlobalHeader`/`DefaultUmask`/`maxTarSymlinkTargetSize`
constants, all exported error vars, doc comments (including the racy-git
and security notes).

Callers: `upload-archive` session handlers. In-tree tests removed: 1
(`archive_test.go`).
