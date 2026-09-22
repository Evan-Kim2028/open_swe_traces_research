# Bug report

View projection and HTTP body computation panic when they copy
attribute graphs. Expected: copies preserve shared-subtree identity,
record original links, and detach every mutable value so edits to the
copy never affect the design. Got: panics.

Reproduce with:

```
go test -count=1 ./expr/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
