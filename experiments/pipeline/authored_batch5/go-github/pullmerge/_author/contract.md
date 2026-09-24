# Contract — pullmerge

`PullRequestsService.Merge` controls whether an explicit empty commit
message is sent to the merge endpoint. Every commitment below is covered
by a hidden test; every hidden test maps to a commitment.

## Commitments

1. **Empty message omitted.** When `commitMessage` is empty and `options`
   is nil or has `DontDefaultIfBlank == false`, the request body omits
   `commit_message`. Covered by `TestDetail01`.
2. **Explicit empty sent.** When `commitMessage` is empty and
   `options.DontDefaultIfBlank` is true, the request body carries
   `commit_message` as an explicit empty string — asserted as
   present-and-empty in the sent JSON. Covered by `TestDetail02`.
3. **Non-empty always sent.** A non-empty `commitMessage` is sent
   regardless of the flag. Covered by `TestDetail03`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | partially |
| TestDetail02 | 2 | no — asserted as field present-and-empty |
| TestDetail03 | 3 | yes |
