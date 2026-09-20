# Exported API — sigblock

Package `plumbing/object` (module `example.internal/gitkit/v6`) — armored signature-block
detection and stripping used by commit/tag signature handling.

All excised functions are unexported but load-bearing for exported callers:
`Commit.EncodeWithoutSignature`, `Tag.EncodeWithoutSignature`, `Verify` paths. Kept
callers in `commit.go`/`tag.go` depend on `parseSignedBytes`, `stripObjectSignatures`,
`isSignatureHeader`, `countSignatureBlocks`, `typeForSignature` semantics. Format tables
(`-----BEGIN PGP SIGNATURE-----`, `-----BEGIN PGP MESSAGE-----`, `-----BEGIN SIGNED
MESSAGE-----`, `-----BEGIN SSH SIGNATURE-----`) stay visible.

In-tree tests removed: 1.
