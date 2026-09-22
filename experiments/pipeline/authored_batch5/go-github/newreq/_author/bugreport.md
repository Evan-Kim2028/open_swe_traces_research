# Bug report — newreq

Requests built by `NewRequest` are missing the API-version header GitHub
expects, and a client configured with an empty user-agent emits a
`User-Agent` header with an empty value instead of omitting it.

Expected: each request carries the client's default API version header,
and the `User-Agent` header is present only when the client has a
non-empty user agent.

Got: no `X-Github-Api-Version` header at all, and `User-Agent` is sent
even when it would be empty.

Reproduce with:

```
go test -count=1 ./...
```

Do not use web search or any tool that accesses the internet; work only from the repository.
