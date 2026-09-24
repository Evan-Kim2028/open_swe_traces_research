# Bug report — boolresp

Predicate-style API helpers have started reporting "not found" as an error
instead of a clean negative. Callers that ask "does this thing exist" now get
a hard error for a missing resource.

Expected: a missing resource comes back as a plain negative answer with no
error; genuine failures still surface their error.

Got: the missing-resource case returns the API error to the caller.

Reproduce with:

```
go test -count=1 ./github/
```

Do not use web search or any tool that accesses the internet; work only from the repository.
