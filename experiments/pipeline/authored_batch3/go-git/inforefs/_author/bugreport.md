# Bug report

Reading the reference list a dumb HTTP server advertises is broken.
Advertisements containing a malformed line are decoded as if nothing were
wrong — a body that is not a ref list at all yields plausible-looking
references, and a list hit by one bad line comes back truncated or
garbage instead of failing. Refs with names the server should never have
sent are kept rather than dropped, peeled entries lose the marker that
distinguishes them from their base ref, and a truncated body is accepted
as a complete advertisement. Writing the list back out drops entries or
mangles the line format.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
