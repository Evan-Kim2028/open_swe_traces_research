# Closure — pullmerge

Package: github (root). File: github/pulls.go.
Removed bodies: PullRequestsService.Merge — stubbed to a variant that
omits `commit_message` whenever the caller passes an empty string,
regardless of `PullRequestOptions.DontDefaultIfBlank`. Drops the
`DontDefaultIfBlank && commitMessage == "" → CommitMessage = &""` block.
CommitTitle/MergeMethod/SHA plumbing and the PUT itself keep working.
Kept: MergeAsync, pullRequestMergeRequest fields, all option fields.
Tests removed: 1 func in github/pulls_test.go
(TestPullRequestsService_Merge_Blank_Message).
