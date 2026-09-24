# Contract — searchq

`SearchService.search` assembles the Accept header from per-search-type
media types (plus a text-match media type when requested) and flattens
search parameters into the query. Every commitment below is covered by a
hidden test; every hidden test maps to a commitment.

## Commitments

1. **TextMatch adds a media type.** `opts.TextMatch == true` adds a
   text-match media type to the `Accept` header — asserted as a
   text-match value being present, not the exact literal. Covered by
   `TestDetail01`.
2. **Per-type preview (shape).** Each previewed search type contributes a
   media type to `Accept` — asserted as a `vnd.github` media type beyond
   the bare default being present, not the specific preview names.
   Covered by `TestDetail02`.
3. **RepositoryID flattened.** `parameters.RepositoryID` (non-nil)
   becomes the `repository_id` query parameter. Covered by
   `TestDetail03`.
4. **Single joined header.** Accept values are joined into one
   comma-separated header line — not repeated headers. Covered by
   `TestDetail04`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | doc — asserted on the text-match marker, not the literal |
| TestDetail02 | 2 | no — shape only |
| TestDetail03 | 3 | partially |
| TestDetail04 | 4 | partially |
