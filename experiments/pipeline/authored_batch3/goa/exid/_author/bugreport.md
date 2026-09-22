# Bug report

Example generation panics when it derives repeatable keys for methods,
bodies, errors, or type members. Expected: each generated example draws
from a stable distinct seed so examples don't change when unrelated
parts of the design change. Got: panics.

Reproduce with:

```
go test -count=1 ./expr/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
