# Closure — sigblock

Package: `plumbing/object`. File: `signature.go`.

Removed (6 functions stubbed): `typeForSignature`, `parseSignedBytes`,
`countSignatureBlocks`, `isSignatureHeader`, `stripObjectSignatures`,
`stripHeaderSignatures`.

Kept: signature type constants + all three format tables (the armor prefixes are
visible), `signatureType`/`signatureFormat` types, all doc comments (which cite upstream
behaviour — the `doc` inferables).

Tests deleted: `signature_test.go` only.
