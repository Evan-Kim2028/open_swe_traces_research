# Bug report

any multi-key txn write fails: prewrite/commit never reaches the store (panic on dispatch).

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository and test output.
