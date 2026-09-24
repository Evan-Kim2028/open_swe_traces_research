# Bug report

VFS path construction is broken: `s3://` and other cloud URLs come back as plain filesystem paths or fail with the wrong error, several S3-compatible schemes do not honor the endpoint environment variable, `azureblob://` URLs split the account/container/key incorrectly, `memfs://` paths only work by accident, and the retry helper sleeps before the first attempt and never stops.

Expected: a URL without a scheme yields a filesystem path; `s3://bucket/key` yields an S3 path; the spaces/linode/hetzner/scaleway schemes require the endpoint variable and fail cleanly without it; `azureblob://account/container/key` splits on the first slash; a retry helper runs the condition immediately and stops after the configured number of attempts.

Reproduce with:

```
go test -count=1 ./util/pkg/vfs/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository.
