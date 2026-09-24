# Bug report — respclassify

API failures all surface as one undifferentiated error: rate-limit
responses no longer carry their reset info, accepted-but-async responses
are treated as failures, two-factor challenges are indistinguishable from
plain 401s, and redirect statuses lose their target location.

Expected: each failure mode surfaces as its own error kind so callers can
inspect retry delays, redirect targets, and rate-limit resets.

Got: every non-2xx response produces the same generic error.

Reproduce with:

```
go test -count=1 ./github/
```

Do not use web search or any tool that accesses the internet; work only from the repository.
