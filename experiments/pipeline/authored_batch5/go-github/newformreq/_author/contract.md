# Contract — newformreq

`Client.NewFormRequest` builds POST requests for urlencoded form bodies,
gated to destinations the client was configured for. Every commitment below
is covered by a hidden test; every hidden test maps to a commitment.

## Commitments

1. **Untrusted destination refused.** A `urlStr` resolving to a host outside
   the configured origins is rejected with `ErrUntrustedDestination` before
   the form body is posted. Covered by `TestDetail01`.
2. **Configured destination accepted.** A `urlStr` resolving to a configured
   destination — relative, or absolute at the client's own origin — is
   accepted. Covered by `TestDetail02`.
3. **API version header.** `X-Github-Api-Version` carries the client's
   configured default API version on every form request. Covered by
   `TestDetail03`.
4. **User-Agent omitted when empty (shape).** When the client's user-agent
   is empty the header is absent, not present-empty. Covered by
   `TestDetail04`.
5. **Content-Type.** The request's `Content-Type` is the urlencoded form
   media type per the method's documented contract. Covered by
   `TestDetail05`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | doc |
| TestDetail02 | 2 | yes |
| TestDetail03 | 3 | doc |
| TestDetail04 | 4 | partially |
| TestDetail05 | 5 | yes — asserted as the documented urlencoded media type |
