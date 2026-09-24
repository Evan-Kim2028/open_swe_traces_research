# Bug report

`Status` reports clean files as modified after any operation that rewrites
the index (checkout, add), and ignored directories still get walked —
`node_modules` shows up in scans and status takes far longer than it
should. Files touched in the same second as an index write sometimes show
as unmodified even when their content changed. With `core.autocrlf` set,
files that differ only in line endings are reported as modified.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
