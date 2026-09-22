# Bug report

Reading packfiles through the handle leaks and races. File descriptors
stay open forever after the last reader goes away, or get closed while a
cursor is still mid-read — reads fail spuriously under concurrency.
Closing the handle twice panics. The reported pack metadata accepts files
whose trailing hash doesn't match the pack name, and a header with a
wrong magic or version is taken at face value. Seeking past the start of
the file silently wraps instead of erroring.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
