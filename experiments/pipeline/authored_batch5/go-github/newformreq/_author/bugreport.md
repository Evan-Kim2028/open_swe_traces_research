# Bug report — newformreq

`NewFormRequest` posts multipart bodies to arbitrary hosts: a `urlStr`
resolving outside the configured origins is honored, so the form
payload (which can carry credentials via headers) can be sent to an
attacker-chosen destination. Form requests are also missing the API
version header, and send an empty `User-Agent` header when the client
has none.

Expected: foreign form destinations rejected, version header present,
and `User-Agent` omitted when empty.

Got: no destination check, no version header, empty `User-Agent` sent.

Reproduce with:

```
go test -count=1 ./...
```

Do not use web search or any tool that accesses the internet; work only from the repository.
