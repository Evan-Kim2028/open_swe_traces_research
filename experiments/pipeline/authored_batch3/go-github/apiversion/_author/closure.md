# Closure — apiversion

Package: github (root). File: github/github.go.
Removed bodies: Client.checkRequestAPIVersionBeforeDo — stubbed to `return nil`
(accepts every version).
Kept: bareDo call site, ErrUnsupportedAPIVersion sentinel, api20221128 /
api20260310 constants, apiVersionMin/Max/Default fields and option plumbing.
Tests removed: 2 funcs in github/github_test.go
(TestClient_checkRequestAPIVersionBeforeDo,
TestClient_bareDo_errors_with_unsupported_api_version).
