# Closure — credhdrs

Package: github (root). File: github/github.go.
Removed bodies: setCredentialsAsHeaders — stubbed to clone the request without
attaching credentials (drops the SetBasicAuth call).
Kept: request copying, transports, origin-scope plumbing (sameOrigin etc.
belong to the urlpolicy unit).
Tests removed: 6 funcs in github/github_test.go — TestBasicAuthTransport,
TestBasicAuthTransport_originScope, TestBasicAuthTransport_transport,
TestSetCredentialsAsHeaders, TestUnauthenticatedRateLimitedTransport,
TestUnauthenticatedRateLimitedTransport_originScope. The two *_originScope
tests are also deleted by the urlpolicy unit (shared helper coverage).
