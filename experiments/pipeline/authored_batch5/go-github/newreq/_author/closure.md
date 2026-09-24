# Closure — newreq

Package: github (root). File: github/github.go.
Removed bodies: Client.NewRequest — stubbed to a variant that always sets
`User-Agent: c.userAgent` (even when empty) and never sets the
`X-Github-Api-Version` header. Keeps baseURL trailing-slash check,
path-traversal check, JSON body encode, Accept/Content-Type headers, and
the RequestOption application loop, so every NewRequest consumer keeps
working; only the two header details are excised.
Kept: NewFormRequest, NewUploadRequest, BareDo/Do, all Client fields.
Tests removed: 2 funcs in github/github_test.go (TestNewRequest,
TestNewRequest_emptyUserAgent).
