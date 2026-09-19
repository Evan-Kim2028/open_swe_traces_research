# Bug report

struct validation after binding is broken: invalid objects sail through (or the bind panics) so required-field and format constraints are never enforced.
Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository and test output.
