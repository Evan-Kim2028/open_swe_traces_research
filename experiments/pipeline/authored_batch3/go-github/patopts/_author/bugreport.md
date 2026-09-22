# Bug report — patopts

Filtering the personal-access-token listing by owners or token IDs silently
does nothing: the request goes out without the array filters, so the API
returns unfiltered results.

Expected: each requested owner and token ID reaches the server as a separate
array query parameter, alongside the other list options.

Got: the owner/token filters never appear in the request URL.

Reproduce with:

```
go test -count=1 ./github/
```

Do not use web search or any tool that accesses the internet; work only from the repository.
