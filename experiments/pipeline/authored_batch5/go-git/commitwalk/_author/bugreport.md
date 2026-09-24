# Bug report

History traversal is misbehaving across the board. Walking a commit's
parents either revisits merge ancestors endlessly or skips them entirely.
Path-filtered logs report commits that never touched the path, and
time-bounded logs ignore the bounds. Ordering by committer time is not
respected — results come out breadth-first regardless. Errors mid-walk
are swallowed instead of surfacing through the iterator.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
