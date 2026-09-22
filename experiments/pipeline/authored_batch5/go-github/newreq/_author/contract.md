# Contract — newreq

`Client.NewRequest` builds API requests carrying the client's configured
headers. Every commitment below is covered by a hidden test; every hidden
test maps to a commitment.

## Commitments

1. **API version header.** Every request carries `X-Github-Api-Version`
   whose value is the client's configured default API version — the
   configured value is asserted, not a literal. Covered by `TestDetail01`.
2. **User-Agent omitted when empty (shape).** When the client's user-agent
   string is empty, the `User-Agent` header is absent from the request
   rather than present with an empty value. Covered by `TestDetail02`.
3. **User-Agent verbatim.** A non-empty user-agent is sent verbatim.
   Covered by `TestDetail03`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | doc |
| TestDetail02 | 2 | partially |
| TestDetail03 | 3 | yes |
