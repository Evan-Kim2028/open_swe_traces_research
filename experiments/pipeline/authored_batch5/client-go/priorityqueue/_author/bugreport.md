# Bug report

`internal/client`'s `PriorityQueue` panics on every operation — the heap
wrapper, `Take`, `highestPriority`, `clean`, and `reset` are all stubbed —
so batch command scheduling cannot order work by priority.

Expected (in-package probes with an `Item` whose `priority()` is a uint64):
after pushing priorities 1..5, `Len()` is 5 and `highestPriority()` is 5;
`Take(1)` returns the priority-5 item and leaves `highestPriority()` 4;
`Take(2)` returns `[4, 3]`; a final `Take(5)` returns the remaining two
items and empties the queue. `Take(0)` and `Take(-1)` return nil. Pushing a
canceled item then calling `clean()` drops it (`Len()` 5 -> ... -> 0 in the
sequence above).

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
