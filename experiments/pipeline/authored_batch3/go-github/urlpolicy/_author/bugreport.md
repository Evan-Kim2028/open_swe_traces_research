# Bug report — urlpolicy

Request-safety checks have gone slack: URLs with parent-path segments are
accepted, credentials are offered to the wrong hosts, and bodies can be
posted to destinations outside the configured origins without complaint.

Expected: traversal-shaped URLs are refused, credential forwarding is
scoped to the configured origins (matched tolerantly), and out-of-scope
uploads are rejected with a clear error.

Got: all of those requests sail through unchallenged.

Reproduce with:

```
go test -count=1 ./github/
```

Do not use web search or any tool that accesses the internet; work only from the repository.
