# Exported API — pullmerge

`PullRequestsService.Merge(ctx, owner, repo, number, commitMessage,
options)` performs the merge via PUT. The request body is a
`pullRequestMergeRequest` with `commit_message`, `commit_title`,
`merge_method`, `sha`. `PullRequestOptions.DontDefaultIfBlank` controls
whether an explicit empty `commit_message` is sent so GitHub does not
substitute its own default message.
