# Contract — downloadasset

`RepositoriesService.DownloadReleaseAsset` either returns the asset stream
directly or resolves a redirect — returning the URL, or fetching it through
a caller-supplied client. Every commitment below is covered by a hidden
test; every hidden test maps to a commitment.

## Commitments

1. **Direct response.** When the API responds without a redirect, the
   response body is returned directly as the stream. Covered by
   `TestDetail01`.
2. **Followed redirect.** On redirect with a non-nil
   `followRedirectsClient`, the redirect target is fetched through that
   client — not the API client — and the fetched body is returned with an
   empty `redirectURL`. Covered by `TestDetail02`.
3. **Returned redirect.** On redirect with a nil `followRedirectsClient`,
   `redirectURL` is returned and no fetch occurs. Covered by
   `TestDetail03`.
4. **Redirect fetch validated.** The redirect-fetch response is validated:
   an error status propagates as an error and no body is returned; a
   success returns its body. Covered by `TestDetail04`.
5. **Original body closed.** When a redirect is followed, the original API
   response body is closed first. Covered by `TestDetail05`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | yes |
| TestDetail02 | 2 | doc |
| TestDetail03 | 3 | yes |
| TestDetail04 | 4 | partially |
| TestDetail05 | 5 | partially |
