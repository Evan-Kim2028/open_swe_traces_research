# Bug report — clientclone

`Client.Clone` no longer clones: the returned client drops the
receiver's configuration (base/upload URLs, user agent, API version
bounds, auth token, rate-limit flags, marketplace stub), builds its own
bare HTTP client without `CheckRedirect`/`Jar`/`Timeout`, and starts
with empty rate-limit state instead of sharing the parent's. A
token-scoped clone also loses the re-scoping fix, and cloning an
uninitialized client panics instead of returning `errUninitialized`.

Expected: the clone mirrors the receiver's configuration, shares its
rate budget, and re-scopes credentials to its own origins.

Got: a mostly-default client.

Reproduce with:

```
go test -count=1 ./...
```

Do not use web search or any tool that accesses the internet; work only from the repository.
