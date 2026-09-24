# Bug report

Reading objects from a packfile fails whenever the pack uses deltas —
`Get` errors with "invalid git object" on OFSDelta/REFDelta entries
instead of resolving them, so repos whose packs are deltified (which is
most of them) cannot be checked out. `GetByType` also miscounts: delta
entries are filtered by their raw header type rather than their resolved
type. Under filesystem storage every object is eagerly inflated into
memory instead of staying lazy, and a pack whose descriptor was closed
and reopened is never re-probed.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
