# Exported API — comfortfade

Pull-request reviews accept two comment encodings: the legacy `Position` form
and the newer "comfort fade" multi-line form (`Side`/`Line`/`StartSide`/
`StartLine`). The client detects which style a review uses and selects the
matching `Accept` preview header; mixed reviews are rejected.

- `PullRequestReviewRequest.Comments` — `[]*DraftReviewComment`.
- `DraftReviewComment` fields: `Position` (legacy) vs `Side`, `Line`,
  `StartSide`, `StartLine` (comfort fade).
- `ErrMixedCommentStyles` — sentinel returned when a review mixes the two
  comment styles.
- `mediaTypeMultiLineCommentsPreview` — preview Accept header used for
  comfort-fade reviews.
- `isComfortFadePreview` (excised body) runs inside `CreateReview`.
