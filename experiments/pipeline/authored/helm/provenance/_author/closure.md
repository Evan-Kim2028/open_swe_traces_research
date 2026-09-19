# Closure — provenance

Package: pkg/provenance. Files: sign.go.

Removed: 10 functions (bodies stubbed to `panic("excised: <name>")`, signatures and doc comments preserved, compiles clean).

`NewFromFiles(keyfile, keyringfile)` / `NewFromKeyring(keyringfile, id)` -> *Signatory; `DecryptKey(PassphraseFetcher)`; `ClearSign(archiveData, filename, metadataBytes)` -> string; `Verify(archiveData, provData, filename)` -> *Verification; `ParseMessageBlock(data, metadata, *SumCollection)`; `DigestFile`/`Digest` -> hex sha256.
