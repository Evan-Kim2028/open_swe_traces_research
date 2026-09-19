# Bug report

HTML rendering broke: pages panic or return the not-configured error even though templates were loaded, and the debug renderer never picks up its template source.
Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository and test output.
