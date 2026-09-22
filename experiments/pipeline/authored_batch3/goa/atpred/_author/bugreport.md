# Bug report

HTTP body computation, security analysis, and codegen panic when they
inspect attribute requiredness, defaults, tags, or member lookup.
Expected: required/default/tag predicates follow user types and
references, pointer decisions respect defaults, and deletion cleans
required lists and examples. Got: panics.

Reproduce with:

```
go test -count=1 ./expr/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
