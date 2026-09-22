# Closure — enturls

Package: github (root). Files: github/url.go, github/github.go.
Removed bodies: parseURL — stubbed to bare url.Parse (drops empty-check,
error wrapping, trailing-slash). WithEnterpriseURLs — stubbed to plain parse
of both URLs (drops the api/v3 and api/uploads path conventions).
Kept: url.Parse usage, option plumbing, WithURLs wrapper path.
Tests removed: github/github_test.go — TestWithEnterpriseURLs, TestWithURLs,
TestClientCopy_leak_transport, TestClient_CloneReScopesToken,
TestClient_tokenNotForwardedOnCrossOriginRedirect, TestClient_tokenOriginScope
(shared with urlpolicy); github/url_test.go — Test_parseURL.
