# Bug report

Code generation panics when producing identifiers, comments, or tag
names. Expected: identifiers are valid Go names honoring metadata and
reserved words, comments wrap and indent correctly, and snake/kebab
names handle acronym runs. Got: panics.

Reproduce with:

```
go test -count=1 ./codegen/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
