# Bug report

pessimistic lock acquisition always errors/panics: LockKeys cannot proceed, so pessimistic txns abort.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository and test output.
