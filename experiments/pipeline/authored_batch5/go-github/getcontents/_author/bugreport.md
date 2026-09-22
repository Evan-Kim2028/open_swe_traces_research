# Bug report — getcontents

`GetContents` breaks on two fronts: directory paths return a decode
error (the response array cannot fit a `*RepositoryContent`), and paths
containing spaces, `+`, or `..` are sent unescaped — producing the
wrong route or an invalid request.

Expected: the path is URL-escaped with its trailing slash trimmed, and
the response is decoded as file or directory, populating exactly one of
the two return values.

Got: directories fail to decode; unsafe paths are sent raw.

Reproduce with:

```
go test -count=1 ./...
```

Do not use web search or any tool that accesses the internet; work only from the repository.
