# Bug report

GCE naming/error helpers are broken: `IsNotFound`/`IsNotReady` misclassify googleapi errors,
`ClusterPrefixedName`/`ClusterSuffixedName` produce names exceeding `maxLength` or keep dots,
`SafeClusterName`/`SafeObjectName`/`SafeTruncatedClusterName` don't sanitize,
`ServiceAccountName` doesn't respect the 30-char cap, `LastComponent` doesn't strip the path,
`SSHUsernameForImage` returns `admin` for Ubuntu images, and `ZoneToRegion` returns the whole
zone instead of the region.

Expected: 404 / `resourceNotReady` detection; dot→dash sanitization with length-capped
prefix/suffix composition; last-path-component extraction; `ubuntu` for `ubuntu-*` images;
`a-b-c` zone → `a-b` region.

Reproduce with:

```
go test -count=1 ./upup/pkg/fi/cloudup/gce/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the
internet; work only from the repository.
