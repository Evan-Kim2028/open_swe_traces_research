# Exported API — provenance

`NewFromFiles(keyfile, keyringfile)` / `NewFromKeyring(keyringfile, id)` -> *Signatory; `DecryptKey(PassphraseFetcher)`; `ClearSign(archiveData, filename, metadataBytes)` -> string; `Verify(archiveData, provData, filename)` -> *Verification; `ParseMessageBlock(data, metadata, *SumCollection)`; `DigestFile`/`Digest` -> hex sha256.
Callers: pkg/downloader verify, repo provenance checks.
