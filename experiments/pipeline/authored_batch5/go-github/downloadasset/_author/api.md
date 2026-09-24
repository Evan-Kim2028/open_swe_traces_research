# Exported API — downloadasset

`RepositoriesService.DownloadReleaseAsset(ctx, owner, repo, id,
followRedirectsClient)` returns either the asset's `io.ReadCloser`
stream or its redirect URL:

- No redirect: the API response body is streamed.
- Redirect + `followRedirectsClient != nil`: the redirect target is
  fetched through that (separate, credential-free) client and its
  validated body is returned; `redirectURL` is empty.
- Redirect + nil client: `redirectURL` is returned for the caller.
- Error responses on the redirect fetch surface as errors with the
  original body closed.
