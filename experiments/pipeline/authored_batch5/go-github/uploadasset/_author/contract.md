# Contract — uploadasset

`RepositoriesService.UploadReleaseAsset` and
`UploadReleaseAssetFromRelease` validate inputs, normalize the release's
upload URL, and produce a media type for the upload. Every commitment
below is covered by a hidden test; every hidden test maps to a
commitment.

## Commitments

1. **Directory rejected.** A directory passed as `file` is rejected
   before any request — asserted as a pre-request rejection (not a
   transport error) with no network call, not a specific literal. Covered
   by `TestDetail01`.
2. **Invalid inputs rejected.** A nil `reader`, a negative `size`, or a
   release with empty `UploadURL` is rejected before any request.
   Covered by `TestDetail02`.
3. **URI-template stripped.** `release.UploadURL`'s `{?name,label}`
   suffix is stripped before the upload URL is used — asserted via the
   request path and the name/label query parameters arriving cleanly.
   Covered by `TestDetail03`.
4. **Relative URL normalized.** A relative upload URL is normalized
   (leading `/` trimmed) so it composes with the client's URL path
   prefix — asserted via the request landing on the prefixed route.
   Covered by `TestDetail04`.
5. **Media type produced (shape).** The upload request always carries a
   `Content-Type` — asserted as non-empty, not the precedence-resolved
   value. Covered by `TestDetail05`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | partially — shape only |
| TestDetail02 | 2 | partially |
| TestDetail03 | 3 | doc |
| TestDetail04 | 4 | partially |
| TestDetail05 | 5 | no — shape only |
