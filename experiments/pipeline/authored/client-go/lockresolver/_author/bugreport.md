# Bug report

any encounter with another txn's lock fails (panic or 'cannot resolve') — reads and writes never get past a leftover lock even after the owner is gone.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository and test output.
