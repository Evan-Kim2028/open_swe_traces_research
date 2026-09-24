# Closure — gethdr

Package: github (root). File: github/repos_hooks_deliveries.go.
Removed body: getHeader — stubbed to exact-match map access.
Kept: HookDelivery, HookRequest, HookResponse types and both GetHeader
methods (they delegate to getHeader).
Tests removed: TestHookRequest_GetHeader, TestHookResponse_GetHeader in
github/repos_hooks_deliveries_test.go.
