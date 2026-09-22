# Bug report — apiversion

Requests pinned to an API version are no longer validated: a version outside
the client's supported window goes straight to the network instead of being
rejected up front.

Expected: a request carrying an explicit API-version header fails fast with
the unsupported-version error when the version falls outside the supported
window, and succeeds when inside it.

Got: every version is accepted; the unsupported-version error is never
returned.

Reproduce with:

```
go test -count=1 ./github/
```

Do not use web search or any tool that accesses the internet; work only from the repository.
