# Exported API — ratecat

`GetRateLimitCategory(method, path)` classifies a REST endpoint into the
rate-limit bucket GitHub meters it under, so `Response.Rate` can be read
against the right pool.

- `GetRateLimitCategory` — exported classifier used by `Client.Do` before
  and after each request.
- `RateLimitCategory` constants (Core, Search, CodeSearch, Graphql,
  IntegrationManifest, SourceImport, CodeScanningUpload, Scim,
  DependencySnapshots, AuditLog, DependencySBOM) — kept, define the
  vocabulary.
