# Bug report

Loose object files round-trip badly. Reading an object fails when the
type and size header is well formed but long, accepts negative declared
sizes, and conflates a truncated header with an oversized one. The
computed object id excludes the header so it never matches the object's
real identity, and asking for it before the header is read crashes.
Writing accepts a declared size but then silently drops bytes past the
limit without reporting it, and closing the writer more than once
re-triggers the underlying error instead of returning the first result.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
