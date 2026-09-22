# Bug report

Packs produced by the encoder are corrupt or unverifiable. The trailer
checksum does not match the bytes that precede it on sha-256 repositories,
and sometimes not at all. Delta entries come out with the wrong reference
form — bases listed by id when the pack was asked to be offset-relative —
or a delta lands in the stream before the base it needs, so readers fail
to resolve it. When a selected base representation disappears mid-write,
the writer recurses forever instead of recovering, and objects report
their identity inconsistently once deltas are involved.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
