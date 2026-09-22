# Closure — errfmt

Package: github (root). File: github/github.go.
Removed bodies: Error() of ErrorResponse, RateLimitError, AbuseRateLimitError,
RedirectionError, AcceptedError, Error; Error.UnmarshalJSON; sanitizeURL;
formatRateReset — each stubbed to a degenerate format.
Kept: all error types, CheckResponse, sensitiveParams list.
Tests removed: 11 funcs in github/github_test.go — TestErrorResponse_Error,
TestError_Error, TestError_UnmarshalJSON, TestSanitizeURL, TestDo_sanitizeURL,
TestFormatRateReset, TestTwoFactorAuthError, TestRateLimitError,
TestAcceptedError, TestAbuseRateLimitError,
TestCheckResponse_unexpectedErrorStructure (shared with respclassify/erris).
