# Bug report

JSON-family responses are wrong: bodies panic or come out empty/garbled — secure prefix missing, JSONP unwrapped, non-ASCII unescaped, or wrong content types.
Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository and test output.
