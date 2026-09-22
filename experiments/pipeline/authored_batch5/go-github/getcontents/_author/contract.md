# Contract — getcontents

`RepositoriesService.GetContents` fetches file or directory metadata,
escaping the path segment and decoding whichever of the two response
shapes the API returned. Every commitment below is covered by a hidden
test; every hidden test maps to a commitment.

## Commitments

1. **Path segment escaped.** The path is escaped as a URL path segment —
   a literal `?` stays inside the path rather than becoming a query — and
   a trailing `/` is trimmed, so `path` never breaks the route. Covered
   by `TestDetail01`.
2. **Two-shape decode.** The response body is decoded as object →
   `fileContent`, else array → `directoryContent`; exactly one is non-nil
   on success. Covered by `TestDetail02`.
3. **Undecodable body errors (shape).** When neither shape decodes, an
   error surfaces — the combined-error wording is not pinned. Covered by
   `TestDetail03`.
4. **Options applied.** `opts.Ref` applies to the request URL as the
   `ref` query parameter. Covered by `TestDetail04`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | partially |
| TestDetail02 | 2 | yes |
| TestDetail03 | 3 | no — shape only |
| TestDetail04 | 4 | yes |
