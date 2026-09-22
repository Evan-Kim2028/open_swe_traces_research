# Bug report — ratehdrs

Token-expiration metadata never reaches callers: responses that carry a
valid expiration header report an unset expiration, and a degenerate
rate-limit reset value is treated as a real epoch.

Expected: expiration headers parse into the response's expiration
timestamp, in whichever zone format the server sends; absent or junk
headers still yield the unset value.

Got: expiration is always reported unset, and a zero reset is surfaced as a
real time.

Reproduce with:

```
go test -count=1 ./github/
```

Do not use web search or any tool that accesses the internet; work only from the repository.
