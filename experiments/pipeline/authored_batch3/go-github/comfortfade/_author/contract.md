# Contract (L2) — comfortfade

- A review comment using the multi-line-comment fields (side/line family)
  selects the multi-line-comments preview Accept header; a comment using the
  legacy `Position` field does not. The exact membership of the field set is
  conventional; the contract pins the two representative fields.
- A single comment carrying both styles is rejected with
  `ErrMixedCommentStyles` before any network call.
- Styles must be consistent across the whole comment list — a `Position`
  comment alongside a side/line comment is rejected before any network call.
  The shape contracted: an error surfaces and no request leaves the client.
- Nil comments in the slice are skipped without panic and do not poison style
  detection.
- An all-fade review emits the preview Accept header; an all-position or
  commentless review emits the default header.
- The preview media type committed by the suite is the surviving preview
  constant's value; a mixed-style error aborts the request before any network
  call.

## Coverage

| hidden test | DETAILS row |
|---|---|
| `TestDetail01` | 1 — line-field comment selects preview Accept; position comment does not |
| `TestDetail02` | 2 — single comment with both styles → `errors.Is(err, ErrMixedCommentStyles)`, no network |
| `TestDetail03` | 3 — cross-comment style mix errors pre-network (shape only) |
| `TestDetail04` | 4 — nil entries skipped, detection unaffected (shape only) |
| `TestDetail05` | 5 — all-fade → preview Accept; all-position/empty → default Accept |
| `TestDetail06` | 6 — preview media type literal; error path aborts before network |
