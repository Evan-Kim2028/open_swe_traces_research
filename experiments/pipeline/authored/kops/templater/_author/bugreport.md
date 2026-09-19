# Bug report

Manifest templating panics. Interpolation, indent, snippet include, and the channel “recommended version / image” helpers are missing. Turning on “fail on missing keys” has no effect.

Reproduce with:

```
go test -count=1 ./pkg/util/templater/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository.
