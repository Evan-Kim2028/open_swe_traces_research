# Bug report

Pack index lookups are unreliable. Objects present in the pack come back
as not-found, offsets occasionally return values from the wrong object —
especially for packs with large files — and reverse lookups (offset to
hash) never terminate or return stale results. Iterating entries by
offset order comes back hash-ordered instead, prefix scans return entries
outside the prefix, and building an index while scanning a pack either
crashes early or produces an index whose fanout disagrees with the object
count.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
