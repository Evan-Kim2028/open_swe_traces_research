# Contract — downloadcontents

`RepositoriesService.DownloadContentsWithMeta` resolves file content to a
readable stream — via `download_url` fetch, inline `Content`, or a typed
sentinel — while always returning the file metadata. Every commitment
below is covered by a hidden test; every hidden test maps to a commitment.

## Commitments

1. **download_url fetched.** When the API response carries a
   `download_url`, its bytes are fetched with a plain GET and the body is
   returned to the caller, with the file metadata populated. Covered by
   `TestDetail01`.
2. **No download source.** When `download_url` is absent and the file has
   no inline `Content`, the call returns `ErrContentsNoDownloadURL` with
   the metadata still populated. Covered by `TestDetail02`.
3. **Early returns.** A directory path returns `ErrContentsDirectory` and
   a submodule path returns `ErrContentsSubmodule` — without any download.
   Covered by `TestDetail03`.
4. **Inline content.** A file whose `Content` is populated inline returns
   that content wrapped as a reader without a second request. Covered by
   `TestDetail04`.
5. **Transport error shape.** On download transport error, a `*Response`
   is returned and the error is the transport failure — not the
   no-download-url sentinel. Covered by `TestDetail05`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | doc |
| TestDetail02 | 2 | partially |
| TestDetail03 | 3 | yes |
| TestDetail04 | 4 | partially |
| TestDetail05 | 5 | no — shape only (a response is returned, the transport error surfaces) |
