# Bug report

form, query-string, header and URI-parameter binding into structs is broken: bound objects come back with every field at its zero value (or the call panics), and the data parsed from the request never reaches the destination.
Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository and test output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
