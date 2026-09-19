# Bug report

binding request data into structs fails across sources: forms, query strings, headers and path parameters all come back empty or panic instead of filling the destination object.
Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository and test output.
