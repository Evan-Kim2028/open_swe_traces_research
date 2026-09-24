# Closure — uploadasset

Package: github (root). File: github/repos_releases.go.
Removed bodies: UploadReleaseAsset and UploadReleaseAssetFromRelease —
stubbed to variants that drop file validation and media-type inference:
the first loses the `stat.IsDir` rejection, the extension-based
`mime.TypeByExtension` default, and the `opts == nil` guard around
`opts.MediaType`; the second loses the `nil release`/`nil reader`/
negative-`size` guards, the `{?name,label}` URI-template strip, the
leading-`/` normalization for relative upload URLs, and the
`opts.MediaType` → `opts.Name`-extension fallback chain. Request
building and Do keep working.
Kept: NewUploadRequest, ReleaseAsset fields, UploadOptions.
Tests removed: 8 funcs in github/repos_releases_test.go covering the
upload path, the FromRelease helper, and the guard cases.
