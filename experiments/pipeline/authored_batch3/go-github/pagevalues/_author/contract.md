# Contract (L2) — pagevalues

- A pagination link carrying a `cursor` parameter populates the response's
  `Cursor` field, and a cursor-only link is not consulted for the integer
  page fields. Whether `Cursor` is additionally populated from non-`next`
  rels is an implementation choice the suite does not pin.
- A link carrying a `since` parameter but no `page` paginates the link's
  page field as if `page` had been given; when both are present, `page`
  wins.
- A link carrying none of the recognized pagination parameters leaves every
  page/cursor field at its zero value; links with unrelated parameters are
  likewise skipped.
- A `page` value that fails integer parsing lands in `NextPageToken` while
  `NextPage` stays zero.
- Malformed link segments are skipped without error and without disturbing
  the parsing of well-formed links in the same header.
- `before` and `after` parameters populate the response's `Before`/`After`
  fields from the committed rel pairings regardless of numeric parsing.

## Coverage

| hidden test | DETAILS row |
|---|---|
| `TestDetail01` | 1 — `cursor=` on next rel populates `Cursor`; page fields untouched (rel-exclusivity not pinned) |
| `TestDetail02` | 2 — `since=` falls back into the rel's page field; explicit `page=` wins |
| `TestDetail03` | 3 — param-free / unrelated-param links leave all fields zero (shape) |
| `TestDetail04` | 4 — non-integer `page=` → `NextPageToken`, `NextPage` stays 0 |
| `TestDetail05` | 5 — malformed segments skipped; well-formed links still parse (shape) |
| `TestDetail06` | 6 — `before=`/`after=` populate `Before`/`After` via their rels |
