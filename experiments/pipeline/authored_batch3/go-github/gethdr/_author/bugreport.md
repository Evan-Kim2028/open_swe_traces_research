# Bug report — gethdr

Reading headers off a webhook delivery only works when the caller guesses
the exact letter-casing the server used; the same header asked for with a
different casing comes back empty.

Expected: header reads succeed regardless of how the key is cased, and
genuinely missing headers still come back empty.

Got: lookups miss unless the caller's casing matches the stored casing
byte-for-byte.

Reproduce with:

```
go test -count=1 ./github/
```

Do not use web search or any tool that accesses the internet; work only from the repository.
