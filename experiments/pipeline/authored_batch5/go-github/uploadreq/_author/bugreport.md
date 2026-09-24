# Bug report — uploadreq

`NewUploadRequest` accepts upload URLs it should refuse: paths with
`..` segments resolve without error, and absolute URLs pointing at
unconfigured hosts are honored — so a server-supplied upload URL could
send the caller's bytes anywhere. Uploads also lost their replayable
body: requests built from a seekable file no longer expose `GetBody`,
so HTTP/2 retries after a refused stream fail with "cannot retry".

Expected: traversal rejected, absolute destinations limited to
configured origins, and `GetBody` populated whenever the reader can be
rewound without buffering.

Got: no validation, no `GetBody`.

Reproduce with:

```
go test -count=1 ./...
```

Do not use web search or any tool that accesses the internet; work only from the repository.
