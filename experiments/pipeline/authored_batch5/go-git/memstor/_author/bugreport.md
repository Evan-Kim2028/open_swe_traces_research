# Bug report

In-memory storage loses ref updates under concurrency —
`CheckAndSetReference` overwrites a ref even when the stored hash moved
since the caller read it, so a fetch can clobber a concurrent push.
Objects written through a transaction are still visible in the base
storage before Commit. `SetIndex` leaves `ModTime` zero so worktree
status rehashes every file. Objects of an unknown type are silently
stored, and `ForEachObjectHash` propagates `storer.ErrStop` as a
failure instead of a clean stop.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
