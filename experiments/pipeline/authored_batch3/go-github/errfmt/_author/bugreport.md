# Bug report — errfmt

API errors have gone unreadable: error strings lost the failing request's
method, URL and status, the rate-limit countdown vanished, and some payloads
that used to decode now fail. Worse, tokens in request URLs are showing up in
error text.

Expected: errors carry their request context and countdown information, URLs
embedded in error text have credentials scrubbed, and loosely-structured API
error payloads still decode.

Got: messages are bare or wrong, the countdown is missing, credentials leak
into error text, and string-shaped error payloads no longer parse.

Reproduce with:

```
go test -count=1 ./github/
```

Do not use web search or any tool that accesses the internet; work only from the repository.
