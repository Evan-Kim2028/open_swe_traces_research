# Contract (L2) — jsonrenders

All JSON renderers write 'application/json; charset=utf-8' unless the header is already set. WriteJSON marshals and writes the bytes. IndentedJSON marshals with a 4-space indent. SecureJSON writes its prefix only when the marshaled top-level value is a JSON array ([...]) — objects and scalars get no prefix. JsonpJSON uses content type 'application/javascript; charset=utf-8': with an empty callback it writes the raw JSON; otherwise it writes JSEscapeString(callback) + '(' + json + ');'. AsciiJSON uses bare 'application/json' (no charset) and rewrites every non-ASCII rune in the marshaled output as a \uXXXX escape (lowercase hex), leaving ASCII bytes untouched. PureJSON streams through an Encoder with HTML escaping disabled (so <, >, & stay literal and the output ends with the encoder's newline). Marshal/encode errors propagate and happen after the content type is set.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestRenderJSON/TestRenderJSONError` | plain marshal+write with the right content type; marshal errors propagate |
| `TestRenderIndentedJSON(+Panics)` | 4-space-indented output |
| `TestRenderSecureJSON(+Fail)` | prefix emitted only for top-level arrays |
| `TestRenderJsonpJSON(+Error,+Error2,+Fail)` | callback escaping, '(' json ');' wrapping, empty-callback passthrough, javascript content type |
| `TestRenderAsciiJSON(+Fail)` | non-ASCII runes become \uXXXX escapes; charset-less content type |
| `TestRenderPureJSON` | encoder path with HTML escaping off and trailing newline |
