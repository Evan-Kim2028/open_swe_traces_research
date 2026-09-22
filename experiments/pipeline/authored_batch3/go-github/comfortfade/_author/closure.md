# Closure — comfortfade

Package: github (root). File: github/pulls_reviews.go.
Removed bodies: PullRequestReviewRequest.isComfortFadePreview — stubbed to
`return false, nil` (no style detection, no mixed-style rejection, no preview
header).
Kept: ErrMixedCommentStyles, mediaTypeMultiLineCommentsPreview, the
CreateReview call site, DraftReviewComment fields.
Tests removed: 3 funcs in github/pulls_reviews_test.go
(TestPullRequestReviewRequest_isComfortFadePreview,
TestPullRequestsService_CreateReview_badReview,
TestPullRequestsService_CreateReview_addHeader).
