# Closure — dounit

Package: github (root). File: github/github.go.
Removed bodies: Client.Do — stubbed to a streaming decoder variant:
`io.Writer` sinks get a raw copy, any other non-nil v gets
`json.NewDecoder(resp.Body).Decode(v)` inline. Drops the pooled-buffer
read-then-decode path and, with it, the rule that a whitespace-only body
is not an error. Every response the endpoint leaves blank or whitespace
(`200 {}` handlers, comment-delete, etc.) now surfaces `EOF`/`unexpected
end of JSON input`.
Kept: BareDo/bareDo plumbing, Response construction, all callers.
Tests removed: 1 func in github/github_test.go
(TestDo_whitespaceOnlyBody).
