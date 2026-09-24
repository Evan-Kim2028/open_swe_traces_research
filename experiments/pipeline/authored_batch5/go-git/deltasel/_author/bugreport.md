# Bug report

Pack compression is broken. Objects that should delta against each other
are written whole, so packs are much larger than they should be. When
deltas do get reused from an existing pack, some whose base object isn't
being sent are emitted as-is and the result is undecodable on the other
end. A stored delta chain that references itself hangs the encoder
indefinitely. And the compression window doesn't seem to bound memory —
peak usage grows with object count rather than staying flat.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
