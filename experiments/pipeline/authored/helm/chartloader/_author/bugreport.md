# Bug report

`helm template`/`lint` on any chart fails or panics while reading the chart; loading a packaged chart or a chart directory returns an error or unusable chart. Users cannot render or install anything.

Reproduce with:

```
go test -count=1 ./internal/chart/v3/loader/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository.
