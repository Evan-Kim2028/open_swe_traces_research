# Bug report — createfork

`RepositoriesService.CreateFork` loses the deferred fork's metadata:
when GitHub answers `202 Accepted`, the returned `*Repository` is empty
even though the response body contains the pending fork's JSON.

Expected: the accepted response's payload is decoded into the returned
repository, so callers can inspect the fork being created while
handling `*AcceptedError`.

Got: repository fields are all zero when `*AcceptedError` is returned.

Reproduce with:

```
go test -count=1 ./...
```

Do not use web search or any tool that accesses the internet; work only from the repository.
