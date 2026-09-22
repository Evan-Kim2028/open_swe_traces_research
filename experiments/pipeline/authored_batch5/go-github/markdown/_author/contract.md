# Contract — markdown

`MarkdownService.Render` posts text (with optional mode/context) to the
markdown endpoint and returns the rendered HTML body. Every commitment below
is covered by a hidden test; every hidden test maps to a commitment.

## Commitments

1. **Mode passed through.** A non-empty `opts.Mode` is sent as the request's
   `mode` field. Covered by `TestDetail01`.
2. **Context passed through.** A non-empty `opts.Context` is sent as the
   request's `context` field. Covered by `TestDetail02`.
3. **Empty fields omitted (shape).** Empty option fields are not sent as
   empty strings, and `opts == nil` sends only `text`. Covered by
   `TestDetail03`.
4. **Rendered body returned.** The rendered HTML response body is returned
   as a string. Covered by `TestDetail04`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | doc |
| TestDetail02 | 2 | doc |
| TestDetail03 | 3 | partially |
| TestDetail04 | 4 | yes |
