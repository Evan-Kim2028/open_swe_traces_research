# Bug report

Incremental fetches transfer far too many objects — the pack sent after
a `have` negotiation contains commits, trees and blobs the client already
has, as if the haves were ignored entirely. Fetching into a shallow
repository also fails or walks past the shallow boundary. In some setups
the result contains duplicate or out-of-order objects.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
