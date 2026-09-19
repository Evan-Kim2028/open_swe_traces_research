# Bug report

form, query-string, header and URI-parameter binding into structs is broken: bound objects come back with every field at its zero value (or the call panics), and the data parsed from the request never reaches the destination.
Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository and test output.
