# Bug report

requests are bound with the wrong binder (or none): JSON posts get treated as forms and vice versa, so handlers see empty or wrongly-parsed objects depending on content type.
Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository and test output.
