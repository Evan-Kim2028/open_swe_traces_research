# Exported API — downloadcontents

`RepositoriesService.DownloadContents` /
`DownloadContentsWithMeta(ctx, owner, repo, path, opts)` fetch a file's
bytes. The API's `download_url` is cross-origin by design (raw content
hosts, pre-signed CDNs), so the fetch goes through the client's own
`http.Client` — credentials attach only to configured origins — rather
than being refused as a foreign destination. Callers rely on:

- `(nil, meta, resp, ErrContentsNoDownloadURL)` when the API supplies no
  download URL and no inline content.
- `(body, meta, resp, nil)` with `body` already open on success.
- The download HTTP response exposed via the returned `*Response`.
