# Bug report

package-level route registration is dead: calling the top-level GET/POST/etc. panics or silently registers nothing, and Routes() reports an empty table.
Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository and test output.
