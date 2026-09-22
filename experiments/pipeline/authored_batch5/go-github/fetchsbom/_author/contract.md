# Contract — fetchsbom

`DependencyGraphService.FetchSBOM` resolves an SBOM fetch-report redirect —
returning the URL to the caller, or fetching it through a caller-supplied
client and decoding the report. Every commitment below is covered by a
hidden test; every hidden test maps to a commitment.

## Commitments

1. **Followed redirect.** With a non-nil `followRedirectsClient`, the
   redirect target is fetched through that client — not the API client —
   and the decoded `*SBOM` is returned with an empty `redirectURL`.
   Covered by `TestDetail01`.
2. **Returned redirect.** With a nil `followRedirectsClient`,
   `redirectURL` is returned and no fetch occurs. Covered by
   `TestDetail02`.
3. **Redirect fetch validated.** An error HTTP status on the redirect
   fetch propagates as an error rather than decoding the error page —
   asserted with an error page that would otherwise decode cleanly.
   Covered by `TestDetail03`.
4. **Bodies closed on every path.** The original API response body and the
   downloaded body are closed on both the success and error paths.
   Covered by `TestDetail04`.
5. **Payload decodes.** The downloaded payload decodes as a populated
   SBOM — asserted on populated fields, not the wrapper shape. Covered by
   `TestDetail05`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | doc |
| TestDetail02 | 2 | yes |
| TestDetail03 | 3 | partially |
| TestDetail04 | 4 | yes |
| TestDetail05 | 5 | partially |
