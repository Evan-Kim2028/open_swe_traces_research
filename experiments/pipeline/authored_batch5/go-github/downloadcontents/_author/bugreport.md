# Bug report — downloadcontents

`DownloadContents` / `DownloadContentsWithMeta` always fail for real
files: every file path returns `ErrContentsNoDownloadURL` even when the
API supplied a `download_url`, so downloading any file through these
methods is impossible.

Expected: the API-provided `download_url` is fetched through the
client's credential-scoped HTTP client and the body streamed back;
`ErrContentsNoDownloadURL` only when the API gave no URL and no inline
content.

Got: `ErrContentsNoDownloadURL` unconditionally for files.

Reproduce with:

```
go test -count=1 ./...
```

Do not use web search or any tool that accesses the internet; work only from the repository.
