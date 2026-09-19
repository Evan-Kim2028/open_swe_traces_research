# Bug report

`helm package` includes files it should skip (or skips files it should keep): .helmignore rules are not honored — VCS dirs, editor backups, or explicitly ignored paths leak into packaged charts.

Reproduce with:

```
go test -count=1 ./pkg/ignore/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository.
