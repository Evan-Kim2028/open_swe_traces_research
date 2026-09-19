# Bug report

Install/upgrade/uninstall applies resources in the wrong order: namespaces, CRDs and workloads are not ordered correctly, hooks fire out of weight order, and release history lists come back unsorted.

Reproduce with:

```
go test -count=1 ./pkg/release/v1/util/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository.
