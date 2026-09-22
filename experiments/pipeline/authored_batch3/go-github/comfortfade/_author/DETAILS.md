# DETAILS — comfortfade

1. A comment is comfort-fade style iff any of `StartSide`, `Side`,
   `StartLine`, `Line` is set; `Position` set means legacy style. Inferable:
   partially — the field split is documented on the struct, the exact
   four-field set is arbitrary.
2. A single comment carrying both styles → `ErrMixedCommentStyles`. Inferable:
   yes — the sentinel's name and the struct comments state the two styles
   can't mix.
3. Styles must be consistent across the whole comment list — a `Position`
   comment followed by a `Line`/`Side` comment, or vice versa, is also
   `ErrMixedCommentStyles`. Inferable: no — cross-comment mixing is an
   arbitrary extension of the rule.
4. Nil comments in the slice are skipped. Inferable: no — silent tolerance of
   nil entries is an arbitrary edge.
5. An all-comfort-fade review returns true; an all-position or empty review
   returns false. Inferable: yes — forced by the header-selection use site.
6. A true result makes `CreateReview` set `Accept` to
   `mediaTypeMultiLineCommentsPreview`; an error aborts the request before any
   network call. Inferable: yes — the call site and const remain.
