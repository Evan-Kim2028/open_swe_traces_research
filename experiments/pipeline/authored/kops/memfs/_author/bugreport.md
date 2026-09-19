# Bug report

The in-memory filesystem panics on join/create/read. Exclusive create no longer returns “already exists”, directory listings are empty, and recursive tree walks either miss nested files or include directories as files.

Reproduce with:

```
go test -count=1 ./util/pkg/vfs/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository.
