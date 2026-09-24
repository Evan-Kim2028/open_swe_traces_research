# Details — vfspaths

1. `BuildVfsPath`: no `://` → FSPath on the raw string; `file://` → FSPath on the stripped path; the ten cloud schemes each route to a builder; anything else is an error. Inferable: yes — scheme list is the visible API surface.
2. Every cloud builder parses the URL, rejects a mismatched scheme, and takes the bucket from `u.Host` minus a trailing slash — empty bucket is an error. Inferable: partially — repeated pattern, but each has its own error wording.
3. `buildS3Path` is the ONLY S3-family builder that works without `S3_ENDPOINT` (uses a default endpoint resolver); `do`, `linode`, `hos`, `scw` all REQUIRE `S3_ENDPOINT` and error without it. Inferable: no — which schemes require the env var is arbitrary.
4. All `S3_ENDPOINT` consumers set BaseEndpoint + UsePathStyle + a checksum-validation flag; linode ADDITIONALLY pins request/response checksum modes to when-required. Inferable: no.
5. `buildAzureBlobPath` ERRORS when `AZURE_STORAGE_ACCOUNT` is set (inverted — the account must come from the URL); host=account, first path segment=container, remainder=key; missing either is an error. Inferable: no — the inverted env check is surprising.
6. `buildMemFSPath` errors when the memfs context is uninitialized (production default) — `NewTestingVFSContext`/`ResetMemfsContext` initialize it. Inferable: partially.
7. `RetryWithBackoff` runs the condition BEFORE the first sleep, sleeps `duration` (jittered) between attempts, counts attempts against `Steps`, and returns the condition's (done, err) on exhaustion — not a timeout error. Inferable: no — returning the last error vs a timeout sentinel is a choice.
8. `nextBackoffDuration` multiplies by `Factor` then clamps to `Cap` only when `Cap > 0` — an uncapped backoff grows without bound. Inferable: partially.
