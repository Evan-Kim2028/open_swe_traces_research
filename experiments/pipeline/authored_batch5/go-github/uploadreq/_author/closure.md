# Closure — uploadreq

Package: github (root). File: github/github.go.
Removed bodies: Client.NewUploadRequest — stubbed to a variant that
resolves `urlStr` against uploadURL with no `..` traversal check and no
destination-origin gate, hands the caller's `reader` to
`http.NewRequestWithContext` unwrapped, and never sets `GetBody`.
Path traversal, foreign-host upload destinations, the concrete-reader
hiding wrapper, and the seeker+readerAt GetBody are all excised; the
trailing-slash check, ContentLength, mediaType defaulting, and opts loop
keep working.
Kept: NewRequest, NewFormRequest, checkURLPathTraversal,
checkBodyDestination (still used by NewFormRequest),
uploadRequestBodyReader type, RequestOption.
Tests removed: 6 funcs in github/github_test.go (TestNewUploadRequest_*)
plus 1 in github/repos_releases_test.go
(TestRepositoriesService_UploadReleaseAssetFromRelease_ForeignHostIsRejected).
