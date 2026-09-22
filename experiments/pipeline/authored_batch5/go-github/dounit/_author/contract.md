# Contract — dounit

`Client.Do` decodes the API response into the target value, tolerating
empty bodies, streaming to `io.Writer` targets, and surfacing read/decode
failures. Every commitment below is covered by a hidden test; every hidden
test maps to a commitment.

## Commitments

1. **Empty/whitespace body tolerated.** A response body that is empty or
   contains only whitespace produces no decode error for a non-nil,
   non-`io.Writer` target — `Do` returns `err == nil`. Covered by
   `TestDetail01`.
2. **Decode errors surface.** A non-empty body that fails JSON decode
   surfaces the decode error. Covered by `TestDetail02`.
3. **Read errors surface.** A body that cannot be read (transport error
   mid-body) surfaces the read error. Covered by `TestDetail03`.
4. **Raw bytes to writers.** `io.Writer` targets receive the raw body
   bytes with no JSON decode. Covered by `TestDetail04`.
5. **Nil target.** `v == nil` performs no decode at all — even an
   undecodable body is not an error. Covered by `TestDetail05`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | partially |
| TestDetail02 | 2 | yes |
| TestDetail03 | 3 | yes |
| TestDetail04 | 4 | doc |
| TestDetail05 | 5 | doc |
