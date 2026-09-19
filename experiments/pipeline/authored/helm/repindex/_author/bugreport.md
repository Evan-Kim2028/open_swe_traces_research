# Bug report

`helm search repo`, dependency resolution, and `helm repo index` break: the index lookup fails or panics, charts cannot be found by version or version range, and generated indexes are empty or malformed.

Reproduce with:

```
go test -count=1 ./pkg/repo/v1/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository.
