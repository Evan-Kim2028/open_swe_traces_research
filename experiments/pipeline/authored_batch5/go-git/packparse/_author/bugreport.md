# Bug report

Pack ingestion mis-handles deltas and leaks memory. Delta entries are
resolved in scan order rather than walking out from real bases, so a
delta that arrives before its base — or chains through a differently-
encoded sibling — fails or resolves against the wrong content. Delta
chains of unbounded depth are accepted, thin-pack bases that live outside
the pack are treated as hard errors, and the same parser object can be
run twice producing nonsense. On constrained storage the inflated object
contents are never released, and an empty input reports a low-level read
error instead of a recognizable empty pack.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
