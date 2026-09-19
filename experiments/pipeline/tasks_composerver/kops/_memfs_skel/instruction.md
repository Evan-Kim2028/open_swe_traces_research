# Bug report

The in-memory filesystem panics on join/create/read. Exclusive create no longer returns “already exists”, directory listings are empty, and recursive tree walks either miss nested files or include directories as files.

Reproduce with:

```
go test -count=1 ./util/pkg/vfs/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
