# Closure — downloadcontents

Package: github (root). File: github/repos_contents.go.
Removed bodies: DownloadContentsWithMeta — stubbed to return
`ErrContentsNoDownloadURL` for every file, dropping the
`fileContent.DownloadURL` fetch through `s.client.client` (the
credential-scoped HTTP client): a plain GET, error propagation with the
raw `*http.Response` wrapped, and the body handed to the caller.
Directory/submodule early-outs and the inline `Content` fallback keep
working.
Kept: GetContents, DownloadContents thin wrapper, ErrContentsNoDownloadURL,
RepositoryContent fields.
Tests removed: 4 funcs in github/repos_contents_test.go
(TestRepositoriesService_DownloadContents_SuccessByDownload,
_FailedDownloadResponse, and the same pair for WithMeta).
