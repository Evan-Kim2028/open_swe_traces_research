# Closure — newformreq

Package: github (root). File: github/github.go.
Removed bodies: Client.NewFormRequest — stubbed to a variant that always
sets `User-Agent: c.userAgent` (even when empty), never sets
`X-Github-Api-Version`, and skips the `checkBodyDestination` gate on the
resolved form URL. Multipart body creation, Accept, and Content-Type
keep working.
Kept: NewRequest, NewUploadRequest, checkBodyDestination (still used
elsewhere), RequestOption.
Tests removed: 5 funcs in github/github_test.go
(TestNewFormRequest_RejectsForeignFormDestination,
TestNewFormRequest_RejectsBodyDestination,
TestNewFormRequest_bodyDestination, TestNewFormRequest,
TestNewRequest_emptyUserAgent).
