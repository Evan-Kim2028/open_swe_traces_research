# Contract — gitrefs

`GitService`'s ref operations validate inputs, normalize the `refs/`
prefix, and path-escape refs into routes. Every commitment below is
covered by a hidden test; every hidden test maps to a commitment.

## Commitments

1. **CreateRef validates.** An empty `Ref` and an empty `SHA` are rejected
   before any request is built — asserted as an error with no network
   call, not a specific error literal. Covered by `TestDetail01`.
2. **CreateRef normalizes.** `body.Ref` is normalized to begin with
   `refs/` — an already-prefixed ref passes through unchanged. Covered by
   `TestDetail02`.
3. **UpdateRef validates.** An empty `ref` and an empty `body.SHA` are
   rejected before the request is built. Covered by `TestDetail03`.
4. **Prefix stripped on write/delete.** `UpdateRef` and `DeleteRef` strip
   a leading `refs/` from `ref` before escaping it into the route —
   `refs/heads/x` and `heads/x` resolve identically. Covered by
   `TestDetail04`.
5. **Path escaping.** The ref is path-escaped so a ref containing a space
   (or similar) cannot break the URL — asserted on the escaped route.
   Covered by `TestDetail05`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | partially — shape only |
| TestDetail02 | 2 | partially |
| TestDetail03 | 3 | partially — shape only |
| TestDetail04 | 4 | partially |
| TestDetail05 | 5 | yes |
