# Contract (L2) — fidownload

`DownloadURL` skips the fetch entirely when the destination already matches the expected hash. `downloadURLToWriter` streams bytes while hashing (sha256 or the expected algorithm), dispatches `gs`/`s3`/`azureblob` through VFS `WriteToWithContext` and everything else through `OpenURL`, and fails after writing when the hash mismatches. `downloadURLToFile` stages a temp file, chmods 0644, renames. `OpenURL` uses the hardened client (timeouts, proxy-from-env), rejects non-2xx, and returns a reader whose Close cancels the request context.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestDownloadURLRejectsNon2xxAndPreservesDestination` | non-2xx is an error; destination untouched |
| `TestDownloadURLToWriterVerifiesHash` | matching hash returns the actual hash |
| `TestDownloadURLToWriterRejectsHashMismatch` | mismatch errors after streaming |
