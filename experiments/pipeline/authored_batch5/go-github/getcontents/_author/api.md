# Exported API — getcontents

`RepositoriesService.GetContents(ctx, owner, repo, path, opts)` returns
either `*RepositoryContent` (a file) or `[]*RepositoryContent` (a
directory listing) for the same endpoint. Callers rely on:

- `path` escaped as a URL path segment: spaces, `+`, `..`, and a
  trailing `/` normalized away.
- A file response populating `fileContent`; an array response
  populating `directoryContent`; only one non-nil.
- An un-decodable body surfacing a combined unmarshal error.
