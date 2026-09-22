# Exported API — enturls

Enterprise (GHE) clients need API-root and upload URLs derived from the
instance hostname; plain `WithURLs` accepts them verbatim while
`WithEnterpriseURLs` applies the conventional `api/v3` and `api/uploads`
layout.

- `WithURLs(baseURL, uploadURL *string) ClientOptionsFunc` — set both URLs
  verbatim (empty string → error, missing trailing slash → added).
- `WithEnterpriseURLs(baseURL, uploadURL string) ClientOptionsFunc` — same
  parsing plus the enterprise path conventions.
- `parseURL` (unexported, excised body) — shared URL normalizer.
- `Client.BaseURL` / `Client.UploadURL` — consumed by every request builder.
