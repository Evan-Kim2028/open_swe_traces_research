# Bug report

Two overlapping holds on the same key both entered the critical section.
expected the second to wait until the first released, actual both
proceeded (or the race detector fired).

Reproduce with:

```
go test -count=1 -timeout 15m -race ./internal/latch/
```

Do not skip, delete, or weaken the tests. Do not change test assertions
or testdata just to make them green.

Work in `/app`. Keep unrelated tests passing.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
