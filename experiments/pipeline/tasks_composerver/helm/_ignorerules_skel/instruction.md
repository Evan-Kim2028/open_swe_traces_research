# Bug report

`helm package` includes files it should skip (or skips files it should keep): .helmignore rules are not honored — VCS dirs, editor backups, or explicitly ignored paths leak into packaged charts.

Reproduce with:

```
go test -count=1 ./pkg/ignore/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
