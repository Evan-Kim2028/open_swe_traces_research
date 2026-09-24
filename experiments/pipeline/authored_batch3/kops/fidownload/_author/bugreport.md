# Bug report

File downloading is broken: existing files are re-downloaded even when their content already matches, downloaded bytes are not hash-verified, cloud-storage URLs (`s3://`, `gs://`, `azureblob://`) are fetched over plain HTTP instead of through the VFS layer, non-2xx HTTP responses are treated as success, and downloaded files land with the wrong permissions or in the wrong place.

Expected: a destination whose hash already matches is left untouched; a `file://` or `https://` URL streams through the HTTP path while `s3://`/`gs://`/`azureblob://` go through the VFS writer; a 404 is an error, not an empty file; the destination file ends up mode `0644` under the requested name.

Reproduce with:

```
go test -count=1 ./upup/pkg/fi/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository.
