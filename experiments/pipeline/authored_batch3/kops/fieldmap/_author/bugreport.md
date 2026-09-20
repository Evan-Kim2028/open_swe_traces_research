# Bug report

Cluster field paths come back in the wrong API version: paths shown to users print the new-style spelling instead of the old one, paths stored internally keep the old spelling instead of the new one, and fields with no mapping lose their spelling entirely or get translated when they should pass through.

Expected: the new-style `spec.api.publicName` displays to users as `spec.masterPublicName`; the old-style `spec.masterPublicName` resolves internally to `spec.api.publicName`; unmapped paths pass through unchanged in both directions.

Reproduce with:

```
go test -count=1 ./pkg/apis/kops/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository.
