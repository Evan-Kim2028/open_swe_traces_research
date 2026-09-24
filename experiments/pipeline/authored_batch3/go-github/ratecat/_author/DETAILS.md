# DETAILS — ratecat

1. `/search/code` GET requests are CodeSearch; other `/search/` requests are
   Search regardless of method. Inferable: partially — the category set is
   exported, the path routing is an API convention.
2. `/graphql` is its own category. Inferable: partially.
3. POSTs to `/app-manifests/*/conversions` are IntegrationManifest; PUTs to
   `/repos/*​/import` are SourceImport. Inferable: no — method+suffix
   pairs are arbitrary endpoint conventions.
4. Any path ending `/code-scanning/sarifs` is CodeScanningUpload; `/scim/`
   prefixed paths are Scim; POSTs ending `/dependency-graph/snapshots` are
   DependencySnapshots. Inferable: no.
5. Any path ending `/audit-log` is AuditLog, and paths ending
   `/dependency-graph/sbom` or `/dependency-graph/sbom/generate-report`
   under `/repos/` are DependencySBOM. Inferable: no.
6. Everything else — including runner-registration endpoints — falls
   through to Core. Inferable: yes — the default bucket.
7. Classification is a pure function of method and path; no client state is
   consulted. Inferable: yes — signature has no receiver.
