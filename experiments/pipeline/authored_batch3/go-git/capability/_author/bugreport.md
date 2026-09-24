# Bug report

The capability list used by every protocol message is broken. Encoded
capability strings come out sorted alphabetically instead of in the order
they were declared, a capability holding several values collapses to one
token, and an explicit empty value is decoded as if none were present.
Updating a capability loses its declared position, and removing then
re-adding it resurrects it in the wrong slot. Validation refuses
perfectly valid declarations — flag capabilities carrying arguments are
accepted while required-argument capabilities without one pass — and the
session identifier check misses embedded whitespace. The reported agent
string ignores the documented environment override.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
