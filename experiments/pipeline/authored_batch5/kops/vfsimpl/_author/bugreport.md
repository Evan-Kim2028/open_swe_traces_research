# Bug report

The vfs path implementations are broken: `Path`/`String` render malformed URLs (missing
scheme, bucket, or container segments), `Join` drops the context/bucket fields or doesn't
normalize, `Base` returns the whole key, `GetHTTPsUrl` produces the wrong host format,
`isGCSNotFound` misses sentinel errors, `GSAcl.String` renders the wrong shape, and
`AWSErrorCode`/`TerraformLink` return junk.

Expected: `<scheme>://<bucket>/<key>`-family formatting; `path.Join` semantics preserving
bucket/context; regional S3 endpoint resolution and `storage.googleapis.com` GCS URLs;
sentinel+404 not-found detection.

Reproduce with:

```
go test -count=1 ./util/pkg/vfs/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the
internet; work only from the repository.
