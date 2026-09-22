# Closure — downloadasset

Package: github (root). File: github/repos_releases.go.
Removed bodies: DownloadReleaseAsset and downloadReleaseAssetFromURL —
stubbed to variants that stop at the redirect URL: the first always
returns `loc.String()` (never follows it, even when a
`followRedirectsClient` was supplied), and the helper returns the
redirected response body without `CheckResponse` validation or the
original-body close on error. Direct (non-redirect) downloads still
stream `resp.Body`.
Kept: GetReleaseAsset, the API request plumbing, Response.
Tests removed: 4 funcs in github/repos_releases_test.go
(TestRepositoriesService_DownloadReleaseAsset_FollowRedirect,
_FollowMultipleRedirects, _FollowRedirectToError,
_FollowRedirectToErrorClosesOriginalBody).
