# Exported API — fetchsbom

`DependencyGraphService.FetchSBOM(ctx, owner, repo, sbomUUID,
followRedirectsClient)` retrieves a generated SBOM. The API responds
with a redirect to a pre-signed download URL:

- `followRedirectsClient == nil`: `redirectURL` is returned for the
  caller.
- Non-nil: the URL is fetched through that client — deliberately not
  `s.client`, because the pre-signed host rejects authenticated
  requests — the response is validated, decoded as `SBOMInfo`, and
  wrapped in `*SBOM`.
- Non-redirect API responses are an error; the original network body is
  always closed.
