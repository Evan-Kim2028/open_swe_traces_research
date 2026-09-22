# Closure — urlpolicy

Package: github (root). File: github/github.go.
Removed bodies: checkURLPathTraversal (no-op), sameOrigin (exact
case-sensitive host compare), normalizedPort (no scheme defaults),
isAllowedOrigin (no default list), shouldAuthorizeRequest (base origin
only), checkBodyDestination (no-op).
Kept: ErrPathForbidden, ErrUntrustedDestination, defaultAuthOrigins,
NewRequest/NewFormRequest/NewUploadRequest call sites.
Tests removed: 14 funcs in github/github_test.go (traversal, origin,
port-normalization, token-scope, unconfigured-destination tests) plus
TestRepositoriesService_UploadReleaseAssetFromRelease_ForeignHostIsRejected
in github/repos_releases_test.go.
