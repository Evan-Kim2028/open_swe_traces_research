# Bug report — uploadasset

`UploadReleaseAsset` / `UploadReleaseAssetFromRelease` accept inputs
they should reject: a directory uploads as if it were a file, nil
readers and negative sizes sail through, and a release with no
`UploadURL` produces a request anyway. The `{?name,label}` template in
`release.UploadURL` is sent verbatim, breaking the URL, and no media
type is inferred from the file's extension.

Expected: invalid inputs rejected, the template stripped, media type
inferred from `opts.MediaType` or the file/name extension.

Got: no guards, raw template URL, no media-type inference.

Reproduce with:

```
go test -count=1 ./...
```

Do not use web search or any tool that accesses the internet; work only from the repository.
