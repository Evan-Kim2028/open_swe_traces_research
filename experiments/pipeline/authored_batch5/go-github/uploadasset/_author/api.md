# Exported API — uploadasset

`RepositoriesService.UploadReleaseAsset(ctx, owner, repo, id, opts,
file)` uploads `*os.File` to a release; `UploadReleaseAssetFromRelease`
does the same starting from a `*RepositoryRelease` and an
`io.Reader`+size pair. Callers rely on:

- Rejecting directories, nil readers, negative sizes, and releases with
  no `UploadURL`.
- Stripping the `{?name,label}` URI-template suffix from
  `release.UploadURL` before use.
- Media type: `opts.MediaType` wins; else inferred from the file's (or
  `opts.Name`'s) extension.
