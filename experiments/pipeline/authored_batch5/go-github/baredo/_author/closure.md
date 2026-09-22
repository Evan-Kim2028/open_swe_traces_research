# Closure — baredo

Package: github (root). File: github/github.go.
Removed bodies: Client.bareDo — stubbed to a minimal send: one
`client.Do(req)`, `newResponse` wrap, `CheckResponse`, return. Drops the
rate-limit read/update around the call, the per-category limiter,
accepted-error body capture (`AcceptedError.Raw`), URL sanitization for
error messages, context-cancellation mapping, retry-on-secondary-limit
sleep accounting, and the original-body close on error paths. Response
construction and CheckResponse keep working; all rate bookkeeping,
error metadata, and cancellation semantics are gone.
Kept: Client.Do, newResponse, CheckResponse, Rate fields, sanitizeURL,
GetRateLimitCategory, checkRateLimitBeforeDo /
checkSecondaryRateLimitBeforeDo, rate-limit structs.
Tests removed: 26 funcs across github/github_test.go and
github/code-scanning_test.go covering rate limits, accepted errors,
sanitization, retries, cancellation, body closing, and the sarif upload
caller.
