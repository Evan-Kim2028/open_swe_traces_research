# Bug report

The mockstore deadlock detector panics: `Detect`, the wait-for graph, and
its cleanup/expiry are stubbed, so mock transactions never report deadlocks.

Expected (in-package probes): `NewDetector()` then `Detect(1, 2, 100)`
returns nil; a following `Detect(2, 1, 200)` returns an error rendering as
`deadlock(100)`. After `CleanUp(1)`, `Detect(2, 1, 99)` returns nil. With
edges 3->9 and 7->9 registered, `Expire(5)` drops the 3-entry; `Detect(9, 7,
50)` then returns `deadlock(1)`. `(&ErrDeadlock{KeyHash: 42}).Error()` is
`"deadlock(42)"`.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
