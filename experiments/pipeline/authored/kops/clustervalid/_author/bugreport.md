# Bug report

Cluster validation is broken: a check that used to report missing cloud groups, undersized groups, unready nodes, missing control-plane static pods, and failed critical pods now panics (or returns nothing). Detached and bastion machines are treated as ordinary nodes, and the “allow N unready workers” allowance is ignored.

Reproduce with:

```
go test -count=1 ./pkg/validation/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository.
