# Bug report — pullmerge

`PullRequests.Merge` ignores `PullRequestOptions.DontDefaultIfBlank`:
merging with an empty `commitMessage` never sends `commit_message`, so
GitHub substitutes its default merge message even when the caller asked
for a blank one.

Expected: with `DontDefaultIfBlank` set, an empty message is sent as an
explicit empty `commit_message`.

Got: `commit_message` is omitted whenever the message is empty, so the
server-side default always wins.

Reproduce with:

```
go test -count=1 ./...
```

Do not use web search or any tool that accesses the internet; work only from the repository.
