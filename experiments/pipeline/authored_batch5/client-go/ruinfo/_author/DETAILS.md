# Details — ruinfo

1. `MakeRequestInfo` marks `bypass` when the request source contains
   `"internal_others"`; a non-write request gets `writeBytes = -1`
   (sentinel — `IsWrite()` is `writeBytes > -1`, so a zero-byte WRITE still
   reports `IsWrite()==true` while `WriteBytes()` returns 0). Inferable:
   no — the -1 sentinel and the substring match are arbitrary.
2. Write bytes are summed only for `PrewriteRequest` (mutation keys+values
   + primary lock + secondaries) and `CommitRequest` (keys); any other
   write-typed request counts 0. Inferable: no.
3. `MakeResponseInfo` reads `Data.Size()` for coprocessor/stream responses
   and whole-response `Size()` for Scan; `ScanDetailV2.ProcessedVersionsSize`
   OVERRIDES `readBytes` whenever present. Inferable: no.
4. `getKVCPU` prefers `TimeDetailV2.ProcessWallTimeNs`, then
   `TimeDetailV2`-era `ProcessWallTimeMs` (x1e6), then legacy
   `details.TimeDetail.ProcessWallTimeMs`. Inferable: no — precedence is a
   choice.
5. GetResponse/BatchGetResponse contribute `detailsV2` only (no
   readBytes); nil `resp.Resp` and unlisted types yield an empty
   `ResponseInfo`; `Succeed()` is unconditionally true. Inferable:
   partially.
